package handlers

import (
	"github.com/gin-gonic/gin"
	"io"
)

type Handler interface {
	Handle(c *gin.Context, w io.Writer) error
}
