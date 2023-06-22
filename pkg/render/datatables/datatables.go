package datatables

import (
	"context"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"html/template"
	"io"
)

// DataTables describes the data passed to  the template displaying the results of a glazed command.
// It's meant to be a more structured layer on top of the HTMLOutputTemplateFormatter
// that parka offers for having users provide their own template formatting.
type DataTables struct {
	Command *cmds.CommandDescription
	// LongDescription is the HTML of the rendered markdown of the long description of the command.
	LongDescription string

	Layout *layout.Layout
	Links  []layout.Link

	// Stream provides a channel where each element represents a row of the table
	// to be rendered, already formatted.
	// Per default, we will render the individual rows as HTML, but the JSRendering
	// flag will make this output individual entries of a JS array.
	//
	// TODO(manuel, 2023-06-04): Maybe we could make this be an iterator of rows that provide access to the individual
	// columns for more interesting HTML shenanigans too.
	JSStream   <-chan template.JS
	HTMLStream <-chan template.HTML
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

type DataTablesOutputFormatter struct {
	*render.HTMLTemplateOutputFormatter
	dataTablesData *DataTables
}

type DataTablesOutputFormatterOption func(*DataTablesOutputFormatter)

func WithCommand(cmd *cmds.CommandDescription) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Command = cmd
	}
}

func WithLongDescription(desc string) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.LongDescription = desc
	}
}

func WithReplaceAdditionalData(data map[string]interface{}) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.AdditionalData = data
	}
}

func WithAdditionalData(data map[string]interface{}) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		if d.dataTablesData.AdditionalData == nil {
			d.dataTablesData.AdditionalData = data
		} else {
			for k, v := range data {
				d.dataTablesData.AdditionalData[k] = v
			}
		}
	}
}

// WithJSRendering enables JS rendering for the DataTables renderer.
// This means that we will render the table into the toplevel element
// `tableData` in javascript, and not call the parent output formatter
func WithJSRendering() DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.JSRendering = true
	}
}

func WithLayout(l *layout.Layout) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Layout = l
	}
}

func WithLinks(links ...layout.Link) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Links = links
	}
}

func WithAppendLinks(links ...layout.Link) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Links = append(d.dataTablesData.Links, links...)
	}
}

func WithPrependLinks(links ...layout.Link) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Links = append(links, d.dataTablesData.Links...)
	}
}

func WithColumns(columns ...string) DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.Columns = columns
	}
}

func WithUseDataTables() DataTablesOutputFormatterOption {
	return func(d *DataTablesOutputFormatter) {
		d.dataTablesData.UseDataTables = true
	}
}

//go:embed templates/*
var templateFS embed.FS

func NewDataTablesHTMLTemplateCreateProcessorFunc(
	options ...render.HTMLTemplateOutputFormatterOption,
) (glazed.CreateProcessorFunc, error) {
	l := NewDataTablesLookupTemplate()
	return render.NewHTMLTemplateLookupCreateProcessorFunc(l, "data-tables.tmpl.html", options...), nil
}

func NewDataTablesLookupTemplate() *render.LookupTemplateFromFS {
	l := render.NewLookupTemplateFromFS(
		render.WithFS(templateFS),
		render.WithBaseDir("templates/"),
		render.WithPatterns("**/*.tmpl.html"),
	)

	_ = l.Reload()

	return l
}

func NewDataTablesOutputFormatter(
	t *template.Template,
	of *table.OutputFormatter,
	options ...DataTablesOutputFormatterOption,
) *DataTablesOutputFormatter {
	ret := &DataTablesOutputFormatter{
		HTMLTemplateOutputFormatter: render.NewHTMLTemplateOutputFormatter(t, of),
		dataTablesData:              &DataTables{},
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func (d *DataTablesOutputFormatter) Output(ctx context.Context, w io.Writer) error {
	dt := d.dataTablesData
	if dt.JSRendering {
		jsonOutputFormatter := json.NewOutputFormatter(json.WithTable(d.OutputFormatter.Table))
		dt.JSStream = formatters.StartFormatIntoChannel[template.JS](ctx, jsonOutputFormatter)
	} else {
		dt.HTMLStream = formatters.StartFormatIntoChannel[template.HTML](ctx, d.OutputFormatter)
	}

	// TODO(manuel, 2023-06-20) We need to properly pass the columns here, which can't be set upstream
	// since we already pass in the JSStream here and we keep it, I think we are better off cloning the
	// DataTables struct, or even separating it out to make d.dataTablesData immutable and just contain the
	// toplevel config.
	if d.OutputFormatter.Table != nil {
		dt.Columns = d.OutputFormatter.Table.Columns
	}

	err := d.HTMLTemplateOutputFormatter.Template.Execute(w, dt)

	if err != nil {
		return err
	}

	return nil
}

// NewDataTablesCreateOutputProcessorFunc creates a glazed.CreateProcessorFunc based on a TemplateLookup
// and a template name.
func NewDataTablesCreateOutputProcessorFunc(
	lookup render.TemplateLookup,
	templateName string,
	options ...DataTablesOutputFormatterOption,
) glazed.CreateProcessorFunc {
	return func(c *gin.Context, pc *glazed.CommandContext) (
		processor.Processor,
		string, // content type
		error,
	) {
		contextType := "text/html"

		// Lookup template on every request, not up front. That way, templates can be reloaded without recreating the gin
		// server.
		t, err := lookup.Lookup(templateName)
		if err != nil {
			return nil, contextType, err
		}

		l, ok := pc.ParsedLayers["glazed"]
		var gp *processor.GlazeProcessor

		if ok {
			l.Parameters["output"] = "table"
			l.Parameters["table-format"] = "html"

			gp, err = settings.SetupProcessor(l.Parameters)
		} else {
			gp, err = settings.SetupProcessor(map[string]interface{}{
				"output":       "table",
				"table-format": "html",
			})
		}

		if err != nil {
			return nil, contextType, err
		}

		layout_, err := layout.ComputeLayout(pc)
		if err != nil {
			return nil, contextType, err
		}

		description := pc.Cmd.Description()

		longHTML, err := render.RenderMarkdownToHTML(description.Long)
		if err != nil {
			return nil, contextType, err
		}

		options_ := []DataTablesOutputFormatterOption{
			WithCommand(description),
			WithLongDescription(longHTML),
			WithLayout(layout_),
		}
		options_ = append(options_, options...)

		of := NewDataTablesOutputFormatter(
			t,
			gp.OutputFormatter().(*table.OutputFormatter),
			options_...,
		)

		gp2 := processor.NewGlazeProcessor(of)

		return gp2, contextType, nil
	}
}
