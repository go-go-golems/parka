package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// TODO(manuel, 2023-06-21) This part of the API is a complete mess, I'm not even sure what it is supposed to do overall
// Well worth refactoring

type LayerParseState struct {
	Slug string
	// Defaults contains the default values for the parameters, as strings to be parsed
	// NOTE(manuel, 2023-06-21) Why are these strings?
	// See also https://github.com/go-go-golems/glazed/issues/239
	Defaults map[string]string
	// Parameters contains the parsed parameters so far
	Parameters map[string]interface{}
	// ParameterDefinitions contains the parameter definitions that can still be parsed
	ParameterDefinitions map[string]*parameters.ParameterDefinition
}

type ParseState struct {
	FlagsAndArguments *LayerParseState
	Layers            map[string]*LayerParseState
}

func NewParseStateFromCommandDescription(d *cmds.CommandDescription) *ParseState {
	ret := &ParseState{
		FlagsAndArguments: &LayerParseState{
			Defaults:             map[string]string{},
			Parameters:           map[string]interface{}{},
			ParameterDefinitions: map[string]*parameters.ParameterDefinition{},
		},
		Layers: map[string]*LayerParseState{},
	}

	for _, p := range d.Flags {
		ret.FlagsAndArguments.ParameterDefinitions[p.Name] = p
	}
	for _, p := range d.Arguments {
		ret.FlagsAndArguments.ParameterDefinitions[p.Name] = p
	}

	for _, l := range d.Layers {
		ret.Layers[l.GetSlug()] = &LayerParseState{
			Slug:                 l.GetSlug(),
			Defaults:             map[string]string{},
			Parameters:           map[string]interface{}{},
			ParameterDefinitions: map[string]*parameters.ParameterDefinition{},
		}

		for _, p := range l.GetParameterDefinitions() {
			ret.Layers[l.GetSlug()].ParameterDefinitions[p.Name] = p
		}
	}

	return ret
}

// ParseStep is used to parse parameters out of a gin.Context (meaning most certainly out of an incoming *http.Request).
// These parsed parameters are stored in the ParseState structure.
// A ParseStep can only parse parameters that are given in the ParameterDefinitions field of the ParseState.
type ParseStep interface {
	Parse(c *gin.Context, result *LayerParseState) error
}

// Parser is contains a list of ParserFunc that are used to parse an incoming
// request into a proper CommandContext, and ultimately be used to Run a glazed Command.
//
// These ParserFunc can be operating on the general parameters as well as per layer.
// The flexibility is there so that more complicated commands can ultimately be built that leverage
// different validations and rewrite rules.
//
// NOTE(manuel, 2023-04-16) I wonder when I will queue multiple ParserFunc and LayerParser Func.
// We might actually already do this by leveraging it to overwrite layer parameters (say, sqleton
// connection parameters).
type Parser struct {
	Parsers            []ParseStep
	LayerParsersBySlug map[string][]ParseStep
}

type ParserOption func(*Parser)

func NewParser(options ...ParserOption) *Parser {
	ph := &Parser{
		Parsers:            []ParseStep{},
		LayerParsersBySlug: map[string][]ParseStep{},
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

func (p *Parser) Parse(c *gin.Context, state *ParseState) error {
	for _, parser := range p.Parsers {
		if err := parser.Parse(c, state.FlagsAndArguments); err != nil {
			return err
		}
	}

	for _, layer := range state.Layers {
		for _, parser := range p.LayerParsersBySlug[layer.Slug] {
			if err := parser.Parse(c, layer); err != nil {
				return err
			}
		}
	}

	// NOTE(manuel, 2023-06-21) We might have to copy each layer's parsed parameters back to the main Parameters
	// either here or further down the road when calling the glazed command since many commands might still
	// rely on getting layer specific flags in ps[] itself.

	return nil
}

// WithPrependParser adds the given ParserFunc to the beginning of the list of parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithPrependParser(ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = append(ps, ph.Parsers...)
	}
}

// WithAppendParser adds the given ParserFunc to the end of the list of parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithAppendParser(ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = append(ph.Parsers, ps...)
	}
}

// WithReplaceParser replaces the list of parsers with the given ParserFunc.
// This will remove all previously added prepend, replace, append parsers.
func WithReplaceParser(ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = ps
	}
}

// WithPrependLayerParser adds the given ParserFunc to the beginning of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceLayerParser.
func WithPrependLayerParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParseStep{}
		}
		ph.LayerParsersBySlug[slug] = append(ps, ph.LayerParsersBySlug[slug]...)
	}
}

// WithAppendLayerParser adds the given ParserFunc to the end of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceLayerParser.
func WithAppendLayerParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParseStep{}
		}
		ph.LayerParsersBySlug[slug] = append(ph.LayerParsersBySlug[slug], ps...)
	}
}

// WithReplaceLayerParser replaces the list of layer parsers with the given ParserFunc.
func WithReplaceLayerParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		ph.LayerParsersBySlug[slug] = ps
	}
}

// WithGlazeOutputParserOption is a convenience function to override the output and table format glazed settings.
func WithGlazeOutputParserOption(output string, tableFormat string) ParserOption {
	return WithAppendLayerParser(
		"glazed",
		NewStaticParseStep(map[string]interface{}{
			"output":       output,
			"table-format": tableFormat,
		}),
	)
}

// WithReplaceStaticLayer is a convenience function to use static layer parsing.
// This entirely replaces current layer parsers, but can later on be amended with other parsers,
// for example with WithAppendOverrideLayer.
func WithReplaceStaticLayer(slug string, overrides map[string]interface{}) ParserOption {
	return WithReplaceLayerParser(
		slug,
		NewStaticParseStep(overrides),
	)
}

// WithAppendOverrideLayer is a convenience function to override the parameters of a layer.
// The overrides are appended past currently present parser functions.
func WithAppendOverrideLayer(slug string, overrides map[string]interface{}) ParserOption {
	return WithAppendLayerParser(
		slug,
		NewStaticParseStep(overrides),
	)
}
