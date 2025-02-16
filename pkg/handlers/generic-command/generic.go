package generic_command

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/trie"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
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
	"github.com/rs/zerolog/log"
)

type GenericCommandHandler struct {
	// If true, all glazed outputs will try to use a row output if possible.
	// This means that "ragged" objects (where columns might not all be present)
	// will have missing columns, only the fields of the first object will be used
	// as rows.
	//
	// This is true per default, and needs to be explicitly set to false to use
	// a normal TableMiddleware oriented output.
	Stream bool

	// AdditionalData is passed to the template being rendered.
	AdditionalData map[string]interface{}

	ParameterFilter *config.ParameterFilter

	// TemplateName is the name of the template that is lookup up through the given TemplateLookup
	// used to render the glazed command.
	TemplateName string
	// IndexTemplateName is the name of the template that is looked up through TemplateLookup to render
	// command indexes. Leave empty to not render index pages at all.
	IndexTemplateName string
	// TemplateLookup is used to look up both TemplateName and IndexTemplateName
	TemplateLookup render.TemplateLookup

	// path under which the command handler is served
	BasePath string

	// preMiddlewares are run before the parameter filter middlewares
	preMiddlewares []middlewares.Middleware
	// postMiddlewares are run after the parameter filter middlewares
	postMiddlewares []middlewares.Middleware
	// middlewares contains all middlewares in order: pre + parameter filter + post
	middlewares []middlewares.Middleware
}

func NewGenericCommandHandler(options ...GenericCommandHandlerOption) (*GenericCommandHandler, error) {
	handler := &GenericCommandHandler{
		AdditionalData:  map[string]interface{}{},
		TemplateLookup:  datatables.NewDataTablesLookupTemplate(),
		ParameterFilter: &config.ParameterFilter{},
		preMiddlewares:  []middlewares.Middleware{},
		postMiddlewares: []middlewares.Middleware{},
	}

	for _, opt := range options {
		opt(handler)
	}

	// Compute the final middleware chain
	parameterMiddlewares := handler.ParameterFilter.ComputeMiddlewares(handler.Stream)
	handler.middlewares = append(handler.preMiddlewares, parameterMiddlewares...)
	handler.middlewares = append(handler.middlewares, handler.postMiddlewares...)

	if handler.TemplateLookup == nil {
		handler.TemplateLookup = datatables.NewDataTablesLookupTemplate()
	}

	err := handler.TemplateLookup.Reload()
	if err != nil {
		return nil, err
	}

	return handler, nil
}

type GenericCommandHandlerOption func(handler *GenericCommandHandler)

func WithTemplateName(name string) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.TemplateName = name
	}
}

func WithParameterFilter(overridesAndDefaults *config.ParameterFilter) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.ParameterFilter = overridesAndDefaults
	}
}

func WithParameterFilterOptions(opts ...config.ParameterFilterOption) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		for _, opt := range opts {
			opt(handler.ParameterFilter)
		}
	}
}

func WithDefaultTemplateName(name string) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		if handler.TemplateName == "" {
			handler.TemplateName = name
		}
	}
}

func WithIndexTemplateName(name string) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.IndexTemplateName = name
	}
}

func WithDefaultIndexTemplateName(name string) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		if handler.IndexTemplateName == "" {
			handler.IndexTemplateName = name
		}
	}
}

// WithMergeAdditionalData merges the passed in map with the handler's AdditionalData map.
// If a value is already set in the AdditionalData map and override is true, it will get overwritten.
func WithMergeAdditionalData(data map[string]interface{}, override bool) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
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

func WithTemplateLookup(lookup render.TemplateLookup) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.TemplateLookup = lookup
	}
}

func WithPreMiddlewares(middlewares ...middlewares.Middleware) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.preMiddlewares = append(handler.preMiddlewares, middlewares...)
	}
}

func WithPostMiddlewares(middlewares ...middlewares.Middleware) GenericCommandHandlerOption {
	return func(handler *GenericCommandHandler) {
		handler.postMiddlewares = append(handler.postMiddlewares, middlewares...)
	}
}

