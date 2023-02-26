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
	"os"
)

func (s *Server) LoadTemplateFS(_fs fs.FS, patterns ...string) (*template.Template, error) {
	tmpl := helpers.CreateHTMLTemplate("")
	t, err := tmpl.ParseFS(_fs, patterns...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *Server) LookupTemplate(name ...string) (*template.Template, error) {
	var t *template.Template

	if s.devMode {
		possibleFileNames := []string{}
		for _, n := range name {
			possibleFileNames = append(possibleFileNames,
				s.devTemplateDir+"/"+n,
				s.devParkaTemplateDir+"/"+n,
			)
		}

		for _, fileName := range possibleFileNames {
			// lookup in s.devTemplateDir
			_, err := os.Stat(fileName)
			if err == nil {
				b, err := os.ReadFile(fileName)
				if err != nil {
					return nil, err
				}
				t, err = helpers.CreateHTMLTemplate("").Parse(string(b))
				if err != nil {
					return nil, err
				}

				return t, nil
			}
		}
	} else {
		for _, n := range name {
			// check if markdown file exists
			t = s.Template.Lookup(n)
			if t == nil {
				t = s.ParkaTemplate.Lookup(n)
			}
			if t != nil {
				break
			}
		}
	}

	return t, nil

}

func (s *Server) serveMarkdownTemplate(c *gin.Context, page string, data interface{}) {
	t, err := s.LookupTemplate(page+".tmpl.md", page+".md")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}

	if t != nil {
		markdown, err := s.RenderMarkdownTemplateToHTML(t, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		baseTemplate, err := s.LookupTemplate("base.tmpl.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err = baseTemplate.Execute(
			c.Writer,
			map[string]interface{}{
				"markdown": template.HTML(markdown),
			})
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	} else {
		t, err = s.LookupTemplate(page+".tmpl.html", page+".html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
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

func (s *Server) RenderMarkdownTemplateToHTML(t *template.Template, data interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	rendered := buf.String()

	return s.RenderMarkdownToHTML(rendered)
}

func (s *Server) RenderMarkdownToHTML(rendered string) (string, error) {
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
