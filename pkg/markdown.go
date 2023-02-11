package pkg

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	html2 "github.com/yuin/goldmark/renderer/html"
	"html/template"
	"net/http"
	"os"
)

func (s *Server) renderTemplate(c *gin.Context, page string, data interface{}) {
	// check if markdown file exists
	markdownFile := s.TemplateDir + "/" + page + ".md"
	_, err := os.Stat(markdownFile)
	var t *template.Template

	if err == nil {
		markdown, err := s.renderMarkdownToHTML(markdownFile, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		t, err = template.ParseFiles(s.TemplateDir + "/base.tmpl.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err = t.Execute(c.Writer, map[string]interface{}{
			"markdown": template.HTML(markdown),
		})
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

	} else {
		t, err = template.ParseFiles(s.TemplateDir + "/" + page + ".tmpl.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err = t.Execute(c.Writer, data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	}

}

func (s *Server) renderMarkdownToHTML(file string, data interface{}) (string, error) {
	f, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	// parse as template
	t, err := template.New("markdown").Parse(string(f))
	if err != nil {
		return "", err
	}

	// execute template
	buf := new(bytes.Buffer)
	err = t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	rendered := buf.String()

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

	buf = new(bytes.Buffer)
	err = engine.Convert([]byte(rendered), buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
