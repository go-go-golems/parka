package handlers

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/labstack/echo/v4"
)

// Handler wraps the normal handle method to allow rendering into a different writer,
// so that we can provide file downloads.
//
// TODO(manuel, 2024-05-07) I don't think we actually need this
type Handler interface {
	Handle(c echo.Context) error
}

type UnsupportedCommandError struct {
	Command cmds.Command
}

func (e *UnsupportedCommandError) Error() string {
	return "unsupported command: " + e.Command.Description().Name
}
