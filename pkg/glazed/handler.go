package glazed

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/csv"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/formatters/template"
	"github.com/go-go-golems/glazed/pkg/formatters/yaml"
	"net/http"
)

func NewGinHandlerFromCommandHandlers(
	cmd cmds.GlazeCommand,
	// NOTE(manuel, 2023-04-16) Weird to use the ... here
	handlers ...CommandHandlerFunc,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// NOTE(manuel, 2023-02-28) Add initial middleware handlers here
		//
		// It probably makes sense to give the user control over the initial parka
		// context before passing it downstream
		pc := NewCommandContext(cmd)

		for _, h := range handlers {
			err := h(c, pc)
			if err != nil {
				_ = c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		gp, of, err := SetupProcessor(pc)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// TODO(manuel, 2023-03-02) We might want to switch on the requested content type here too

		var contentType string

		switch of_ := of.(type) {
		case *json.OutputFormatter:
			contentType = "application/json"
		case *csv.OutputFormatter:
			contentType = "text/csv"
		case *table.OutputFormatter:
			//exhaustive:ignore
			switch of_.TableFormat {
			case "html":
				contentType = "text/html"
			case "markdown":
				contentType = "text/markdown"
			default:
			}
		case *yaml.OutputFormatter:
			contentType = "application/x-yaml"
		case *template.OutputFormatter:
			// TODO(manuel, 2023-03-02) Unclear how to render HTML templates or text templates here
			// probably the best idea is to have the formatter return a content type anyway
			contentType = "text/html"
		}

		// get gp output
		s, err := of.Output()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Status(200)
		c.Writer.Header().Set("Content-Type", contentType)
		_, err = c.Writer.Write([]byte(s))
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

func SetupProcessor(pc *CommandContext) (*cmds.GlazeProcessor, formatters.OutputFormatter, error) {
	// TODO(manuel, 2023-02-11) For now, create a raw JSON output formatter. We will want more nuance here
	// See https://github.com/go-go-golems/parka/issues/8

	l, ok := pc.ParsedLayers["glazed"]
	if ok {
		return cli.SetupProcessor(l.Parameters)
	}

	of := json.NewOutputFormatter(
		json.WithOutputIndividualRows(true),
	)
	gp := cmds.NewGlazeProcessor(of)

	return gp, of, nil
}
