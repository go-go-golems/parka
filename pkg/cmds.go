package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/parka/pkg/glazed"
	"html/template"
)

type JSONMarshaler interface {
	MarshalJSON() ([]byte, error)
}

// HTMLTemplateHandler is a handler that renders a template
// and also provides affordances to control what input parameters are passed downstream.
// Its main purpose is to be used as a fragment renderer for htmx calls.
type HTMLTemplateHandler struct {
	Template *template.Template
}

func (s *Server) HandleSimpleQueryCommand(
	cmd cmds.GlazeCommand,
	options ...glazed.HandleOption,
) gin.HandlerFunc {
	opts := glazed.NewHandleOptions(options)
	opts.Handlers = append(opts.Handlers,
		glazed.NewCommandHandlerFunc(cmd,
			glazed.NewCommandQueryParser(cmd, opts.ParserOptions...)),
	)
	return glazed.GinHandleGlazedCommand(cmd, opts)
}

func (s *Server) HandleSimpleQueryOutputFileCommand(
	cmd cmds.GlazeCommand,
	outputFile string,
	fileName string,
	options ...glazed.HandleOption,
) gin.HandlerFunc {
	opts := glazed.NewHandleOptions(options)
	opts.Handlers = append(opts.Handlers,
		glazed.NewCommandHandlerFunc(cmd, glazed.NewCommandQueryParser(cmd, opts.ParserOptions...)),
	)
	return glazed.GinHandleGlazedCommandWithOutputFile(cmd, outputFile, fileName, opts)
}

// TODO(manuel, 2023-02-28) We want to provide a handler to catch errors while parsing parameters

func (s *Server) HandleSimpleFormCommand(
	cmd cmds.GlazeCommand,
	options ...glazed.HandleOption,
) gin.HandlerFunc {
	opts := &glazed.HandleOptions{}
	for _, option := range options {
		option(opts)
	}
	opts.Handlers = append(opts.Handlers,
		glazed.NewCommandHandlerFunc(cmd, glazed.NewCommandFormParser(cmd, opts.ParserOptions...)),
	)
	return glazed.GinHandleGlazedCommand(cmd, opts)
}
