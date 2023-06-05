package render

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"html/template"
	"io"
)

// NOTE(manuel, 2023-06-04): I don'Template think any of this is necessary.
//
// So it looks like the steps to output glazed data is to use a CreateProcessorFunc to create
// a processor.Processor. Here we create a processor that uses a HTMLTemplateOutputFormatter (which
// we are converting to a more specialized DataTableOutputFormatter), and then wrap all this through a
// HTMLTableProcessor. But really the HTMLTableProcessor is just there to wrap the output formatter and
// the template used. But the template used should be captured by the OutputFormatter in the first place.
//
// As such, we can use a generic Processor (why is there even a processor to be overloaded, if the definition of
// processor.Processor is the following:
//
//type Processor interface {
//	ProcessInputObject(ctx context.Context, obj map[string]interface{}) error
//	OutputFormatter() formatters.OutputFormatter
//}
//
// Probably because we use the processor.GlazeProcessor class as a helper, which is able to handle the different
// middlewares and output formatters. This means we can kill HTMLTemplateProcessor, capture the template in the
// HTMLTemplateOutputFormatter and then use the standard GlazeProcessor.

// HTMLTemplateOutputFormatter wraps a normal HTML table output formatter, and allows
// a template to be added in the back in the front.
type HTMLTemplateOutputFormatter struct {
	// We use a table outputFormatter because we need to access the Table itself.
	*table.OutputFormatter
	Template *template.Template
	Data     map[string]interface{}
}

type HTMLTemplateOutputFormatterOption func(*HTMLTemplateOutputFormatter)

func WithHTMLTemplateOutputFormatterData(data map[string]interface{}) HTMLTemplateOutputFormatterOption {
	return func(of *HTMLTemplateOutputFormatter) {
		if of.Data == nil {
			of.Data = map[string]interface{}{}
		}
		for k, v := range data {
			of.Data[k] = v
		}
	}
}

func NewHTMLTemplateOutputFormatter(
	t *template.Template,
	of *table.OutputFormatter,
	options ...HTMLTemplateOutputFormatterOption,
) *HTMLTemplateOutputFormatter {
	ret := &HTMLTemplateOutputFormatter{
		OutputFormatter: of,
		Template:        t,
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func (H *HTMLTemplateOutputFormatter) Output(ctx context.Context, w io.Writer) error {
	data := map[string]interface{}{}
	for k, v := range H.Data {

		data[k] = v
	}
	data["Columns"] = H.OutputFormatter.Table.Columns
	data["Table"] = H.OutputFormatter.Table

	err := H.Template.Execute(w, data)

	if err != nil {
		return err
	}

	return err
}

// NewHTMLTemplateLookupCreateProcessorFunc creates a glazed.CreateProcessorFunc based on a TemplateLookup
// and a template name.
func NewHTMLTemplateLookupCreateProcessorFunc(
	lookup TemplateLookup,
	templateName string,
	options ...HTMLTemplateOutputFormatterOption,
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

		// Here, we use the parsed layer to configure the glazed middlewares.
		// We then use the created formatters.OutputFormatter as a basis for
		// our own output formatter that renders into an HTML template.
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

		longHTML, err := RenderMarkdownToHTML(description.Long)
		if err != nil {
			return nil, contextType, err
		}

		options_ := []HTMLTemplateOutputFormatterOption{
			WithHTMLTemplateOutputFormatterData(
				map[string]interface{}{
					"Command":         description,
					"LongDescription": template.HTML(longHTML),
					"Layout":          layout_,
				}),
		}
		options_ = append(options_, options...)

		of := NewHTMLTemplateOutputFormatter(t, gp.OutputFormatter().(*table.OutputFormatter), options_...)
		gp2 := processor.NewGlazeProcessor(of)

		return gp2, contextType, nil
	}
}
