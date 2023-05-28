package render

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	html2 "github.com/yuin/goldmark/renderer/html"
	"html/template"
)

func RenderMarkdownTemplateToHTML(t *template.Template, data interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	rendered := buf.String()

	return RenderMarkdownToHTML(rendered)
}

func RenderMarkdownToHTML(rendered string) (string, error) {
	engine := goldmark.New(
		goldmark.WithExtensions(
			// add tables
			extension.NewTable(),
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithRendererOptions(
			html2.WithUnsafe()))

	buf := new(bytes.Buffer)
	err := engine.Convert([]byte(rendered), buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
