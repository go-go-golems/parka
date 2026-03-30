package middlewares

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func UpdateFromQueryParameters(c echo.Context, options ...fields.ParseOption) sources.Middleware {
	return func(next sources.HandlerFunc) sources.HandlerFunc {
		return func(schema_ *schema.Schema, parsedValues *values.Values) error {
			err := next(schema_, parsedValues)
			if err != nil {
				return err
			}

			err = schema_.ForEachE(func(_ string, section schema.Section) error {
				sectionValues := parsedValues.GetOrCreate(section)
				defs := section.GetDefinitions()
				err := defs.ForEachE(func(p *fields.Definition) error {
					if p.Type.IsFile() {
						return errors.New("file parameters are not supported in query parameters")
					}

					if p.Type.IsList() {
						// check p.Name[] parameter
						values_, ok := c.QueryParams()[fmt.Sprintf("%s[]", p.Name)]
						if ok {
							parsedField, err := p.ParseField(values_, options...)
							if err != nil {
								return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, values_)
							}
							sectionValues.Fields.Update(p.Name, parsedField)
							return nil
						}
					}
					value := c.QueryParam(p.Name)
					if value == "" {
						if p.Required {
							return errors.Errorf("required parameter '%s' is missing", p.Name)
						}
						return nil
					}

					if p.Type.NeedsFileContent("") {
						f := strings.NewReader(value)
						fileName := "query.txt"
						if p.Type.IsObject() {
							fileName = "query.json"
						}
						parsedField, err := p.ParseFromReader(f, fileName, options...)
						if err != nil {
							return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
						}
						sectionValues.Fields.Update(p.Name, parsedField)
					} else {
						var values_ []string
						if p.Type.IsList() {
							values_ = strings.Split(value, ",")
						} else {
							values_ = []string{value}
						}
						parsedField, err := p.ParseField(values_, options...)
						if err != nil {
							return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
						}
						sectionValues.Fields.Update(p.Name, parsedField)
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
