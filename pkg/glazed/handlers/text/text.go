package text

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
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
			parka_middlewares.UpdateFromQueryParameters(c,
				parameters.WithParseStepSource("query"),
			),
		},
		h.middlewares...,
	)
	middlewares_ = append(middlewares_, middlewares.SetFromDefaults())

	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")

	ctx := c.Request().Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		err := cmd.RunIntoWriter(ctx, parsedLayers, c.Response())
		if err != nil {
			return err
		}

	case cmds.GlazeCommand:
		gp, err := handlers.CreateTableProcessorWithOutput(parsedLayers, "table", "ascii")
		if err != nil {
			return err
		}

		glazedLayer, ok := parsedLayers.Get("glazed")
		if !ok {
			return errors.New("glazed layer not found")
		}

		of, err := settings.SetupTableOutputFormatter(glazedLayer)
		if err != nil {
			return err
		}
		err = of.RegisterTableMiddlewares(gp)
		if err != nil {
			return err
		}

		gp.AddTableMiddleware(table.NewOutputMiddleware(of, c.Response()))

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

func CreateQueryHandler(
	cmd cmds.Command,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd, options...)
	return handler.Handle
}
