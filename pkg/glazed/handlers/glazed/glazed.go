package glazed

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

type QueryHandler struct {
	cmd         cmds.GlazeCommand
	middlewares []middlewares.Middleware
}

type QueryHandlerOption func(*QueryHandler)

func WithMiddlewares(middlewares ...middlewares.Middleware) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.middlewares = middlewares
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
	parsedLayers := layers.NewParsedLayers()

	middlewares_ := append(h.middlewares,
		middlewares2.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
		middlewares.SetFromDefaults(),
	)
	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}

	glazedLayer, ok := parsedLayers.Get("glazed")
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
	err = h.cmd.RunIntoGlazeProcessor(ctx, parsedLayers, gp)
	if err != nil {
		return err
	}

	err = gp.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}
