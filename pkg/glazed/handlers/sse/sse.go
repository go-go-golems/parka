package sse

// Implement a streaming SSE handler

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
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

func (h *QueryHandler) Handle(c echo.Context) error {
	description := h.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	middlewares_ := append([]middlewares.Middleware{
		middlewares2.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
	}, h.middlewares...)
	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers, middlewares_...)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
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
					_, err := fmt.Fprintf(c.Response(), "data: %s\n\n", msg)
					if err != nil {
						return err
					}
					c.Response().Flush()
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
		eventChan := make(chan string, 100)
		r := row.NewOutputChannelMiddleware(json2.NewOutputFormatter(), eventChan)
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
				case row := <-eventChan:
					_, err := c.Response().Write([]byte(row))
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
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd,
		WithMiddlewares(middlewares_...),
	)
	return handler.Handle
}
