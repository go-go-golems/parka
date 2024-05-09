package command

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	generic_command "github.com/go-go-golems/parka/pkg/handlers/generic-command"
	"github.com/go-go-golems/parka/pkg/render"
	parka "github.com/go-go-golems/parka/pkg/server"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type CommandHandler struct {
	generic_command.GenericCommandHandler
	DevMode bool

	// can be any of BareCommand, WriterCommand or GlazeCommand
	Command cmds.Command
}

type CommandHandlerOption func(*CommandHandler)

func WithDevMode(devMode bool) CommandHandlerOption {
	return func(handler *CommandHandler) {
		handler.DevMode = devMode
	}
}

func WithGenericCommandHandlerOptions(options ...generic_command.GenericCommandHandlerOption) CommandHandlerOption {
	return func(handler *CommandHandler) {
		for _, option := range options {
			option(&handler.GenericCommandHandler)
		}
	}
}

func NewCommandHandler(
	command cmds.Command,
	options ...CommandHandlerOption,
) *CommandHandler {
	c := &CommandHandler{
		GenericCommandHandler: *generic_command.NewGenericCommandHandler(),
		Command:               command,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

func LoadCommandFromFile(path string, loader loaders.CommandLoader) (cmds.Command, error) {
	fs_, filePath, err := loaders.FileNameToFsFilePath(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get absolute path")
	}

	cmds_, err := loaders.LoadCommandsFromFS(
		fs_, filePath, path,
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

	return allCmds[0], nil
}

func NewCommandHandlerFromConfig(
	config_ *config.Command,
	loader loaders.CommandLoader,
	options ...CommandHandlerOption,
) (*CommandHandler, error) {
	genericOptions := []generic_command.GenericCommandHandlerOption{
		generic_command.WithTemplateName(config_.TemplateName),
		generic_command.WithMergeAdditionalData(config_.AdditionalData, true),
	}
	// TODO(manuel, 2024-05-09) To make this reloadable on dev mode, we would actually need to thunk this and pass the thunk to the GenericCommandHandler
	cmd, err := LoadCommandFromFile(config_.File, loader)
	if err != nil {
		return nil, err
	}

	c := NewCommandHandler(cmd, WithGenericCommandHandlerOptions(genericOptions...))
	// TODO(manuel, 2024-05-09) Handle devmode
	c.Command = cmd

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

	// TODO(manuel, 2024-05-09) The TemplateLookup initialization probably makes sense to be extracted as a reusable component
	// to load templates dynamically in dev mode, since it's shared across quite a few handlers (see command-dir).

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

	return ch.GenericCommandHandler.ServeSingleCommand(server, path, ch.Command)
}
