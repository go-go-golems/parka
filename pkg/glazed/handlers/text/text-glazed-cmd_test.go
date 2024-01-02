package text

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TextHandlerTest describes a test for the `text` handler. This handler
// takes a GET HTTP query and parses the url parameters to render
// the template after parsing according it to the parameter definitions.
type TextHandlerGlazedCommandTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	QueryParameters []utils.QueryParameter       `yaml:"queryParameters"`
	ExpectedOutput  interface{}                  `yaml:"expectedOutput"`
	ExpectedError   bool                         `yaml:"expectedError"`
	ErrorString     string                       `yaml:"errorString,omitempty"`
}

//go:embed test-data/text-handler-glazed-command.yaml
var textHandlerGlazedCommandTestsYAML string

func TestTextHandlerGlazeCommand(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]TextHandlerGlazedCommandTest](textHandlerGlazedCommandTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := utils.MockGinContextWithQueryParameters(tt.QueryParameters)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			cmd, err := NewTestGlazedCommand(cmds.WithLayers(layers_))
			require.NoError(t, err)

			router := gin.Default()
			router.GET("/", CreateQueryHandler(cmd))

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, c.Request)

			// Check for expected error
			if tt.ExpectedError {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
				var json_ map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &json_)
				require.NoError(t, err)
				if tt.ErrorString != "" {
					assert.Equal(t, tt.ErrorString, json_["error"])
				}
			} else {
				if resp.Header().Get("Content-Type") == "application/json" {
					assert.Equal(t, http.StatusOK, resp.Code)
					var json_ []map[string]interface{}
					err := json.Unmarshal(resp.Body.Bytes(), &json_)
					require.NoError(t, err)
					assert.Equal(t, tt.ExpectedOutput, json_)

				} else if resp.Header().Get("Content-Type") == "text/plain; charset=utf-8" {
					assert.Equal(t, http.StatusOK, resp.Code)
					assert.Equal(t, tt.ExpectedOutput, resp.Body.String())
				}
			}
		})
	}
}

type TestGlazedCommand struct {
	description *cmds.CommandDescription
}

func NewTestGlazedCommand(options ...cmds.CommandDescriptionOption) (*TestGlazedCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	options = append(options, cmds.WithLayersList(glazedLayer))

	description := cmds.NewCommandDescription("test-glazed-command", options...)
	return &TestGlazedCommand{
		description: description,
	}, nil
}

func (t *TestGlazedCommand) Description() *cmds.CommandDescription {
	return t.description
}

func (t *TestGlazedCommand) ToYAML(w io.Writer) error {
	return t.Description().ToYAML(w)
}

func (t *TestGlazedCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	v, ok := parsedLayers.Get(layers.DefaultSlug)
	if !ok {
		return fmt.Errorf("default layer not found")
	}

	m := v.Parameters.ToMap()

	for i := 0; i < 3; i++ {
		row := types.NewRow(
			types.MRP("test", i),
			types.MRP("test2", fmt.Sprintf("test-%d", i)),
			types.MRP("test3", fmt.Sprintf("test3-%d", i)),
		)
		for k, v := range m {
			row.Set(k, v)
		}
		err := gp.AddRow(ctx,
			row,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ cmds.GlazeCommand = (*TestGlazedCommand)(nil)
