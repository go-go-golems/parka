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
	ParsedLayers map[string]*layers.ParsedParameterLayer
	// ParsedParameters contains the map of parsed parameters parsed so far
	ParsedParameters map[string]interface{}
}

// NewCommandContext creates a new CommandContext for the given command.
func NewCommandContext(cmd cmds.Command) *CommandContext {
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

// ContextMiddleware is used to build up a CommandContext. It is similar to the HandlerFunc
// in gin, but instead of handling the HTTP response, it is only used to build up the CommandContext.
type ContextMiddleware interface {
	Handle(*gin.Context, *CommandContext) error
}

// PrepopulatedContextMiddleware is a ContextMiddleware that prepopulates the CommandContext with
// the given map of parameters. It overwrites existing parameters in pc.ParsedParameters,
// and overwrites parameters in individual parser layers if they have already been set.
type PrepopulatedContextMiddleware struct {
	ps map[string]interface{}
}

func (p *PrepopulatedContextMiddleware) Handle(c *gin.Context, pc *CommandContext) error {
	for k, v := range p.ps {
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

func NewCommandFormParser(cmd cmds.Command, options ...parser.ParserOption) *parser.Parser {
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

func NewContextParserMiddleware(cmd cmds.Command, parser *parser.Parser) *ContextParserMiddleware {
	return &ContextParserMiddleware{
		command: cmd,
		parser:  parser,
	}
}
