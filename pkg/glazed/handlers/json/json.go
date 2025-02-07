package json

import (
	"bytes"
	"encoding/json"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"net/http"
)

type QueryHandler struct {
	cmd         cmds.Command
	middlewares []middlewares.Middleware
}

type QueryHandlerOption func(*QueryHandler)

func NewQueryHandler(cmd cmds.Command, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd: cmd,
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

var _ handlers.Handler = (*QueryHandler)(nil)

func (h *QueryHandler) Handle(c echo.Context) error {
	description := h.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	middlewares_ := append(
		[]middlewares.Middleware{
			middlewares2.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
		},
		h.middlewares...,
	)
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

		foo := struct {
			Data string `json:"data"`
		}{
			Data: buf.String(),
		}
		encoder := json.NewEncoder(c.Response())
		encoder.SetIndent("", "  ")
		err = encoder.Encode(foo)
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

func CreateJSONQueryHandler(
	cmd cmds.Command,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd, options...)
	return handler.Handle
}
