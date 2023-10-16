package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"io"
)

type Handler interface {
	Handle(c *gin.Context, w io.Writer) error
}

type UnsupportedCommandError struct {
	Command cmds.Command
}

func (e *UnsupportedCommandError) Error() string {
	return "unsupported command: " + e.Command.Description().Name
}
