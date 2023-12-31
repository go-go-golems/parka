package middlewares

import (
	"bytes"
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/yaml"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test-data/update-from-form-query.yaml
var updateFromFormQueryTestsYAML string

// UpdateFromFormQueryTest represents a single test case for UpdateFromFormQuery.
type UpdateFromFormQueryTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	ParsedLayers    []helpers.TestParsedLayer    `yaml:"parsedLayers"`
	Form            MultipartForm                `yaml:"form"`
	ExpectedLayers  []helpers.TestExpectedLayer  `yaml:"expectedLayers"`
	ExpectedError   bool                         `yaml:"expectedError"`
}

type Field struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type File struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}

type MultipartForm struct {
	Fields []Field           `yaml:"fields"` // Regular form fields
	Files  map[string][]File `yaml:"files"`  // File fields with file content
}

// mockGinContextWithMultipartForm creates a mock gin.Context with multipart form data.
func mockGinContextWithMultipartForm(form MultipartForm) (*gin.Context, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for _, kv := range form.Fields {
		err := writer.WriteField(kv.Name, kv.Value)
		if err != nil {
			return nil, err
		}
	}

	// Add file fields with content
	for key, fileContents := range form.Files {
		for _, file := range fileContents {
			part, err := writer.CreateFormFile(key, file.Name) // Use a dummy filename
			if err != nil {
				return nil, err
			}
			_, err = part.Write([]byte(file.Content)) // Write the actual content provided in the test case
			if err != nil {
				return nil, err
			}
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c, nil
}

// TestUpdateFromFormQuery runs the table-driven tests for UpdateFromFormQuery.
func TestUpdateFromFormQuery(t *testing.T) {
	tests, err := yaml.LoadTestFromYAML[[]UpdateFromFormQueryTest](updateFromFormQueryTestsYAML)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create a mock gin.Context with the multipart form data
			gin.SetMode(gin.TestMode)
			c, _ := mockGinContextWithMultipartForm(tt.Form)

			// Create ParameterLayers and ParsedLayers from test definitions
			layers_ := helpers.NewTestParameterLayers(tt.ParameterLayers)
			parsedLayers := helpers.NewTestParsedLayers(layers_, tt.ParsedLayers)

			// Create the middleware and execute it
			middleware := UpdateFromFormQuery(c)
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
