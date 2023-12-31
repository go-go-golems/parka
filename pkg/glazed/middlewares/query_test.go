package middlewares

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UpdateFromQueryParametersTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	ParsedLayers    []helpers.TestParsedLayer    `yaml:"parsedLayers"`
	QueryParameters []QueryParameter             `yaml:"queryParameters"`
	ExpectedLayers  []helpers.TestExpectedLayer  `yaml:"expectedLayers"`
	ExpectedError   bool                         `yaml:"expectedError"`
}

type QueryParameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func mockGinContextWithQueryParameters(parameters []QueryParameter) (*gin.Context, error) {
	// Create a new HTTP request
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create a Values map to hold the query parameters
	values := url.Values{}

	// Add each parameter to the Values map
	for _, param := range parameters {
		values.Add(param.Name, param.Value)
	}

	// Set the RawQuery field of the request URL
	req.URL.RawQuery = values.Encode()

	// Create a new gin context with the request
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	return c, nil
}

//go:embed test-data/update-from-query-parameters.yaml
var updateFromQueryParametersTestsYAML string

// TestUpdateFromFormQuery runs the table-driven tests for UpdateFromFormQuery.
func TestUpdateFromQueryParameters(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]UpdateFromQueryParametersTest](updateFromQueryParametersTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create a mock gin.Context with the multipart form data
			gin.SetMode(gin.TestMode)
			c, _ := mockGinContextWithQueryParameters(tt.QueryParameters)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			parsedLayers := helpers.NewTestParsedLayers(layers_, tt.ParsedLayers)

			// Create the middleware and execute it
			middleware := UpdateFromQueryParameters(c)
			err := middleware(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
				return nil
			})(layers_, parsedLayers)

			// Check for expected error
			if tt.ExpectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Check expected outputs
				helpers.TestExpectedOutputs(t, tt.ExpectedLayers, parsedLayers)
			}
		})
	}
}
