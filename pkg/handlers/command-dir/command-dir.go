package command_dir

// Package command_dir provides an HTTP interface for exposing commands from a
// repository in various formats such as API responses, downloadable files, or
// rendered in a DataTables UI. It allows users to interact with commands through
// a web interface, offering different endpoints for accessing command data in
// text, JSON, streaming, and file download formats.
//
// The handler integrates with a command repository to serve command outputs over
// HTTP and supports the following output formats:
// - `/data/*path` for JSON output.
// - `/text/*path` for plain text output.
// - `/streaming/*path` for streaming output using server-sent events.
// - `/download/*path` for downloading command output as a file.
// - `/datatables/*path` for rendering commands in a DataTables UI.
//
// Configuration options include:
// - TemplateName: Specifies the template for rendering command outputs.
// - IndexTemplateName: Specifies the template for rendering command indexes.
// - TemplateLookup: Interface for finding and reloading templates.
// - Repository: The command repository to expose over HTTP.
// - AdditionalData: Extra data to be passed to the template.
// - ParameterFilter: Filters for command parameters, including overrides,
//   defaults, blacklist, and whitelist.
// - Stream: Use a channel to stream row results to the HTML template render. Easy to get into concurrency deadlocks, use with care.
//
// Edge cases and potential exceptions are handled as follows:
// - If a command is not found, a `404` error is returned.
// - Ambiguous commands result in a `404` error with an appropriate message.
// - Errors during file download handling or template execution result in a `500`
//   error with an error message.
// - The handler expects exactly one directory for template lookup; otherwise, it
//   returns an error.

import (
	"context"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/handlers/generic-command"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
	"os"
)

type CommandDirHandler struct {
	generic_command.GenericCommandHandler

	DevMode bool

	// Repository is the command repository that is exposed over HTTP through this handler.
	Repository *repositories.Repository
}

type CommandDirHandlerOption func(handler *CommandDirHandler)

func WithDevMode(devMode bool) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.DevMode = devMode
	}
}

func WithRepository(r *repositories.Repository) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.Repository = r
	}
}

func WithGenericCommandHandlerOptions(options ...generic_command.GenericCommandHandlerOption) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		for _, option := range options {
			option(&handler.GenericCommandHandler)
		}
	}
}

func NewCommandDirHandlerFromConfig(
	config_ *config.CommandDir,
	options ...CommandDirHandlerOption,
) (*CommandDirHandler, error) {
	genericOptions := []generic_command.GenericCommandHandlerOption{
		generic_command.WithTemplateName(config_.TemplateName),
		generic_command.WithIndexTemplateName(config_.IndexTemplateName),
		generic_command.WithMergeAdditionalData(config_.AdditionalData, true),
	}
	genericHandler, err := generic_command.NewGenericCommandHandler(genericOptions...)
	if err != nil {
		return nil, err
	}
	cd := &CommandDirHandler{
		GenericCommandHandler: *genericHandler,
	}

	cd.ParameterFilter.Overrides = config_.Overrides
	cd.ParameterFilter.Defaults = config_.Defaults
	cd.ParameterFilter.Blacklist = config_.Blacklist
	cd.ParameterFilter.Whitelist = config_.Whitelist
	// by default, we stream when outputting to datatables too
	if config_.Stream != nil {
		cd.Stream = *config_.Stream
	} else {
		cd.Stream = true
	}

	for _, option := range options {
		option(cd)
	}

	// we run this after the options in order to get the DevMode value
	if cd.TemplateLookup == nil {
		if config_.TemplateLookup != nil {
			patterns := config_.TemplateLookup.Patterns
			if len(patterns) == 0 {
				patterns = []string{"**/*.tmpl.md", "**/*.tmpl.html"}
			}
			// we currently only support a single directory
			if len(config_.TemplateLookup.Directories) != 1 {
				return nil, errors.New("template lookup directories must be exactly one")
			}
			cd.TemplateLookup = render.NewLookupTemplateFromFS(
				render.WithFS(os.DirFS(config_.TemplateLookup.Directories[0])),
				render.WithBaseDir(""),
				render.WithPatterns(patterns...),
				render.WithAlwaysReload(cd.DevMode),
			)
		} else {
			cd.TemplateLookup = datatables.NewDataTablesLookupTemplate()
		}
	}

	err = cd.TemplateLookup.Reload()
	if err != nil {
		return nil, err
	}

	return cd, nil
}

func (cd *CommandDirHandler) Watch(ctx context.Context) error {
	return cd.Repository.Watch(ctx)
}

func (cd *CommandDirHandler) Serve(server *parka.Server, basePath string) error {
	if cd.Repository == nil {
		return errors.New("no repository configured")
	}

	return cd.GenericCommandHandler.ServeRepository(server, basePath, cd.Repository)
}
