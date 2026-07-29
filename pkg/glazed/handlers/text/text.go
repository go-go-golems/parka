package text

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
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
		glazedLayer, ok := parsedValues.Get(settings.StructuredOutputSlug)
		if !ok {
			// No structured-output section was provided; create a default one
			// with table format (what the text handler wants).
			section, err := settings.NewStructuredOutputSection(
				schema.WithDefaults(map[string]interface{}{"format": "table"}),
			)
			if err != nil {
				return err
			}
			glazedLayer, err = values.NewSectionValues(section)
			if err != nil {
				return err
			}
			// Apply the section's field defaults to the values.
			defaults, err := section.GetDefinitions().FieldValuesFromDefaults()
			if err != nil {
				return err
			}
			if _, err := glazedLayer.Fields.Merge(defaults); err != nil {
				return err
			}
		} else {
			// Force table output for the text handler regardless of the request.
			if _, err := glazedLayer.Fields.UpdateExistingValue("format", "table", fields.WithSource("parka-text-handler")); err != nil {
				return err
			}
		}

		gp, _, err := settings.SetupStructuredOutput(glazedLayer, c.Response())
		if err != nil {
			return err
		}

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
