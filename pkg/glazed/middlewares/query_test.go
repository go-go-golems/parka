package middlewares

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UpdateFromQueryParametersTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	ParsedLayers    []helpers.TestParsedLayer    `yaml:"parsedLayers"`
	QueryParameters []utils.QueryParameter       `yaml:"queryParameters"`
	ExpectedLayers  []helpers.TestExpectedLayer  `yaml:"expectedLayers"`
	ExpectedError   bool                         `yaml:"expectedError"`
	ErrorString     string                       `yaml:"errorString,omitempty"`
}

//go:embed test-data/update-from-query-parameters.yaml
var updateFromQueryParametersTestsYAML string

// TestUpdateFromFormQuery runs the table-driven tests for UpdateFromFormQuery.
func TestUpdateFromQueryParameters(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]UpdateFromQueryParametersTest](updateFromQueryParametersTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req := utils.NewRequestWithQueryParameters(tt.QueryParameters)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			parsedLayers := helpers.NewTestParsedLayers(layers_, tt.ParsedLayers...)

			resp := httptest.NewRecorder()
			e := echo.New()
			e.ServeHTTP(resp, req)
			c := e.NewContext(req, resp)

			// Create the middleware and execute it
			middleware := UpdateFromQueryParameters(c)
			err := middleware(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
				return nil
			})(layers_, parsedLayers)

			// Check for expected error
			if tt.ExpectedError {
				assert.Error(t, err)
				if tt.ErrorString != "" {
					assert.Equal(t, tt.ErrorString, err.Error())
				}
			} else {
				require.NoError(t, err)
				// Check expected outputs
				helpers.TestExpectedOutputs(t, tt.ExpectedLayers, parsedLayers)
			}
		})
	}
}
