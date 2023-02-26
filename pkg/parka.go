package pkg

import (
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed "web/src/templates/*"
var templateFS embed.FS

//go:embed "web/dist/*"
var distFS embed.FS

type StaticPath struct {
	LocalPath string
	UrlPath   string
}

type Server struct {
	Router   *gin.Engine
	Commands []ParkaCommand

	// used by HTML() calls to render a template
	Template *template.Template

	// parka bundled templates
	ParkaTemplate *template.Template
}

type ServerOption = func(*Server)

func WithCommands(commands ...ParkaCommand) ServerOption {
	return func(s *Server) {
		s.Commands = append(s.Commands, commands...)
	}
}

func NewServer(options ...ServerOption) (*Server, error) {
	router := gin.Default()

	s := &Server{
		Router: router,
	}

	for _, option := range options {
		option(s)
	}

	return s, nil
}

func (s *Server) SetTemplate(t *template.Template) {
	s.Template = t
}

func (s *Server) SetParkaTemplate(t *template.Template) {
	s.ParkaTemplate = t
}

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
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
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

func (s *Server) ServeEmbeddedAssets() error {
	s.Router.StaticFS("/dist", NewEmbedFileSystem(distFS, "web/dist/"))

	t := helpers.CreateHTMLTemplate("templates")
	err := helpers.ParseHTMLFS(t, templateFS, "**/*.tmpl.*", "web/src/templates/")
	if err != nil {
		return err
	}
	s.SetParkaTemplate(t)
	return nil
}

func (s *Server) ServeAssets(distPath string, templatePath string) error {
	s.Router.Static("/dist", distPath)

	t := helpers.CreateHTMLTemplate("templates")
	if !strings.HasSuffix(templatePath, "/") {
		templatePath += "/"
	}
	err := helpers.ParseHTMLFS(t, os.DirFS(templatePath), "**/*.tmpl.*", templatePath)
	if err != nil {
		return err
	}
	s.SetParkaTemplate(t)
	return nil
}

func (s *Server) Run() error {

	s.Router.GET("/", func(c *gin.Context) {
		s.renderMarkdownTemplate(c, "index", nil)
	})
	s.Router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		s.renderMarkdownTemplate(c, page, nil)
	})

	s.serveCommands()

	return s.Router.Run()
}
