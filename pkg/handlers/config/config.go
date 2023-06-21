package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

// Route represents a single sub-route of the server.
// Only one of the booleans or one of the pointers should be true or non-nil.
// This is the first attempt at the structure of a config file, and is bound to change.
type Route struct {
	Path              string       `yaml:"path"`
	CommandDirectory  *CommandDir  `yaml:"commandDirectory,omitempty"`
	Command           *Command     `yaml:"command,omitempty"`
	Static            *Static      `yaml:"static,omitempty"`
	StaticFile        *StaticFile  `yaml:"staticFile,omitempty"`
	TemplateDirectory *TemplateDir `yaml:"templateDirectory,omitempty"`
	Template          *Template    `yaml:"template,omitempty"`
}

// RouteHandlerConfiguration is the interface that all route handler configurations must implement.
// By RouteHandlerConfiguration, we mean things like CommandDir, Command, Static, etc...
type RouteHandlerConfiguration interface {
	//
	ExpandPaths() error
}

func (r *Route) HandlesCommand() bool {
	return r.Command != nil || r.CommandDirectory != nil
}

func (r *Route) HandlesStatic() bool {
	return r.Static != nil || r.StaticFile != nil
}

func (r *Route) HandlesTemplate() bool {
	return r.Template != nil || r.TemplateDirectory != nil
}

func (r *Route) IsCommandRoute() bool {
	return r.Command != nil
}

func (r *Route) IsCommandDirRoute() bool {
	return r.CommandDirectory != nil
}

func (r *Route) IsStaticRoute() bool {
	return r.Static != nil
}

func (r *Route) IsStaticFileRoute() bool {
	return r.StaticFile != nil
}

func (r *Route) IsTemplateRoute() bool {
	return r.Template != nil
}

func (r *Route) IsTemplateDirRoute() bool {
	return r.TemplateDirectory != nil
}

func expandPath(path string) string {
	// expand ~
	if len(path) >= 2 && path[:2] == "~/" {
		path = path[2:]
		path = "$HOME/" + path
	}

	// expand env vars
	path = os.ExpandEnv(path)
	return path
}

// TODO(manuel, 2023-06-20) We should probably allow for environment values to be passed as data as well

type CommandDir struct {
	Repositories               []string          `yaml:"repositories"`
	IncludeDefaultRepositories bool              `yaml:"includeDefaultRepositories"`
	TemplateDirectory          string            `yaml:"templateDirectory,omitempty"`
	TemplateName               string            `yaml:"templateName,omitempty"`
	IndexTemplateName          string            `yaml:"indexTemplateName,omitempty"`
	AdditionalData             map[string]string `yaml:"additionalData,omitempty"`
	Defaults                   *LayerParams      `yaml:"defaults,omitempty"`
	Overrides                  *LayerParams      `yaml:"overrides,omitempty"`
}

func (c *CommandDir) ExpandPaths() error {
	c.TemplateDirectory = expandPath(c.TemplateDirectory)
	repositories := []string{}

	for _, repository := range c.Repositories {
		repository = expandPath(repository)

		// skip if path doesn't exist
		if _, err := os.Stat(repository); os.IsNotExist(err) {
			continue
		}

		repositories = append(repositories, repository)
	}

	if len(repositories) == 0 {
		return errors.Errorf("no repositories found: %s", strings.Join(c.Repositories, ", "))
	}
	c.Repositories = repositories
	return nil
}

type Command struct {
	File           string            `yaml:"file"`
	TemplateName   string            `yaml:"templateName"`
	AdditionalData map[string]string `yaml:"additionalData,omitempty"`
	Defaults       *LayerParams      `yaml:"defaults,omitempty"`
	Overrides      *LayerParams      `yaml:"overrides,omitempty"`
}

func (c *Command) ExpandPaths() error {
	c.File = expandPath(c.File)
	return nil
}

type Static struct {
	LocalPath string `yaml:"localPath"`
}

func (s *Static) ExpandPaths() error {
	s.LocalPath = expandPath(s.LocalPath)
	return nil
}

type StaticFile struct {
	LocalPath string `yaml:"localPath"`
}

func (s *StaticFile) ExpandPaths() error {
	s.LocalPath = expandPath(s.LocalPath)
	return nil
}

// TemplateDir serves a directory of html, md, .tmpl.md, .tmpl.html files.
// Markdown files are renderer using the given MarkdownBaseTemplateName, which will be
// looked up in the TemplateDir itself, or using the default renderer if empty.
type TemplateDir struct {
	LocalDirectory           string                 `yaml:"localDirectory"`
	IndexTemplateName        string                 `yaml:"indexTemplateName,omitempty"`
	MarkdownBaseTemplateName string                 `yaml:"markdownBaseTemplateName,omitempty"`
	AdditionalData           map[string]interface{} `yaml:"additionalData,omitempty"`
}

