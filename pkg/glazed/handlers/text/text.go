package text

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
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

func (h *QueryHandler) Handle(c *gin.Context, writer io.Writer) error {
	description := h.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	middlewares_ := append([]middlewares.Middleware{
		parka_middlewares.UpdateFromQueryParameters(c,
			parameters.WithParseStepSource("query"),
		),
		middlewares.SetFromDefaults(),
	}, h.middlewares...)
	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")

	ctx := c.Request.Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		err := cmd.RunIntoWriter(ctx, parsedLayers, writer)
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

		gp.AddTableMiddleware(table.NewOutputMiddleware(of, writer))

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
	middlewares_ ...middlewares.Middleware,
) gin.HandlerFunc {
	handler := NewQueryHandler(cmd,
		WithMiddlewares(middlewares_...),
	)
	return func(c *gin.Context) {
		err := handler.Handle(c, c.Writer)
		if err != nil {
			log.Error().Err(err).Msg("failed to handle query")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
	}
}
