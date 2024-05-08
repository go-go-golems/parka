package command

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	output_file "github.com/go-go-golems/parka/pkg/glazed/handlers/output-file"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"strings"
)

type CommandHandler struct {
	DevMode bool

	// TemplateName is the name of the template that is lookup up through the given TemplateLookup
	// used to render the glazed command.
	TemplateName string
	// TemplateLookup is used to look up both TemplateName and IndexTemplateName
	TemplateLookup render.TemplateLookup

	// can be any of BareCommand, WriterCommand or GlazeCommand
	Command cmds.GlazeCommand

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

type CommandHandlerOption func(*CommandHandler)

func WithDevMode(devMode bool) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.DevMode = devMode
	}
}

func WithTemplateName(templateName string) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.TemplateName = templateName
	}
}

func WithDefaultTemplateName(defaultTemplateName string) CommandHandlerOption {
	return func(handler *CommandHandler) {
		if handler.TemplateName == "" {
			handler.TemplateName = defaultTemplateName
		}
	}
}

func WithTemplateLookup(templateLookup render.TemplateLookup) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.TemplateLookup = templateLookup
	}
}

// WithMergeAdditionalData merges the passed in map with the handler's AdditionalData map.
// If a value is already set in the AdditionalData map and override is true, it will get overwritten.
func WithMergeAdditionalData(data map[string]interface{}, override bool) CommandHandlerOption {
	return func(handler *CommandHandler) {
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

func WithParameterFilter(parameterFilter *config.ParameterFilter) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.ParameterFilter = parameterFilter
	}
}

func WithParameterFilterOptions(opts ...config.ParameterFilterOption) CommandHandlerOption {
	return func(handler *CommandHandler) {
		for _, opt := range opts {
			opt(handler.ParameterFilter)
		}
	}
}

func NewCommandHandler(
	command cmds.GlazeCommand,
	options ...CommandHandlerOption,
) *CommandHandler {
	c := &CommandHandler{
		Command:         command,
		TemplateName:    "",
		AdditionalData:  map[string]interface{}{},
		ParameterFilter: &config.ParameterFilter{},
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func NewCommandHandlerFromConfig(
	config_ *config.Command,
	loader loaders.CommandLoader,
	options ...CommandHandlerOption,
) (*CommandHandler, error) {
	c := &CommandHandler{
		TemplateName:    config_.TemplateName,
		AdditionalData:  config_.AdditionalData,
		ParameterFilter: &config.ParameterFilter{},
	}

	fs_, filePath, err := loaders.FileNameToFsFilePath(config_.File)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get absolute path")
	}

	cmds_, err := loaders.LoadCommandsFromFS(
		fs_, filePath, config_.File,
		loader, []cmds.CommandDescriptionOption{}, []alias.Option{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load commands from file")
	}

	allCmds := []cmds.GlazeCommand{}
	for _, cmd := range cmds_ {
		switch v := cmd.(type) {
		case *alias.CommandAlias:
			allCmds = append(allCmds, v)
		case cmds.GlazeCommand:
			allCmds = append(allCmds, v)
		default:
			return nil, errors.Errorf(
				"command %s loaded from %s is not a GlazeCommand",
				cmd.Description().Name,
				filePath,
			)
		}
	}
	if len(allCmds) == 0 {
		return nil, errors.Errorf(
			"no commands found in %s",
			filePath,
		)
	}

	if len(allCmds) > 1 {
		return nil, errors.Errorf(
			"more than one command found in %s, please specify which one to use",
			filePath,
		)
	}

	c.Command = allCmds[0]

	// TODO(manuel, 2023-08-06) I think we hav eto find the proper command here

	// NOTE(manuel, 2023-08-03) most of this matches CommandDirHandler, maybe at some point we could unify them both
	// Let's see when this starts causing trouble again
	c.ParameterFilter.Overrides = config_.Overrides
	c.ParameterFilter.Defaults = config_.Defaults
	c.ParameterFilter.Whitelist = config_.Whitelist
	c.ParameterFilter.Blacklist = config_.Blacklist

	// by default, we stream
	if config_.Stream != nil {
		c.Stream = *config_.Stream
	} else {
		c.Stream = true
	}

	for _, option := range options {
		option(c)
	}

	// we run this after the options in order to get the DevMode value
	if c.TemplateLookup == nil {
		if config_.TemplateLookup != nil {
			patterns := config_.TemplateLookup.Patterns
			if len(patterns) == 0 {
				patterns = []string{"**/*.tmpl.md", "**/*.tmpl.html"}
			}
			// we currently only support a single directory
			if len(config_.TemplateLookup.Directories) != 1 {
				return nil, errors.New("template lookup directories must be exactly one")
			}
			c.TemplateLookup = render.NewLookupTemplateFromFS(
				render.WithFS(os.DirFS(config_.TemplateLookup.Directories[0])),
				render.WithBaseDir(""),
				render.WithPatterns(patterns...),
				render.WithAlwaysReload(c.DevMode),
			)
		} else {
			c.TemplateLookup = datatables.NewDataTablesLookupTemplate()
		}
	}

	err = c.TemplateLookup.Reload()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (ch *CommandHandler) Serve(server *parka.Server, path string) error {
	path = strings.TrimSuffix(path, "/")

	middlewares_ := ch.ParameterFilter.ComputeMiddlewares(ch.Stream)

	server.Router.GET(path+"/data", func(c echo.Context) error {
		options := []json.QueryHandlerOption{
			json.WithMiddlewares(middlewares_...),
		}
		return json.CreateJSONQueryHandler(ch.Command, options...)(c)
	})
	// TODO(manuel, 2024-01-17) This doesn't seem to match what is in command-dir
	server.Router.GET(path+"/glazed", func(c echo.Context) error {
		options := []datatables.QueryHandlerOption{
			datatables.WithMiddlewares(middlewares_...),
			datatables.WithTemplateLookup(ch.TemplateLookup),
			datatables.WithTemplateName(ch.TemplateName),
			datatables.WithAdditionalData(ch.AdditionalData),
			datatables.WithStreamRows(ch.Stream),
		}

		return datatables.CreateDataTablesHandler(ch.Command, path, "", options...)(c)
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

		return output_file.CreateGlazedFileHandler(
			ch.Command,
			fileName,
			middlewares_...,
		)(c)
	})

	return nil
}
