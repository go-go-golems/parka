package template_dir

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/server"
	"io/fs"
	"path/filepath"
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
			handler.fs, handler.LocalDirectory, err = loaders.FileNameToFsFilePath(p)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func NewTemplateDirHandler(options ...TemplateDirHandlerOption) (*TemplateDirHandler, error) {
	handler := &TemplateDirHandler{}
	for _, option := range options {
		err := option(handler)
		if err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func NewTemplateDirHandlerFromConfig(td *config.TemplateDir, options ...TemplateDirHandlerOption) (*TemplateDirHandler, error) {
	handler := &TemplateDirHandler{
		IndexTemplateName: td.IndexTemplateName,
	}
	err := WithLocalDirectory(td.LocalDirectory)(handler)
	if err != nil {
		return nil, err
	}

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
	err = templateLookup.Reload()

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
	// The use of a middleware to handle something that could be handled by the routing framework itself
	// is because gin (which really should get replaced because we actually go against its grain heavily)
	// does not allow routes to overlap.
	server.Router.Use(td.renderer.HandleWithTrimPrefix(path, nil))

	return nil
}