func (gch *GenericCommandHandler) ServeSingleCommand(server *parka.Server, basePath string, command cmds.Command) error {
	gch.BasePath = basePath

	server.Router.GET(basePath+"/data", func(c echo.Context) error {
		return gch.ServeData(c, command)
	})
	server.Router.GET(basePath+"/text", func(c echo.Context) error {
		return gch.ServeText(c, command)
	})
	server.Router.GET(basePath+"/stream", func(c echo.Context) error {
		return gch.ServeStreaming(c, command)
	})
	server.Router.GET(basePath+"/download/*", func(c echo.Context) error {
		return gch.ServeDownload(c, command)
	})
	// don't use a specific datatables path here
	server.Router.GET(basePath, func(c echo.Context) error {
		return gch.ServeDataTables(c, command, basePath+"/download")
	})

	return nil
}

func (gch *GenericCommandHandler) ServeRepository(server *parka.Server, basePath string, repository *repositories.Repository) error {
	basePath = strings.TrimSuffix(basePath, "/")
	gch.BasePath = basePath

	server.Router.GET(basePath+"/data/*", func(c echo.Context) error {
		commandPath := c.Param("*")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(repository, commandPath)
		if err != nil {
			log.Debug().
				Str("commandPath", commandPath).
				Str("basePath", basePath).
				Msg("could not find command")
			return err
		}

		return gch.ServeData(c, command)
	})

	server.Router.GET(basePath+"/text/*", func(c echo.Context) error {
		commandPath := c.Param("*")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(repository, commandPath)
		if err != nil {
			log.Debug().
				Str("commandPath", commandPath).
				Str("basePath", basePath).
				Msg("could not find command")
			return err
		}

		return gch.ServeText(c, command)
	})

	server.Router.GET(basePath+"/streaming/*", func(c echo.Context) error {
		commandPath := c.Param("*")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, err := getRepositoryCommand(repository, commandPath)
		if err != nil {
			log.Debug().
				Str("commandPath", commandPath).
				Str("basePath", basePath).
				Msg("could not find command")
			return err
		}

		return gch.ServeStreaming(c, command)
	})

	server.Router.GET(basePath+"/datatables/*", func(c echo.Context) error {
		commandPath := c.Param("*")
		commandPath = strings.TrimPrefix(commandPath, "/")

		// Get repository command
		command, err := getRepositoryCommand(repository, commandPath)
		if err != nil {
			log.Debug().
				Str("commandPath", commandPath).
				Str("basePath", basePath).
				Msg("could not find command")
			return err
		}

		return gch.ServeDataTables(c, command, basePath+"/download/"+commandPath)
	})

	server.Router.GET(basePath+"/download/*", func(c echo.Context) error {
		commandPath := c.Param("*")
		commandPath = strings.TrimPrefix(commandPath, "/")
		// strip file name from path
		index := strings.LastIndex(commandPath, "/")
		if index == -1 {
			return errors.New("could not find file name")
		}
		if index >= len(commandPath)-1 {
			return errors.New("could not find file name")
		}
		commandPath = commandPath[:index]

		command, err := getRepositoryCommand(repository, commandPath)
		if err != nil {
			log.Debug().
				Str("commandPath", commandPath).
				Str("basePath", basePath).
				Msg("could not find command")
			return err
		}

		return gch.ServeDownload(c, command)
	})

	server.Router.GET(basePath+"/commands/*", func(c echo.Context) error {
		path_ := c.Param("*")
		path_ = strings.TrimPrefix(path_, "/")
		path_ = strings.TrimSuffix(path_, "/")
		splitPath := strings.Split(path_, "/")
		if path_ == "" {
			splitPath = []string{}
		}
		renderNode, ok := repository.GetRenderNode(splitPath)
		if !ok {
			return errors.Errorf("command %s not found", path_)
		}
		templateName := gch.IndexTemplateName
		if gch.IndexTemplateName == "" {
			templateName = "commands.tmpl.html"
		}
		templ, err := gch.TemplateLookup.Lookup(templateName)
		if err != nil {
			return err
		}

		var nodes []*trie.RenderNode

		if renderNode.Command != nil {
			nodes = append(nodes, renderNode)
		} else {
			nodes = append(nodes, renderNode.Children...)
		}
		err = templ.Execute(c.Response(), utils.H{
			"nodes": nodes,
			"path":  basePath,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return nil
}

// computeDataTablesOptions returns the options used for DataTables handlers
func (gch *GenericCommandHandler) computeDataTablesOptions() []datatables.QueryHandlerOption {
	return []datatables.QueryHandlerOption{
		datatables.WithMiddlewares(gch.preMiddlewares...),
		datatables.WithMiddlewares(gch.middlewares...),
		datatables.WithMiddlewares(gch.postMiddlewares...),
		datatables.WithTemplateLookup(gch.TemplateLookup),
		datatables.WithTemplateName(gch.TemplateName),
		datatables.WithAdditionalData(gch.AdditionalData),
		datatables.WithStreamRows(gch.Stream),
	}
}

// computeJSONOptions returns the options used for JSON handlers
func (gch *GenericCommandHandler) computeJSONOptions() []json.QueryHandlerOption {
	return []json.QueryHandlerOption{
		json.WithMiddlewares(gch.preMiddlewares...),
		json.WithMiddlewares(gch.middlewares...),
		json.WithMiddlewares(gch.postMiddlewares...),
	}
}

// computeTextOptions returns the options used for text handlers
func (gch *GenericCommandHandler) computeTextOptions() []text.QueryHandlerOption {
	return []text.QueryHandlerOption{
		text.WithMiddlewares(gch.preMiddlewares...),
		text.WithMiddlewares(gch.middlewares...),
		text.WithMiddlewares(gch.postMiddlewares...),
	}
}

// computeSSEOptions returns the options used for SSE handlers
func (gch *GenericCommandHandler) computeSSEOptions() []sse.QueryHandlerOption {
	return []sse.QueryHandlerOption{
		sse.WithMiddlewares(gch.preMiddlewares...),
		sse.WithMiddlewares(gch.middlewares...),
		sse.WithMiddlewares(gch.postMiddlewares...),
	}
}

// computeOutputFileOptions returns the options used for output file handlers
func (gch *GenericCommandHandler) computeOutputFileOptions() []output_file.QueryHandlerOption {
	return []output_file.QueryHandlerOption{
		output_file.WithMiddlewares(gch.preMiddlewares...),
		output_file.WithMiddlewares(gch.middlewares...),
		output_file.WithMiddlewares(gch.postMiddlewares...),
	}
}

func (gch *GenericCommandHandler) ServeData(c echo.Context, command cmds.Command) error {
	switch v := command.(type) {
	case cmds.GlazeCommand:
		return json.CreateJSONQueryHandler(v, gch.computeJSONOptions()...)(c)
	default:
		return text.CreateQueryHandler(v, gch.computeTextOptions()...)(c)
	}
}

func (gch *GenericCommandHandler) ServeText(c echo.Context, command cmds.Command) error {
	return text.CreateQueryHandler(command, gch.computeTextOptions()...)(c)
}

func (gch *GenericCommandHandler) ServeStreaming(c echo.Context, command cmds.Command) error {
	return sse.CreateQueryHandler(command, gch.computeSSEOptions()...)(c)
}

func (gch *GenericCommandHandler) ServeDataTables(c echo.Context, command cmds.Command, downloadPath string) error {
	switch v := command.(type) {
	case cmds.GlazeCommand:
		return datatables.CreateDataTablesHandler(v, gch.BasePath, downloadPath, gch.computeDataTablesOptions()...)(c)
	default:
		return c.JSON(http.StatusInternalServerError, utils.H{"error": "command is not a glazed command"})
	}
}

func (gch *GenericCommandHandler) ServeDownload(c echo.Context, command cmds.Command) error {
	path_ := c.Request().URL.Path
	index := strings.LastIndex(path_, "/")
	if index == -1 {
		return c.JSON(http.StatusInternalServerError, utils.H{"error": "could not find file name"})
	}
	if index >= len(path_)-1 {
		return c.JSON(http.StatusInternalServerError, utils.H{"error": "could not find file name"})
	}
	fileName := path_[index+1:]

	switch v := command.(type) {
	case cmds.GlazeCommand:
		return output_file.CreateGlazedFileHandler(
			v,
			fileName,
			gch.computeOutputFileOptions()...,
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
		return c.JSON(http.StatusInternalServerError, utils.H{"error": "command is not a glazed/writer command"})
	}
}

// getRepositoryCommand lookups a command in the given repository and return success as bool and the given command,
// or sends an error code over HTTP using the gin.Context.
func getRepositoryCommand(r *repositories.Repository, commandPath string) (cmds.Command, error) {
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
