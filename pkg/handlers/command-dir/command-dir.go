package command_dir

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/fs"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	output_file "github.com/go-go-golems/parka/pkg/glazed/handlers/output-file"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/sse"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/text"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
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
	Repository *fs.Repository

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

type CommandDirHandlerOption func(handler *CommandDirHandler)

func WithTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateName = name
	}
}

func WithOverridesAndDefaults(overridesAndDefaults *config.OverridesAndDefaults) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.OverridesAndDefaults = overridesAndDefaults
	}
}

func WithOverridesAndDefaultsOptions(opts ...config.OverridesAndDefaultsOption) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		for _, opt := range opts {
			opt(handler.OverridesAndDefaults)
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

func WithRepository(r *fs.Repository) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.Repository = r
	}
}

func NewCommandDirHandlerFromConfig(
	config_ *config.CommandDir,
	options ...CommandDirHandlerOption,
) (*CommandDirHandler, error) {
	cd := &CommandDirHandler{
		TemplateName:         config_.TemplateName,
		IndexTemplateName:    config_.IndexTemplateName,
		AdditionalData:       config_.AdditionalData,
		OverridesAndDefaults: &config.OverridesAndDefaults{},
	}

	if config_.Overrides != nil {
		cd.OverridesAndDefaults.Overrides = &config.HandlerParameters{
			Flags:     config_.Overrides.Flags,
			Arguments: config_.Overrides.Arguments,
			Layers:    config_.Overrides.Layers,
		}
	} else {
		cd.OverridesAndDefaults.Overrides = &config.HandlerParameters{
			Flags:     map[string]interface{}{},
			Arguments: map[string]interface{}{},
			Layers:    map[string]map[string]interface{}{},
		}
	}

	if config_.Defaults != nil {
		cd.OverridesAndDefaults.Defaults = &config.HandlerParameters{
			Flags:     config_.Defaults.Flags,
			Arguments: config_.Defaults.Arguments,
			Layers:    config_.Defaults.Layers,
		}
	} else {
		cd.OverridesAndDefaults.Defaults = &config.HandlerParameters{
			Flags:     map[string]interface{}{},
			Arguments: map[string]interface{}{},
			Layers:    map[string]map[string]interface{}{},
		}
	}

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

	server.Router.GET(path+"/data/*path", func(c *gin.Context) {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": fmt.Sprintf("command %s not found", commandPath)})
			return
		}

		switch v := command.(type) {
		case cmds.GlazeCommand:
			json.CreateJSONQueryHandler(v)(c)
		default:
			text.CreateQueryHandler(v)(c)
		}
	})

	server.Router.GET(path+"/text/*path", func(c *gin.Context) {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": fmt.Sprintf("command %s not found", commandPath)})
			return
		}

		parserOptions := cd.OverridesAndDefaults.ComputeParserOptions(cd.Stream)
		text.CreateQueryHandler(command, parserOptions...)(c)
	})

	server.Router.GET(path+"/streaming/*path", func(c *gin.Context) {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, "/")
		command, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": fmt.Sprintf("command %s not found", commandPath)})
			return
		}

		parserOptions := cd.OverridesAndDefaults.ComputeParserOptions(cd.Stream)
		sse.CreateQueryHandler(command, parserOptions...)(c)
	})

	server.Router.GET(path+"/datatables/*path",
		func(c *gin.Context) {
			commandPath := c.Param("path")
			commandPath = strings.TrimPrefix(commandPath, "/")

			// Get repository command
			command, ok := getRepositoryCommand(c, cd.Repository, commandPath)
			if !ok {
				c.JSON(404, gin.H{"error": fmt.Sprintf("command %s not found", commandPath)})
				return
			}
			switch v := command.(type) {
			case cmds.GlazeCommand:
				options := []datatables.QueryHandlerOption{
					datatables.WithParserOptions(cd.OverridesAndDefaults.ComputeParserOptions(cd.Stream)...),
					datatables.WithTemplateLookup(cd.TemplateLookup),
					datatables.WithTemplateName(cd.TemplateName),
					datatables.WithAdditionalData(cd.AdditionalData),
					datatables.WithStreamRows(cd.Stream),
				}

				datatables.CreateDataTablesHandler(v, path, commandPath, options...)(c)
			default:
				c.JSON(500, gin.H{"error": fmt.Sprintf("command %s is not a glazed command", commandPath)})
			}
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

		commandPath := strings.TrimPrefix(path_[:index], "/")
		command, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			// JSON output and error code already handled by getRepositoryCommand
			return
		}
		parserOptions := cd.OverridesAndDefaults.ComputeParserOptions(cd.Stream)

		switch v := command.(type) {
		case cmds.GlazeCommand:
			output_file.CreateGlazedFileHandler(
				v,
				fileName,
				parserOptions...,
			)(c)

		case cmds.WriterCommand:
			handler := text.NewQueryHandler(command)

			baseName := filepath.Base(fileName)
			c.Writer.Header().Set("Content-Disposition", "attachment; filename="+baseName)

			err := handler.Handle(c, c.Writer)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

		default:
			c.JSON(500, gin.H{"error": fmt.Sprintf("command %s is not a glazed/writer command", commandPath)})
		}
	})

	return nil
}

// getRepositoryCommand lookups a command in the given repository and return success as bool and the given command,
// or sends an error code over HTTP using the gin.Context.
func getRepositoryCommand(c *gin.Context, r repositories.Repository, commandPath string) (
	cmds.Command,
	bool,
) {
	path := strings.Split(commandPath, "/")
	commands := r.CollectCommands(path, false)
	if len(commands) == 0 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("command %s not found", commandPath)})
		return nil, false
	}

	if len(commands) > 1 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("ambiguous command %s", commandPath)})
		return nil, false
	}

	// NOTE(manuel, 2023-05-15) Check if this is actually an alias, and populate the defaults from the alias flags
	// This could potentially be moved to the repository code itself

	return commands[0], true
}
