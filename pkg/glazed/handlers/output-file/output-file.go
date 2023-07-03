package output_file

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/parka/pkg/glazed/handlers"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/glazed"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type OutputFileHandler struct {
	handler        handlers.Handler
	outputFileName string
}

func NewOutputFileHandler(handler handlers.Handler, outputFileName string) *OutputFileHandler {
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

func CreateGlazedFileHandler(
	cmd cmds.GlazeCommand,
	fileName string,
	parserOptions ...parser.ParserOption,
) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			glazedOverrides["output"] = "yaml"
		} else if strings.HasSuffix(fileName, ".xlsx") {
			glazedOverrides["output"] = "excel"
		} else if strings.HasSuffix(fileName, ".txt") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "ascii"
		} else {
			c.JSON(500, gin.H{"error": "could not determine output format"})
			return
		}

		// override parameter layers at the end
		parserOptions = append(parserOptions, parser.WithAppendOverrides("glazed", glazedOverrides))

		handler := glazed.NewQueryHandler(cmd, glazed.WithQueryHandlerParserOptions(parserOptions...))
		outputFileHandler := NewOutputFileHandler(handler, tmpFile.Name())

		err = outputFileHandler.Handle(c, c.Writer)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

	}
}
