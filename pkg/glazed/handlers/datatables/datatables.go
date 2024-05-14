package datatables

import (
	"context"
	"embed"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	table_formatter "github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"github.com/kucherenkovova/safegroup"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"html/template"
	"io"
	"time"
)

// DataTables describes the data passed to  the template displaying the results of a glazed command.
// It's meant to be a more structured layer on top of the HTMLOutputTemplateFormatter
// that parka offers for having users provide their own template formatting.
type DataTables struct {
	Command *cmds.CommandDescription
	// LongDescription is the HTML of the rendered markdown of the long description of the command.
	LongDescription template.HTML

	Layout *layout.Layout
	Links  []layout.Link

	BasePath string

	// Stream provides a channel where each element represents a row of the table
	// to be rendered, already formatted.
	// Per default, we will render the individual rows as HTML, but the JSRendering
	// flag will make this output individual entries of a JS array.
	//
	// TODO(manuel, 2023-06-04): Maybe we could make this be an iterator of rows that provide access to the individual
	// columns for more interesting HTML shenanigans too.
	JSStream    chan template.JS
	HTMLStream  chan template.HTML
	ErrorStream chan string
	// Configuring the template to load the table data through javascript, and provide the data itself
	// as a JSON array inlined in the HTML of the page.
	JSRendering bool

	Columns []string

	// UseDataTables is using the datatables.net framework.
	// This is an opinionated way of proposing different table layouts and javascript functionality
	// (for now). If a user wants more advanced customization, they can use the HTMLTemplateOutputFormatter
	// or use this implementation for inspiration.
	UseDataTables bool

	// AdditionalData to be passed to the rendering engine
	AdditionalData map[string]interface{}
	// StreamRows is used to control whether the rows should be streamed or not.
	// If set to false, a TableMiddleware used (which collects all rows, but thus also all column names)
	// into memory before passing them to the template.
	// This is useful when the rows are "ragged" (i.e. not all rows have the same number of columns).
	StreamRows      bool
	CommandMetadata map[string]interface{}
}

func NewDataTables() *DataTables {
	return &DataTables{
		AdditionalData:  make(map[string]interface{}),
		CommandMetadata: make(map[string]interface{}),
	}
}

//go:embed templates/*
var templateFS embed.FS

func NewDataTablesLookupTemplate() *render.LookupTemplateFromFS {
	l := render.NewLookupTemplateFromFS(
		render.WithFS(templateFS),
		render.WithBaseDir("templates/"),
		render.WithPatterns("**/*.tmpl.html"),
	)

	_ = l.Reload()

	return l
}

func (dt *DataTables) Clone() *DataTables {
	ret := *dt
	return &ret
}

type QueryHandler struct {
	cmd         cmds.GlazeCommand
	middlewares []middlewares.Middleware

	templateName string
	lookup       render.TemplateLookup

	dt *DataTables
}

type QueryHandlerOption func(qh *QueryHandler)

func NewQueryHandler(
	cmd cmds.GlazeCommand,
	options ...QueryHandlerOption,
) *QueryHandler {
	qh := &QueryHandler{
		cmd:          cmd,
		dt:           NewDataTables(),
		lookup:       NewDataTablesLookupTemplate(),
		templateName: "data-tables.tmpl.html",
	}

	for _, option := range options {
		option(qh)
	}

	return qh
}

func WithDataTables(dt *DataTables) QueryHandlerOption {
	return func(qh *QueryHandler) {
		qh.dt = dt
	}
}

func WithMiddlewares(middlewares ...middlewares.Middleware) QueryHandlerOption {
	return func(qh *QueryHandler) {
		qh.middlewares = middlewares
	}
}

func WithTemplateLookup(lookup render.TemplateLookup) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.lookup = lookup
	}
}

func WithTemplateName(templateName string) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.templateName = templateName
	}
}

func WithAdditionalData(data map[string]interface{}) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.dt.AdditionalData = data
	}
}

func WithStreamRows(streamRows bool) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.dt.StreamRows = streamRows
	}
}

var _ handlers.Handler = &QueryHandler{}
var _ echo.HandlerFunc = (&QueryHandler{}).Handle

