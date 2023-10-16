package sse

// Implement a streaming SSE handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type QueryHandler struct {
	cmd                cmds.Command
	contextMiddlewares []glazed.ContextMiddleware
	parserOptions      []parser.ParserOption
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

func (h *QueryHandler) Handle(c *gin.Context, writer gin.ResponseWriter) error {
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

	c.Header("Content-Type", "text/event-stream")

	ctx := c.Request.Context()
	switch cmd := h.cmd.(type) {
	case cmds.WriterCommand:
		// Create a writer that on every read amount of bytes sends an sse message
		// to the client
		sseWriter := NewSSEWriter()

		eg := errgroup.Group{}
		eg.Go(func() error {
			defer sseWriter.Close()
			return cmd.RunIntoWriter(
				ctx,
				pc.ParsedLayers,
				pc.ParsedParameters,
				sseWriter,
			)
		})

		eg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case msg, ok := <-sseWriter.ch:
					if !ok {
						return nil
					}
					// write SSE event to writer
					s := fmt.Sprintf("data: %s\n\n", msg)
					_, err := writer.Write([]byte(s))
					if err != nil {
						return err
					}
					writer.Flush()
				}
			}
		})

		return eg.Wait()

	case cmds.GlazeCommand:
		gp, err := handlers.CreateTableProcessorWithOutput(pc, "json", "")

		gp.ReplaceTableMiddleware()
		c := make(chan string, 100)
		r := row.NewOutputChannelMiddleware(json2.NewOutputFormatter(), c)
		gp.AddRowMiddleware(r)

		eg := errgroup.Group{}
		eg.Go(func() error {
			err := cmd.Run(ctx, pc.ParsedLayers, pc.ParsedParameters, gp)
			if err != nil {
				return err
			}
			err = gp.Close(ctx)
			if err != nil {
				return err
			}
			return nil
		})

		eg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case row := <-c:
					_, err := writer.Write([]byte(row))
					if err != nil {
						return err
					}
				}
			}
		})

		err = eg.Wait()
		if err != nil {
			return err
		}

	case cmds.BareCommand:
		err := cmd.Run(ctx, pc.ParsedLayers, pc.ParsedParameters)
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
