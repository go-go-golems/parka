package command

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	output_file "github.com/go-go-golems/parka/pkg/glazed/handlers/output-file"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
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

	OverridesAndDefaults *config.OverridesAndDefaults

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

func WithOverridesAndDefaults(overridesAndDefaults *config.OverridesAndDefaults) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.OverridesAndDefaults = overridesAndDefaults
	}
}

func WithOverridesAndDefaultsOptions(opts ...config.OverridesAndDefaultsOption) CommandHandlerOption {
	return func(handler *CommandHandler) {
		for _, opt := range opts {
			opt(handler.OverridesAndDefaults)
		}
	}
}

func NewCommandHandler(
	command cmds.GlazeCommand,
	options ...CommandHandlerOption,
) *CommandHandler {
	c := &CommandHandler{
		Command:              command,
		TemplateName:         "",
		AdditionalData:       map[string]interface{}{},
		OverridesAndDefaults: &config.OverridesAndDefaults{},
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func NewCommandHandlerFromConfig(
	config_ *config.Command,
	loader loaders.FSCommandLoader,
	options ...CommandHandlerOption,
) (*CommandHandler, error) {
	c := &CommandHandler{
		TemplateName:         config_.TemplateName,
		AdditionalData:       config_.AdditionalData,
		OverridesAndDefaults: &config.OverridesAndDefaults{},
	}

	// get absolute path from config_.File
	absPath, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}
	var filePath string
	if strings.HasPrefix(config_.File, "/") {
		filePath = config_.File
	} else {
		filePath = absPath + config_.File
	}

	cmds_, aliases, err := loader.LoadCommandsFromFS(os.DirFS("/"), filePath, []cmds.CommandDescriptionOption{}, []alias.Option{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load commands from file")
	}

	allCmds := []cmds.GlazeCommand{}
	for _, cmd := range cmds_ {
		glazeCommand, ok := cmd.(cmds.GlazeCommand)
		if !ok {
			return nil, errors.Errorf(
				"command %s loaded from %s is not a GlazeCommand",
				cmd.Description().Name,
				filePath,
			)
		}
		allCmds = append(allCmds, glazeCommand)
	}
	for _, alias := range aliases {
		allCmds = append(allCmds, alias)
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
	if config_.Overrides != nil {
		c.OverridesAndDefaults.Overrides = &config.HandlerParameters{
			Flags:     config_.Overrides.Flags,
			Arguments: config_.Overrides.Arguments,
			Layers:    config_.Overrides.Layers,
		}
	} else {
		c.OverridesAndDefaults.Overrides = &config.HandlerParameters{
			Flags:     map[string]interface{}{},
			Arguments: map[string]interface{}{},
			Layers:    map[string]map[string]interface{}{},
		}
	}

	if config_.Defaults != nil {
		c.OverridesAndDefaults.Defaults = &config.HandlerParameters{
			Flags:     config_.Defaults.Flags,
			Arguments: config_.Defaults.Arguments,
			Layers:    config_.Defaults.Layers,
		}
	} else {
		c.OverridesAndDefaults.Defaults = &config.HandlerParameters{
			Flags:     map[string]interface{}{},
			Arguments: map[string]interface{}{},
			Layers:    map[string]map[string]interface{}{},
		}
	}

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

	server.Router.GET(path+"/data", func(c *gin.Context) {
		json.CreateJSONQueryHandler(ch.Command)(c)
	})
	server.Router.GET(path+"/sqleton", func(c *gin.Context) {
		options := []datatables.QueryHandlerOption{
			datatables.WithParserOptions(ch.OverridesAndDefaults.ComputeParserOptions(ch.Stream)...),
			datatables.WithTemplateLookup(ch.TemplateLookup),
			datatables.WithTemplateName(ch.TemplateName),
			datatables.WithAdditionalData(ch.AdditionalData),
			datatables.WithStreamRows(ch.Stream),
		}

		datatables.CreateDataTablesHandler(ch.Command, path, "", options...)(c)
	})
	server.Router.GET(path+"/download/*path", func(c *gin.Context) {
		path_ := c.Param("path")
		path_ = strings.TrimPrefix(path_, path+"/")
		index := strings.LastIndex(path_, "/")
		if index == -1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		if index >= len(path_)-1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		fileName := path_[index+1:]

		parserOptions := ch.OverridesAndDefaults.ComputeParserOptions(ch.Stream)

		output_file.CreateGlazedFileHandler(
			ch.Command,
			fileName,
			parserOptions...,
		)(c)
	})

	return nil
}
