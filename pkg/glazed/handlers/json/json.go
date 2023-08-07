package json

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

type QueryHandler struct {
	cmd                cmds.GlazeCommand
	contextMiddlewares []glazed.ContextMiddleware
	parserOptions      []parser.ParserOption
}

type QueryHandlerOption func(*QueryHandler)

func NewQueryHandler(cmd cmds.GlazeCommand, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd: cmd,
	}

	for _, option := range options {
		option(h)
	}

	return h
}

func WithQueryHandlerContextMiddlewares(middlewares ...glazed.ContextMiddleware) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.contextMiddlewares = middlewares
	}
}

// WithQueryHandlerParserOptions sets the parser options for the QueryHandler
func WithQueryHandlerParserOptions(options ...parser.ParserOption) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.parserOptions = options
	}
}

func (h *QueryHandler) Handle(c *gin.Context, writer io.Writer) error {
	pc := glazed.NewCommandContext(h.cmd)

	h.contextMiddlewares = append(
		h.contextMiddlewares,
		glazed.NewContextParserMiddleware(
			h.cmd,
			glazed.NewCommandQueryParser(h.cmd, h.parserOptions...),
		),
	)

	for _, h := range h.contextMiddlewares {
		err := h.Handle(c, pc)
		if err != nil {
			return err
		}
	}

	gp, err := handlers.CreateTableProcessorWithOutput(pc, "json", "")
	if err != nil {
		return err
	}

	// remove table middlewares because we are a streaming handler
	gp.ReplaceTableMiddleware()
	gp.AddRowMiddleware(row.NewOutputMiddleware(json2.NewOutputFormatter(), writer))

	c.Header("Content-Type", "application/json")

	ctx := c.Request.Context()
	err = h.cmd.Run(ctx, pc.ParsedLayers, pc.ParsedParameters, gp)
	if err != nil {
		return err
	}

	err = gp.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateJSONQueryHandler(
	cmd cmds.GlazeCommand,
	parserOptions ...parser.ParserOption,
) gin.HandlerFunc {
	handler := NewQueryHandler(cmd,
		WithQueryHandlerParserOptions(parserOptions...),
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
