package text

import (
	_ "embed"
	"encoding/json"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			req := utils.NewRequestWithQueryParameters(tt.QueryParameters)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			cmd, err := utils.NewTestGlazedCommand(cmds.WithLayers(layers_))
			require.NoError(t, err)

			resp := httptest.NewRecorder()
			e := echo.New()
			e.GET("/", CreateQueryHandler(cmd))
			e.ServeHTTP(resp, req)

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
