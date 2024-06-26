package handlers

import (
	"context"
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/helpers/strings"
	"github.com/go-go-golems/parka/pkg/handlers/command"
	"github.com/go-go-golems/parka/pkg/handlers/command-dir"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/handlers/static-dir"
	"github.com/go-go-golems/parka/pkg/handlers/static-file"
	"github.com/go-go-golems/parka/pkg/handlers/template"
	"github.com/go-go-golems/parka/pkg/handlers/template-dir"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
)

// TODO(manuel, 2023-05-31) For multi command serves, we should be able to configure
// the command repository type in the config file (for example, sqleton, pinocchio, escuse-me, etc...)

// RepositoryFactory is a function that returns a repository given a list of directories.
// This is used to provision the CommandDir handlers.
type RepositoryFactory func(dirs []string) (*repositories.Repository, error)

// TODO(manuel, 2023-12-13) This currently uses a ReaderCommandLoader which assumes that there is a command in a single file
// THat's however not the case when loading fat commands (like in escuse-me), so we might need something
// like a reader loader that is able to get other loaders for side files too, maybe, or a map of strings or a map of readers

func NewRepositoryFactoryFromReaderLoaders(
	fsLoader loaders.CommandLoader,
) RepositoryFactory {
	return func(dirs []string) (*repositories.Repository, error) {
		directories := []repositories.Directory{}
		for _, dir := range dirs {
			directories = append(directories, repositories.Directory{
				FS:               os.DirFS(dir),
				RootDirectory:    ".",
				RootDocDirectory: "doc",
				WatchDirectory:   dir,
				Name:             dir,
				SourcePrefix:     "file",
			})
		}
		r := repositories.NewRepository(
			repositories.WithDirectories(directories...),
			repositories.WithUpdateCallback(func(cmd cmds.Command) error {
				description := cmd.Description()
				log.Info().Str("name", description.Name).
					Str("source", description.Source).
					Msg("Updating cmd")
				// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
				return nil
			}),
			repositories.WithRemoveCallback(func(cmd cmds.Command) error {
				description := cmd.Description()
				log.Info().Str("name", description.Name).
					Str("source", description.Source).
					Msg("Removing cmd")
				// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
				// NOTE(manuel, 2023-05-25) Regarding the above TODO, why?
				// We don't need to recompute the func, since it fetches the command at runtime.
				return nil
			}),
			repositories.WithCommandLoader(fsLoader),
		)

		// TODO(manuel, 2024-01-18) Properly integrate help system into parka
		helpSystem := help.NewHelpSystem()
		err := r.LoadCommands(helpSystem)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
			os.Exit(1)
		}

		return r, nil
	}
}

// ConfigFileHandler contains everything needed to serve a config file
type ConfigFileHandler struct {
	Config *config.Config

	RepositoryFactory RepositoryFactory

	CommandDirectoryOptions  []command_dir.CommandDirHandlerOption
	TemplateDirectoryOptions []template_dir.TemplateDirHandlerOption
	TemplateOptions          []template.TemplateHandlerOption
	CommandOptions           []command.CommandHandlerOption

	// ConfigFileLocation is an optional path to the config file on disk in case it needs to be reloaded
	ConfigFileLocation        string
	commandDirectoryHandlers  []*command_dir.CommandDirHandler
	templateDirectoryHandlers []*template_dir.TemplateDirHandler
	templateHandlers          []*template.TemplateHandler

	DevMode bool
}

type ConfigFileHandlerOption func(*ConfigFileHandler)

func WithDevMode(devMode bool) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.DevMode = devMode
	}
}

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

func WithAppendCommandHandlerOptions(options ...command.CommandHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.CommandOptions = append(handler.CommandOptions, options...)
	}
}

