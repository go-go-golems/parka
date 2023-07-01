package render

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"html/template"
	"io"
)

// NOTE(manuel, 2023-06-04): I don'Template think any of this is necessary.
//
// So it looks like the steps to output glazed data is to use a CreateProcessorFunc to create
// a processor.TableProcessor. Here we create a processor that uses a HTMLTemplateOutputFormatter (which
// we are converting to a more specialized DataTableOutputFormatter), and then wrap all this through a
// HTMLTableProcessor. But really the HTMLTableProcessor is just there to wrap the output formatter and
// the template used. But the template used should be captured by the TableOutputFormatter in the first place.
//
// As such, we can use a generic TableProcessor (why is there even a processor to be overloaded, if the definition of
// processor.TableProcessor is the following:
//
//type TableProcessor interface {
//	AddRow(ctx context.Context, obj map[string]interface{}) error
//	TableOutputFormatter() formatters.TableOutputFormatter
//}
//
// Probably because we use the processor.GlazeProcessor class as a helper, which is able to handle the different
// middlewares and output formatters. This means we can kill HTMLTemplateProcessor, capture the template in the
// HTMLTemplateOutputFormatter and then use the standard GlazeProcessor.

// HTMLTemplateOutputFormatter wraps a normal HTML table output formatter, and allows
// a template to be added in the back in the front.
type HTMLTemplateOutputFormatter struct {
	TemplateName string
	Lookup       TemplateLookup
	Data         map[string]interface{}
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
	lookup TemplateLookup,
	templateName string,
	options ...HTMLTemplateOutputFormatterOption,
) *HTMLTemplateOutputFormatter {
	ret := &HTMLTemplateOutputFormatter{
		TemplateName: templateName,
		Lookup:       lookup,
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func (H *HTMLTemplateOutputFormatter) Output(c *gin.Context, pc *glazed.CommandContext, w io.Writer) error {
	// Here, we use the parsed layer to configure the glazed middlewares.
	// We then use the created formatters.TableOutputFormatter as a basis for
	// our own output formatter that renders into an HTML template.
	var err error

	layout_, err := layout.ComputeLayout(pc)
	if err != nil {
		return err
	}

	description := pc.Cmd.Description()

	longHTML, err := RenderMarkdownToHTML(description.Long)
	if err != nil {
		return err
	}

	data := map[string]interface{}{}
	for k, v := range H.Data {
		data[k] = v
	}
	data["Command"] = description
	data["LongDescription"] = template.HTML(longHTML)
	data["Layout"] = layout_

	// TODO(manuel, 2023-06-30) Get the column names out of a RowOutputMiddleware
	//data["Columns"] = table.Columns

	t, err := H.Lookup.Lookup(H.TemplateName)
	if err != nil {
		return err
	}

	// TODO: we are missing the background processing of the rows here

	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return err
}
