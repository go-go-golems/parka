package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"strings"
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

func UpdateFromFormQuery(c *gin.Context, options ...parameters.ParseStepOption) middlewares.Middleware {
	return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
		return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
			err := layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
				parsedLayer := parsedLayers.GetOrCreate(l)

				pds := l.GetParameterDefinitions()
				err := pds.ForEachE(func(p *parameters.ParameterDefinition) error {
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

					switch {
					case !p.Type.IsFileLoading(value):
						v := []string{value}
						// TODO(manuel, 2023-12-22) There should be a more robust way to send these values as form values
						if p.Type == parameters.ParameterTypeStringList ||
							p.Type == parameters.ParameterTypeIntegerList ||
							p.Type == parameters.ParameterTypeFloatList {
							v = strings.Split(value, ",")
						}

						parsedParameter, err := p.ParseParameter(v, options...)
						if err != nil {
							return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
						}
						parsedLayer.Parameters.Update(p.Name, parsedParameter)
					case p.Type == parameters.ParameterTypeStringFromFile:
						s, err := ParseStringFromFile(c, p.Name)
						if err != nil {
							return err
						}
						parsedLayer.Parameters.UpdateValue(p.Name, p, s, options...)
					case p.Type == parameters.ParameterTypeObjectFromFile:
						obj, err := ParseObjectFromFile(c, p.Name)
						if err != nil {
							return err
						}
						parsedLayer.Parameters.UpdateValue(p.Name, p, obj, options...)
					case p.Type == parameters.ParameterTypeStringListFromFile,
						p.Type == parameters.ParameterTypeObjectListFromFile:
						fallthrough
					default:
						// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
						// See: https://github.com/go-go-golems/parka/issues/23
						return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, "invalid file type")
					}

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
