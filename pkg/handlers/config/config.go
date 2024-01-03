package config

import (
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
)

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

func expandPaths(paths []string) ([]string, error) {
	expandedPaths := []string{}
	for _, path := range paths {
		path_, err := EvaluateConfigEntry(path)
		if err != nil {
			return nil, err
		}
		path = expandPath(path_.(string))

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Warn().Str("path", path).Msg("path does not exist")
			continue
		}

		expandedPaths = append(expandedPaths, expandPath(path))
	}

	return expandedPaths, nil
}

// TemplateLookupConfig is used to configured a directory based template lookup.
type TemplateLookupConfig struct {
	// Directories is a list of directories that will be searched for templates.
	Directories []string `yaml:"directories,omitempty"`
	// Patterns is a list of glob patterns that will be used to match files in the directories.
	// If the list is empty, the default of **/*.tmpl.md and **/*.tmpl.html will be used
	Patterns []string `yaml:"patterns,omitempty"`
}

// ParameterFilterList are used to configure whitelists and blacklists.
// Entire layers as well as individual flags and arguments can be whitelisted or blacklisted.
// Params is used for the default layer.
type ParameterFilterList struct {
	Layers          []string            `yaml:"layers,omitempty"`
	LayerParameters map[string][]string `yaml:"layerParameters,omitempty"`
	Parameters      []string            `yaml:"parameters,omitempty"`
}

func (p *ParameterFilterList) GetAllLayerParameters() map[string][]string {
	ret := map[string][]string{}
	for layer, params := range p.LayerParameters {
		ret[layer] = params
	}
	if _, ok := ret[layers.DefaultSlug]; !ok {
		ret[layers.DefaultSlug] = []string{}
	}
	ret[layers.DefaultSlug] = append(ret[layers.DefaultSlug], p.Parameters...)
	return ret
}

type LayerParameters struct {
	Layers     map[string]map[string]interface{} `yaml:"layers,omitempty"`
	Parameters map[string]interface{}            `yaml:"parameters,omitempty"`
}

func NewLayerParameters() *LayerParameters {
	return &LayerParameters{
		Layers:     map[string]map[string]interface{}{},
		Parameters: map[string]interface{}{},
	}
}

// Merge merges the two LayerParameters, with the overrides taking precedence.
// It merges all the layers, flags, and arguments. For each layer, the layer flags are merged as well,
// overrides taking precedence.
func (p *LayerParameters) Merge(overrides *LayerParameters) {
	for k, v := range overrides.Layers {
		if _, ok := p.Layers[k]; !ok {
			p.Layers[k] = map[string]interface{}{}
		}
		for k2, v2 := range v {
			p.Layers[k][k2] = v2
		}
	}

	for k, v := range overrides.Parameters {
		p.Parameters[k] = v
	}
}

func (p *LayerParameters) Clone() *LayerParameters {
	ret := NewLayerParameters()
	ret.Merge(p)
	return ret
}

func (p *LayerParameters) GetParameterMap() map[string]map[string]interface{} {
	r := p.Clone()
	ret := r.Layers
	if _, ok := ret[layers.DefaultSlug]; !ok {
		ret[layers.DefaultSlug] = map[string]interface{}{}
	}
	for k, v := range r.Parameters {
		ret[layers.DefaultSlug][k] = v
	}

	return ret
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
	UseDefaultParkaRenderer *bool `yaml:"useDefaultParkaRenderer,omitempty"`
	// TODO(manuel, 2023-06-21) These two options are not implemented yet
	// It is not so much that they are hard to implement, but rather that they are annoying to test.
	// See: https://github.com/go-go-golems/parka/issues/56
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

	err = cfg.Initialize()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (cfg *Config) Initialize() error {
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
			if cfg.Defaults.Renderer.UseDefaultParkaRenderer == nil {
				if cfg.Defaults.Renderer.TemplateDirectory == "" {
					cfg.Defaults.Renderer.UseDefaultParkaRenderer = boolPtr(true)
				} else {
					cfg.Defaults.Renderer.UseDefaultParkaRenderer = boolPtr(false)
				}
			}

			if cfg.Defaults.Renderer.TemplateDirectory != "" {
				cfg.Defaults.Renderer.TemplateDirectory = expandPath(cfg.Defaults.Renderer.TemplateDirectory)
			}
		}
	}
	var err error
	for _, route := range cfg.Routes {
		if route.CommandDirectory != nil {
			err = route.CommandDirectory.ExpandPaths()
			if err != nil {
				return err
			}
		}
		if route.Command != nil {
			err = route.Command.ExpandPaths()
			if err != nil {
				return err
			}
		}
		if route.Static != nil {
			err = route.Static.ExpandPaths()
			if err != nil {
				return err
			}
		}
		if route.StaticFile != nil {
			err = route.StaticFile.ExpandPaths()
			if err != nil {
				return err
			}
		}
		if route.Template != nil {
			err = route.Template.ExpandPaths()
			if err != nil {
				return err
			}
		}
		if route.TemplateDirectory != nil {
			err = route.TemplateDirectory.ExpandPaths()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
