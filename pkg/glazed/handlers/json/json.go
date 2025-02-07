package json

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
)

type QueryHandler struct {
	cmd         cmds.Command
	middlewares []middlewares.Middleware
	// useJSONBody determines whether to use JSON body parsing (POST) or query parameters (GET)
	useJSONBody bool
}

type QueryHandlerOption func(*QueryHandler)

func NewQueryHandler(cmd cmds.Command, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd:         cmd,
		useJSONBody: false, // default to query parameters
	}

	for _, option := range options {
		option(h)
	}

	return h
}

func WithMiddlewares(middlewares ...middlewares.Middleware) QueryHandlerOption {
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

var _ handlers.Handler = (*QueryHandler)(nil)

func (h *QueryHandler) Handle(c echo.Context) error {
	description := h.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	var jsonMiddleware *parka_middlewares.JSONBodyMiddleware
	if h.useJSONBody {
		jsonMiddleware = parka_middlewares.NewJSONBodyMiddleware(c, parameters.WithParseStepSource("json"))
		defer func() {
			if err := jsonMiddleware.Close(); err != nil {
				// We can only log this error since we're in a defer
				c.Logger().Errorf("failed to cleanup JSON middleware: %v", err)
			}
		}()
	}

	// Build the middleware chain
	middlewares_ := make([]middlewares.Middleware, 0)
	if h.useJSONBody {
		middlewares_ = append(middlewares_, jsonMiddleware.Middleware())
	} else {
		middlewares_ = append(middlewares_, parka_middlewares.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")))
	}

	middlewares_ = append(middlewares_, h.middlewares...)
	middlewares_ = append(middlewares_, middlewares.SetFromDefaults())

	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		buf := bytes.Buffer{}
		err := cmd.RunIntoWriter(ctx, parsedLayers, &buf)
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
		gp, err := handlers.CreateTableProcessorWithOutput(parsedLayers, "json", "")
		if err != nil {
			return err
		}

		// remove table middlewares because we are a streaming handler
		gp.ReplaceTableMiddleware()
		gp.AddRowMiddleware(row.NewOutputMiddleware(json2.NewOutputFormatter(), c.Response()))

		if err != nil {
			return err
		}

		err = cmd.RunIntoGlazeProcessor(ctx, parsedLayers, gp)
		if err != nil {
			return err
		}

		err = gp.Close(ctx)
		if err != nil {
			return err
		}

	case cmds.BareCommand:
		err := cmd.Run(ctx, parsedLayers)
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
