package middlewares

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// FormMiddleware is a struct-based middleware that handles form data and file uploads
// and manages temporary files created during processing.
type FormMiddleware struct {
	c       echo.Context
	options []parameters.ParseStepOption
	files   []string
	mu      sync.Mutex
}

// NewFormMiddleware creates a new FormMiddleware instance
func NewFormMiddleware(c echo.Context, options ...parameters.ParseStepOption) *FormMiddleware {
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
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, r)
	if err != nil {
		return "", errors.Wrap(err, "could not write to temporary file")
	}

	return tmpFile.Name(), nil
}

func (m *FormMiddleware) getListParameterFromForm(p *parameters.ParameterDefinition) (*parameters.ParsedParameter, error) {
	if p.Type.IsList() {
		// check p.Name[] parameter
		values_, err := m.c.FormParams()
		if err != nil {
			return nil, err
		}
		values, ok := values_[fmt.Sprintf("%s[]", p.Name)]
		if ok {
			pValue, err := p.ParseParameter(values, m.options...)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, values)
			}
			return pValue, nil
		}

		return nil, nil
	} else {
		return nil, errors.Errorf("parameter '%s' is not a list parameter", p.Name)
	}
}

func (m *FormMiddleware) getFileParameterFromForm(p *parameters.ParameterDefinition) (interface{}, error) {
	form, err := m.c.MultipartForm()
	if err != nil {
		return nil, err
	}
	headers := form.File[p.Name]
	if len(headers) == 0 {
		if p.Required {
			return nil, errors.Errorf("required parameter '%s' is missing", p.Name)
		}

		return nil, nil
	}

	values := []interface{}{}
	for _, h := range headers {
		err = func() error {
			f, err := h.Open()
			if err != nil {
				return err
			}
			defer f.Close()

			// For ParameterTypeFile, we need to create a temporary file
			if p.Type == parameters.ParameterTypeFile || p.Type == parameters.ParameterTypeFileList {
				tmpPath, err := m.createTempFileFromReader(f)
				if err != nil {
					return err
				}
				m.addFile(tmpPath)
				values = append(values, tmpPath)
				return nil
			}

			// For other file-like parameters, parse the content
			v, err := p.ParseFromReader(f, h.Filename, m.options...)
			if err != nil {
				return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, h.Filename)
			}

			values = append(values, v.Value)
			return nil
		}()

		if err != nil {
			return nil, err
		}
	}

	var v interface{}

	//exhaustive:ignore
	switch {
	case p.Type == parameters.ParameterTypeFile:
		if len(values) == 0 {
			return nil, nil
		}
		v = values[0]

	case p.Type == parameters.ParameterTypeFileList:
		v = values

	case p.Type.IsList():
		vs := []interface{}{}
		for _, v_ := range values {
			vss, err := cast.CastListToInterfaceList(v_)
			if err != nil {
				return nil, err
			}
			vs = append(vs, vss...)
		}
		v = vs

	case p.Type == parameters.ParameterTypeStringFromFile,
		p.Type == parameters.ParameterTypeStringFromFiles:
		s := ""
		for _, v_ := range values {
			ss, ok := v_.(string)
			if !ok {
				return nil, errors.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, v_, "expected string")
			}
			s += ss
		}
		v = s

	case p.Type == parameters.ParameterTypeObjectFromFile:
		v = values[0]

	default:
		return nil, errors.Errorf("invalid type for parameter '%s': (%v) %s", p.Name, p.Type, "expected string or list")
	}

	return v, nil
}

// Middleware returns the actual middleware function
func (m *FormMiddleware) Middleware() middlewares.Middleware {
	return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
		return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
			err := layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
				parsedLayer := parsedLayers.GetOrCreate(l)

				pds := l.GetParameterDefinitions()
				err := pds.ForEachE(func(p *parameters.ParameterDefinition) error {
					if p.Type.NeedsFileContent("") || p.Type == parameters.ParameterTypeFile || p.Type == parameters.ParameterTypeFileList {
						v, err := m.getFileParameterFromForm(p)
						if err != nil {
							return err
						}

						if v != nil {
							err := parsedLayer.Parameters.UpdateValue(p.Name, p, v, m.options...)
							if err != nil {
								return err
							}
						}

						return nil
					}

					// parse arrays
					if p.Type.IsList() {
						v, err := m.getListParameterFromForm(p)
						if err != nil {
							return err
						}
						if v != nil {
							parsedLayer.Parameters.Update(p.Name, v)
						} else if p.Required {
							return errors.Errorf("required parameter '%s' is missing", p.Name)
						}

						return nil
					}

					value := m.c.FormValue(p.Name)
					if value == "" {
						if p.Required {
							return errors.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					v := []string{value}
					parsedParameter, err := p.ParseParameter(v, m.options...)
					if err != nil {
						return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
					}
					parsedLayer.Parameters.Update(p.Name, parsedParameter)

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

// UpdateFromFormQuery is a convenience function that creates a FormMiddleware and returns its middleware function
func UpdateFromFormQuery(c echo.Context, options ...parameters.ParseStepOption) middlewares.Middleware {
	m := NewFormMiddleware(c, options...)
	defer m.Close()
	return m.Middleware()
}
