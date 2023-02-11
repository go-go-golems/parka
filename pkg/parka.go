package pkg

import (
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

type StaticPath struct {
	LocalPath string
	UrlPath   string
}

type Server struct {
	Router   *gin.Engine
	Commands []ParkaCommand

	TemplateDir string
	StaticPath  []StaticPath
}

type ServerOption = func(*Server)

func WithCommands(commands ...ParkaCommand) ServerOption {
	return func(s *Server) {
		s.Commands = append(s.Commands, commands...)
	}
}

func WithStaticPath(path StaticPath) ServerOption {
	return func(s *Server) {
		s.StaticPath = append(s.StaticPath, path)
	}
}

func WithTemplateDir(dir string) ServerOption {
	return func(s *Server) {
		s.TemplateDir = dir
	}
}

func NewServer(options ...ServerOption) *Server {
	s := &Server{
		Router: gin.Default(),
	}

	for _, option := range options {
		option(s)
	}

	return s
}

func (s *Server) Run() error {
	s.Router.GET("/", func(c *gin.Context) {
		s.renderTemplate(c, "index", nil)
	})
	s.Router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		s.renderTemplate(c, page, nil)
	})
	s.Router.Use(static.Serve("/dist/", static.LocalFile("./web/dist", true)))

	s.serveCommands()

	return s.Router.Run()
}
