package datatables

import (
	"context"
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	table_formatter "github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
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
	cmd                cmds.GlazeCommand
	contextMiddlewares []glazed.ContextMiddleware
	parserOptions      []parser.ParserOption

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

func WithContextMiddlewares(middlewares ...glazed.ContextMiddleware) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.contextMiddlewares = middlewares
	}
}

// WithParserOptions sets the parser options for the QueryHandler
func WithParserOptions(options ...parser.ParserOption) QueryHandlerOption {
	return func(h *QueryHandler) {
		h.parserOptions = options
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

func (qh *QueryHandler) Handle(c *gin.Context, w io.Writer) error {
	pc := glazed.NewCommandContext(qh.cmd)

	qh.contextMiddlewares = append(
		qh.contextMiddlewares,
		glazed.NewContextParserMiddleware(
			qh.cmd,
			glazed.NewCommandQueryParser(qh.cmd, qh.parserOptions...),
		),
	)

	for _, h := range qh.contextMiddlewares {
		err := h.Handle(c, pc)
		if err != nil {
			return err
		}
	}

	// rowC is the channel where the rows are sent to. They will need to get converted
	// to template.JS or template.HTML before being sent to either
	rowC := make(chan string, 100)

	// columnsC is the channel where the column names are sent to. Since the row.ColumnsChannelMiddleware
	// instance that sends columns to this channel is running before the row firmware, we should be careful
	// about not blocking. Potentially, this could be done by starting a goroutine in the middleware,
	// since we have a context there, and there is no need to block the middleware processing.
	columnsC := make(chan []types.FieldName, 10)

	dt_ := qh.dt.Clone()
	var of formatters.RowOutputFormatter
	// buffered so that we don't hang on it when exciting
	dt_.ErrorStream = make(chan string, 1)
	if dt_.JSRendering {
		of = json.NewOutputFormatter(json.WithOutputIndividualRows(true))
		dt_.JSStream = make(chan template.JS, 100)
	} else {
		of = table_formatter.NewOutputFormatter("html")
		dt_.HTMLStream = make(chan template.HTML, 100)
	}

	// manually create a streaming output TableProcessor
	gp, err := handlers.CreateTableProcessorWithOutput(pc, "table", "ascii")
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

	ctx := c.Request.Context()
	ctx2, cancel := context.WithCancel(ctx)
	eg, ctx3 := errgroup.WithContext(ctx2)

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
			close(rowC)
			close(columnsC)
			close(dt_.ErrorStream)
			_ = cancel
		}()

		err = qh.cmd.Run(ctx3, pc.ParsedLayers, pc.ParsedParameters, gp)
		if err != nil {
			dt_.ErrorStream <- err.Error()
			return err
		}

		err = gp.Close(ctx3)
		if err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		// if qh.Cmd implements cmds.CommandWithMetadata, get Metadata
		if cm_, ok := qh.cmd.(cmds.CommandWithMetadata); ok {
			dt_.CommandMetadata, err = cm_.Metadata(c, pc.ParsedLayers, pc.ParsedParameters)
		}
		err := qh.renderTemplate(c, pc, w, dt_, columnsC)
		if err != nil {
			return err
		}

		return nil
	})

	return eg.Wait()
}

func (qh *QueryHandler) renderTemplate(
	c *gin.Context,
	pc *glazed.CommandContext,
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

	layout_, err := layout.ComputeLayout(pc)
	if err != nil {
		return err
	}

	description := pc.Cmd.Description()

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
	path string,
	commandPath string,
	options ...QueryHandlerOption,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := cmd.Description().Name
		dateTime := time.Now().Format("2006-01-02--15-04-05")
		links := []layout.Link{
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.csv", path, commandPath, dateTime, name),
				Text:  "Download CSV",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.json", path, commandPath, dateTime, name),
				Text:  "Download JSON",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.xlsx", path, commandPath, dateTime, name),
				Text:  "Download Excel",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.md", path, commandPath, dateTime, name),
				Text:  "Download Markdown",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.html", path, commandPath, dateTime, name),
				Text:  "Download HTML",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("%s/download/%s/%s-%s.txt", path, commandPath, dateTime, name),
				Text:  "Download Text",
				Class: "download",
			},
		}

		dt := NewDataTables()
		dt.Command = cmd.Description()
		dt.Links = links
		dt.BasePath = path
		dt.JSRendering = true
		dt.UseDataTables = false

		options_ := []QueryHandlerOption{
			WithDataTables(dt),
		}
		options_ = append(options_, options...)

		handler := NewQueryHandler(cmd, options_...)

		err := handler.Handle(c, c.Writer)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("error handling query")
		}
	}
}
