package sse

// Implement a streaming SSE handler

import (
	"fmt"
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	json2 "github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	middlewares2 "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/kucherenkovova/safegroup"
	"github.com/labstack/echo/v4"
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

		eg := safegroup.Group{}
		eg.Go(func() error {
			defer sseWriter.Close()
			return cmd.RunIntoWriter(
				ctx,
				parsedValues,
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
		gp, err := handlers.CreateTableProcessorWithOutput(parsedValues, "json", "")
		if err != nil {
			return err
		}

		gp.ReplaceTableMiddleware()
		eventChan := make(chan string, 100)
		r := row.NewOutputChannelMiddleware(json2.NewOutputFormatter(), eventChan)
		gp.AddRowMiddleware(r)

		eg := safegroup.Group{}
		eg.Go(func() error {
			err := cmd.RunIntoGlazeProcessor(ctx, parsedValues, gp)
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
