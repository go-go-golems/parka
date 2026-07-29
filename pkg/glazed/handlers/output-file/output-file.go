package output_file

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/glazed"
	parka_middlewares "github.com/go-go-golems/parka/pkg/glazed/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type QueryHandler struct {
	cmd         cmds.GlazeCommand
	fileName    string
	middlewares []sources.Middleware
	// whitelistedLayers contains the list of layers that are allowed to be modified through query parameters
	whitelistedLayers []string
}

type QueryHandlerOption func(*QueryHandler)

func NewQueryHandler(cmd cmds.GlazeCommand, fileName string, options ...QueryHandlerOption) *QueryHandler {
	h := &QueryHandler{
		cmd:      cmd,
		fileName: fileName,
	}

	for _, option := range options {
		option(h)
	}

	return h
}

func WithMiddlewares(middlewares ...sources.Middleware) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.middlewares = middlewares
	}
}

func WithWhitelistedLayers(layers ...string) QueryHandlerOption {
	return func(handler *QueryHandler) {
		handler.whitelistedLayers = layers
	}
}

func (h *QueryHandler) Handle(c echo.Context) error {
	queryMiddleware := parka_middlewares.UpdateFromQueryParameters(c, fields.WithSource("query"))
	if len(h.whitelistedLayers) > 0 {
		queryMiddleware = sources.WrapWithWhitelistedSections(h.whitelistedLayers, queryMiddleware)
	}

	glazedOverrides := map[string]interface{}{}
	needsRealFileOutput := false

	// create a temporary file for glazed output
	// The structured-output cleanup removed the table-format flag and the
	// markdown/html/excel table formatters. CSV/TSV are now first-class
	// format values; markdown/html/xlsx output is no longer supported by
	// the structured-output section and must be handled separately.
	if strings.HasSuffix(h.fileName, ".csv") {
		glazedOverrides["format"] = "csv"
	} else if strings.HasSuffix(h.fileName, ".tsv") {
		glazedOverrides["format"] = "tsv"
	} else if strings.HasSuffix(h.fileName, ".md") {
		glazedOverrides["format"] = "table"
	} else if strings.HasSuffix(h.fileName, ".html") {
		glazedOverrides["format"] = "table"
	} else if strings.HasSuffix(h.fileName, ".json") {
		glazedOverrides["format"] = "json"
	} else if strings.HasSuffix(h.fileName, ".yaml") {
		glazedOverrides["format"] = "yaml"
	} else if strings.HasSuffix(h.fileName, ".xlsx") {
		glazedOverrides["format"] = "table"
		needsRealFileOutput = true
	} else if strings.HasSuffix(h.fileName, ".txt") {
		glazedOverrides["format"] = "table"
	} else {
		return errors.New("unsupported file format")
	}

	var tmpFile *os.File
	var err error

	glazedOverride := sources.FromMap(
		map[string]map[string]interface{}{
			settings.StructuredOutputSlug: glazedOverrides,
		},
		fields.WithSource("output-file-glazed-override"),
	)

	middlewares_ := append(
		[]sources.Middleware{
			queryMiddleware,
			glazedOverride,
		},
		h.middlewares...,
	)
	middlewares_ = append(middlewares_, sources.FromDefaults())

	handler := glazed.NewQueryHandler(h.cmd,
		glazed.WithMiddlewares(middlewares_...),
	)

	baseName := filepath.Base(h.fileName)
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+baseName)

	// excel output needs a real output file, otherwise we can go stream to the HTTP response
	if needsRealFileOutput {
		tmpFile, err = os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", h.fileName))
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

func CreateGlazedFileHandler(
	cmd cmds.GlazeCommand,
	fileName string,
	options ...QueryHandlerOption,
) echo.HandlerFunc {
	handler := NewQueryHandler(cmd, fileName, options...)
	return handler.Handle
}
