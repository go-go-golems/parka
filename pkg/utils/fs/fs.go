package fs

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// EmbedFileSystem is a helper to make an embed FS work as a http.FS,
// which allows us to serve embed.FS using gin's `Static` middleware.
type EmbedFileSystem struct {
	f           http.FileSystem
	stripPrefix string
}

// NewEmbedFileSystem will create a new EmbedFileSystem that will serve the given embed.FS
// under the given URL path. stripPrefix will be added to the beginning of all paths when
// looking up files in the embed.FS.
func NewEmbedFileSystem(f fs.FS, stripPrefix string) *EmbedFileSystem {
	if !strings.HasSuffix(stripPrefix, "/") {
		stripPrefix += "/"
	}
	return &EmbedFileSystem{
		f:           http.FS(f),
		stripPrefix: stripPrefix,
	}
}

// Open will open the file with the given name from the embed.FS. The name will be prefixed
// with the stripPrefix that was given when creating the EmbedFileSystem.
func (e *EmbedFileSystem) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")
	return e.f.Open(e.stripPrefix + name)
}

// Exists will check if the given path exists in the embed.FS. The path will be prefixed
// with the stripPrefix that was given when creating the EmbedFileSystem, while prefix will
// be removed from the path.
func (e *EmbedFileSystem) Exists(prefix string, path string) bool {
	if len(path) < len(prefix) {
		return false
	}

	// remove prefix from path
	path = path[len(prefix):]

	f, err := e.f.Open(e.stripPrefix + path)
	if err != nil {
		return false
	}
	defer func(f http.File) {
		_ = f.Close()
	}(f)
	return true
}

// StaticPath allows you to serve static files from a http.FileSystem under a given URL path UrlPath.
type StaticPath struct {
	FS      http.FileSystem
	UrlPath string
}

// NewStaticPath creates a new StaticPath that will serve files from the given http.FileSystem.
func NewStaticPath(fs http.FileSystem, urlPath string) StaticPath {
	return StaticPath{
		FS:      fs,
		UrlPath: urlPath,
	}
}

// AddPrefixPathFS is a helper wrapper that will a prefix to each incoming filename that is to be opened.
// This is useful for embedFS which will keep their prefix. For example, mounting the embed fs go:embed static
// will retain the static/* prefix, while the static gin handler will strip it.
type AddPrefixPathFS struct {
	fs     fs.FS
	prefix string
}

// NewAddPrefixPathFS creates a new AddPrefixPathFS that will add the given prefix to each file that is opened..
func NewAddPrefixPathFS(fs fs.FS, prefix string) AddPrefixPathFS {
	return AddPrefixPathFS{
		fs:     fs,
		prefix: prefix,
	}
}

func (s AddPrefixPathFS) Open(name string) (fs.File, error) {
	return s.fs.Open(filepath.Join(s.prefix, name))
}
