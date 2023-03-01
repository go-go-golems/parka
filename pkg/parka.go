package pkg

import (
	"embed"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed "web/src/templates/*"
var templateFS embed.FS

//go:embed "web/dist/*"
var distFS embed.FS

type StaticPath struct {
	fs      http.FileSystem
	urlPath string
}

func NewStaticPath(fs http.FileSystem, urlPath string) StaticPath {
	return StaticPath{
		fs:      fs,
		urlPath: urlPath,
	}
}

type Server struct {
	Router *gin.Engine

	StaticPaths     []StaticPath
	TemplateLookups []TemplateLookup
}

type ServerOption = func(*Server)

func WithTemplateLookups(lookups ...TemplateLookup) ServerOption {
	return func(s *Server) {
		// prepend lookups to the list
		s.TemplateLookups = append(lookups, s.TemplateLookups...)
	}
}

func WithStaticPaths(paths ...StaticPath) ServerOption {
	return func(s *Server) {
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
	}
}

func NewServer(options ...ServerOption) (*Server, error) {
	router := gin.Default()

	parkaLookup, err := LookupTemplateFromFS(templateFS, "web/src/templates", "**/*.tmpl.*")
	if err != nil {
		return nil, err
	}

	s := &Server{
		Router: router,
		StaticPaths: []StaticPath{
			NewStaticPath(NewEmbedFileSystem(distFS, "web/dist"), "/dist"),
		},
		TemplateLookups: []TemplateLookup{
			parkaLookup,
		},
	}

	for _, option := range options {
		option(s)
	}

	return s, nil
}

// EmbedFileSystem is a helper to make an embed FS work as a http.FS,
// which allows us to serve embed.FS using gin's `Static` middleware.
type EmbedFileSystem struct {
	f           http.FileSystem
	stripPrefix string
}

func NewEmbedFileSystem(f fs.FS, stripPrefix string) *EmbedFileSystem {
	if !strings.HasSuffix(stripPrefix, "/") {
		stripPrefix += "/"
	}
	return &EmbedFileSystem{
		f:           http.FS(f),
		stripPrefix: stripPrefix,
	}
}

func (e *EmbedFileSystem) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")
	return e.f.Open(e.stripPrefix + name)
}

func (e *EmbedFileSystem) Exists(prefix string, path string) bool {
	// remove prefix from path
	path = path[len(prefix):]

	f, err := e.f.Open(e.stripPrefix + path)
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

// LookupTemplate will iterate through the template lookups until it finds one of the
// templates given in name.
func (s *Server) LookupTemplate(name ...string) (*template.Template, error) {
	var t *template.Template

	for _, lookup := range s.TemplateLookups {
		t, err := lookup(name...)
		if err == nil {
			return t, nil
		}
	}

	return t, nil
}

func (s *Server) serveMarkdownTemplatePage(c *gin.Context, page string, data interface{}) {
	t, err := s.LookupTemplate(page+".tmpl.md", page+".md")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}

	if t != nil {
		markdown, err := RenderMarkdownTemplateToHTML(t, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		baseTemplate, err := s.LookupTemplate("base.tmpl.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err = baseTemplate.Execute(
			c.Writer,
			map[string]interface{}{
				"markdown": template.HTML(markdown),
			})
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	} else {
		t, err = s.LookupTemplate(page+".tmpl.html", page+".html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
		if t == nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err := t.Execute(c.Writer, data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}
	}
}

func (s *Server) Run() error {
	for _, path := range s.StaticPaths {
		s.Router.StaticFS(path.urlPath, path.fs)
	}

	s.Router.GET("/", func(c *gin.Context) {
		s.serveMarkdownTemplatePage(c, "index", nil)
	})
	s.Router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		s.serveMarkdownTemplatePage(c, page, nil)
	})

	return s.Router.Run()
}