func (t *TemplateDir) ExpandPaths() error {
	t.LocalDirectory = expandPath(t.LocalDirectory)
	return nil
}

type Template struct {
	// every request will be rendered from the template file, using the default renderer in the case of markdown
	// content.
	TemplateFile string `yaml:"templateFile"`
	// TODO(manuel, 2023-06-20) Add the option to pass in data to the template
	AdditionalData           map[string]interface{} `yaml:"additionalData,omitempty"`
	MarkdownBaseTemplateName string                 `yaml:"markdownBaseTemplateName,omitempty"`
}

func (t *Template) ExpandPaths() error {
	t.TemplateFile = expandPath(t.TemplateFile)
	return nil
}

type LayerParams struct {
	Layers    map[string]map[string]interface{} `yaml:"layers,omitempty"`
	Flags     map[string]interface{}            `yaml:"flags,omitempty"`
	Arguments map[string]interface{}            `yaml:"arguments,omitempty"`
}

func NewLayerParams() *LayerParams {
	return &LayerParams{
		Layers:    map[string]map[string]interface{}{},
		Flags:     map[string]interface{}{},
		Arguments: map[string]interface{}{},
	}
}

// Merge merges the two LayerParams, with the overrides taking precedence.
// It merges all the layers, flags, and arguments. For each layer, the layer flags are merged as well,
// overrides taking precedence.
func (p *LayerParams) Merge(overrides *LayerParams) {
	for k, v := range overrides.Layers {
		if _, ok := p.Layers[k]; !ok {
			p.Layers[k] = map[string]interface{}{}
		}
		for k2, v2 := range v {
			p.Layers[k][k2] = v2
		}
	}

	for k, v := range overrides.Flags {
		p.Flags[k] = v
	}

	for k, v := range overrides.Arguments {
		p.Arguments[k] = v
	}
}

// Defaults controls the default renderer and which embedded static files to serve.
type Defaults struct {
	Renderer            *DefaultRendererOptions `yaml:"renderer,omitempty"`
	UseParkaStaticFiles *bool                   `yaml:"useParkaStaticFiles,omitempty"`
}

// DefaultRendererOptions controls the default renderer.
// If UseDefaultParkaRenderer is true, the default parka renderer will be used.
// It renders markdown files using base.tmpl.html and uses a tailwind css stylesheet
// which has to be served under dist/output.css.
type DefaultRendererOptions struct {
	UseDefaultParkaRenderer  *bool  `yaml:"useDefaultParkaRenderer,omitempty"`
	TemplateDirectory        string `yaml:"templateDirectory,omitempty"`
	MarkdownBaseTemplateName string `yaml:"markdownBaseTemplateName,omitempty"`
}

type Config struct {
	Routes   []*Route  `yaml:"routes"`
	Defaults *Defaults `yaml:"defaults,omitempty"`
}

func boolPtr(b bool) *bool {
	return &b
}

func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Defaults == nil {
		cfg.Defaults = &Defaults{
			UseParkaStaticFiles: boolPtr(true),
			Renderer: &DefaultRendererOptions{
				UseDefaultParkaRenderer: boolPtr(true),
			},
		}
	} else {
		if cfg.Defaults.UseParkaStaticFiles == nil {
			cfg.Defaults.UseParkaStaticFiles = boolPtr(true)
		}

		if cfg.Defaults.Renderer == nil {
			cfg.Defaults.Renderer = &DefaultRendererOptions{
				UseDefaultParkaRenderer: boolPtr(true),
			}
		} else {
			if cfg.Defaults.Renderer.UseDefaultParkaRenderer == nil &&
				cfg.Defaults.UseParkaStaticFiles != nil &&
				*cfg.Defaults.UseParkaStaticFiles {
				cfg.Defaults.Renderer.UseDefaultParkaRenderer = boolPtr(true)
			}
		}
	}
	for _, route := range cfg.Routes {
		if route.CommandDirectory != nil {
			err = route.CommandDirectory.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
		if route.Command != nil {
			err = route.Command.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
		if route.Static != nil {
			err = route.Static.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
		if route.StaticFile != nil {
			err = route.StaticFile.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
		if route.Template != nil {
			err = route.Template.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
		if route.TemplateDirectory != nil {
			err = route.TemplateDirectory.ExpandPaths()
			if err != nil {
				return nil, err
			}
		}
	}

	return &cfg, nil
}
