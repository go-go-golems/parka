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
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	output_file "github.com/go-go-golems/parka/pkg/glazed/handlers/output-file"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/sse"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/text"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type CommandDirHandler struct {
	DevMode bool

	// TemplateName is the name of the template that is lookup up through the given TemplateLookup
	// used to render the glazed command.
	TemplateName string
	// IndexTemplateName is the name of the template that is looked up through TemplateLookup to render
	// command indexes. Leave empty to not render index pages at all.
	IndexTemplateName string
	// TemplateLookup is used to look up both TemplateName and IndexTemplateName
	TemplateLookup render.TemplateLookup

	// Repository is the command repository that is exposed over HTTP through this handler.
	Repository *repositories.Repository

	// AdditionalData is passed to the template being rendered.
	AdditionalData map[string]interface{}

	ParameterFilter *config.ParameterFilter

	// If true, all glazed outputs will try to use a row output if possible.
	// This means that "ragged" objects (where columns might not all be present)
	// will have missing columns, only the fields of the first object will be used
	// as rows.
	//
	// This is true per default, and needs to be explicitly set to false to use
	// a normal TableMiddleware oriented output.
	Stream bool
}

type CommandDirHandlerOption func(handler *CommandDirHandler)

func WithTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateName = name
	}
}

func WithParameterFilter(overridesAndDefaults *config.ParameterFilter) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.ParameterFilter = overridesAndDefaults
	}
}

func WithParameterFilterOptions(opts ...config.ParameterFilterOption) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		for _, opt := range opts {
			opt(handler.ParameterFilter)
		}
	}
}

func WithDefaultTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.TemplateName == "" {
			handler.TemplateName = name
		}
	}
}

func WithIndexTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.IndexTemplateName = name
	}
}

func WithDefaultIndexTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.IndexTemplateName == "" {
			handler.IndexTemplateName = name
		}
	}
}

// WithMergeAdditionalData merges the passed in map with the handler's AdditionalData map.
// If a value is already set in the AdditionalData map and override is true, it will get overwritten.
func WithMergeAdditionalData(data map[string]interface{}, override bool) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.AdditionalData == nil {
			handler.AdditionalData = data
		} else {
			for k, v := range data {
				if _, ok := handler.AdditionalData[k]; !ok || override {
					handler.AdditionalData[k] = v
				}
			}
		}
	}
}

func WithTemplateLookup(lookup render.TemplateLookup) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateLookup = lookup
	}
}

// handling all the ways to configure overrides
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

