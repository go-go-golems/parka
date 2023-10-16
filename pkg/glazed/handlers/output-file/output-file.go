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

	_, err = io.Copy(c.Writer, f)
	if err != nil {
		return err
	}

	return nil
}

// CreateGlazedFileHandler creates a handler that will run a glazed command and write the output
// with a Content-Disposition header to the response writer.
//
// If an output format requires writing to a temporary file locally, such as excel,
// the handler is wrapped in a temporary file handler.
func CreateGlazedFileHandler(
	cmd cmds.GlazeCommand,
	fileName string,
	parserOptions ...parser.ParserOption,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		glazedOverrides := map[string]interface{}{}
		needsRealFileOutput := false

		// create a temporary file for glazed output
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
			needsRealFileOutput = true
		} else if strings.HasSuffix(fileName, ".txt") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "ascii"
		} else {
			c.JSON(500, gin.H{"error": "could not determine output format"})
			return
		}

		var tmpFile *os.File
		var err error

		// excel output needs a real output file, otherwise we can go stream to the HTTP response
		if needsRealFileOutput {
			tmpFile, err = os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", fileName))
			if err != nil {
				c.JSON(500, gin.H{"error": "could not create temporary file"})
				return
			}
			defer func(name string) {
				_ = os.Remove(name)
			}(tmpFile.Name())

			// now check file suffix for content-type
			glazedOverrides["output-file"] = tmpFile.Name()
		}

		parserOptions = append(parserOptions, parser.WithAppendOverrides("glazed", glazedOverrides))
		handler := glazed.NewQueryHandler(cmd, glazed.WithQueryHandlerParserOptions(parserOptions...))

		baseName := filepath.Base(fileName)
		c.Writer.Header().Set("Content-Disposition", "attachment; filename="+baseName)

		if needsRealFileOutput {
			outputFileHandler := NewOutputFileHandler(handler, tmpFile.Name())

			err = outputFileHandler.Handle(c, c.Writer)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		} else {
			err = handler.Handle(c, c.Writer)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		// override parameter layers at the end

	}
}
