package command_dir

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type HandlerParameters struct {
	Layers    map[string]map[string]interface{}
	Flags     map[string]interface{}
	Arguments map[string]interface{}
}

func NewHandlerParameters() *HandlerParameters {
	return &HandlerParameters{
		Layers:    map[string]map[string]interface{}{},
		Flags:     map[string]interface{}{},
		Arguments: map[string]interface{}{},
	}
}

// NewHandlerParametersFromLayerParams creates a new HandlerParameters from the config file.
// It currently requires a list of layerDefinitions in order to lookup the correct
// layers to stored as ParsedParameterLayer. It doesn't fail if configured layers don't exist.
//
// TODO(manuel, 2023-05-31) Add a way to validate the fact that overrides in a config file might
// have a typo and don't correspond to existing layer definitions in the application.
func NewHandlerParametersFromLayerParams(p *config.LayerParams) {
	ret := NewHandlerParameters()
	for name, l := range p.Layers {
		ret.Layers[name] = map[string]interface{}{}
		for k, v := range l {
			ret.Layers[name][k] = v
		}
	}

	for name, v := range p.Flags {
		ret.Flags[name] = v
	}

	for name, v := range p.Arguments {
		ret.Arguments[name] = v
	}
}

// Merge merges the given overrides into this one.
// If a layer is already present, it is merged with the given one.
// Flags and arguments are merged, overrides taking precedence.
func (ho *HandlerParameters) Merge(other *HandlerParameters) {
	for k, v := range other.Layers {
		if _, ok := ho.Layers[k]; !ok {
			ho.Layers[k] = map[string]interface{}{}
		}
		for k2, v2 := range v {
			ho.Layers[k][k2] = v2
		}
	}
	for k, v := range other.Flags {
		ho.Flags[k] = v
	}
	for k, v := range other.Arguments {
		ho.Arguments[k] = v
	}
}

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

	Overrides *HandlerParameters
	Defaults  *HandlerParameters
}

type CommandDirHandlerOption func(handler *CommandDirHandler)

func WithTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateName = name
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

func WithReplaceOverrides(overrides *HandlerParameters) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.Overrides = overrides
	}
}

func WithMergeOverrides(overrides *HandlerParameters) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = overrides
		} else {
			handler.Overrides.Merge(overrides)
		}
	}
}

func WithOverrideFlag(name string, value string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Flags[name] = value
	}
}

func WithOverrideArgument(name string, value string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Arguments[name] = value
	}
}

func WithMergeOverrideLayer(name string, layer map[string]interface{}) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Overrides.Layers[name]; !ok {
				handler.Overrides.Layers[name] = map[string]interface{}{}
			}
			handler.Overrides.Layers[name][k] = v
		}
	}
}

// WithLayerDefaults populates the defaults for the given layer. If a value is already set, the value is skipped.
func WithLayerDefaults(name string, layer map[string]interface{}) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Overrides.Layers[name]; !ok {
				handler.Overrides.Layers[name] = map[string]interface{}{}
			}
			if _, ok := handler.Overrides.Layers[name][k]; !ok {
				handler.Overrides.Layers[name][k] = v
			}
		}
	}
}

func WithReplaceOverrideLayer(name string, layer map[string]interface{}) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Layers[name] = layer
	}
}

// TODO(manuel, 2023-05-25) We can't currently override defaults, since they are parsed up front.
// For that we would need https://github.com/go-go-golems/glazed/issues/239
// So for now, we only deal with overrides.
//
// Handling all the way to configure defaults.

func WithReplaceDefaults(defaults *HandlerParameters) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.Defaults = defaults
	}
}

func WithMergeDefaults(defaults *HandlerParameters) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Defaults == nil {
			handler.Defaults = defaults
		} else {
			handler.Defaults.Merge(defaults)
		}
	}
}

func WithDefaultFlag(name string, value string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Flags[name] = value
	}
}

func WithDefaultArgument(name string, value string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Arguments[name] = value
	}
}

