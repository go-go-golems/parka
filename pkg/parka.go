package pkg

import (
	"context"
	"embed"
	"fmt"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/parka/pkg/render"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed "web/src/templates/*"
var templateFS embed.FS

//go:embed "web/dist/*"
var distFS embed.FS

// StaticPath allows you to serve static files from a http.FileSystem under a given URL path urlPath.
type StaticPath struct {
	fs      http.FileSystem
	urlPath string
}

// NewStaticPath creates a new StaticPath that will serve files from the given http.FileSystem.
func NewStaticPath(fs http.FileSystem, urlPath string) StaticPath {
	return StaticPath{
		fs:      fs,
		urlPath: urlPath,
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
	return s.fs.Open(s.prefix + name)
}

// Server is the main class that parka uses to serve static and templated content.
// It is a wrapper around gin.Engine.
//
// It is meant to be quite flexible, allowing you to add static paths and template lookups
// that can provide different fs and template backends.
//
// Router is the gin.Engine that is used to serve the content, and it is exposed so that you
// can use it as just a gin.Engine if you want to.
type Server struct {
	Router *gin.Engine

	StaticPaths     []StaticPath
	DefaultRenderer *render.Renderer

	Port    uint16
	Address string
}

type ServerOption = func(*Server) error

// WithStaticPaths will add the given static paths to the list of static paths.
// If a path with the same URL path already exists, it will be replaced.
func WithStaticPaths(paths ...StaticPath) ServerOption {
	return func(s *Server) error {
		// prepend paths to the list
	pathLoop:
		for _, path := range paths {
			for i, existingPath := range s.StaticPaths {
				if existingPath.urlPath == path.urlPath {
					s.StaticPaths[i] = path
					continue pathLoop
				}
			}
			s.StaticPaths = append(s.StaticPaths, path)
		}

		return nil
	}
}

// WithPort will set the port that the server will listen on.
func WithPort(port uint16) ServerOption {
	return func(s *Server) error {
		s.Port = port
		return nil
	}
}

// WithAddress will set the address that the server will listen on.
func WithAddress(address string) ServerOption {
	return func(s *Server) error {
		s.Address = address
		return nil
	}
}

func WithFailOption(err error) ServerOption {
	return func(_ *Server) error {
		return err
	}
}

func WithDefaultRenderer(r *render.Renderer) ServerOption {
	return func(s *Server) error {
		s.DefaultRenderer = r
		return nil
	}
}

func GetDefaultParkaRendererOptions() ([]render.RendererOption, error) {
	// this should be overloaded too
	parkaLookup, err := render.LookupTemplateFromFS(templateFS, "web/src/templates", "**/*.tmpl.*")
	if err != nil {
		return nil, err
	}

	return []render.RendererOption{
		render.WithAppendTemplateLookups(parkaLookup),
		render.WithMarkdownBaseTemplateName("base.tmpl.html"),
	}, nil
}

func WithDefaultParkaLookup(options ...render.RendererOption) ServerOption {
	options_, err := GetDefaultParkaRendererOptions()
	if err != nil {
		return WithFailOption(err)
	}
	options_ = append(options_, options...)

	renderer, err := render.NewRenderer(options_...)
	if err != nil {
		return WithFailOption(err)
	}

	return WithDefaultRenderer(renderer)
}

func GetParkaStaticFS() http.FileSystem {
	return NewEmbedFileSystem(distFS, "web/dist")
}

func WithDefaultParkaStaticPaths() ServerOption {
	return WithStaticPaths(
		NewStaticPath(GetParkaStaticFS(), "/dist"),
	)
}

func WithGzip() ServerOption {
	return func(s *Server) error {
		s.Router.Use(gzip.Gzip(gzip.DefaultCompression))
		return nil
	}
}

// NewServer will create a new Server with the given options.
// This loads a fixed set of files and templates from the embed.FS.
// These files provide tailwind support for Markdown rendering and a standard index and base page template.
// NOTE(manuel, 2023-04-16) This is definitely ripe to be removed.
func NewServer(options ...ServerOption) (*Server, error) {
	router := gin.Default()

	s := &Server{
		Router:      router,
		StaticPaths: []StaticPath{},
	}

	for _, option := range options {
		err := option(s)
		if err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return s, nil
}

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
	defer f.Close()
	return true
}

// Run will start the server and listen on the given address and port.
func (s *Server) Run(ctx context.Context) error {
	for _, path := range s.StaticPaths {
		s.Router.StaticFS(path.urlPath, path.fs)
	}

	// match all remaining paths to the templates
	if s.DefaultRenderer != nil {
		s.Router.GET("/", s.DefaultRenderer.HandleWithTemplate("index", nil))
		s.Router.Use(s.DefaultRenderer.Handle(nil))
	}

	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	eg := errgroup.Group{}
	eg.Go(func() error {
		<-ctx.Done()
		return srv.Shutdown(ctx)
	})
	eg.Go(func() error {
		return srv.ListenAndServe()
	})

	return eg.Wait()
}
