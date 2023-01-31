package cmds

import (
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"html/template"
	"net/http"
)

type Server struct {
	TemplateDir string
	port        uint16
}

func (s *Server) renderTemplate(c *gin.Context, tmpl string, data interface{}) {
	t, err := template.ParseFiles(s.TemplateDir + "/" + tmpl)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}

	err = t.Execute(c.Writer, data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}
}

func (s *Server) Run() error {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		s.renderTemplate(c, "index.tmpl.html", nil)
	})
	r.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		s.renderTemplate(c, page+".tmpl.html", nil)
	})
	r.Use(static.Serve("/dist/", static.LocalFile("./web/dist", true)))

	return r.Run()
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetUint16("port")
		cobra.CheckErr(err)
		templateDir, err := cmd.Flags().GetString("template-dir")
		cobra.CheckErr(err)

		s := &Server{
			port: port,

			TemplateDir: templateDir,
		}
		err = s.Run()
		cobra.CheckErr(err)
	},
}

func init() {
	ServeCmd.Flags().Uint16("port", 8080, "Port to listen on")
	ServeCmd.Flags().String("template-dir", "web/src/templates", "Directory containing templates")
}
