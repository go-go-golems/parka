package middlewares

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// JSONBodyMiddleware is a struct-based middleware that handles JSON POST requests
// and manages temporary files created during processing.
type JSONBodyMiddleware struct {
	c       echo.Context
	options []parameters.ParseStepOption
	files   []string
	mu      sync.Mutex
}

// NewJSONBodyMiddleware creates a new JSONBodyMiddleware instance
func NewJSONBodyMiddleware(c echo.Context, options ...parameters.ParseStepOption) *JSONBodyMiddleware {
	return &JSONBodyMiddleware{
		c:       c,
		options: options,
		files:   make([]string, 0),
	}
}

// Close cleans up any temporary files created during processing
func (m *JSONBodyMiddleware) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, f := range m.files {
		if err := os.Remove(f); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to remove temporary file %s", f))
		}
	}
	m.files = m.files[:0] // Clear the files list

	if len(errs) > 0 {
		return errors.Errorf("failed to clean up some temporary files: %v", errs)
	}
	return nil
}

// addFile adds a temporary file to be cleaned up later
func (m *JSONBodyMiddleware) addFile(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = append(m.files, path)
}

// createTempFileFromString creates a temporary file with the given content
func (m *JSONBodyMiddleware) createTempFileFromString(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "parka-json-*")
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary file")
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	if err != nil {
		return "", errors.Wrap(err, "could not write to temporary file")
	}

	return tmpFile.Name(), nil
}

// Middleware returns the actual middleware function
func (m *JSONBodyMiddleware) Middleware() middlewares.Middleware {
	return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
		return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
			// Read the request body
			body, err := io.ReadAll(m.c.Request().Body)
			if err != nil {
				return errors.Wrap(err, "could not read request body")
			}

			// Parse JSON
			var jsonData map[string]interface{}
			if err := json.Unmarshal(body, &jsonData); err != nil {
				return errors.Wrap(err, "could not parse JSON body")
			}

			err = layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
				parsedLayer := parsedLayers.GetOrCreate(l)

				pds := l.GetParameterDefinitions()
				err := pds.ForEachE(func(p *parameters.ParameterDefinition) error {
					value, exists := jsonData[p.Name]
					if !exists {
						if p.Required {
							return errors.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					// Handle file-like parameters
					if p.Type.NeedsFileContent("") {
						switch v := value.(type) {
						case string:
							// Create a temporary file with the content
							tmpPath, err := m.createTempFileFromString(v)
							if err != nil {
								return err
							}
							m.addFile(tmpPath)

							// Parse the file content
							f, err := os.Open(tmpPath)
							if err != nil {
								return errors.Wrapf(err, "could not open temporary file for parameter '%s'", p.Name)
							}
							defer f.Close()

							parsed, err := p.ParseFromReader(f, filepath.Base(tmpPath), m.options...)
							if err != nil {
								return errors.Wrapf(err, "invalid value for parameter '%s'", p.Name)
							}

							err = parsedLayer.Parameters.UpdateValue(p.Name, p, parsed.Value, m.options...)
							if err != nil {
								return err
							}
							return nil
						default:
							return errors.Errorf("invalid type for file parameter '%s': expected string", p.Name)
						}
					}

					// Handle regular parameters
					var stringValue string
					switch v := value.(type) {
					case string:
						stringValue = v
					case float64:
						stringValue = fmt.Sprintf("%v", v)
					case bool:
						stringValue = fmt.Sprintf("%v", v)
					case []interface{}:
						// Handle array parameters
						if p.Type.IsList() {
							strValues := make([]string, len(v))
							for i, item := range v {
								strValues[i] = fmt.Sprintf("%v", item)
							}
							parsedParam, err := p.ParseParameter(strValues, m.options...)
							if err != nil {
								return errors.Wrapf(err, "invalid value for parameter '%s'", p.Name)
							}
							parsedLayer.Parameters.Update(p.Name, parsedParam)
							return nil
						}
						return errors.Errorf("received array for non-array parameter '%s'", p.Name)
					default:
						return errors.Errorf("unsupported type for parameter '%s'", p.Name)
					}

					parsedParam, err := p.ParseParameter([]string{stringValue}, m.options...)
					if err != nil {
						return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, stringValue)
					}
					parsedLayer.Parameters.Update(p.Name, parsedParam)

					return nil
				})

				if err != nil {
					return err
				}
				return nil
			})

			if err != nil {
				return err
			}

			return next(layers_, parsedLayers)
		}
	}
}
