package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"net/http"
)

type JSONMarshaler interface {
	MarshalJSON() ([]byte, error)
}

// so what would a parsing middleware for REST and co look like.
//
// we have:
// - query parameters
// - form parameters
// - REST JSON body
//
// On the cobra side, when we have a cobra command, we register a layer and this
// returns a parser function that can be called later on. This is done because
// we have a cobra *cmd object that we need to create, and only later
// actually parse. This doesn't really apply to the web scenario, I think.
//
// We also want a way for these layers and parameters to provide HTML form elements
// or a way to render themselves in response to a request.
//
// ## Parsing incoming parameters
//
// Ultimately, when a request comes in to the API, we want to go through all the
// "middlewares" and ultimately get a map of parsed layers and a map of parameters,
// and call Run.
//
// ## Rendering the glazed output
//
// The response is then rendered by the command / command wrapper, potentially rendering
// out the individual layers to give some feedback on the request.
//
// - [ ] Parse the content-type of an incoming request to configure the output type for glazed
// - [ ] Allow some parameters of the incoming request to configure the glazed output
//  - this might not be good for SimpleCommand (but maybe?)
//  - there might be something there to be done with the layer wrapper discussed in the section
//    below
//
// ## Filtering parameters
//
// One very important part of parka is that it should allow for filtering and validating
// in addition to what the actual command layers give us.
//
// I'm not entirely sure how this fits in with the current ParkaContext that I build,
// but I think it might be possible by defining somethings that's a "wrapper layer".
// It takes another layer, potentially a set of defaults (although it might be easiest to
// just pass that straight to the underlying layer anyway), and it does then
// expose its own set of flags, and then once parsed, update the lower layer.
//
// The command itself would actually just know its own layers, but when exposed over REST,
// it will be wrapped in whatever the developer specifies.
//
// The problem with this approach however is that we might need to dissect the lower layers of the
// command, which is maybe not so elegant. But let's try that.
//
// While this seems like something that could be useful in glazed itself, I'm going to
// start building it only in parka for now, and then see how far it goes.
//
// ## Configuring parka handlers declaratively
//
// If we want to follow the pattern of declaratively based commands,
// we could make a folder that specifies the templates to be loaded,
// and the parameters and their types to be exposed, in a yaml.
//
// This could be loaded, and then used to wrap *another* command (which might or might not
// be declarative). Probably the best is to be a bit less bullish on the yaml flag declarations,
// but maybe just leverage the recursive template loading that I now have. This makes it possible
// to load plenty of smaller fragments for reuse.
//
// ## Rendering parameter widgets
//
// One really important part of parka is being able to render exposed parameters
// as HTML widgets. This might be possible to achieve by the handler chain by providing
// renderers for either a layer or individual flags or by default.
//
// When a request comes in to the HTML side, we want to render out the form elements
// for each layer, but ultimately it comes down to the command itself to render the final picture.
//
// It can of course call on the HTML forms of the individual layers. Of course,
// we provide standard form elements for each parameter type as well, so that there is a
// quick way to render the different parameters.
//
// ## Using templates for the rendering of parameters
//
// We should capitalize on the heave use of templates through the go go golems ecosystem
// and make it easy for people to provide overrides for parameter types, individual flags
// in the form of reusable templates that take metadata describing the parameter.
//
// Maybe this kind of metadata could be part of the original parameter definitions / command description,
// and there is a way to create an extensible mechanism right from the start?
//
// Let's not worry too much about that for now, and stay typed and potentially verbose and
// not that generic for parka, not hesitating to create our own structs and parallel constructions
// until we better understand what is different about the parka use cases.
//
// ## Exposing the flag metadata
//
// Another interesting to do would be to recreate voodoo in golang. For that, it should be
// possible to query the metadata about the arguments that a HTTP API command accepts, so that
// they can be used to configure a client side command line app dynamically.
//
// - [ ] Build voodoo on top of parka, see https://github.com/go-go-golems/parka/issues/15

// ParkaContext keeps the context for execution of a parka command,
// and can be worked upon by ParkaHandlerFuncs.
type ParkaContext struct {
	// Cmd is the command that will be executed
	Cmd cmds.Command
	// ParsedLayers contains the map of parsed layers parsed so far
	ParsedLayers map[string]*layers.ParsedParameterLayer
	// ParsedParameters contains the map of parsed parameters parsed so far
	ParsedParameters map[string]interface{}
}

func NewParkaContext(cmd cmds.Command) *ParkaContext {
	return &ParkaContext{
		Cmd:              cmd,
		ParsedLayers:     map[string]*layers.ParsedParameterLayer{},
		ParsedParameters: map[string]interface{}{},
	}
}

type ParkaHandlerFunc func(*gin.Context, *ParkaContext) error

func HandleParsedParametersFromQuery(
	ps map[string]*parameters.ParameterDefinition,
	onlyDefined bool,
) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
		parsedParameters, err := parseQueryFromParameterDefinitions(c, ps, onlyDefined)
		if err != nil {
			return err
		}
		for k, v := range parsedParameters {
			pc.ParsedParameters[k] = v
		}

		return nil
	}
}

