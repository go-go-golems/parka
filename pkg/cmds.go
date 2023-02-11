package pkg

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"strings"
)

type JSONMarshaler interface {
	MarshalJSON() ([]byte, error)
}

// parseQueryParameters extracts the query parameters out of a request according to the description in parameters
func parseQueryParameters(c *gin.Context, parameters []*cmds.Parameter) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	for _, p := range parameters {
		value := c.Query(p.Name)
		if value == "" {
			if p.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
			}
			params[p.Name] = p.Default
		} else if p.Type != cmds.ParameterTypeStringFromFile && p.Type != cmds.ParameterTypeObjectFromFile {
			pValue, err := p.ParseParameter([]string{value})
			if err != nil {
				return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
			}
			params[p.Name] = pValue
		} else {
			// TODO(manuel, 2023-02-11) Implement file upload
			// See https://github.com/go-go-golems/parka/issues/10
			_ = 123
		}

	}
	return params, nil
}

type ParkaCommand interface {
	cmds.Command
	RunFromParka(c *gin.Context, parameters map[string]interface{}, gp *cli.GlazeProcessor) error
}

func (s *Server) serveCommands() {
	apiCmds := []interface{}{}

	for _, cmd := range s.Commands {
		description := cmd.Description()

		if jm, ok := cmd.(JSONMarshaler); ok {
			apiCmds = append(apiCmds, jm)
		} else {
			apiCmds = append(apiCmds, description)
		}

		path := "/api/command/" + strings.Join(description.Parents, "/") + "/" + description.Name

		// GET and POST (?)
		s.Router.GET(path, func(c *gin.Context) {
			flags, err := parseQueryParameters(c, description.Flags)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			of, gp, _ := SetupProcessor(c, flags)

			err = cmd.RunFromParka(c, flags, gp)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			// get gp output
			_, err = of.Output()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			rows := []map[string]interface{}{}
			for _, row := range of.Table.Rows {
				rows = append(rows, row.GetValues())
			}

			c.JSON(200, rows)
		})
	}

	s.Router.GET("/api/commands", func(c *gin.Context) {
		c.JSON(200, apiCmds)
	})

}

func SetupProcessor(c *gin.Context, flags map[string]interface{}) (
	*formatters.JSONOutputFormatter,
	*cli.GlazeProcessor,
	error,
) {
	// TODO(manuel, 2023-02-11) For now, create a raw JSON output formatter. We will want more nuance here
	// See https://github.com/go-go-golems/parka/issues/8

	of := formatters.NewJSONOutputFormatter(true)
	gp := cli.NewGlazeProcessor(of, []middlewares.ObjectMiddleware{})

	return of, gp, nil
}
