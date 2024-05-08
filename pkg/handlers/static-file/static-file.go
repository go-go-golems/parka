package static_file

import (
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/labstack/echo/v4"
	"io/fs"
	"os"
	"strings"
)

type StaticFileHandler struct {
	fs        fs.FS
	localPath string
}

type StaticFileHandlerOption func(handler *StaticFileHandler)

func WithDefaultFS(fs fs.FS, localPath string) StaticFileHandlerOption {
	return func(handler *StaticFileHandler) {
		if handler.fs == nil {
			handler.fs = fs
			handler.localPath = localPath
		}
	}
}

func WithLocalPath(localPath string) StaticFileHandlerOption {
	return func(handler *StaticFileHandler) {
		if localPath != "" {
			if localPath[0] == '/' {
				handler.fs = os.DirFS(localPath)
			} else {
				handler.fs = os.DirFS(localPath)
			}
			handler.localPath = strings.TrimPrefix(localPath, "/")
		}
	}
}

func NewStaticFileHandler(options ...StaticFileHandlerOption) *StaticFileHandler {
	handler := &StaticFileHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewStaticFileHandlerFromConfig(shf *config.StaticFile, options ...StaticFileHandlerOption) *StaticFileHandler {
	handler := &StaticFileHandler{}
	for _, option := range options {
		option(handler)
	}
	WithLocalPath(shf.LocalPath)(handler)
	return handler
}

func (s *StaticFileHandler) Serve(server *server.Server, path string) error {
	server.Router.StaticFS(path, echo.MustSubFS(s.fs, s.localPath))
	return nil
}
