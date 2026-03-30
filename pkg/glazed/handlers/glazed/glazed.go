package glazed

import (
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type QueryHandler struct {
	cmd               cmds.GlazeCommand
	middlewares       []sources.Middleware
	whitelistedLayers []string
}

type QueryHandlerOption func(*QueryHandler)

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

func NewQueryHandler(cmd cmds.GlazeCommand, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd: cmd,
	}

	for _, option := range options {
		option(h)
	}

	return h
}

var _ handlers.Handler = (*QueryHandler)(nil)

func (h *QueryHandler) Handle(c echo.Context) error {
	description := h.cmd.Description()
	parsedValues := values.New()

	queryMiddleware := middlewares2.UpdateFromQueryParameters(c, fields.WithSource("query"))
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

	glazedLayer, ok := parsedValues.Get(settings.GlazedSlug)
	if !ok {
		return errors.New("glazed layer not found")
	}

	gp, err := settings.SetupTableProcessor(glazedLayer)
	if err != nil {
		return err
	}

	of, err := settings.SetupProcessorOutput(gp, glazedLayer, c.Response())
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", of.ContentType())
	c.Response().WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
	err = h.cmd.RunIntoGlazeProcessor(ctx, parsedValues, gp)
	if err != nil {
		return err
	}

	err = gp.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}
