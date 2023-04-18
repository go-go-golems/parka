package render

import (
	"bytes"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/pkg/errors"
	"html/template"
)

// This file contains a variety of output renderers for HTML output.
// The idea would to create a set of glazed.CreateProcessorFunc
// that would return an OutputFormatter() that can be used to render
// a command and a table as HTML.

// HTMLTemplateOutputFormatter wraps a normal HTML table output formatter, and allows
// a template to be added in the back in the front.
type HTMLTemplateOutputFormatter struct {
	*table.OutputFormatter

	t *template.Template
}

func NewHTMLTemplateOutputFormatter(t *template.Template, of *table.OutputFormatter) *HTMLTemplateOutputFormatter {
	return &HTMLTemplateOutputFormatter{
		OutputFormatter: of,
		t:               t,
	}
}

func (H *HTMLTemplateOutputFormatter) Output() (string, error) {
	res, err := H.OutputFormatter.Output()
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = H.t.Execute(buf, map[string]interface{}{
		"Table": template.HTML(res),
	})

	if err != nil {
		return "", err
	}

	return buf.String(), err
}

type HTMLTemplateProcessor struct {
	*cmds.GlazeProcessor

	of *HTMLTemplateOutputFormatter
}

func NewHTMLTemplateProcessor(gp *cmds.GlazeProcessor, t *template.Template) (*HTMLTemplateProcessor, error) {
	parentOf, ok := gp.OutputFormatter().(*table.OutputFormatter)
	if !ok {
		return nil, errors.New("parent output formatter is not a table output formatter")
	}

	of := NewHTMLTemplateOutputFormatter(t, parentOf)
	return &HTMLTemplateProcessor{
		GlazeProcessor: gp,
		of:             of,
	}, nil
}

func (H *HTMLTemplateProcessor) OutputFormatter() formatters.OutputFormatter {
	return H.of
}

//go:embed templates/*
var templateFS embed.FS

func RenderDataTables(c *gin.Context, pc *glazed.CommandContext) (
	cmds.Processor,
	string, // content type
	error,
) {
	contextType := "text/html"

	l, ok := pc.ParsedLayers["glazed"]
	l.Parameters["output"] = "table"
	l.Parameters["table-format"] = "html"

	var gp *cmds.GlazeProcessor
	var err error

	if ok {
		gp, err = cli.SetupProcessor(l.Parameters)
	} else {
		gp, err = cli.SetupProcessor(map[string]interface{}{
			"output":       "table",
			"table-format": "html",
		})
	}

	if err != nil {
		return nil, contextType, err
	}

	// NOTE(manuel, 2023-04-18) This loading could potentially be done outside of the Render call (such as not to load this on every call)
	// Potentially, we should use the Watcher and implement a watcher that reloads templates on demand...
	//
	// See https://github.com/go-go-golems/parka/issues/26
	t := templating.CreateHTMLTemplate("data-tables")
	err = templating.ParseHTMLFS(t, templateFS, "templates/**/*.tmpl.html", "templates/")
	if err != nil {
		return nil, contextType, err
	}

	tTables := t.Lookup("data-tables.tmpl.html")
	if tTables == nil {
		return nil, contextType, errors.New("could not find data-tables template")
	}

	gp2, err := NewHTMLTemplateProcessor(gp, tTables)
	if err != nil {
		return nil, contextType, err
	}
	return gp2, contextType, nil
}
