package config

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
