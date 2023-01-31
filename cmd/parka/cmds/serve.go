package cmds

import (
	"bytes"
	html2 "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"net/http"
	"os"
)

type Server struct {
	TemplateDir string
	port        uint16
}

func (s *Server) renderTemplate(c *gin.Context, page string, data interface{}) {
	// check if markdown file exists
	markdownFile := s.TemplateDir + "/" + page + ".md"
	_, err := os.Stat(markdownFile)
	var t *template.Template

	if err == nil {
		markdown, err := s.renderMarkdownToHTML(markdownFile, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		t, err = template.ParseFiles(s.TemplateDir + "/base.tmpl.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

		err = t.Execute(c.Writer, map[string]interface{}{
			"markdown": template.HTML(markdown),
		})
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering template")
			return
		}

	} else {
		t, err = template.ParseFiles(s.TemplateDir + "/" + page + ".tmpl.html")
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

}

func (s *Server) Run() error {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		s.renderTemplate(c, "index", nil)
	})
	r.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		s.renderTemplate(c, page, nil)
	})
	r.Use(static.Serve("/dist/", static.LocalFile("./web/dist", true)))

	return r.Run()
}

func (s *Server) renderMarkdownToHTML(file string, data interface{}) (string, error) {
	f, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	// parse as template
	t, err := template.New("markdown").Parse(string(f))
	if err != nil {
		return "", err
	}

	// execute template
	buf := new(bytes.Buffer)
	err = t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	rendered := buf.String()

	engine := goldmark.New(
		goldmark.WithExtensions(
			// add tables
			extension.NewTable(),
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					html2.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe()))

	buf = new(bytes.Buffer)
	err = engine.Convert([]byte(rendered), buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
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
