package glazed

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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

// HandlePrepopulatedParsedLayers sets the given layers in the CommandContext's ParsedLayers,
// overriding the parameters of any layers that are already present.
// This means that if a parameter is not set in layers_ but is set in the ParsedLayers,
// the value in the ParsedLayers will be kept.
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

func NewCommandQueryParser(cmd cmds.GlazeCommand, options ...ParserOption) *Parser {
	d := cmd.Description()

	ph := NewParser()
	ph.Parsers = []ParserFunc{
		NewQueryParserFunc(false),
	}

	// NOTE(manuel, 2023-04-16) API design: we would probably like to hide layers right here in the handler constructor
	for _, l := range d.Layers {
		slug := l.GetSlug()
		ph.LayerParsersBySlug[slug] = []ParserFunc{
			NewQueryParserFunc(false),
		}
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

func NewCommandFormParser(cmd cmds.GlazeCommand, options ...ParserOption) *Parser {
	d := cmd.Description()

	ph := NewParser()
	ph.Parsers = []ParserFunc{
		NewFormParserFunc(false),
	}

	// NOTE(manuel, 2023-04-16) API design: we would probably like to hide layers right here in the handler constructor
	for _, l := range d.Layers {
		slug := l.GetSlug()
		ph.LayerParsersBySlug[slug] = []ParserFunc{
			NewFormParserFunc(false),
		}
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

func NewCommandHandlerFunc(cmd cmds.GlazeCommand, parserHandler *Parser) CommandHandlerFunc {
	d := cmd.Description()

	var err error

	return func(c *gin.Context, pc *CommandContext) error {
		pds := map[string]*parameters.ParameterDefinition{}
		for _, p := range d.Flags {
			pds[p.Name] = p
		}
		for _, p := range d.Arguments {
			pds[p.Name] = p
		}

		for _, o := range parserHandler.Parsers {
			pds, err = o(c, pc.ParsedParameters, pds)
			if err != nil {
				return err
			}
		}

		for _, l := range d.Layers {
			slug := l.GetSlug()
			parsers, ok := parserHandler.LayerParsersBySlug[slug]
			if !ok {
				continue
			}

			_, ok = pc.ParsedLayers[slug]
			if !ok {
				pc.ParsedLayers[slug] = &layers.ParsedParameterLayer{
					Layer:      l,
					Parameters: map[string]interface{}{},
				}
			}

			pds = l.GetParameterDefinitions()

			for _, o := range parsers {
				pds, err = o(c, pc.ParsedLayers[slug].Parameters, pds)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}
}