func (qh *QueryHandler) Handle(c echo.Context) error {
	description := qh.cmd.Description()
	parsedLayers := layers.NewParsedLayers()

	dt_ := qh.dt.Clone()
	// rowC is the channel where the rows are sent to. They will need to get converted
	// to template.JS or template.HTML before being sent to either
	rowC := make(chan string, 100)

	// buffered so that we don't hang on it when exiting
	dt_.ErrorStream = make(chan string, 10)

	// columnsC is the channel where the column names are sent to. Since the row.ColumnsChannelMiddleware
	// instance that sends columns to this channel is running before the row firmware, we should be careful
	// about not blocking. Potentially, this could be done by starting a goroutine in the middleware,
	// since we have a context there, and there is no need to block the middleware processing.
	columnsC := make(chan []types.FieldName, 10)

	err := middlewares.ExecuteMiddlewares(description.Layers, parsedLayers,
		append(
			qh.middlewares,
			parka_middlewares.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
			middlewares.SetFromDefaults(),
		)...,
	)

	if err != nil {
		log.Debug().Err(err).Msg("error executing middlewares")
		g := &safegroup.Group{}
		g.Go(func() error {
			if dt_.JSStream != nil {
				log.Debug().Msg("Closing JS stream")
				close(dt_.JSStream)
			}
			if dt_.HTMLStream != nil {
				log.Debug().Msg("Closing HTML stream")
				close(dt_.HTMLStream)
			}
			if dt_.ErrorStream != nil {
				v_ := err.Error()
				log.Debug().Str("errorString", v_).Msg("Sending error to error stream")
				select {
				case dt_.ErrorStream <- v_:
					log.Debug().Msg("Error sent to error stream")
				default:
					log.Debug().Msg("Error stream full")
				}
				log.Debug().Msg("Closing error stream")
				close(dt_.ErrorStream)
			}

			return nil
		})
		g.Go(func() error {
			log.Debug().Msg("Closing columns channel")
			columnsC <- []types.FieldName{}
			close(columnsC)
			log.Debug().Msg("Closing row channel")
			close(rowC)
			log.Debug().Msg("Closing columns channel")
			return nil
		})
		g.Go(func() error {
			log.Debug().Msg("Rendering template")
			return qh.renderTemplate(parsedLayers, c.Response(), dt_, columnsC)
		})

		return g.Wait()
	}

	// This needs to run after parsing the layers
	if cm_, ok := qh.cmd.(cmds.CommandWithMetadata); ok {
		dt_.CommandMetadata, err = cm_.Metadata(c.Request().Context(), parsedLayers)
		if err != nil {
			return err
		}
	}

	var of formatters.RowOutputFormatter
	if dt_.JSRendering {
		of = json.NewOutputFormatter(json.WithOutputIndividualRows(true))
		dt_.JSStream = make(chan template.JS, 100)
	} else {
		of = table_formatter.NewOutputFormatter("html")
		dt_.HTMLStream = make(chan template.HTML, 100)
	}

	// manually create a streaming output TableProcessor
	gp, err := handlers.CreateTableProcessorWithOutput(parsedLayers, "table", "ascii")
	if err != nil {
		return err
	}

	if dt_.StreamRows {
		gp.ReplaceTableMiddleware()
		gp.AddRowMiddleware(row.NewColumnsChannelMiddleware(columnsC, true))
		gp.AddRowMiddleware(row.NewOutputChannelMiddleware(of, rowC))
	} else {
		gp.AddTableMiddleware(table.NewColumnsChannelMiddleware(columnsC))
		gp.AddTableMiddleware(table.NewOutputChannelMiddleware(of, rowC))
	}

	ctx := c.Request().Context()
	ctx2, cancel := context.WithCancel(ctx)
	eg, ctx3 := safegroup.WithContext(ctx2)

	// copy the json rows to the template stream
	eg.Go(func() error {
		defer func() {
			if dt_.JSRendering {
				close(dt_.JSStream)
			} else {
				close(dt_.HTMLStream)
			}
		}()
		for {
			select {
			case <-ctx3.Done():
				return ctx3.Err()
			case row_, ok := <-rowC:
				// check if channel is closed
				if !ok {
					return nil
				}

				if dt_.JSRendering {
					dt_.JSStream <- template.JS(row_)
				} else {
					dt_.HTMLStream <- template.HTML(row_)
				}
			}
		}
	})

	// actually run the command
	eg.Go(func() error {
		defer func() {
			if dt_.ErrorStream != nil {
				close(dt_.ErrorStream)
				dt_.ErrorStream = nil
			}
			_ = cancel
		}()

		// NOTE(manuel, 2023-10-16) The GetAllParameterValues is a bit of a hack because really what we want is to only get those flags through the layers
		err = qh.cmd.RunIntoGlazeProcessor(ctx3, parsedLayers, gp)

		g := &safegroup.Group{}
		if err != nil {
			g.Go(func() error {
				// make sure to render the ErrorStream at the bottom, because we would otherwise get into a deadlock with the streaming channels
				// NOTE(manuel, 2024-05-14) I'm not sure if with the addition of goroutines this is actually still necessary
				//
				if dt_.ErrorStream != nil {
					dt_.ErrorStream <- err.Error()
					close(dt_.ErrorStream)
					dt_.ErrorStream = nil
				}
				return nil
			})
		}

		g.Go(func() error {
			err = gp.Close(ctx3)
			if err != nil {
				return err
			}

			close(rowC)
			close(columnsC)

			return nil
		})

		err := g.Wait()
		if err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		// if qh.Cmd implements cmds.CommandWithMetadata, get Metadata
		err := qh.renderTemplate(parsedLayers, c.Response(), dt_, columnsC)
		if err != nil {
			return err
		}

		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (qh *QueryHandler) renderTemplate(
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
	dt_ *DataTables,
	columnsC chan []types.FieldName,
) error {
	// Here, we use the parsed layer to configure the glazed middlewares.
	// We then use the created formatters.TableOutputFormatter as a basis for
	// our own output formatter that renders into an HTML template.
	var err error

	t, err := qh.lookup.Lookup(qh.templateName)
	if err != nil {
		return err
	}

	layout_, err := layout.ComputeLayout(qh.cmd, parsedLayers)
	if err != nil {
		return err
	}

	description := qh.cmd.Description()

	longHTML, err := render.RenderMarkdownToHTML(description.Long)
	if err != nil {
		return err
	}

	dt_.Layout = layout_
	dt_.LongDescription = template.HTML(longHTML)
	dt_.Command = description

	// Wait for the column names to be sent to the channel. This will only
	// take the first row into account.
	columns := <-columnsC
	dt_.Columns = columns

	// start copying from rowC to HTML or JS stream

	err = t.Execute(w, dt_)
	if err != nil {
		return err
	}

	return nil
}

func CreateDataTablesHandler(
	cmd cmds.GlazeCommand,
	basePath string,
	downloadPath string,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := cmd.Description().Name
		dateTime := time.Now().Format("2006-01-02--15-04-05")
		links := []layout.Link{
			{
				Href:  fmt.Sprintf("%s/%s-%s.csv", downloadPath, dateTime, name),
				Text:  "Download CSV",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/%s-%s.json", downloadPath, dateTime, name),
				Text:  "Download JSON",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/%s-%s.xlsx", downloadPath, dateTime, name),
				Text:  "Download Excel",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/%s-%s.md", downloadPath, dateTime, name),
				Text:  "Download Markdown",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/%s-%s.html", downloadPath, dateTime, name),
				Text:  "Download HTML",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/%s-%s.txt", downloadPath, dateTime, name),
				Text:  "Download Text",
				Class: "download",
			},
		}

		dt := NewDataTables()
		dt.Command = cmd.Description()
		dt.Links = links
		dt.BasePath = basePath
		dt.JSRendering = true
		dt.UseDataTables = false

		options_ := []QueryHandlerOption{
			WithDataTables(dt),
		}
		options_ = append(options_, options...)

		handler := NewQueryHandler(cmd, options_...)

		err := handler.Handle(c)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("error handling query")
			return err
		}

		return nil
	}
}
