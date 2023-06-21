package template_dir

import (
	"fmt"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/server"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TODO(manuel, 2023-05-28) Add a proper Handler interface that also
// deals with devmode / reloading

type TemplateDirHandler struct {
	fs                       fs.FS
	LocalDirectory           string
	IndexTemplateName        string
	MarkdownBaseTemplateName string
	rendererOptions          []render.RendererOption
	renderer                 *render.Renderer
	alwaysReload             bool
}

type TemplateDirHandlerOption func(handler *TemplateDirHandler) error

func WithDefaultFS(fs fs.FS, localPath string) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) error {
		if handler.fs == nil {
			handler.fs = fs
			handler.LocalDirectory = localPath
		}
		return nil
	}
}

func WithAlwaysReload(alwaysReload bool) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) error {
		handler.alwaysReload = alwaysReload
		return nil
	}
}

func WithAppendRendererOptions(rendererOptions ...render.RendererOption) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) error {
		handler.rendererOptions = append(handler.rendererOptions, rendererOptions...)
		return nil
	}
}

func WithLocalDirectory(localPath string) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) error {
		if localPath != "" {
			// try to resolve the localPath to an absolute path, because lookups in relative paths
			// are a bit untested.
			p, err := filepath.Abs(localPath)
			if err != nil {
				return err
			}
			if len(p) == 0 {
				return fmt.Errorf("invalid local path: %s", localPath)
			}
			if p[0] == '/' {
				handler.fs = os.DirFS("/")
			} else {
				handler.fs = os.DirFS(".")
			}
			// We strip the / prefix because once the FS is /, we need to match for "relative" paths within the FS.
			// We can't just simplify the whole thing to be os.DirFS(localPath) because we also need to support
			// embed.FS where we can't do the same path shenanigans, since the embed.FS files reflect the directory
			// from which they were included, which we need to strip at runtime.
			handler.LocalDirectory = strings.TrimPrefix(p, "/")
		}

		return nil
	}
}

func NewTemplateDirHandler(options ...TemplateDirHandlerOption) *TemplateDirHandler {
	handler := &TemplateDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewTemplateDirHandlerFromConfig(td *config.TemplateDir, options ...TemplateDirHandlerOption) (*TemplateDirHandler, error) {
	handler := &TemplateDirHandler{
		IndexTemplateName: td.IndexTemplateName,
	}
	WithLocalDirectory(td.LocalDirectory)(handler)

	for _, option := range options {
		err := option(handler)
		if err != nil {
			return nil, err
		}
	}
	templateLookup := render.NewLookupTemplateFromFS(
		render.WithFS(handler.fs),
		render.WithBaseDir(handler.LocalDirectory),
		render.WithPatterns(
			"**/*.tmpl.md",
			"**/*.md",
			"**/*.tmpl.html",
			"**/*.html",
		),
		render.WithAlwaysReload(handler.alwaysReload),
	)
	err := templateLookup.Reload()

	if err != nil {
		return nil, fmt.Errorf("failed to load local template: %w", err)
	}
	rendererOptions := append(
		handler.rendererOptions,
		render.WithPrependTemplateLookups(templateLookup),
		render.WithIndexTemplateName(handler.IndexTemplateName),
	)
	r, err := render.NewRenderer(rendererOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load local template: %w", err)
	}
	handler.renderer = r

	return handler, nil
}

func (td *TemplateDirHandler) Serve(server *server.Server, path string) error {
	// TODO(manuel, 2023-05-26) This is a hack because we currently mix and match content with commands.
	server.Router.Use(td.renderer.HandleWithTrimPrefix(path, nil))

	return nil
}