func WithAppendTemplateHandlerOptions(options ...template.TemplateHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.TemplateOptions = append(handler.TemplateOptions, options...)
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

// NewConfigFileHandler creates a new config file handler. The actual handlers resulting from the config
// file are actually created in Serve.
//
// It will use the options passed in using WithAppendCommandDirHandlerOptions and WithAppendTemplateDirHandlerOptions
// and pass them to the TemplateDir and CommandDir handlers.
//
// TODO(manuel, 2023-06-20) This doesn't allow taking CommandDirOptions and TemplateDirOptions for individual routes.
//
// Also see https://github.com/go-go-golems/parka/issues/51 to allow the individual config file entries
// to actually provide the options for the handlers.
//
// In a way, the options passed here could be considered "defaults". The order of overrides would be interesting
// to figure out.
func NewConfigFileHandler(
	config *config.Config,
	options ...ConfigFileHandlerOption,
) *ConfigFileHandler {
	handler := &ConfigFileHandler{
		Config: config,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

// Serve serves the config file by registering all the routers.
//
// To create the handlers, it will walk over each individual
// route and create the appropriate handler. For example, if the route contains a CommandDirectory, it will
// create a CommandDirHandler and register it with the server.
//
// NOTE(manuel, 2023-06-20) Creating the handlers late, in the Serve method, is not ideal
// because it makes it hard for the creating function to override specific handler options
// if need be (also this could potentially better be handled by setting the right overrides
// and defaults in the config.Config object upfront).
func (cfh *ConfigFileHandler) Serve(server_ *server.Server) error {
	// TODO(manuel, 2023-06-05) Add default repositories and handle them in Command and CommandDir

	if *cfh.Config.Defaults.UseParkaStaticFiles {
		fs_ := server.GetParkaStaticFS()
		parkaStaticHandler := static_dir.NewStaticDirHandler(
			static_dir.WithDefaultFS(fs_, "web/dist"),
		)
		err := parkaStaticHandler.Serve(server_, "/dist")
		if err != nil {
			return err
		}
	}

	rendererOptionsConfig := cfh.Config.Defaults.Renderer
	rendererOptions := []render.RendererOption{}
	if *rendererOptionsConfig.UseDefaultParkaRenderer {
		parkaDefaultRendererOptions, err := server.GetDefaultParkaRendererOptions()
		if err != nil {
			return err
		}

		rendererOptions = append(rendererOptions, parkaDefaultRendererOptions...)
	} else {
		if rendererOptionsConfig.TemplateDirectory != "" {
			dir, err := filepath.Abs(os.ExpandEnv(rendererOptionsConfig.TemplateDirectory))
			if err != nil {
				return err
			}
			lookup := render.NewLookupTemplateFromFS(
				render.WithFS(os.DirFS(dir)),
				render.WithPatterns("**/*.tmpl.*"),
			)
			err = lookup.Reload()
			if err != nil {
				return err
			}

			markdownBaseTemplateName := "base.tmpl.html"
			if rendererOptionsConfig.MarkdownBaseTemplateName != "" {
				markdownBaseTemplateName = rendererOptionsConfig.MarkdownBaseTemplateName
			}

			rendererOptions = []render.RendererOption{
				render.WithAppendTemplateLookups(lookup),
				render.WithMarkdownBaseTemplateName(markdownBaseTemplateName),
			}
		}
	}

	// prepend the renderer options to the list of options
	// honestly this setting should actually be a setting for each route as well
	cfh.TemplateDirectoryOptions = append([]template_dir.TemplateDirHandlerOption{
		template_dir.WithAppendRendererOptions(rendererOptions...),
	}, cfh.TemplateDirectoryOptions...)
	cfh.TemplateOptions = append([]template.TemplateHandlerOption{
		template.WithAppendRendererOptions(rendererOptions...),
	}, cfh.TemplateOptions...)

	for _, route := range cfh.Config.Routes {
		if route.Command != nil {
			return errors.New("command routes are not yet supported")
		}

		if route.CommandDirectory != nil {
			cd := route.CommandDirectory
			// TODO(manuel, 2023-05-31) We must pass in the RepositoryConstructor here,
			// because we need to create an app specific repository, but the config file
			// contains the directories to load commands from.

			if cfh.RepositoryFactory == nil {
				return ErrNoRepositoryFactory{}
			}

			// TODO(manuel, 2023-06-22) It would be nicer to do that in the constructor for the handler itself
			repositories := []string{}
			if *cd.IncludeDefaultRepositories {
				repositories = viper.GetStringSlice("repositories")
			}
			repositories = append(repositories, cd.Repositories...)
			// remove duplicates
			repositories = strings.UniqueStrings(repositories)

			r, err := cfh.RepositoryFactory(repositories)
			if err != nil {
				return err
			}
			directoryOptions := []command_dir.CommandDirHandlerOption{
				command_dir.WithRepository(r),
			}

			// Because the external options are passed in last, they will overwrite whatever
			// options were set from the config file itself, which is useful when running
			// the config file less version of the serve command in sqleton.
			directoryOptions = append(directoryOptions, cfh.CommandDirectoryOptions...)

			cdh, err := command_dir.NewCommandDirHandlerFromConfig(
				cd,
				directoryOptions...,
			)
			if err != nil {
				return err
			}

			cfh.commandDirectoryHandlers = append(cfh.commandDirectoryHandlers, cdh)

			err = cdh.Serve(server_, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.Template != nil {
			th, err := template.NewTemplateHandlerFromConfig(route.Template, cfh.TemplateOptions...)
			if err != nil {
				return err
			}

			cfh.templateHandlers = append(cfh.templateHandlers, th)

			err = th.Serve(server_, route.Path)
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

			// NOTE(manuel, 2023-06-20) I don't think we need to keep track of these
			cfh.templateDirectoryHandlers = append(cfh.templateDirectoryHandlers, tdh)

			err = tdh.Serve(server_, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.StaticFile != nil {
			sfh := static_file.NewStaticFileHandlerFromConfig(route.StaticFile)
			err := sfh.Serve(server_, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.Static != nil {
			sdh := static_dir.NewStaticDirHandlerFromConfig(route.Static)
			err := sdh.Serve(server_, route.Path)
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
		cdh_ := cdh
		errGroup.Go(func() error {
			return cdh_.Watch(ctx2)
		})
	}

	// TODO(manuel, 2023-05-31) What happens if we wait on an empty errgroup?
	return errGroup.Wait()
}
