package glazed

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
)

// CommandContext keeps the context for execution of a glazed command,
// and can be worked upon by CommandHandlerFunc.
//
// A CommandContext is progressively built up from the query by passing through a list of registered
// ParkaHandlerFuncs. These handler functions are registered to a specific route.
type CommandContext struct {
	// Cmd is the command that will be executed
	Cmd cmds.GlazeCommand
	// ParsedLayers contains the map of parsed layers parsed so far
	ParsedLayers map[string]*layers.ParsedParameterLayer
	// ParsedParameters contains the map of parsed parameters parsed so far
	ParsedParameters map[string]interface{}
}

// NewCommandContext creates a new CommandContext for the given command.
func NewCommandContext(cmd cmds.GlazeCommand) *CommandContext {
	return &CommandContext{
		Cmd:              cmd,
		ParsedLayers:     map[string]*layers.ParsedParameterLayer{},
		ParsedParameters: map[string]interface{}{},
	}
}

// GetAllParameterDefinitions returns a map of all parameter definitions for the command.
// This includes flags, arguments and all layers.
func (pc *CommandContext) GetAllParameterDefinitions() []*parameters.ParameterDefinition {
	description := pc.Cmd.Description()

	ret := pc.GetFlagsAndArgumentsParameterDefinitions()

	for _, l := range description.Layers {
		for _, p := range l.GetParameterDefinitions() {
			ret = append(ret, p)
		}
	}

	return ret
}

func (pc *CommandContext) GetFlagsAndArgumentsParameterDefinitions() []*parameters.ParameterDefinition {
	ret := []*parameters.ParameterDefinition{}

	description := pc.Cmd.Description()

	ret = append(ret, description.Flags...)
	ret = append(ret, description.Arguments...)

	return ret
}

func (pc *CommandContext) GetAllParameterValues() map[string]interface{} {
	ret := map[string]interface{}{}

	for k, v := range pc.ParsedParameters {
		ret[k] = v
	}

	for _, l := range pc.ParsedLayers {
		for k, v := range l.Parameters {
			ret[k] = v
		}
	}

	return ret
}

// CommandHandlerFunc mirrors gin's HandlerFunc, but also gets passed a CommandContext.
// That allows it to reuse data from the gin.Context, most importantly the request itself.
type CommandHandlerFunc func(*gin.Context, *CommandContext) error

// HandlePrepopulatedParameters sets the given parameters in the CommandContext's ParsedParameters.
// If any of the given parameters also belong to a layer, they are also set there.
func HandlePrepopulatedParameters(ps map[string]interface{}) CommandHandlerFunc {
	return func(c *gin.Context, pc *CommandContext) error {
		for k, v := range ps {
			pc.ParsedParameters[k] = v

			// Now check if the parameter is in any of the layers, and if so, set it there as well
			for _, layer := range pc.ParsedLayers {
				if _, ok := layer.Parameters[k]; ok {
					layer.Parameters[k] = v
				}
			}
		}
		return nil
	}
}

// HandlePrepopulatedParsedLayers sets the given layers in the CommandContext's Layers,
// overriding the parameters of any layers that are already present.
// This means that if a parameter is not set in layers_ but is set in the Layers,
// the value in the Layers will be kept.
func HandlePrepopulatedParsedLayers(layers_ map[string]*layers.ParsedParameterLayer) CommandHandlerFunc {
	return func(c *gin.Context, pc *CommandContext) error {
		for k, v := range layers_ {
			parsedLayer, ok := pc.ParsedLayers[k]
			if ok {
				for k2, v2 := range v.Parameters {
					parsedLayer.Parameters[k2] = v2
				}
			} else {
				pc.ParsedLayers[k] = v
			}
		}
		return nil
	}
}

func NewCommandQueryParser(cmd cmds.GlazeCommand, options ...parser.ParserOption) *parser.Parser {
	d := cmd.Description()

	// NOTE(manuel, 2023-06-21) We could pass the parser options here, but then we wouldn't be able to
	// override the layer parser. Or we could pass the QueryParsers right here.
	ph := parser.NewParser()
	ph.Parsers = []parser.ParseStep{
		parser.NewQueryParseStep(false),
	}

	// NOTE(manuel, 2023-04-16) API design: we would probably like to hide layers right here in the handler constructor
	for _, l := range d.Layers {
		slug := l.GetSlug()
		ph.LayerParsersBySlug[slug] = []parser.ParseStep{
			parser.NewQueryParseStep(false),
		}
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

func NewCommandFormParser(cmd cmds.GlazeCommand, options ...parser.ParserOption) *parser.Parser {
	d := cmd.Description()

	ph := parser.NewParser()
	ph.Parsers = []parser.ParseStep{
		parser.NewFormParseStep(false),
	}

	// TODO(manuel, 2023-06-21) This is probably not necessary if the FormParseStep handles layers by itself
	for _, l := range d.Layers {
		slug := l.GetSlug()
		ph.LayerParsersBySlug[slug] = []parser.ParseStep{
			parser.NewFormParseStep(false),
		}
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

// NewCommandHandlerFunc creates a CommandHandlerFunc using the given Parser struct.
// This first establishes a set of defaults by loading them from an alias definition.
//
// When the CommandHandler is invoked, we first gather all the parameterDefinitions from the
// cmd (fresh on every invocation, because the parsers are allowed to modify them).
func NewCommandHandlerFunc(cmd cmds.GlazeCommand, parserHandler *parser.Parser) CommandHandlerFunc {
	d := cmd.Description()

	defaults := map[string]string{}

	// check if we are an alias
	alias_, ok := cmd.(*alias.CommandAlias)
	if ok {
		defaults = alias_.Flags
		for idx, v := range alias_.Arguments {
			if len(d.Arguments) <= idx {
				defaults[d.Arguments[idx].Name] = v
			}
		}
	}

	// TODO(manuel, 2023-05-25) This is where we should handle default values provided from the config file
	//
	// See https://github.com/go-go-golems/sqleton/issues/161
	//
	// We should clearly establish a precedence scheme, something like:
	// - alias defaults (loaded from repository)
	// - overwritten by defaults set in code
	// - overwritten by defaults set from config file
	//
	// See also https://github.com/go-go-golems/glazed/issues/139
	//
	// ## hack notes
	//
	// I think that parser handler could actually override / fill out the defaults here,
	// since we just pass the map around. That's probably not the smart way to do it though,
	// and would warrant revisiting.
	//
	// Actually, the parsers return updated ParameterDefinitions, which means that we should be
	// able to override the defaults in those directly.
	var err error

	return func(c *gin.Context, pc *CommandContext) error {
		parseState := parser.NewParseStateFromCommandDescription(d)
		err = parserHandler.Parse(c, parseState)
		if err != nil {
			return err
		}

		pc.ParsedParameters = parseState.FlagsAndArguments.Parameters
		pc.ParsedLayers = map[string]*layers.ParsedParameterLayer{}
		commandLayers := pc.Cmd.Description().Layers
		for _, v := range commandLayers {
			parsedParameterLayer := &layers.ParsedParameterLayer{
				Layer:      v,
				Parameters: map[string]interface{}{},
			}
			parsedLayers, ok := parseState.Layers[v.GetSlug()]
			if ok {
				parsedParameterLayer.Parameters = parsedLayers.Parameters
			}
			pc.ParsedLayers[v.GetSlug()] = parsedParameterLayer
		}

		return nil
	}
}
