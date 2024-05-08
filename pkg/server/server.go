package server

import (
	"context"
	"embed"
	"fmt"
	"github.com/go-go-golems/parka/pkg/render"
	utils_fs "github.com/go-go-golems/parka/pkg/utils/fs"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"net/http"
	"net/http/pprof"
)

//go:embed "web/src/templates/*"
var templateFS embed.FS

//go:embed "web/dist/*"
var distFS embed.FS

// Server is the main class that parka uses to serve static and templated content.
// It is a wrapper around gin.Engine.
//
// It is meant to be quite flexible, allowing you to add static paths and template lookups
// that can provide different fs and template backends.
//
// Router is the gin.Engine that is used to serve the content, and it is exposed so that you
// can use it as just a gin.Engine if you want to.
type Server struct {
	Router *echo.Echo

	// TODO(manuel, 2023-06-05) This should become a standard Static handler to be added to the Routes
	StaticPaths []utils_fs.StaticPath
	// TODO(manuel, 2023-06-05) This could potentially be replaced with a fallback Handler
	DefaultRenderer *render.Renderer

	Port    uint16
	Address string
}

type ServerOption = func(*Server) error

// WithStaticPaths will add the given static paths to the list of static paths.
// If a path with the same URL path already exists, it will be replaced.
func WithStaticPaths(paths ...utils_fs.StaticPath) ServerOption {
	return func(s *Server) error {
		// prepend paths to the list
	pathLoop:
		for _, path := range paths {
			for i, existingPath := range s.StaticPaths {
				if existingPath.UrlPath == path.UrlPath {
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

func GetDefaultParkaTemplateLookup() (render.TemplateLookup, error) {
	// this should be overloaded too
	parkaLookup := render.NewLookupTemplateFromFS(
		render.WithFS(templateFS),
		render.WithBaseDir("web/src/templates"),
		render.WithPatterns("**/*.tmpl.*"),
	)
	err := parkaLookup.Reload()
	if err != nil {
		return nil, err
	}

	return parkaLookup, nil
}

// GetDefaultParkaRendererOptions will return the default options for the parka renderer.
// This includes looking up templates from the embedded templateFS to provide support for
// markdown rendering with tailwind. This includes css files.
// It also sets base.tmpl.html as the base template for wrapping rendered markdown.
func GetDefaultParkaRendererOptions() ([]render.RendererOption, error) {
	// this should be overloaded too
	parkaLookup := render.NewLookupTemplateFromFS(
		render.WithFS(templateFS),
		render.WithBaseDir("web/src/templates"),
		render.WithPatterns("**/*.tmpl.*"),
	)
	err := parkaLookup.Reload()
	if err != nil {
		return nil, err
	}

	return []render.RendererOption{
		render.WithAppendTemplateLookups(parkaLookup),
		render.WithMarkdownBaseTemplateName("base.tmpl.html"),
	}, nil
}

func WithDefaultParkaRenderer(options ...render.RendererOption) ServerOption {
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

func GetParkaStaticHttpFS() fs.FS {
	return utils_fs.NewEmbedFileSystem(distFS, "web/dist")
}

func GetParkaStaticFS() fs.FS {
	return distFS
}

func WithDefaultParkaStaticPaths() ServerOption {
	return WithStaticPaths(
		utils_fs.NewStaticPath(GetParkaStaticHttpFS(), "/dist"),
	)
}

func WithGzip() ServerOption {
	return func(s *Server) error {
		s.Router.Use(middleware.Gzip())
		return nil
	}
}

// NewServer will create a new Server with the given options.
// This loads a fixed set of files and templates from the embed.FS.
// These files provide tailwind support for Markdown rendering and a standard index and base page template.
// NOTE(manuel, 2023-04-16) This is definitely ripe to be removed.
func NewServer(options ...ServerOption) (*Server, error) {
	router := echo.New()

	s := &Server{
		Router:      router,
		StaticPaths: []utils_fs.StaticPath{},
	}

	for _, option := range options {
		err := option(s)
		if err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return s, nil
}

func (s *Server) RegisterDebugRoutes() {
	handlers_ := map[string]http.HandlerFunc{
		"/debug/pprof/cmdline":   pprof.Cmdline,
		"/debug/pprof/profile":   pprof.Profile,
		"/debug/pprof/symbol":    pprof.Symbol,
		"/debug/pprof/trace":     pprof.Trace,
		"/debug/pprof/mutex":     pprof.Index,
		"/debug/pprof/allocs":    pprof.Index,
		"/debug/pprof/block":     pprof.Index,
		"/debug/pprof/goroutine": pprof.Index,
	}

	for route, handler := range handlers_ {
		route_ := route
		handler_ := handler
		s.Router.GET(route_, echo.WrapHandler(handler_))
	}
}

// Run will start the server and listen on the given address and port.
func (s *Server) Run(ctx context.Context) error {
	for _, path := range s.StaticPaths {
		s.Router.StaticFS(path.UrlPath, path.FS)
	}

	// match all remaining paths to the templates
	if s.DefaultRenderer != nil {
		s.Router.Pre(s.DefaultRenderer.HandleWithTemplateMiddleware("/", "index", nil))
		s.Router.Pre(s.DefaultRenderer.HandleWithTrimPrefixMiddleware("", nil))
	}

	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	eg := errgroup.Group{}
	eg.Go(func() error {
		<-ctx.Done()
		fmt.Println("Shutting down server")
		return srv.Shutdown(ctx)
	})
	eg.Go(func() error {
		fmt.Printf("Starting server on %s\n", addr)
		return srv.ListenAndServe()
	})

	return eg.Wait()
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	// Create a custom error response
	errorResponse := map[string]interface{}{
		"error": err.Error(),
	}

	// Send the custom error response
	if !c.Response().Committed {
		_ = c.JSON(code, errorResponse)
	}

	c.Logger().Error(err)
}
