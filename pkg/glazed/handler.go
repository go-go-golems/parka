package glazed

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
)

type Handler interface {
	Handle(c *gin.Context, w io.Writer) error
}

type OutputFileHandler struct {
	handler        Handler
	outputFileName string
}

func NewOutputFileHandler(handler Handler, outputFileName string) *OutputFileHandler {
	h := &OutputFileHandler{
		handler:        handler,
		outputFileName: outputFileName,
	}

	return h
}

func (h *OutputFileHandler) Handle(c *gin.Context, w io.Writer) error {
	buf := &bytes.Buffer{}
	err := h.handler.Handle(c, buf)
	if err != nil {
		return err
	}

	c.Status(200)

	f, err := os.Open(h.outputFileName)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	baseName := filepath.Base(h.outputFileName)

	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+baseName)

	_, err = io.Copy(c.Writer, f)
	if err != nil {
		return err
	}

	return nil
}
