package config

import (
	"github.com/pkg/errors"
	"strings"
)

// CommandDir represents the config file entry for a command directory route.
type CommandDir struct {
	Repositories               []string `yaml:"repositories"`
	IncludeDefaultRepositories *bool    `yaml:"includeDefaultRepositories"`

	TemplateLookup *TemplateLookupConfig `yaml:"templateLookup,omitempty"`

	TemplateName      string `yaml:"templateName,omitempty"`
	IndexTemplateName string `yaml:"indexTemplateName,omitempty"`

	AdditionalData map[string]interface{} `yaml:"additionalData,omitempty"`

	Defaults  *LayerParameters     `yaml:"defaults,omitempty"`
	Overrides *LayerParameters     `yaml:"overrides,omitempty"`
	Blacklist *ParameterFilterList `yaml:"blackList,omitempty"`
	Whitelist *ParameterFilterList `yaml:"whiteList,omitempty"`

	Stream *bool `yaml:"stream,omitempty"`
}

func (c *CommandDir) ExpandPaths() error {
	var err error

	if c.TemplateLookup != nil {
		c.TemplateLookup.Directories, err = expandPaths(c.TemplateLookup.Directories)
		if err != nil {
			return err
		}
	}

	if c.IncludeDefaultRepositories == nil {
		c.IncludeDefaultRepositories = boolPtr(true)
	}

	repositories, err := expandPaths(c.Repositories)
	if err != nil {
		return err
	}

	if len(repositories) == 0 && !*c.IncludeDefaultRepositories {
		return errors.Errorf("no repositories found: %s", strings.Join(repositories, ", "))
	}
	c.Repositories = repositories

	evaluatedData, err := EvaluateConfigEntry(c.AdditionalData)
	if err != nil {
		return err
	}
	c.AdditionalData = evaluatedData.(map[string]interface{})

	if c.Defaults != nil {
		evaluatedDefaults, err := evaluateLayerParams(c.Defaults)
		if err != nil {
			return err
		}
		c.Defaults = evaluatedDefaults
	}
	if c.Overrides != nil {
		evaluatedOverrides, err := evaluateLayerParams(c.Overrides)
		if err != nil {
			return err
		}
		c.Overrides = evaluatedOverrides
	}

	return nil
}

type Command struct {
	File         string `yaml:"file"`
	TemplateName string `yaml:"templateName"`

	TemplateLookup *TemplateLookupConfig `yaml:"templateLookup,omitempty"`

	AdditionalData map[string]interface{} `yaml:"additionalData,omitempty"`
	Defaults       *LayerParameters       `yaml:"defaults,omitempty"`
	Overrides      *LayerParameters       `yaml:"overrides,omitempty"`
	Whitelist      *ParameterFilterList   `yaml:"whitelist,omitempty"`
	Blacklist      *ParameterFilterList   `yaml:"blacklist,omitempty"`

	Stream *bool `yaml:"stream,omitempty"`
}

func (c *Command) ExpandPaths() error {
	c.File = expandPath(c.File)

	evaluatedData, err := EvaluateConfigEntry(c.AdditionalData)
	if err != nil {
		return err
	}
	c.AdditionalData = evaluatedData.(map[string]interface{})

	if c.Defaults != nil {
		evaluatedDefaults, err := evaluateLayerParams(c.Defaults)
		if err != nil {
			return err
		}
		c.Defaults = evaluatedDefaults
	}
	if c.Overrides != nil {
		evaluatedOverrides, err := evaluateLayerParams(c.Overrides)
		if err != nil {
			return err
		}
		c.Overrides = evaluatedOverrides
	}

	if c.TemplateLookup != nil {
		c.TemplateLookup.Directories, err = expandPaths(c.TemplateLookup.Directories)
		if err != nil {
			return err
		}
	}

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
	LocalDirectory    string                 `yaml:"localDirectory"`
	IndexTemplateName string                 `yaml:"indexTemplateName,omitempty"`
	AdditionalData    map[string]interface{} `yaml:"additionalData,omitempty"`
}

func (t *TemplateDir) ExpandPaths() error {
	t.LocalDirectory = expandPath(t.LocalDirectory)

	evaluatedData, err := EvaluateConfigEntry(t.AdditionalData)
	if err != nil {
		return err
	}
	t.AdditionalData = evaluatedData.(map[string]interface{})

	return nil
}

type Template struct {
	// every request will be rendered from the template file, using the default renderer in the case of markdown
	// content.
	TemplateFile   string                 `yaml:"templateFile"`
	AdditionalData map[string]interface{} `yaml:"additionalData,omitempty"`
}

func (t *Template) ExpandPaths() error {
	t.TemplateFile = expandPath(t.TemplateFile)

	evaluatedData, err := EvaluateConfigEntry(t.AdditionalData)
	if err != nil {
		return err
	}
	t.AdditionalData = evaluatedData.(map[string]interface{})

	return nil
}
