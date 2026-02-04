package middlewares

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// FormMiddleware is a struct-based middleware that handles form data and file uploads
// and manages temporary files created during processing.
type FormMiddleware struct {
	c       echo.Context
	options []fields.ParseOption
	files   []string
	mu      sync.Mutex
}

// NewFormMiddleware creates a new FormMiddleware instance
func NewFormMiddleware(c echo.Context, options ...fields.ParseOption) *FormMiddleware {
	return &FormMiddleware{
		c:       c,
		options: options,
		files:   make([]string, 0),
	}
}

// Close cleans up any temporary files created during processing
func (m *FormMiddleware) Close() error {
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
func (m *FormMiddleware) addFile(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = append(m.files, path)
}

// createTempFileFromReader creates a temporary file from a reader
func (m *FormMiddleware) createTempFileFromReader(r io.Reader) (string, error) {
	tmpFile, err := os.CreateTemp("", "parka-form-*")
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary file")
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	_, err = io.Copy(tmpFile, r)
	if err != nil {
		return "", errors.Wrap(err, "could not write to temporary file")
	}

	return tmpFile.Name(), nil
}

func (m *FormMiddleware) getFilePathsFromForm(p *fields.Definition) ([]string, error) {
	form, err := m.c.MultipartForm()
	if err != nil {
		return nil, err
	}
	headers := form.File[p.Name]
	if len(headers) == 0 {
		return nil, nil
	}

	paths := []string{}
	for _, h := range headers {
		err = func() error {
			f, err := h.Open()
			if err != nil {
				return err
			}
			defer func() {
				_ = f.Close()
			}()

			tmpPath, err := m.createTempFileFromReader(f)
			if err != nil {
				return err
			}
			m.addFile(tmpPath)
			paths = append(paths, tmpPath)
			return nil
		}()

		if err != nil {
			return nil, err
		}
	}

	return paths, nil
}

// Middleware returns the actual middleware function
func (m *FormMiddleware) Middleware() sources.Middleware {
	return func(next sources.HandlerFunc) sources.HandlerFunc {
		return func(schema_ *schema.Schema, parsedValues *values.Values) error {
			err := schema_.ForEachE(func(_ string, section schema.Section) error {
				sectionValues := parsedValues.GetOrCreate(section)

				defs := section.GetDefinitions()
				err := defs.ForEachE(func(p *fields.Definition) error {
					if p.Type.NeedsFileContent("") || p.Type.IsFile() {
						paths, err := m.getFilePathsFromForm(p)
						if err != nil {
							return err
						}

						if len(paths) == 0 {
							if p.Required {
								return errors.Errorf("required parameter '%s' is missing", p.Name)
							}
							return nil
						}

						parsedField, err := p.ParseField(paths, m.options...)
						if err != nil {
							return errors.Wrapf(err, "invalid value for parameter '%s'", p.Name)
						}
						sectionValues.Fields.Update(p.Name, parsedField)
						return nil
					}

					if p.Type.IsList() {
						values_, err := m.c.FormParams()
						if err != nil {
							return err
						}
						if values, ok := values_[fmt.Sprintf("%s[]", p.Name)]; ok {
							parsedField, err := p.ParseField(values, m.options...)
							if err != nil {
								return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, values)
							}
							sectionValues.Fields.Update(p.Name, parsedField)
							return nil
						}
						value := m.c.FormValue(p.Name)
						if value == "" {
							if p.Required {
								return errors.Errorf("required parameter '%s' is missing", p.Name)
							}
							return nil
						}
						parsedField, err := p.ParseField([]string{value}, m.options...)
						if err != nil {
							return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
						}
						sectionValues.Fields.Update(p.Name, parsedField)
						return nil
					}

					value := m.c.FormValue(p.Name)
					if value == "" {
						if p.Required {
							return errors.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					parsedField, err := p.ParseField([]string{value}, m.options...)
					if err != nil {
						return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
					}
					sectionValues.Fields.Update(p.Name, parsedField)

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

			return next(schema_, parsedValues)
		}
	}
}

// UpdateFromFormQuery is a convenience function that creates a FormMiddleware and returns its middleware function
func UpdateFromFormQuery(c echo.Context, options ...fields.ParseOption) sources.Middleware {
	m := NewFormMiddleware(c, options...)
	defer func() {
		_ = m.Close()
	}()
	return m.Middleware()
}
