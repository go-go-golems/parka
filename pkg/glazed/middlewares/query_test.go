package middlewares

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UpdateFromQueryParametersTest struct {
	Name             string                        `yaml:"name"`
	Description      string                        `yaml:"description"`
	Sections         []helpers.TestSection         `yaml:"sections"`
	Values           []helpers.TestSectionValues   `yaml:"values"`
	QueryParameters  []utils.QueryParameter        `yaml:"queryParameters"`
	ExpectedSections []helpers.TestExpectedSection `yaml:"expectedSections"`
	ExpectedError    bool                          `yaml:"expectedError"`
	ErrorString      string                        `yaml:"errorString,omitempty"`
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

			schema_ := helpers.NewTestSchema(tt.Sections)
			parsedValues := helpers.NewTestValues(schema_, tt.Values...)

			resp := httptest.NewRecorder()
			e := echo.New()
			e.ServeHTTP(resp, req)
			c := e.NewContext(req, resp)

			// Create the middleware and execute it
			middleware := UpdateFromQueryParameters(c)
			err := middleware(func(schema_ *schema.Schema, parsedValues *values.Values) error {
				return nil
			})(schema_, parsedValues)

			// Check for expected error
			if tt.ExpectedError {
				assert.Error(t, err)
				if tt.ErrorString != "" {
					assert.Equal(t, tt.ErrorString, err.Error())
				}
			} else {
				require.NoError(t, err)
				helpers.TestExpectedOutputs(t, tt.ExpectedSections, parsedValues)
			}
		})
	}
}
