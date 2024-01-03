package text

import (
	_ "embed"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TextHandlerTest describes a test for the `text` handler. This handler
// takes a GET HTTP query and parses the url parameters to render
// the template after parsing according it to the parameter definitions.
type TextHandlerTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	QueryParameters []utils.QueryParameter       `yaml:"queryParameters"`
	Template        string                       `yaml:"template"`
	ExpectedOutput  string                       `yaml:"expectedOutput"`
	ExpectedError   bool                         `yaml:"expectedError"`
	ErrorString     string                       `yaml:"errorString,omitempty"`
}

//go:embed test-data/text-handler.yaml
var textHandlerTestsYAML string

func TestTextHandler(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]TextHandlerTest](textHandlerTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := utils.MockGinContextWithQueryParameters(tt.QueryParameters)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)

			// TODO(manuel, 2024-01-02) We also need to test with glazed commands
			cmd := cmds.NewTemplateCommand(tt.Name, tt.Template, cmds.WithLayers(layers_))

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
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Equal(t, tt.ExpectedOutput, resp.Body.String())
				assert.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
			}
		})
	}
}
