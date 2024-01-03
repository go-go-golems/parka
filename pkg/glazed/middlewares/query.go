package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"strings"
)

func UpdateFromQueryParameters(c *gin.Context, options ...parameters.ParseStepOption) middlewares.Middleware {
	return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
		return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
			err := next(layers_, parsedLayers)
			if err != nil {
				return err
			}

			err = layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
				parsedLayer := parsedLayers.GetOrCreate(l)

				pds := l.GetParameterDefinitions()
				err := pds.ForEachE(func(p *parameters.ParameterDefinition) error {
					if p.Type.IsFile() {
						return fmt.Errorf("file parameters are not supported in query parameters")
					}

					if p.Type.IsList() {
						// check p.Name[] parameter
						values, ok := c.GetQueryArray(fmt.Sprintf("%s[]", p.Name))
						if ok {
							// TODO(manuel, 2023-12-25) Need to pass in options to ParseParameter
							pp, err := p.ParseParameter(values, options...)
							if err != nil {
								return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, values, err.Error())
							}
							parsedLayer.Parameters.Update(p.Name, pp)
							return nil
						}
					}
					value, ok := c.GetQuery(p.Name)
					if !ok || value == "" {
						if p.Required {
							return fmt.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					if p.Type.NeedsFileContent("") {
						f := strings.NewReader(value)
						// TODO(manuel, 2024-01-01) Use json only for the object types
						fileName := "test.txt"
						if p.Type.IsObject() {
							fileName = "test.json"
						}
						pp, err := p.ParseFromReader(f, fileName, options...)
						if err != nil {
							return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
						}
						parsedLayer.Parameters.Update(p.Name, pp)
					} else {
						var values []string
						if p.Type.IsList() {
							values = strings.Split(value, ",")
						} else {
							values = []string{value}
						}
						// TODO(manuel, 2023-12-25) Need to pass in options to ParseParameter
						pp, err := p.ParseParameter(values, options...)
						if err != nil {
							return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
						}
						parsedLayer.Parameters.Update(p.Name, pp)
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
