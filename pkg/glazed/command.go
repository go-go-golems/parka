package glazed

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
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
	Cmd cmds.Command
	// ParsedLayers contains the map of parsed layers parsed so far
	ParsedLayers *layers.ParsedParameterLayers
}

// NewCommandContext creates a new CommandContext for the given command.
func NewCommandContext(cmd cmds.Command) *CommandContext {
	return &CommandContext{
		Cmd:          cmd,
		ParsedLayers: layers.NewParsedParameterLayers(),
	}
}

// GetAllParameterDefinitions returns a map of all parameter definitions for the command.
// This includes flags, arguments and all layers.
func (pc *CommandContext) GetAllParameterDefinitions() parameters.ParameterDefinitions {
	description := pc.Cmd.Description()
	ret := parameters.NewParameterDefinitions()

	for _, l := range description.Layers {
		ret.Merge(l.GetParameterDefinitions())
	}

	return ret
}

// ContextMiddleware is used to build up a CommandContext. It is similar to the HandlerFunc
// in gin, but instead of handling the HTTP response, it is only used to build up the CommandContext.
type ContextMiddleware interface {
	Handle(*gin.Context, *CommandContext) error
}

// PrepopulatedContextMiddleware is a ContextMiddleware that prepopulates the CommandContext with
// the given map of parameters. It overwrites existing parameters in pc.ParsedParameters,
// and overwrites parameters in individual parser layers if they have already been set.
//
// TODO(manuel, 2023-12-22) This will become a more standard Parameters middleware
type PrepopulatedContextMiddleware struct {
	ps map[string]interface{}
}

func (p *PrepopulatedContextMiddleware) Handle(c *gin.Context, pc *CommandContext) error {
	for k, v := range p.ps {
		pc.ParsedLayers.ForEach(func(_ string, v_ *layers.ParsedParameterLayer) {
			p, ok := v_.Parameters.Get(k)
			if ok {
				p.Set("prepopulated-context", v)
			}
		})
	}
	return nil
}

func NewPrepopulatedContextMiddleware(ps map[string]interface{}) *PrepopulatedContextMiddleware {
	return &PrepopulatedContextMiddleware{
		ps: ps,
	}
}

// PrepopulatedParsedLayersContextMiddleware is a ContextMiddleware that prepopulates the CommandContext with
// the given map of layers. If a layer already exists, it overwrites its parameters.
type PrepopulatedParsedLayersContextMiddleware struct {
	layers map[string]*layers.ParsedParameterLayer
}

func (p *PrepopulatedParsedLayersContextMiddleware) Handle(c *gin.Context, pc *CommandContext) error {
	for k, v := range p.layers {
		parsedLayer, ok := pc.ParsedLayers.Get(k)
		if ok {
			v.Parameters.ForEach(func(k2 string, v2 *parameters.ParsedParameter) {
				parsedLayer.Parameters.Set(k2, v2.Clone())
			})
		} else {
			pc.ParsedLayers.Set(k, v.Clone())
		}
	}
	return nil
}

func NewPrepopulatedParsedLayersContextMiddleware(
	layers map[string]*layers.ParsedParameterLayer,
) *PrepopulatedParsedLayersContextMiddleware {
	return &PrepopulatedParsedLayersContextMiddleware{
		layers: layers,
	}
}

func NewCommandQueryParser(cmd cmds.Command, options ...parser.ParserOption) *parser.Parser {
	d := cmd.Description()

	// NOTE(manuel, 2023-06-21) We could pass the parser options here, but then we wouldn't be able to
	// override the layer parser. Or we could pass the QueryParsers right here.
	ph := parser.NewParser()

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

func NewCommandFormParser(cmd cmds.Command, options ...parser.ParserOption) *parser.Parser {
	d := cmd.Description()

	ph := parser.NewParser()

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

type ContextParserMiddleware struct {
	command cmds.Command
	parser  *parser.Parser
}

func (cpm *ContextParserMiddleware) Handle(c *gin.Context, pc *CommandContext) error {
	parseState := parser.NewParseStateFromCommandDescription(cpm.command)
	err := cpm.parser.Parse(c, parseState)
	if err != nil {
		return err
	}

	pc.ParsedLayers = layers.NewParsedParameterLayers()
	commandLayers := pc.Cmd.Description().Layers
	for _, v := range commandLayers {
		parsedParameterLayer := layers.NewParsedParameterLayer(v)
		parsedLayer, ok := parseState.Layers[v.GetSlug()]
		if ok {
			parsedParameterLayer.Parameters = parsedLayer.ParsedParameters
		}
		pc.ParsedLayers.Set(v.GetSlug(), parsedParameterLayer)
	}

	return nil
}

func NewContextParserMiddleware(cmd cmds.Command, parser *parser.Parser) *ContextParserMiddleware {
	return &ContextParserMiddleware{
		command: cmd,
		parser:  parser,
	}
}
