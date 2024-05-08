package utils

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
)

// QueryParameter holds a key-value pair for a query parameter.
type QueryParameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Field holds a key-value pair for a form field.
type Field struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// MultipartForm holds form fields and files for a multipart form.
type MultipartForm struct {
	Fields []Field           `yaml:"fields"` // Regular form fields
	Files  map[string][]File `yaml:"files"`  // File fields with file content
}

// File contains metadata and content for a file in a multipart form.
type File struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}

// NewRequestWithQueryParameters creates a mock echo.Context with the
// provided query parameters set. It is intended for use in tests.
//
// It returns a GET request for the path "/"
func NewRequestWithQueryParameters(parameters []QueryParameter) *http.Request {
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

	return req
}

// NewRequestWithMultipartForm creates a mock gin.Context populated
// with the provided multipart form data. It constructs the multipart
// form, attaches it to a mock request, and creates a gin.Context using
// that request. Returns the created context and any error.
//
// The request is a POST request to "/" with the provided form data.
func NewRequestWithMultipartForm(form MultipartForm) (*http.Request, error) {
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
	return req, nil
}
