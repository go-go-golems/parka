package render

import (
	"context"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render/layout"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"html/template"
	"io"
)

// This file contains a variety of output renderers for HTML output.
// The idea would to create a set of glazed.CreateProcessorFunc
// that would return an OutputFormatter() that can be used to render
// a command and a table as HTML.

// HTMLTemplateOutputFormatter wraps a normal HTML table output formatter, and allows
// a template to be added in the back in the front.
type HTMLTemplateOutputFormatter struct {
	*table.OutputFormatter
	t    *template.Template
	data map[string]interface{}
	// renderAsJavascript means that we will render the table into the toplevel element
	// `tableData` in javascript, and not call the parent output formatter
	renderAsJavascript bool
}

type HTMLTemplateOutputFormatterOption func(*HTMLTemplateOutputFormatter)

func WithJavascriptRendering() HTMLTemplateOutputFormatterOption {
	return func(of *HTMLTemplateOutputFormatter) {
		of.renderAsJavascript = true
	}
}

func WithHTMLTemplateOutputFormatterData(data map[string]interface{}) HTMLTemplateOutputFormatterOption {
	return func(of *HTMLTemplateOutputFormatter) {
		if of.data == nil {
			of.data = map[string]interface{}{}
		}
		for k, v := range data {
			of.data[k] = v
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
		t:               t,
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func StartFormatIntoChannel[T interface{ ~string }](
	ctx context.Context,
	formatter formatters.OutputFormatter,
) <-chan T {
	reader, writer := io.Pipe()
	c := make(chan T)

	eg, ctx2 := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer close(c)

		// read 8k chunks from reader
		buf := make([]byte, 8192)
		for {
			select {
			case <-ctx2.Done():
				return nil
			default:
				n, err := reader.Read(buf)
				if err != nil {
					return err
				}

				c <- T(buf[:n])
			}
		}
	})

	eg.Go(func() error {
		err := formatter.Output(ctx2, writer)
		defer writer.Close()
		if err != nil {
			writer.CloseWithError(err)
			return err
		}
		return nil
	})

	go func() {
		err := eg.Wait()
		if err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("error in stream formatter")
			}
		}
	}()

	return c
}

func (H *HTMLTemplateOutputFormatter) Output(ctx context.Context, w io.Writer) error {
	data := map[string]interface{}{}
	for k, v := range H.data {
		data[k] = v
	}
	data["Columns"] = H.OutputFormatter.Table.Columns
	data["Table"] = H.OutputFormatter.Table
	data["RenderAsJavascript"] = H.renderAsJavascript

	if H.renderAsJavascript {
		jsonOutputFormatter := json.NewOutputFormatter(json.WithTable(H.OutputFormatter.Table))
		c := StartFormatIntoChannel[template.JS](ctx, jsonOutputFormatter)
		data["JSTableStream"] = c
	} else {
		c := StartFormatIntoChannel[template.HTML](ctx, H.OutputFormatter)
		data["HTMLTableStream"] = c
	}

	err := H.t.Execute(w, data)

	if err != nil {
		return err
	}

	return err
}

type HTMLTemplateProcessor struct {
	*processor.GlazeProcessor

	of *HTMLTemplateOutputFormatter
}

func NewHTMLTemplateProcessor(
	gp *processor.GlazeProcessor,
	t *template.Template,
	options ...HTMLTemplateOutputFormatterOption,
) (*HTMLTemplateProcessor, error) {
	parentOf, ok := gp.OutputFormatter().(*table.OutputFormatter)
	if !ok {
		return nil, errors.New("parent output formatter is not a table output formatter")
	}

	of := NewHTMLTemplateOutputFormatter(t, parentOf, options...)

	ret := &HTMLTemplateProcessor{
		GlazeProcessor: gp,
		of:             of,
	}
	return ret, nil
}

func (H *HTMLTemplateProcessor) OutputFormatter() formatters.OutputFormatter {
	return H.of
}

// NewHTMLTemplateLookupCreateProcessorFunc creates a CreateProcessorFunc based on a TemplateLookup
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

		// lookup on every request, not up front.
		//
		// NOTE(manuel, 2023-04-19) This currently is nailed to a single static templateName passed at configuration time.
		// potentially, templateName could also be dynamic based on the incoming request, but we'll leave
		// that flexibility for later.
		t, err := lookup(templateName)
		if err != nil {
			return nil, contextType, err
		}

		// NOTE(manuel, 2023-04-18) We use glazed to render the actual HTML table.
		// But really, we could allow the user to specify the actual HTML rendering as well.
		// This is currently just a convenience to get started.
		//
		// They can actually do that by passing in their own glazed.CreateProcessorFunc.
		l, ok := pc.ParsedLayers["glazed"]
		l.Parameters["output"] = "table"
		l.Parameters["table-format"] = "html"

		var gp *processor.GlazeProcessor

		if ok {
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

		// NOTE(manuel, 2023-04-19) This layout structure is probably something we should extract out to glazed, as it can also be used for a bubbletea interface.
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

		gp2, err := NewHTMLTemplateProcessor(gp, t, options_...)
		if err != nil {
			return nil, contextType, err
		}
		return gp2, contextType, nil
	}
}

//go:embed templates/*
var templateFS embed.FS

func NewDataTablesHTMLTemplateCreateProcessorFunc(
	options ...HTMLTemplateOutputFormatterOption,
) (glazed.CreateProcessorFunc, error) {
	templateLookup, err := LookupTemplateFromFSReloadable(templateFS, "templates/", "templates/**/*.tmpl.html")
	if err != nil {
		return nil, err
	}

	return NewHTMLTemplateLookupCreateProcessorFunc(templateLookup, "data-tables.tmpl.html", options...), nil
}