func NewCommandDirHandlerFromConfig(
	config_ *config.CommandDir,
	options ...CommandDirHandlerOption,
) (*CommandDirHandler, error) {
	cd := &CommandDirHandler{
		TemplateName:      config_.TemplateName,
		IndexTemplateName: config_.IndexTemplateName,
		AdditionalData:    config_.AdditionalData,
		ParameterFilter:   &config.ParameterFilter{},
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

	err := cd.TemplateLookup.Reload()
	if err != nil {
		return nil, err
	}

	return cd, nil
}

func (cd *CommandDirHandler) Serve(server *parka.Server, path string) error {
	if cd.Repository == nil {
		return fmt.Errorf("no repository configured")
	}

	path = strings.TrimSuffix(path, "/")

	middlewares_ := cd.ParameterFilter.ComputeMiddlewares(cd.Stream)
	server.Router.GET(path+"/data/*path", func(c echo.Context) error {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(c, cd.Repository, commandPath)
		if err != nil {
			return err
		}

		switch v := command.(type) {
		case cmds.GlazeCommand:
			return json.CreateJSONQueryHandler(v, json.WithMiddlewares(middlewares_...))(c)
		default:
			return text.CreateQueryHandler(v)(c)
		}
	})

	server.Router.GET(path+"/text/*path", func(c echo.Context) error {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(c, cd.Repository, commandPath)
		if err != nil {
			return err
		}

		return text.CreateQueryHandler(command, middlewares_...)(c)
	})

	server.Router.GET(path+"/streaming/*path", func(c echo.Context) error {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(c, cd.Repository, commandPath)
		if err != nil {
			return err
		}

		return sse.CreateQueryHandler(command, middlewares_...)(c)
	})

	server.Router.GET(path+"/datatables/*path",
		func(c echo.Context) error {
			commandPath := c.Param("path")
			commandPath = strings.TrimPrefix(commandPath, "/")

			// Get repository command
			command, err := getRepositoryCommand(c, cd.Repository, commandPath)
			if err != nil {
				return err
			}

			switch v := command.(type) {
			case cmds.GlazeCommand:
				options := []datatables.QueryHandlerOption{
					datatables.WithMiddlewares(middlewares_...),
					datatables.WithTemplateLookup(cd.TemplateLookup),
					datatables.WithTemplateName(cd.TemplateName),
					datatables.WithAdditionalData(cd.AdditionalData),
					datatables.WithStreamRows(cd.Stream),
				}

				return datatables.CreateDataTablesHandler(v, path, commandPath, options...)(c)
			default:
				return c.JSON(http.StatusInternalServerError, utils.H{"error": fmt.Sprintf("command %s is not a glazed command", commandPath)})
			}
		})

	server.Router.GET(path+"/download/*path", func(c echo.Context) error {
		path_ := c.Param("path")
		path_ = strings.TrimPrefix(path_, "/")
		index := strings.LastIndex(path_, "/")
		if index == -1 {
			return c.JSON(http.StatusInternalServerError, utils.H{"error": "could not find file name"})
		}
		if index >= len(path_)-1 {
			return c.JSON(http.StatusInternalServerError, utils.H{"error": "could not find file name"})
		}
		fileName := path_[index+1:]

		commandPath := strings.TrimPrefix(path_[:index], "/")
		command, err := getRepositoryCommand(c, cd.Repository, commandPath)
		if err != nil {
			return err
		}

		switch v := command.(type) {
		case cmds.GlazeCommand:
			return output_file.CreateGlazedFileHandler(
				v,
				fileName,
				middlewares_...,
			)(c)

		case cmds.WriterCommand:
			handler := text.NewQueryHandler(command)

			baseName := filepath.Base(fileName)
			c.Response().Header().Set("Content-Disposition", "attachment; filename="+baseName)

			err := handler.Handle(c)
			if err != nil {
				return err
			}

			return nil

		default:
			return c.JSON(http.StatusInternalServerError, utils.H{"error": fmt.Sprintf("command %s is not a glazed/writer command", commandPath)})
		}

	})

	server.Router.GET(path+"/commands/*path", func(c echo.Context) error {
		path_ := c.Param("path")
		path_ = strings.TrimPrefix(path_, "/")
		path_ = strings.TrimSuffix(path_, "/")
		splitPath := strings.Split(path_, "/")
		if path_ == "" {
			splitPath = []string{}
		}
		renderNode, ok := cd.Repository.GetRenderNode(splitPath)
		if !ok {
			return errors.Errorf("command %s not found", path_)
		}
		templateName := cd.IndexTemplateName
		if cd.IndexTemplateName == "" {
			templateName = "commands.tmpl.html"
		}
		templ, err := cd.TemplateLookup.Lookup(templateName)
		if err != nil {
			return err
		}

		var nodes []*repositories.RenderNode

		if renderNode.Command != nil {
			nodes = append(nodes, renderNode)
		} else {
			nodes = append(nodes, renderNode.Children...)
		}
		err = templ.Execute(c.Response(), utils.H{
			"nodes": nodes,
			"path":  path,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return nil
}

type CommandNotFound struct {
	CommandPath string
}

func (e CommandNotFound) Error() string {
	return fmt.Sprintf("command %s not found", e.CommandPath)
}

type AmbiguousCommand struct {
	CommandPath       string
	PotentialCommands []string
}

func (e AmbiguousCommand) Error() string {
	return fmt.Sprintf("command %s is ambiguous, could be one of: %s", e.CommandPath, strings.Join(e.PotentialCommands, ", "))

}

// getRepositoryCommand lookups a command in the given repository and return success as bool and the given command,
// or sends an error code over HTTP using the gin.Context.
func getRepositoryCommand(c echo.Context, r *repositories.Repository, commandPath string) (
	cmds.Command,
	error,
) {
	path := strings.Split(commandPath, "/")
	commands := r.CollectCommands(path, false)
	if len(commands) == 0 {
		return nil, CommandNotFound{CommandPath: commandPath}
	}

	if len(commands) > 1 {
		err := &AmbiguousCommand{
			CommandPath: commandPath,
		}
		for _, command := range commands {
			description := command.Description()
			err.PotentialCommands = append(err.PotentialCommands, strings.Join(description.Parents, " ")+" "+description.Name)
		}
		return nil, err
	}

	// NOTE(manuel, 2023-05-15) Check if this is actually an alias, and populate the defaults from the alias flags
	// This could potentially be moved to the repository code itself

	return commands[0], nil
}
