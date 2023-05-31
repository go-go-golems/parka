package handlers

import (
	"context"
	"github.com/go-go-golems/clay/pkg/repositories"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/handlers/command-dir"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/handlers/static-dir"
	"github.com/go-go-golems/parka/pkg/handlers/static-file"
	"github.com/go-go-golems/parka/pkg/handlers/template-dir"
	"golang.org/x/sync/errgroup"
)

// TODO(manuel, 2023-05-31) For multi command serves, we should be able to configure
// the command repository type in the config file (for example, sqleton, pinocchio, escuse-me, etc...)

// RepositoryFactory is a function that returns a repository given a list of directories.
// This is used to provision the CommandDir handlers.
type RepositoryFactory func(dirs []string) (*repositories.Repository, error)

// ConfigFileHandler contains everything needed to serve a config file
type ConfigFileHandler struct {
	Config *config.Config

	RepositoryFactory RepositoryFactory

	CommandDirectoryOptions  []command_dir.CommandDirHandlerOption
	TemplateDirectoryOptions []template_dir.TemplateDirHandlerOption

	// ConfigFileLocation is an optional path to the config file on disk in case it needs to be reloaded
	ConfigFileLocation        string
	commandDirectoryHandlers  []*command_dir.CommandDirHandler
	templateDirectoryHandlers []*template_dir.TemplateDirHandler
}

type ConfigFileHandlerOption func(*ConfigFileHandler)

func WithAppendCommandDirHandlerOptions(options ...command_dir.CommandDirHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.CommandDirectoryOptions = append(handler.CommandDirectoryOptions, options...)
	}
}

func WithAppendTemplateDirHandlerOptions(options ...template_dir.TemplateDirHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.TemplateDirectoryOptions = append(handler.TemplateDirectoryOptions, options...)
	}
}

func WithConfigFileLocation(location string) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.ConfigFileLocation = location
	}
}

func WithRepositoryFactory(rf RepositoryFactory) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.RepositoryFactory = rf
	}
}

type ErrNoRepositoryFactory struct{}

func (e ErrNoRepositoryFactory) Error() string {
	return "no repository factory provided"
}

func NewConfigFileHandler(config *config.Config, options ...ConfigFileHandlerOption) *ConfigFileHandler {
	handler := &ConfigFileHandler{
		Config: config,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (cfh *ConfigFileHandler) Serve(server *parka.Server) error {
	// NOTE(manuel, 2023-05-26)
	// This could be extracted to a "parseConfigFile", so that we can easily add preconfigured handlers that
	// can deal with embeddedFS

	for _, route := range cfh.Config.Routes {
		if route.CommandDirectory != nil {
			// TODO(manuel, 2023-05-31) We must pass in the RepositoryConstructor here,
			// because we need to create an app specific repository, but the config file
			// contains the directories to load commands from.

			if cfh.RepositoryFactory == nil {
				return ErrNoRepositoryFactory{}
			}

			r, err := cfh.RepositoryFactory(route.CommandDirectory.Repositories)
			if err != nil {
				return err
			}
			directoryOptions := append(cfh.CommandDirectoryOptions, command_dir.WithRepository(r))

			cdh, err := command_dir.NewCommandDirHandlerFromConfig(
				route.CommandDirectory,
				directoryOptions...,
			)
			if err != nil {
				return err
			}

			cfh.commandDirectoryHandlers = append(cfh.commandDirectoryHandlers, cdh)

			err = cdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.TemplateDirectory != nil {
			tdh, err := template_dir.NewTemplateDirHandlerFromConfig(
				route.TemplateDirectory,
				cfh.TemplateDirectoryOptions...,
			)
			if err != nil {
				return err
			}

			cfh.templateDirectoryHandlers = append(cfh.templateDirectoryHandlers, tdh)

			err = tdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.StaticFile != nil {
			sfh := static_file.NewStaticFileHandlerFromConfig(route.StaticFile)
			err := sfh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.Static != nil {
			sdh := static_dir.NewStaticDirHandlerFromConfig(route.Static)
			err := sdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}
	}

	return nil
}

// Watch watches the config for changes and updates the server accordingly.
// Because this will register / unregister routes, this will probably need to be handled
// at a level where we can restart the gin server altogether.
func (cfh *ConfigFileHandler) Watch(ctx context.Context) error {
	errGroup, ctx2 := errgroup.WithContext(ctx)
	for _, cdh := range cfh.commandDirectoryHandlers {
		if cdh.Repository == nil {
			continue
		}
		errGroup.Go(func() error {
			return cdh.Repository.Watch(ctx2)
		})
	}

	// TODO(manuel, 2023-05-31) What happens if we wait on an empty errgroup?
	return errGroup.Wait()
}
