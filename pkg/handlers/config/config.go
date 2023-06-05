package config

import (
	"gopkg.in/yaml.v3"
)

type Route struct {
	Path              string       `yaml:"path"`
	CommandDirectory  *CommandDir  `yaml:"commandDirectory,omitempty"`
	Command           *Command     `yaml:"command,omitempty"`
	Static            *Static      `yaml:"static,omitempty"`
	StaticFile        *StaticFile  `yaml:"staticFile,omitempty"`
	TemplateDirectory *TemplateDir `yaml:"templateDirectory,omitempty"`
	Template          *Template    `yaml:"template,omitempty"`
}

type CommandDir struct {
	Repositories      []string          `yaml:"repositories"`
	TemplateDirectory string            `yaml:"templateDirectory,omitempty"`
	TemplateName      string            `yaml:"templateName,omitempty"`
	IndexTemplateName string            `yaml:"indexTemplateName,omitempty"`
	AdditionalData    map[string]string `yaml:"additionalData,omitempty"`
	Defaults          *LayerParams      `yaml:"defaults,omitempty"`
	Overrides         *LayerParams      `yaml:"overrides,omitempty"`
}

type Command struct {
	File           string            `yaml:"file"`
	TemplateName   string            `yaml:"templateName"`
	AdditionalData map[string]string `yaml:"additionalData,omitempty"`
	Defaults       *LayerParams      `yaml:"defaults,omitempty"`
	Overrides      *LayerParams      `yaml:"overrides,omitempty"`
}

type Static struct {
	LocalPath string `yaml:"localPath"`
}

type StaticFile struct {
	LocalPath string `yaml:"localPath"`
}

type TemplateDir struct {
	LocalDirectory    string                 `yaml:"localDirectory"`
	IndexTemplateName string                 `yaml:"indexTemplateName,omitempty"`
	AdditionalData    map[string]interface{} `yaml:"additionalData,omitempty"`
}

type Template struct {
	TemplateFile string `yaml:"templateFile"`
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

type Config struct {
	Routes []*Route `yaml:"routes"`
}

func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}