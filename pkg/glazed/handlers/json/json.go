package json

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
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
		middlewares2.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
	}, h.middlewares...)
	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}

	c.Header("Content-Type", "application/json")

	ctx := c.Request.Context()
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
		encoder := json.NewEncoder(writer)
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
		gp.AddRowMiddleware(row.NewOutputMiddleware(json2.NewOutputFormatter(), writer))

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
	middlewares ...middlewares.Middleware,
) gin.HandlerFunc {
	handler := NewQueryHandler(cmd,
		WithMiddlewares(middlewares...),
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
