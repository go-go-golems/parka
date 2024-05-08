package static_dir

import (
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/server"
	fs2 "github.com/go-go-golems/parka/pkg/utils/fs"
	"io/fs"
	"os"
	"strings"
)

type StaticDirHandler struct {
	fs        fs.FS
	localPath string
}

type StaticDirHandlerOption func(handler *StaticDirHandler)

func WithDefaultFS(fs fs.FS, localPath string) StaticDirHandlerOption {
	return func(handler *StaticDirHandler) {
		if handler.fs == nil {
			handler.fs = fs
			handler.localPath = localPath
		}
	}
}

func WithLocalPath(localPath string) StaticDirHandlerOption {
	return func(handler *StaticDirHandler) {
		if localPath != "" {
			if !strings.HasSuffix(localPath, "/") {
				localPath = localPath + "/"
			}
			if localPath[0] == '/' {
				handler.fs = os.DirFS(localPath)
			} else {
				handler.fs = os.DirFS(localPath)
			}
		}
	}
}

func NewStaticDirHandler(options ...StaticDirHandlerOption) *StaticDirHandler {
	handler := &StaticDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewStaticDirHandlerFromConfig(sh *config.Static, options ...StaticDirHandlerOption) *StaticDirHandler {
	handler := &StaticDirHandler{}
	WithLocalPath(sh.LocalPath)(handler)
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (s *StaticDirHandler) Serve(server *server.Server, path string) error {
	fs_ := s.fs
	if s.localPath != "" {
		fs_ = fs2.NewAddPrefixPathFS(s.fs, s.localPath)
	}
	server.Router.StaticFS(path, fs_)
	return nil
}
