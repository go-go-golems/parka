package pkg

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	html2 "github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io/fs"
	"net/http"
)

func (s *Server) LoadTemplateFS(_fs fs.FS, patterns ...string) (*template.Template, error) {
	tmpl := helpers.CreateHTMLTemplate("")
	t, err := tmpl.ParseFS(_fs, patterns...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *Server) renderMarkdownTemplate(c *gin.Context, page string, data interface{}) {
	// check if markdown file exists
	t := s.Template.Lookup(page + ".tmpl.md")
	if t == nil {
		t = s.ParkaTemplate.Lookup(page + ".tmpl.md")
	}

	if t == nil {
		t = s.Template.Lookup(page + ".md")
		if t == nil {
			t = s.ParkaTemplate.Lookup(page + ".md")
		}
	}

	if t != nil {
		markdown, err := s.renderMarkdownTemplateToHTML(t, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		err = s.ParkaTemplate.ExecuteTemplate(
			c.Writer,
			"base.tmpl.html",
			map[string]interface{}{
				"markdown": template.HTML(markdown),
			})
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	} else {
		t = s.Template.Lookup(page + ".tmpl.html")
		if t == nil {
			t = s.ParkaTemplate.Lookup(page + ".tmpl.html")
		}
		if t == nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err := t.Execute(c.Writer, data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	}
}

func (s *Server) renderMarkdownTemplateToHTML(t *template.Template, data interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	rendered := buf.String()

	return s.renderMarkdownToHTML(rendered)
}

func (s *Server) renderMarkdownToHTML(rendered string) (string, error) {
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
