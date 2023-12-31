package sse

// Implement a streaming SSE handler

import (
	"fmt"
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
	"golang.org/x/sync/errgroup"
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

func (h *QueryHandler) Handle(c *gin.Context, writer gin.ResponseWriter) error {
	description := h.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	middlewares_ := append([]middlewares.Middleware{
		middlewares2.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
	}, h.middlewares...)
	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
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
				parsedLayers,
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
		gp, err := handlers.CreateTableProcessorWithOutput(parsedLayers, "json", "")
		if err != nil {
			return err
		}

		gp.ReplaceTableMiddleware()
		c := make(chan string, 100)
		r := row.NewOutputChannelMiddleware(json2.NewOutputFormatter(), c)
		gp.AddRowMiddleware(r)

		eg := errgroup.Group{}
		eg.Go(func() error {
			err := cmd.RunIntoGlazeProcessor(ctx, parsedLayers, gp)
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
