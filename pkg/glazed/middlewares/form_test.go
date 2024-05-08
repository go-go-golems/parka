package middlewares

import (
	_ "embed"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UpdateFromFormQueryTest represents a single test case for UpdateFromFormQuery.
type UpdateFromFormQueryTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	ParsedLayers    []helpers.TestParsedLayer    `yaml:"parsedLayers"`
	Form            utils.MultipartForm          `yaml:"form"`
	ExpectedLayers  []helpers.TestExpectedLayer  `yaml:"expectedLayers"`
	ExpectedError   bool                         `yaml:"expectedError"`
}

//go:embed test-data/update-from-form-query.yaml
var updateFromFormQueryTestsYAML string

// TestUpdateFromFormQuery runs the table-driven tests for UpdateFromFormQuery.
func TestUpdateFromFormQuery(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]UpdateFromFormQueryTest](updateFromFormQueryTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := utils.NewRequestWithMultipartForm(tt.Form)
			require.NoError(t, err)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			parsedLayers := helpers.NewTestParsedLayers(layers_, tt.ParsedLayers...)

			resp := httptest.NewRecorder()
			e := echo.New()

			c := e.NewContext(req, resp)

			// Create the middleware and execute it
			middleware := UpdateFromFormQuery(c)
			err = middleware(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
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
