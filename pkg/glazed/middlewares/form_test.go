package middlewares

import (
	_ "embed"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/labstack/echo/v4"

	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UpdateFromFormQueryTest represents a single test case for UpdateFromFormQuery.
type UpdateFromFormQueryTest struct {
	Name             string                        `yaml:"name"`
	Description      string                        `yaml:"description"`
	Sections         []helpers.TestSection         `yaml:"sections"`
	Values           []helpers.TestSectionValues   `yaml:"values"`
	Form             utils.MultipartForm           `yaml:"form"`
	ExpectedSections []helpers.TestExpectedSection `yaml:"expectedSections"`
	ExpectedError    bool                          `yaml:"expectedError"`
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

			schema_ := helpers.NewTestSchema(tt.Sections)
			parsedValues := helpers.NewTestValues(schema_, tt.Values...)

			resp := httptest.NewRecorder()
			e := echo.New()
			c := e.NewContext(req, resp)

			// Create the middleware instance
			middleware := NewFormMiddleware(c)
			defer func() {
				_ = middleware.Close()
			}()

			// Execute the middleware
			err = middleware.Middleware()(func(schema_ *schema.Schema, parsedValues *values.Values) error {
				return nil
			})(schema_, parsedValues)

			// Check for expected error
			if tt.ExpectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				helpers.TestExpectedOutputs(t, tt.ExpectedSections, parsedValues)
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
	err = middleware.Middleware()(func(schema_ *schema.Schema, parsedValues *values.Values) error {

		return nil
	})(schema.NewSchema(), values.New())
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
	defer func() {
		_ = middleware.Close()
	}()

	// Execute the middleware
	err = middleware.Middleware()(func(schema_ *schema.Schema, parsedValues *values.Values) error {
		return nil
	})(schema.NewSchema(), values.New())
	require.NoError(t, err)
}

// TestFormMiddlewareWithDefaultLayer tests that the form middleware correctly processes
// form data and updates the parsed values for a configuration section.
func TestFormMiddlewareWithDefaultLayer(t *testing.T) {
	section, err := schema.NewSection("config", "Configuration",
		schema.WithDescription("Configuration options for testing"),
		schema.WithFields(
			fields.New(
				"name",
				fields.TypeString,
				fields.WithHelp("Test name parameter"),
				fields.WithRequired(true),
			),
			fields.New(
				"count",
				fields.TypeInteger,
				fields.WithHelp("Test count parameter"),
				fields.WithDefault(42),
			),
			fields.New(
				"enabled",
				fields.TypeBool,
				fields.WithHelp("Test boolean parameter"),
				fields.WithDefault(false),
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

	schema_ := schema.NewSchema(schema.WithSections(section))
	parsedValues := values.New()

	// Create and execute the middleware
	middleware := NewFormMiddleware(c)
	defer func() {
		_ = middleware.Close()
	}()

	err = middleware.Middleware()(func(schema_ *schema.Schema, parsedValues *values.Values) error {
		return nil
	})(schema_, parsedValues)
	require.NoError(t, err)

	nameParam, ok := parsedValues.GetField("config", "name")
	require.True(t, ok)
	assert.Equal(t, "test-value", nameParam.Value)

	countParam, ok := parsedValues.GetField("config", "count")
	require.True(t, ok)
	assert.Equal(t, 123, countParam.Value)

	enabledParam, ok := parsedValues.GetField("config", "enabled")
	require.True(t, ok)
	assert.Equal(t, true, enabledParam.Value)
}
