package middlewares

import (
	_ "embed"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"

	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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

			// Create the middleware instance
			middleware := NewFormMiddleware(c)
			defer middleware.Close()

			// Execute the middleware
			err = middleware.Middleware()(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
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

// TestFormMiddlewareCleanup tests that temporary files are properly cleaned up
func TestFormMiddlewareCleanup(t *testing.T) {
	// Create a test file
	content := []byte("test content")

	// Create a multipart form with the test file
	form := utils.MultipartForm{
		Files: map[string][]utils.File{
			"file": {
				{
					Name:    "test.txt",
					Content: string(content),
				},
			},
		},
	}

	req, err := utils.NewRequestWithMultipartForm(form)
	require.NoError(t, err)

	resp := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, resp)

	// Create the middleware instance
	middleware := NewFormMiddleware(c)

	// Execute the middleware
	err = middleware.Middleware()(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {

		return nil
	})(layers.NewParameterLayers(), layers.NewParsedLayers())
	require.NoError(t, err)

	// Close the middleware (this should clean up files)
	err = middleware.Close()
	require.NoError(t, err)
}

// TestFormMiddlewareErrorHandling tests error handling in the form middleware
func TestFormMiddlewareErrorHandling(t *testing.T) {
	// Create a test file that will be deleted before processing
	content := []byte("test content")

	// Create a multipart form with the non-existent file
	form := utils.MultipartForm{
		Files: map[string][]utils.File{
			"file": {
				{
					Name:    "test.txt",
					Content: string(content),
				},
			},
		},
	}

	req, err := utils.NewRequestWithMultipartForm(form)
	require.NoError(t, err)

	resp := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, resp)

	// Create the middleware instance
	middleware := NewFormMiddleware(c)
	defer middleware.Close()

	// Execute the middleware
	err = middleware.Middleware()(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
		return nil
	})(layers.NewParameterLayers(), layers.NewParsedLayers())
	require.NoError(t, err)
}

// TestFormMiddlewareWithDefaultLayer tests that the form middleware correctly processes
// form data and updates the parsed layers with a default parameter layer.
func TestFormMiddlewareWithDefaultLayer(t *testing.T) {
	// Create a default parameter layer with some test parameters
	layer, err := layers.NewParameterLayer("config", "Configuration",
		layers.WithDescription("Configuration options for testing"),
		layers.WithParameterDefinitions(
			parameters.NewParameterDefinition(
				"name",
				parameters.ParameterTypeString,
				parameters.WithHelp("Test name parameter"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"count",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Test count parameter"),
				parameters.WithDefault(42),
			),
			parameters.NewParameterDefinition(
				"enabled",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Test boolean parameter"),
				parameters.WithDefault(false),
			),
		),
	)
	require.NoError(t, err)

	// Create test form data
	form := utils.MultipartForm{
		Fields: []utils.Field{
			{Name: "name", Value: "test-value"},
			{Name: "count", Value: "123"},
			{Name: "enabled", Value: "true"},
		},
	}

	// Create the request with form data
	req, err := utils.NewRequestWithMultipartForm(form)
	require.NoError(t, err)

	// Setup Echo context
	e := echo.New()
	resp := httptest.NewRecorder()
	c := e.NewContext(req, resp)

	// Create parameter layers and parsed layers
	layers_ := layers.NewParameterLayers(layers.WithLayers(layer))
	parsedLayers := layers.NewParsedLayers()

	// Create and execute the middleware
	middleware := NewFormMiddleware(c)
	defer middleware.Close()

	err = middleware.Middleware()(func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
		return nil
	})(layers_, parsedLayers)
	require.NoError(t, err)

	// Verify the parsed values
	// Check string parameter
	nameParam, ok := parsedLayers.GetParameter("config", "name")
	require.True(t, ok)
	assert.Equal(t, "test-value", nameParam.Value)

	// Check integer parameter
	countParam, ok := parsedLayers.GetParameter("config", "count")
	require.True(t, ok)
	assert.Equal(t, 123, countParam.Value)

	// Check boolean parameter
	enabledParam, ok := parsedLayers.GetParameter("config", "enabled")
	require.True(t, ok)
	assert.Equal(t, true, enabledParam.Value)
}
