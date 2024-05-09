// Package render provides an interface for rendering HTML templates and
// implementations for loading templates from directories or filesystems.
// It supports template lookup, reloading, and efficient template handling.
package render

import (
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"html/template"
	"io/fs"
	"os"
	"path"
	"strings"
)

// TemplateLookup is an interface for objects that can lookup a template by name.
// It is use as an interface to allow different ways of loading templates to be provided
// to a parka application.
type TemplateLookup interface {
	// Lookup returns a template by name. If there are multiple names given,
	// implementations may choose how to handle them.
	Lookup(name ...string) (*template.Template, error)

	// Reload reloads all or partial templates (necessary to render the given templates).
	// `name` can easily be ignored if the implementation doesn'Template support partial reloading.
	// This is useful for example to have server expose a specific route that reloads
	// all templates in case new files get uploaded, without having to restart the server.
	// This is also useful for development, where partial reloading is probably less important
	// and performance is not paramount, and as such a full reload on every request could be configured.
	Reload(name ...string) error
}

// LookupTemplateFromFile will load a template from a filesystem. It will always return
// the content of that file when queried, independent of the actual name requested.
type LookupTemplateFromFile struct {
	File string
	// The templateName to respond to, if empty, all templates request will return the file content.
	TemplateName string
}

func NewLookupTemplateFromFile(file string, templateName string) *LookupTemplateFromFile {
	return &LookupTemplateFromFile{
		File:         file,
		TemplateName: templateName,
	}
}

func (l *LookupTemplateFromFile) Lookup(name ...string) (*template.Template, error) {
	b, err := os.ReadFile(l.File)
	if err != nil {
		return nil, err
	}

	for _, name_ := range name {
		if l.TemplateName == "" || l.TemplateName == name_ {
			templateName := path.Base(l.File)
			t, err := templating.CreateHTMLTemplate(templateName).Parse(string(b))
			if err != nil {
				return nil, err
			}

			return t, nil
		}
	}

	return nil, errors.Errorf("template %s not found", name)
}

func (l *LookupTemplateFromFile) Reload(name ...string) error {
	return nil
}

// LookupTemplateFromDirectory will load a template at runtime. This is useful
// for testing local changes to templates without having to recompile the app.
type LookupTemplateFromDirectory struct {
	Directory string
}

func NewLookupTemplateFromDirectory(directory string) *LookupTemplateFromDirectory {
	return &LookupTemplateFromDirectory{
		Directory: directory,
	}
}

// Lookup will load the matching template file from the given Directory.
// This loads the template file on every lookup, and is thus not very efficient.
//
// The names passed as arguments correspond to the relative paths of the parsed
// files starting from configured template directory.
//
// TODO(manuel, 2023-05-28) Implement a reloadable LookupTemplateFromDirectory
// This would distract too much from the current task which is to implement the high-level
// surface TemplateLookup abstraction and then write up the parka ConfigFile code and documentation.
//
// See https://github.com/go-go-golems/parka/issues/49
func (l *LookupTemplateFromDirectory) Lookup(name ...string) (*template.Template, error) {
	dir := l.Directory
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	for _, n := range name {
		fileName := dir + n
		// lookup in s.devTemplateDir
		_, err := os.Stat(fileName)
		if err == nil {
			b, err := os.ReadFile(fileName)
			if err != nil {
				return nil, err
			}

			templateName := strings.TrimPrefix(fileName, dir)
			t, err := templating.CreateHTMLTemplate(templateName).Parse(string(b))
			if err != nil {
				return nil, err
			}

			return t, nil
		}
	}

	return nil, errors.New("templateFS not found")
}

// Reload has an empty implementation, since the lookup happens on every request.
func (l *LookupTemplateFromDirectory) Reload(_ ...string) error {
	return nil
}

// LookupTemplateFromFS will load a template from a fs.FS.
//
// NOTE: this loads the entire template directory into memory on every lookup.
// This is not great for performance, but it is useful for development.
//
// Per default, this exposes the current directory and uses the *html string pattern
// to load HTML templates.
type LookupTemplateFromFS struct {
	_fs          fs.FS
	baseDir      string
	patterns     []string
	alwaysReload bool
	tmpl         *template.Template
}

// Example usage:
//
//   lookup := NewLookupTemplateFromFS(
//     WithAlwaysReload(true),
//     WithFS(os.DirFS("./templates")),
//     WithBaseDir("base"),
//     WithPatterns("*.html"),
//   )
//   tmpl, err := lookup.Lookup("index.html")

type LookupTemplateFromFSOption func(*LookupTemplateFromFS)

func WithAlwaysReload(alwaysReload bool) LookupTemplateFromFSOption {
	return func(l *LookupTemplateFromFS) {
		l.alwaysReload = alwaysReload
	}
}

func WithFS(_fs fs.FS) LookupTemplateFromFSOption {
	return func(l *LookupTemplateFromFS) {
		l._fs = _fs
	}
}

func WithBaseDir(baseDir string) LookupTemplateFromFSOption {
	return func(l *LookupTemplateFromFS) {
		l.baseDir = baseDir
	}
}

func WithPatterns(patterns ...string) LookupTemplateFromFSOption {
	return func(l *LookupTemplateFromFS) {
		l.patterns = patterns
	}
}

func NewLookupTemplateFromFS(options ...LookupTemplateFromFSOption) *LookupTemplateFromFS {
	ret := &LookupTemplateFromFS{
		_fs:      os.DirFS("."),
		baseDir:  "",
		patterns: []string{"**/*.html"},
	}

	for _, o := range options {
		o(ret)
	}

	return ret
}

// Reload all templates from the fs / basedir. This ignores the partial reload, so
// depending on your setup, be mindful of the performance impact.
func (l *LookupTemplateFromFS) Reload(name ...string) error {
	tmpl, err := LoadTemplateFS(l._fs, l.baseDir, l.patterns...)
	if err != nil {
		return err
	}
	l.tmpl = tmpl
	return nil
}

// Lookup a template by name. If alwaysReload is specific, this will reload the entire
// template directory on every lookup.
func (l *LookupTemplateFromFS) Lookup(name ...string) (*template.Template, error) {
	if l.tmpl == nil {
		err := l.Reload()
		if err != nil {
			return nil, err
		}
	}
	if l.alwaysReload {
		err := l.Reload(name...)
		if err != nil {
			return nil, err
		}
	}

	for _, n := range name {
		t := l.tmpl.Lookup(n)
		if t != nil {
			return t, nil
		}
	}

	return nil, errors.New("template not found")
}

// LoadTemplateFS will load a template from a fs.FS.
func LoadTemplateFS(_fs fs.FS, baseDir string, patterns ...string) (*template.Template, error) {
	tmpl := templating.CreateHTMLTemplate("")
	err := templating.ParseHTMLFS(tmpl, _fs, patterns, baseDir)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
