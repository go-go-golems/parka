package datatables

import (
	"context"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	table_formatter "github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"golang.org/x/sync/errgroup"
	"html/template"
	"io"
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
	JSStream   chan template.JS
	HTMLStream chan template.HTML
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
	return &DataTables{
		Command:         dt.Command,
		LongDescription: dt.LongDescription,
		Layout:          dt.Layout,
		Links:           dt.Links,
		BasePath:        dt.BasePath,
		JSStream:        dt.JSStream,
		HTMLStream:      dt.HTMLStream,
		JSRendering:     dt.JSRendering,
		Columns:         dt.Columns,
		UseDataTables:   dt.UseDataTables,
		AdditionalData:  dt.AdditionalData,
	}
}

type OutputFormatter struct {
	t  *template.Template
	dt *DataTables
	// rowC is the channel where the rows are sent to. They will need to get converted
	// to template.JS or template.HTML before being sent to either
	rowC chan string
	// columnsC is the channel where the column names are sent to. Since the row.ColumnsChannelMiddleware
	// instance that sends columns to this channel is running before the row firmware, we should be careful
	// about not blocking. Potentially, this could be done by starting a goroutine in the middleware,
	// since we have a context there, and there is no need to block the middleware processing.
	columnsC chan []types.FieldName
}

func NewOutputFormatter(
	t *template.Template,
	dt *DataTables) *OutputFormatter {

	// make the NewOutputChannelMiddleware generic to send string/template.JS/template.HTML over the wire
	rowC := make(chan string, 100)

	// make a channel to receive column names
	columnsC := make(chan []types.FieldName, 10)

	// we need to make sure that we are closing the channel correctly. Should middlewares have a Close method?
	// that actually sounds reasonable
	return &OutputFormatter{
		t:        t,
		dt:       dt,
		rowC:     rowC,
		columnsC: columnsC,
	}
}

func (of *OutputFormatter) Close(ctx context.Context) error {
	close(of.rowC)
	close(of.columnsC)
	return nil
}

func (dt *OutputFormatter) RegisterMiddlewares(p *middlewares.TableProcessor, writer io.Writer) error {
	var of formatters.RowOutputFormatter
	if dt.dt.JSRendering {
		of = json.NewOutputFormatter()
		dt.dt.JSStream = make(chan template.JS, 100)
	} else {
		of = table_formatter.NewOutputFormatter("html")
		dt.dt.HTMLStream = make(chan template.HTML, 100)
	}

	p.AddRowMiddleware(row.NewColumnsChannelMiddleware(dt.columnsC, true))
	p.AddRowMiddleware(row.NewOutputChannelMiddleware(of, dt.rowC))

	return nil
}

func (dt *OutputFormatter) Output(c *gin.Context, pc *glazed.CommandContext, w io.Writer) error {
	// Here, we use the parsed layer to configure the glazed middlewares.
	// We then use the created formatters.TableOutputFormatter as a basis for
	// our own output formatter that renders into an HTML template.
	var err error

	layout_, err := layout.ComputeLayout(pc)
	if err != nil {
		return err
	}

	description := pc.Cmd.Description()

	longHTML, err := render.RenderMarkdownToHTML(description.Long)
	if err != nil {
		return err
	}

	dt_ := dt.dt.Clone()
	dt_.Layout = layout_
	dt_.LongDescription = template.HTML(longHTML)
	dt_.Command = description

	// Wait for the column names to be sent to the channel. This will only
	// take the first row into account.
	columns := <-dt.columnsC
	dt_.Columns = columns

	// start copying from rowC to HTML or JS stream

	eg, ctx2 := errgroup.WithContext(c)

	eg.Go(func() error {
		err := dt.t.Execute(w, dt_)
		if err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		for {
			select {
			case <-ctx2.Done():
				return ctx2.Err()
			case row_ := <-dt.rowC:
				if dt.dt.JSRendering {
					dt.dt.JSStream <- template.JS(row_)
				} else {
					dt.dt.HTMLStream <- template.HTML(row_)
				}
			}
		}
	})

	return eg.Wait()
}

type OutputFormatterFactory struct {
	TemplateName string
	Lookup       render.TemplateLookup
	DataTables   *DataTables
}

func (dtoff *OutputFormatterFactory) CreateOutputFormatter(
	c *gin.Context,
	pc *glazed.CommandContext,
) (*OutputFormatter, error) {
	// Lookup template on every request, not up front. That way, templates can be reloaded without recreating the gin
	// server.
	t, err := dtoff.Lookup.Lookup(dtoff.TemplateName)
	if err != nil {
		return nil, err
	}

	layout_, err := layout.ComputeLayout(pc)
	if err != nil {
		return nil, err
	}

	description := pc.Cmd.Description()
	dt_ := dtoff.DataTables.Clone()

	longHTML, err := render.RenderMarkdownToHTML(description.Long)
	if err != nil {
		return nil, err
	}

	dt_.LongDescription = template.HTML(longHTML)

	dt_.Layout = layout_

	return NewOutputFormatter(t, dtoff.DataTables), nil
}

func NewOutputFormatterFactory(
	lookup render.TemplateLookup,
	templateName string,
	dataTables *DataTables,
) *OutputFormatterFactory {
	return &OutputFormatterFactory{
		TemplateName: templateName,
		Lookup:       lookup,
		DataTables:   dataTables,
	}
}