func WithMergeDefaultLayer(name string, layer *layers.ParsedParameterLayer) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		for k, v := range layer.Parameters {
			if _, ok := handler.Defaults.Layers[name]; !ok {
				handler.Defaults.Layers[name] = map[string]interface{}{}
			}
			handler.Defaults.Layers[name][k] = v
		}
	}
}

func WithReplaceDefaultLayer(name string, layer map[string]interface{}) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Layers[name] = layer
	}
}

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
	config *config.CommandDir,
	options ...CommandDirHandlerOption,
) (*CommandDirHandler, error) {
	cd := &CommandDirHandler{
		TemplateName:      config.TemplateName,
		IndexTemplateName: config.IndexTemplateName,
		AdditionalData:    config.AdditionalData,
	}

	if config.Overrides != nil {
		cd.Overrides = &HandlerParameters{
			Flags:     config.Overrides.Flags,
			Arguments: config.Overrides.Arguments,
			Layers:    config.Overrides.Layers,
		}
	}

	if config.Defaults != nil {
		cd.Defaults = &HandlerParameters{
			Flags:     config.Defaults.Flags,
			Arguments: config.Defaults.Arguments,
			Layers:    config.Defaults.Layers,
		}
	}

	for _, option := range options {
		option(cd)
	}

	// we run this after the options in order to get the DevMode value

	if cd.TemplateLookup == nil {
		if config.TemplateLookup != nil {
			patterns := config.TemplateLookup.Patterns
			if len(patterns) == 0 {
				patterns = []string{"**/*.tmpl.md", "**/*.tmpl.html"}
			}
			// we currently only support a single directory
			if len(config.TemplateLookup.Directories) != 1 {
				return nil, errors.New("template lookup directories must be exactly one")
			}
			cd.TemplateLookup = render.NewLookupTemplateFromFS(
				render.WithFS(os.DirFS(config.TemplateLookup.Directories[0])),
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

func (cd *CommandDirHandler) computeParserOptions() []parser.ParserOption {
	parserOptions := []parser.ParserOption{}

	// TODO(manuel, 2023-06-21) This needs to be handled for each backend, not just the HTML one
	if cd.Overrides != nil {
		if cd.Overrides.Flags != nil && len(cd.Overrides.Flags) > 0 {
			parserOptions = append(parserOptions,
				parser.WithAppendOverrides(parser.DefaultSlug, cd.Overrides.Flags),
			)
		}
		if cd.Overrides.Arguments != nil && len(cd.Overrides.Arguments) > 0 {
			parserOptions = append(parserOptions,
				parser.WithAppendOverrides(parser.DefaultSlug, cd.Overrides.Arguments),
			)
		}
		for slug, layer := range cd.Overrides.Layers {
			parserOptions = append(parserOptions, parser.WithAppendOverrides(slug, layer))
		}
	}

	if cd.Defaults != nil {
		if cd.Defaults.Flags != nil && len(cd.Defaults.Flags) > 0 {
			parserOptions = append(parserOptions,
				parser.WithPrependDefaults(parser.DefaultSlug, cd.Defaults.Flags),
			)
		}
		if cd.Defaults.Arguments != nil && len(cd.Defaults.Arguments) > 0 {
			parserOptions = append(parserOptions,
				parser.WithPrependDefaults(parser.DefaultSlug, cd.Defaults.Arguments),
			)
		}
		for slug, layer := range cd.Defaults.Layers {
			// we use prepend because that way, later options will actually override earlier flag values,
			// since they will be applied earlier.
			parserOptions = append(parserOptions, parser.WithPrependDefaults(slug, layer))
		}
	}

	return parserOptions
}

func (cd *CommandDirHandler) Serve(server *parka.Server, path string) error {
	if cd.Repository == nil {
		return fmt.Errorf("no repository configured")
	}

	path = strings.TrimSuffix(path, "/")

	server.Router.GET(path+"/data/*path", func(c *gin.Context) {
		commandPath := c.Param("path")
		commandPath = strings.TrimPrefix(commandPath, path)
		sqlCommand, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		json.HandleJSONQueryHandler(sqlCommand)
	})

	server.Router.GET(path+"/sqleton/*path",
		func(c *gin.Context) {
			commandPath := c.Param("path")
			commandPath = strings.TrimPrefix(commandPath, path)

			// Get repository command
			sqlCommand, ok := getRepositoryCommand(c, cd.Repository, commandPath)
			if !ok {
				c.JSON(404, gin.H{"error": "command not found"})
				return
			}

			options := []datatables.QueryHandlerOption{
				datatables.WithParserOptions(cd.computeParserOptions()...),
				datatables.WithTemplateLookup(cd.TemplateLookup),
				datatables.WithTemplateName(cd.TemplateName),
				datatables.WithAdditionalData(cd.AdditionalData),
			}

			datatables.HandleDataTables(sqlCommand, path, commandPath, options...)
		})

	server.Router.GET(path+"/download/*path", func(c *gin.Context) {
		// get file name at end of path
		index := strings.LastIndex(path, "/")
		if index == -1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		if index >= len(path)-1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		fileName := path[index+1:]

		commandPath := strings.TrimPrefix(path[:index], "/")
		sqlCommand, ok := getRepositoryCommand(c, cd.Repository, commandPath)
		if !ok {
			// JSON output and error code already handled by getRepositoryCommand
			return
		}

		// create a temporary file for glazed output
		tmpFile, err := os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", fileName))
		if err != nil {
			c.JSON(500, gin.H{"error": "could not create temporary file"})
			return
		}
		defer func(name string) {
			_ = os.Remove(name)
		}(tmpFile.Name())

		// now check file suffix for content-type
		glazedOverrides := map[string]interface{}{
			"output-file": tmpFile.Name(),
		}
		if strings.HasSuffix(fileName, ".csv") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "csv"
		} else if strings.HasSuffix(fileName, ".tsv") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "tsv"
		} else if strings.HasSuffix(fileName, ".md") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "markdown"
		} else if strings.HasSuffix(fileName, ".html") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "html"
		} else if strings.HasSuffix(fileName, ".json") {
			glazedOverrides["output"] = "json"
		} else if strings.HasSuffix(fileName, ".yaml") {
			glazedOverrides["yaml"] = "yaml"
		} else if strings.HasSuffix(fileName, ".xlsx") {
			glazedOverrides["output"] = "excel"
		} else if strings.HasSuffix(fileName, ".txt") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "ascii"
		} else {
			c.JSON(500, gin.H{"error": "could not determine output format"})
			return
		}

		parserOptions := cd.computeParserOptions()
		// override parameter layers at the end
		parserOptions = append(parserOptions, parser.WithAppendOverrides("glazed", glazedOverrides))
		_ = parserOptions

		_ = sqlCommand
		//handle := server.HandleSimpleQueryOutputFileCommand(
		//	sqlCommand,
		//	tmpFile.Name(),
		//	fileName,
		//	glazed.WithParserOptions(parserOptions...),
		//)
		//
		//handle(c)
	})

	return nil
}

// getRepositoryCommand lookups a command in the given repository and return success as bool and the given command,
// or sends an error code over HTTP using the gin.Context.
//
// TODO(manuel, 2023-05-31) This is an odd API, is it necessary?
func getRepositoryCommand(c *gin.Context, r *repositories.Repository, commandPath string) (cmds.GlazeCommand, bool) {
	path := strings.Split(commandPath, "/")
	commands := r.Root.CollectCommands(path, false)
	if len(commands) == 0 {
		return nil, false
	}

	if len(commands) > 1 {
		c.JSON(404, gin.H{"error": "ambiguous command"})
		return nil, false
	}

	// NOTE(manuel, 2023-05-15) Check if this is actually an alias, and populate the defaults from the alias flags
	// This could potentially be moved to the repository code itself

	glazedCommand, ok := commands[0].(cmds.GlazeCommand)
	if !ok || glazedCommand == nil {
		c.JSON(500, gin.H{"error": "command is not a glazed command"})
	}
	return glazedCommand, true
}
