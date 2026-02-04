package json

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
)

type QueryHandler struct {
	cmd         cmds.Command
	middlewares []sources.Middleware
	// useJSONBody determines whether to use JSON body parsing (POST) or query parameters (GET)
	useJSONBody bool
	// parseOptions are options passed to parameter parsing
	parseOptions []fields.ParseOption
	// whitelistedLayers contains the list of layers that are allowed to be modified through query parameters
	whitelistedLayers []string
}

type QueryHandlerOption func(*QueryHandler)

func NewQueryHandler(cmd cmds.Command, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd:          cmd,
		useJSONBody:  false, // default to query parameters
		parseOptions: []fields.ParseOption{},
	}

	for _, option := range options {
		option(h)
	}

	return h
}

func WithMiddlewares(middlewares ...sources.Middleware) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.middlewares = middlewares
	}
}

// WithJSONBody configures the handler to use JSON body parsing instead of query parameters
func WithJSONBody() QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.useJSONBody = true
	}
}

// WithParseOptions adds parameter parse options to the handler
func WithParseOptions(options ...fields.ParseOption) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.parseOptions = append(handler.parseOptions, options...)
	}
}

func WithWhitelistedLayers(layers ...string) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.whitelistedLayers = layers
	}
}

var _ handlers.Handler = (*QueryHandler)(nil)

func (h *QueryHandler) Handle(c echo.Context) error {
	description := h.cmd.Description()
	parsedValues := values.New()

	var jsonMiddleware *parka_middlewares.JSONBodyMiddleware
	if h.useJSONBody {
		jsonMiddleware = parka_middlewares.NewJSONBodyMiddleware(c, append(h.parseOptions, fields.WithSource("json"))...)
		defer func() {
			if err := jsonMiddleware.Close(); err != nil {
				// We can only log this error since we're in a defer
				c.Logger().Errorf("failed to cleanup JSON middleware: %v", err)
			}
		}()
	}

	// Build the middleware chain
	middlewares_ := make([]sources.Middleware, 0)
	if h.useJSONBody {
		middlewares_ = append(middlewares_, jsonMiddleware.Middleware())
	} else {
		queryMiddleware := parka_middlewares.UpdateFromQueryParameters(c, append(h.parseOptions, fields.WithSource("query"))...)
		if len(h.whitelistedLayers) > 0 {
			queryMiddleware = sources.WrapWithWhitelistedSections(h.whitelistedLayers, queryMiddleware)
		}
		middlewares_ = append(middlewares_, queryMiddleware)
	}

	middlewares_ = append(middlewares_, h.middlewares...)
	middlewares_ = append(middlewares_, sources.FromDefaults())

	err := sources.Execute(description.Schema.Clone(), parsedValues, middlewares_...)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		buf := bytes.Buffer{}
		err := cmd.RunIntoWriter(ctx, parsedValues, &buf)
		if err != nil {
			return err
		}

		response := struct {
			Data string `json:"data"`
		}{
			Data: buf.String(),
		}
		encoder := json.NewEncoder(c.Response())
		encoder.SetIndent("", "  ")
		err = encoder.Encode(response)
		if err != nil {
			return err
		}

	case cmds.GlazeCommand:
		gp, err := handlers.CreateTableProcessorWithOutput(parsedValues, "json", "")
		if err != nil {
			return err
		}

		// remove table middlewares because we are a streaming handler
		gp.ReplaceTableMiddleware()
		gp.AddRowMiddleware(row.NewOutputMiddleware(json2.NewOutputFormatter(), c.Response()))

		if err != nil {
			return err
		}

		err = cmd.RunIntoGlazeProcessor(ctx, parsedValues, gp)
		if err != nil {
			return err
		}

		err = gp.Close(ctx)
		if err != nil {
			return err
		}

	case cmds.BareCommand:
		err := cmd.Run(ctx, parsedValues)
		if err != nil {
			return err
		}

	default:
		return &handlers.UnsupportedCommandError{Command: h.cmd}
	}

	return nil
}

// CreateJSONQueryHandler creates a new JSON handler that uses query parameters
func CreateJSONQueryHandler(
	cmd cmds.Command,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd, options...)
	return handler.Handle
}

// CreateJSONBodyHandler creates a new JSON handler that uses POST body parsing
func CreateJSONBodyHandler(
	cmd cmds.Command,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	options = append(options, WithJSONBody())
	handler := NewQueryHandler(cmd, options...)
	return handler.Handle
}
