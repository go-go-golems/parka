package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io"
	"mime/multipart"
	"strings"
)

type JSONMarshaler interface {
	MarshalJSON() ([]byte, error)
}

// parseQueryParameters extracts the query parameters out of a request according to the description in parameters
func parseQueryParameters(c *gin.Context, ps []*parameters.ParameterDefinition) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	for _, p := range ps {
		// TOOD(manuel, 2023-02-26) this is where we catch the fromfile parameters
		if parameters.IsFileLoadingParameter(p.Type, c.Query(p.Name)) {
			// TODO(manuel, 2023-02-11) Implement file upload
			// See https://github.com/go-go-golems/parka/issues/10

			value := c.Query(p.Name)
			if value == "" {
				if p.Required {
					return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				params[p.Name] = p.Default
			} else {
				f := strings.NewReader(value)
				pValue, err := p.ParseFromReader(f, "")
				if err != nil {
					return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				params[p.Name] = pValue
			}
		} else {
			value := c.Query(p.Name)
			if value == "" {
				if p.Required {
					return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				params[p.Name] = p.Default
			} else {
				pValue, err := p.ParseParameter([]string{value})
				if err != nil {
					return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				params[p.Name] = pValue
			}
		}

	}
	return params, nil
}

func parseStringFromFile(c *gin.Context, name string) (string, error) {
	file, _, err := c.Request.FormFile(name)
	if err != nil {
		return "", fmt.Errorf("error retrieving file '%s': %v", name, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msgf("error closing file '%s'", name)
		}
	}(file)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading contents of file '%s': %v", name, err)
	}

	return string(fileBytes), nil
}

func parseObjectFromFile(c *gin.Context, name string) (map[string]interface{}, error) {
	file, fileHeader, err := c.Request.FormFile(name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving file '%s': %v", name, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msgf("error closing file '%s'", name)
		}
	}(file)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading contents of file '%s': %v", name, err)
	}

	var obj map[string]interface{}
	if strings.HasSuffix(fileHeader.Filename, ".json") {
		err = json.Unmarshal(fileBytes, &obj)
		if err != nil {
			return nil, fmt.Errorf("error parsing contents of file '%s' as JSON: %v", fileHeader.Filename, err)
		}
	} else if strings.HasSuffix(fileHeader.Filename, ".yaml") || strings.HasSuffix(fileHeader.Filename, ".yml") {
		err = yaml.Unmarshal(fileBytes, &obj)
		if err != nil {
			return nil, fmt.Errorf("error parsing contents of file '%s' as YAML: %v", fileHeader.Filename, err)
		}
	} else {
		return nil, fmt.Errorf("unsupported file format for file '%s'", fileHeader.Filename)
	}

	return obj, nil
}

func parseFormData(c *gin.Context, ps []*parameters.ParameterDefinition) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	for _, p := range ps {
		value := c.PostForm(p.Name)
		if value == "" {
			if p.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
			}
			params[p.Name] = p.Default
		} else if p.Type != parameters.ParameterTypeStringFromFile && p.Type != parameters.ParameterTypeObjectFromFile {
			pValue, err := p.ParseParameter([]string{value})
			if err != nil {
				return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
			}
			params[p.Name] = pValue
		} else if p.Type == parameters.ParameterTypeStringFromFile {
			s, err := parseStringFromFile(c, p.Name)
			if err != nil {
				return nil, err
			}
			params[p.Name] = s
		} else if p.Type == parameters.ParameterTypeObjectFromFile {
			obj, err := parseObjectFromFile(c, p.Name)
			if err != nil {
				return nil, err
			}
			params[p.Name] = obj
		}

	}
	return params, nil
}

type ParkaCommand interface {
	cmds.Command
	RunFromParka(
		c *gin.Context,
		parsedLayers map[string]*layers.ParsedParameterLayer,
		parameters map[string]interface{},
		gp *cmds.GlazeProcessor,
	) error
}

type SimpleParkaCommand struct {
	cmds.Command
}

func (s *SimpleParkaCommand) RunFromParka(
	c *gin.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	parameters map[string]interface{},
	gp *cmds.GlazeProcessor,
) error {
	// TODO(manuel, 2023-02-27) Add parsed layers support
	return s.Command.Run(c, parsedLayers, parameters, gp)
}

func NewSimpleParkaCommand(c cmds.Command) *SimpleParkaCommand {
	return &SimpleParkaCommand{c}
}

func (s *Server) serveCommands() {
	apiCmds := []interface{}{}

	// this probably should be exposed as a constructor argument, instead
	// making this generic and adding commands up front.
	//
	// Something with either WithCommand() at ocnstructor time,
	// or probably better ServeCommand() once the server is created.
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
			// TODO(manuel, 2023-02-27) Parse layers
			flags, err := parseQueryParameters(c, append(description.Flags, description.Arguments...))
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			parsedLayers := map[string]*layers.ParsedParameterLayer{}

			of, gp, _ := SetupProcessor()

			err = cmd.RunFromParka(c, parsedLayers, flags, gp)
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

		s.Router.POST(path, func(c *gin.Context) {
			// check if we are multiform data POST
			// if so, parse the form data

			if c.ContentType() == "multipart/form-data" {
				// parse the form data
				flags, err := parseFormData(c, description.Flags)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				of, gp, _ := SetupProcessor()

				// TODO(manuel, 2023-02-27) parse layers
				parsedLayers := map[string]*layers.ParsedParameterLayer{}
				err = cmd.RunFromParka(c, parsedLayers, flags, gp)
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
			}

		})
	}

	s.Router.GET("/api/commands", func(c *gin.Context) {
		c.JSON(200, apiCmds)
	})

}

func SetupProcessor() (*formatters.JSONOutputFormatter, *cmds.GlazeProcessor, error) {
	// TODO(manuel, 2023-02-11) For now, create a raw JSON output formatter. We will want more nuance here
	// See https://github.com/go-go-golems/parka/issues/8

	of := formatters.NewJSONOutputFormatter(true)
	gp := cmds.NewGlazeProcessor(of, []middlewares.ObjectMiddleware{})

	return of, gp, nil
}
