package render

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	html2 "github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io/fs"
	"os"
	"strings"
)

// TemplateLookup is a function that will lookup a template by name.
// It is use as an interface to allow different ways of loading templates to be provided
// to a parka application.
type TemplateLookup func(name ...string) (*template.Template, error)

// LookupTemplateFromDirectory will load a template at runtime. This is useful
// for testing local changes to templates without having to recompile the app.
func LookupTemplateFromDirectory(dir string) TemplateLookup {
	return func(name ...string) (*template.Template, error) {
		for _, n := range name {
			fileName := dir + "/" + n
			// lookup in s.devTemplateDir
			_, err := os.Stat(fileName)
			if err == nil {
				b, err := os.ReadFile(fileName)
				if err != nil {
					return nil, err
				}
				t, err := templating.CreateHTMLTemplate("").Parse(string(b))
				if err != nil {
					return nil, err
				}

				return t, nil
			}
		}

		return nil, errors.New("template not found")
	}
}

// LookupTemplateFromFS will load a template from a fs.FS.
func LookupTemplateFromFS(_fs fs.FS, baseDir string, patterns ...string) (TemplateLookup, error) {
	tmpl, err := LoadTemplateFS(_fs, baseDir, patterns...)
	if err != nil {
		return nil, err
	}

	return func(name ...string) (*template.Template, error) {
		for _, n := range name {
			t := tmpl.Lookup(n)
			if t != nil {
				return t, nil
			}
		}

		return nil, errors.New("template not found")
	}, nil
}

// LoadTemplateFS will load a template from a fs.FS.
func LoadTemplateFS(_fs fs.FS, baseDir string, patterns ...string) (*template.Template, error) {
	if !strings.HasSuffix(baseDir, "/") {
		baseDir += "/"
	}
	tmpl := templating.CreateHTMLTemplate("")
	var err error
	for _, p := range patterns {
		err = templating.ParseHTMLFS(tmpl, _fs, p, baseDir)
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

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