func HandleParsedLayersFromQuery(
	layers_ []layers.ParameterLayer,
	onlyDefined bool,
) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
		for _, layer := range layers_ {
			parsedParameters, err := parseQueryFromParameterDefinitions(
				c, layer.GetParameterDefinitions(),
				onlyDefined,
			)
			if err != nil {
				return err
			}
			name := layer.GetName()
			parsedLayer, ok := pc.ParsedLayers[name]
			if ok {
				for k, v := range parsedParameters {
					parsedLayer.Parameters[k] = v
				}
			} else {
				pc.ParsedLayers[name] = &layers.ParsedParameterLayer{
					Layer:      layer,
					Parameters: parsedParameters,
				}
			}
		}
		return nil
	}
}

func HandleParsedParametersFromForm(
	ps map[string]*parameters.ParameterDefinition,
	onlyDefined bool,
) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
		parsedParameters, err := parseFormFromParameterDefinitions(c, ps, onlyDefined)
		if err != nil {
			return err
		}
		for k, v := range parsedParameters {
			pc.ParsedParameters[k] = v
		}

		return nil
	}
}

func HandleParsedLayersFromForm(
	layers_ []layers.ParameterLayer,
	onlyDefined bool,
) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
		for _, layer := range layers_ {
			parsedParameters, err := parseFormFromParameterDefinitions(
				c, layer.GetParameterDefinitions(),
				onlyDefined,
			)
			if err != nil {
				return err
			}
			name := layer.GetName()
			parsedLayer, ok := pc.ParsedLayers[name]
			if ok {
				for k, v := range parsedParameters {
					parsedLayer.Parameters[k] = v
				}
			} else {
				pc.ParsedLayers[name] = &layers.ParsedParameterLayer{
					Layer:      layer,
					Parameters: parsedParameters,
				}
			}
		}
		return nil
	}
}

func HandlePrepopulatedParameters(ps map[string]interface{}) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
		for k, v := range ps {
			pc.ParsedParameters[k] = v
		}
		return nil
	}
}

func HandlePrepopulatedParsedLayers(layers_ map[string]*layers.ParsedParameterLayer) ParkaHandlerFunc {
	return func(c *gin.Context, pc *ParkaContext) error {
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

// TODO(manuel, 2023-02-28) We want to provide a handler to catch errors while parsing parameters

func (s *Server) HandleSimpleFormCommand(
	cmd cmds.Command,
	handlers ...ParkaHandlerFunc,
) gin.HandlerFunc {
	d := cmd.Description()
	pds := map[string]*parameters.ParameterDefinition{}
	for _, p := range d.Flags {
		pds[p.Name] = p
	}
	for _, p := range d.Arguments {
		pds[p.Name] = p
	}

	pdHandler := HandleParsedParametersFromForm(pds, false)
	layersHandler := HandleParsedLayersFromForm(d.Layers, false)

	return NewGinHandlerFromParkaHandlers(cmd, handlers, pdHandler, layersHandler)
}

func (s *Server) HandleSimpleQueryCommand(
	cmd cmds.Command,
	handlers ...ParkaHandlerFunc,
) gin.HandlerFunc {
	d := cmd.Description()
	pds := map[string]*parameters.ParameterDefinition{}
	for _, p := range d.Flags {
		pds[p.Name] = p
	}
	for _, p := range d.Arguments {
		pds[p.Name] = p
	}

	pdHandler := HandleParsedParametersFromQuery(pds, false)
	layersHandler := HandleParsedLayersFromQuery(d.Layers, false)

	return NewGinHandlerFromParkaHandlers(cmd, handlers, pdHandler, layersHandler)
}

func NewGinHandlerFromParkaHandlers(
	cmd cmds.Command,
	handlers []ParkaHandlerFunc,
	pdHandler ParkaHandlerFunc,
	layersHandler ParkaHandlerFunc,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// NOTE(manuel, 2023-02-28) Add initial middleware handlers here
		//
		// It probably makes sense to give the user control over the initial parka
		// context before passing it downstream
		pc := NewParkaContext(cmd)

		err := pdHandler(c, pc)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		err = layersHandler(c, pc)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		for _, h := range handlers {
			err = h(c, pc)
			if err != nil {
				_ = c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		of, gp, err := SetupProcessor()
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// get gp output
		_, err = of.Output()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		rows := []map[string]interface{}{}
		for _, row := range of.Table.Rows {
			rows = append(rows, row.GetValues())
		}

		c.JSON(200, rows)
	}
}

func SetupProcessor() (*formatters.JSONOutputFormatter, *cmds.GlazeProcessor, error) {
	// TODO(manuel, 2023-02-11) For now, create a raw JSON output formatter. We will want more nuance here
	// See https://github.com/go-go-golems/parka/issues/8

	of := formatters.NewJSONOutputFormatter(true)
	gp := cmds.NewGlazeProcessor(of, []middlewares.ObjectMiddleware{})

	return of, gp, nil
}
