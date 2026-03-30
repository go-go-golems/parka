package text

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type QueryHandler struct {
	cmd         cmds.Command
	middlewares []sources.Middleware
	// whitelistedLayers contains the list of layers that are allowed to be modified through query parameters
	whitelistedLayers []string
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

func WithMiddlewares(middlewares ...sources.Middleware) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.middlewares = middlewares
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

	queryMiddleware := parka_middlewares.UpdateFromQueryParameters(c, fields.WithSource("query"))
	if len(h.whitelistedLayers) > 0 {
		queryMiddleware = sources.WrapWithWhitelistedSections(h.whitelistedLayers, queryMiddleware)
	}

	middlewares_ := append(
		[]sources.Middleware{
			queryMiddleware,
		},
		h.middlewares...,
	)
	middlewares_ = append(middlewares_, sources.FromDefaults())

	err := sources.Execute(description.Schema.Clone(), parsedValues, middlewares_...)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")

	ctx := c.Request().Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		err := cmd.RunIntoWriter(ctx, parsedValues, c.Response())
		if err != nil {
			return err
		}

	case cmds.GlazeCommand:
		gp, err := handlers.CreateTableProcessorWithOutput(parsedValues, "table", "ascii")
		if err != nil {
			return err
		}

		glazedLayer, ok := parsedValues.Get(settings.GlazedSlug)
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

func CreateQueryHandler(
	cmd cmds.Command,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd, options...)
	return handler.Handle
}
