package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// TODO(manuel, 2023-06-21) This part of the API is a complete mess, I'm not even sure what it is supposed to do overall
// Well worth refactoring

// DefaultSlug is used as a drop-in to signal that we actually want to parse the
// top-level flags and arguments.
//
// # TODO(manuel, 2023-06-22) This should be removed once we actually turn default flags and arguments into an actual layer
//
// See https://github.com/go-go-golems/glazed/issues/303
const DefaultSlug = "default"

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

// Parser is contains a list of ParseStep that are used to parse an incoming
// request into a proper CommandContext, and ultimately be used to Run a glazed Command.
//
// These ParseStep can be operating on the general parameters as well as per layer.
// The flexibility is there so that more complicated commands can ultimately be built that leverage
// different validations and rewrite rules.
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

// WithPrependParser adds the given ParserFunc to the beginning of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithPrependParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		if slug == DefaultSlug {
			ph.Parsers = append(ps, ph.Parsers...)
			return
		}

		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParseStep{}
		}
		ph.LayerParsersBySlug[slug] = append(ps, ph.LayerParsersBySlug[slug]...)
	}
}

// WithAppendParser adds the given ParserFunc to the end of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithAppendParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		if slug == DefaultSlug {
			ph.Parsers = append(ph.Parsers, ps...)
			return
		}

		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParseStep{}
		}
		ph.LayerParsersBySlug[slug] = append(ph.LayerParsersBySlug[slug], ps...)
	}
}

// WithReplaceParser replaces the list of layer parsers with the given ParserFunc.
func WithReplaceParser(slug string, ps ...ParseStep) ParserOption {
	return func(ph *Parser) {
		if slug == DefaultSlug {
			ph.Parsers = ps
			return
		}

		ph.LayerParsersBySlug[slug] = ps
	}
}

// WithGlazeOutputParserOption is a convenience function to override the output and table format glazed settings.
func WithGlazeOutputParserOption(output string, tableFormat string) ParserOption {
	return WithAppendParser(
		"glazed",
		NewStaticParseStep(map[string]interface{}{
			"output":       output,
			"table-format": tableFormat,
		}),
	)
}

// WithReplaceParameters is a convenience function to use static layer parsing.
// This entirely replaces current layer parsers, but can later on be amended with other parsers,
// for example with WithAppendOverrides.
//
// Note that this also replaces the defaults
func WithReplaceParameters(slug string, overrides map[string]interface{}) ParserOption {
	return WithReplaceParser(
		slug,
		NewStaticParseStep(overrides),
	)
}

// WithAppendOverrides is a convenience function to override the parameters of a layer.
// The overrides are appended past currently present parser functions.
func WithAppendOverrides(slug string, overrides map[string]interface{}) ParserOption {
	return WithAppendParser(
		slug,
		NewStaticParseStep(overrides),
	)
}

// WithPrependDefaults is a convenience function to set the initial parameters of a layer.
// If a value is already set, it won't be overwritten.
func WithPrependDefaults(slug string, defaults map[string]interface{}) ParserOption {
	return WithPrependParser(
		slug,
		NewDefaultParseStep(defaults),
	)
}

// WithStopParsing will stop parsing parameters, even if further parser steps are added at the end
// of the parser chain. This can be used to "seal" parsing and prevent further parameters from being
// overridden.
func WithStopParsing(slug string) ParserOption {
	return WithAppendParser(
		slug,
		NewStopParseStep(),
	)
}
