package output_file

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/list"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/glazed"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

// CreateGlazedFileHandler creates a handler that will run a glazed command and write the output
// with a Content-Disposition header to the response writer.
//
// If an output format requires writing to a temporary file locally, such as excel,
// the handler is wrapped in a temporary file handler.
func CreateGlazedFileHandler(
	cmd cmds.GlazeCommand,
	fileName string,
	middlewares_ ...middlewares.Middleware,
) echo.HandlerFunc {
	return func(c echo.Context) error {
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
			return errors.New("unsupported file format")
		}

		var tmpFile *os.File
		var err error

		glazedOverride := middlewares.UpdateFromMap(
			map[string]map[string]interface{}{
				settings.GlazedSlug: glazedOverrides,
			},
			parameters.WithParseStepSource("output-file-glazed-override"),
		)

		handler := glazed.NewQueryHandler(cmd,
			glazed.WithMiddlewares(
				list.Prepend(middlewares_,
					parka_middlewares.UpdateFromQueryParameters(c, parameters.WithParseStepSource("query")),
					glazedOverride)...,
			))

		baseName := filepath.Base(fileName)
		c.Response().Header().Set("Content-Disposition", "attachment; filename="+baseName)

		// excel output needs a real output file, otherwise we can go stream to the HTTP response
		if needsRealFileOutput {
			tmpFile, err = os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", fileName))
			if err != nil {
				return errors.Wrap(err, "could not create temporary file")
			}
			defer func(name string) {
				_ = os.Remove(name)
			}(tmpFile.Name())

			// now check file suffix for content-type
			glazedOverrides["output-file"] = tmpFile.Name()

			// here we have the output of the handler go to a request that we discard, and
			// we instead copy the temporary file to the response writer
			res := httptest.NewRecorder()
			req := c.Request()
			newCtx := c.Echo().NewContext(req, res)

			err = handler.Handle(newCtx)
			if err != nil {
				return err
			}

			// copy tmpFile to output
			f, err := os.Open(tmpFile.Name())
			if err != nil {
				return errors.Wrap(err, "could not open temporary file")
			}
			defer func(f *os.File) {
				_ = f.Close()
			}(f)

			c.Response().Header().Set("Content-Type", "application/octet-stream")
			c.Response().WriteHeader(http.StatusOK)

			_, err = io.Copy(c.Response().Writer, f)
			if err != nil {
				return err
			}
		} else {
			err = handler.Handle(c)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
