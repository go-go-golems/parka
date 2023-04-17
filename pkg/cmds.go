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
	// NOTE(manuel, 2023-04-16) The better API would be to pass in a list of HandlerOptions

	// This parser handler business is maybe too over-generified, all handlers are currently parser
	// handlers except for the template handler.
	parserOptions []glazed.ParserOption,
	handlers ...glazed.CommandHandlerFunc,
) gin.HandlerFunc {
	handlers_ := []glazed.CommandHandlerFunc{
		glazed.NewCommandHandlerFunc(cmd, glazed.NewCommandQueryParser(cmd, parserOptions...)),
	}
	handlers_ = append(handlers_, handlers...)
	return glazed.NewGinHandlerFromCommandHandlers(cmd, handlers_...)
}

// TODO(manuel, 2023-02-28) We want to provide a handler to catch errors while parsing parameters

func (s *Server) HandleSimpleFormCommand(
	cmd cmds.GlazeCommand,
	handlers ...glazed.CommandHandlerFunc,
) gin.HandlerFunc {
	handlers_ := []glazed.CommandHandlerFunc{
		glazed.NewCommandHandlerFunc(cmd, glazed.NewCommandFormParser(cmd)),
	}
	handlers_ = append(handlers_, handlers...)
	return glazed.NewGinHandlerFromCommandHandlers(cmd, handlers_...)
}
