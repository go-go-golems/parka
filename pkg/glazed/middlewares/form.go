package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/pkg/errors"
)

func getListParameterFromForm(c *gin.Context, p *parameters.ParameterDefinition, options ...parameters.ParseStepOption) (*parameters.ParsedParameter, error) {
	if p.Type.IsList() {
		// check p.Name[] parameter
		values, ok := c.GetPostFormArray(fmt.Sprintf("%s[]", p.Name))
		if ok {
			pValue, err := p.ParseParameter(values, options...)
			if err != nil {
				return nil, errors.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, values, err.Error())
			}
			return pValue, nil
		}

		return nil, nil
	} else {
		return nil, errors.Errorf("parameter '%s' is not a list parameter", p.Name)
	}
}

func getFileParameterFromForm(c *gin.Context, p *parameters.ParameterDefinition) (interface{}, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	headers := form.File[p.Name]
	if len(headers) == 0 {
		if p.Required {
			return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
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
			defer func() {
				_ = f.Close()
			}()

			v, err := p.ParseFromReader(f, h.Filename)
			if err != nil {
				return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, h.Filename, err.Error())
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

func UpdateFromFormQuery(c *gin.Context, options ...parameters.ParseStepOption) middlewares.Middleware {
	return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
		return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
			err := layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
				parsedLayer := parsedLayers.GetOrCreate(l)

				pds := l.GetParameterDefinitions()
				err := pds.ForEachE(func(p *parameters.ParameterDefinition) error {
					if p.Type.NeedsFileContent("") {
						v, err := getFileParameterFromForm(c, p)
						if err != nil {
							return err
						}

						if v != nil {
							parsedLayer.Parameters.UpdateValue(p.Name, p, v, options...)
						}

						return nil
					}

					// parse arrays
					if p.Type.IsList() {
						v, err := getListParameterFromForm(c, p, options...)
						if err != nil {
							return err
						}
						if v != nil {
							parsedLayer.Parameters.Update(p.Name, v)
						} else if p.Required {
							return fmt.Errorf("required parameter '%s' is missing", p.Name)
						}

						return nil
					}

					value, ok := c.GetPostForm(p.Name)
					// TODO(manuel, 2023-02-28) is this enough to check if a file is missing?
					if !ok {
						if p.Required {
							return fmt.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					v := []string{value}
					parsedParameter, err := p.ParseParameter(v, options...)
					if err != nil {
						return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
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

			return nil
		}
	}
}
