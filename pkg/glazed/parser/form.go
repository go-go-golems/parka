package parser

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"strings"
	"time"
)

type FormParseStep struct {
	onlyDefined bool
}

func (f *FormParseStep) ParseLayerState(c *gin.Context, state *LayerParseState) error {
	err := state.ParameterDefinitions.ForEachE(func(p *parameters.ParameterDefinition) error {
		if parameters.IsListParameter(p.Type) {
			// check p.Name[] parameter
			values, ok := c.GetPostFormArray(fmt.Sprintf("%s[]", p.Name))
			if ok {
				pValue, err := p.ParseParameter(values)
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, values, err.Error())
				}
				state.ParsedParameters.UpdateValue(p.Name, p, "form-parse", pValue)
				return nil
			}
		}

		value := c.DefaultPostForm(p.Name, state.Defaults[p.Name])
		// TODO(manuel, 2023-02-28) is this enough to check if a file is missing?
		if value == "" {
			if p.Required {
				return fmt.Errorf("required parameter '%s' is missing", p.Name)
			}
			if !f.onlyDefined {
				// NOTE(manuel, 2023-12-22) It's odd that we have some mechanism here to set defaults, can't we use the standard layer filling?
				if _, ok := state.ParsedParameters.Get(p.Name); !ok {
					if p.Type == parameters.ParameterTypeDate {
						switch v := p.Default.(type) {
						case string:
							parsedDate, err := parameters.ParseDate(v)
							if err != nil {
								return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
							}

							state.ParsedParameters.SetAsDefault(p.Name, p, "form-parse-default", parsedDate)
						case time.Time:
							state.ParsedParameters.SetAsDefault(p.Name, p, "form-parse-default", v)

						}
					} else {
						state.ParsedParameters.SetAsDefault(p.Name, p, "form-parse-default", p.Default)
					}
				}
			}
		} else if !parameters.IsFileLoadingParameter(p.Type, value) {
			v := []string{value}
			// TODO(manuel, 2023-12-22) There should be a more robust way to send these values as form values
			if p.Type == parameters.ParameterTypeStringList ||
				p.Type == parameters.ParameterTypeIntegerList ||
				p.Type == parameters.ParameterTypeFloatList {
				v = strings.Split(value, ",")
			}
			pValue, err := p.ParseParameter(v)
			if err != nil {
				return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
			}
			state.ParsedParameters.UpdateValue(p.Name, p, "form-parse-list", pValue)
		} else if p.Type == parameters.ParameterTypeStringFromFile {
			s, err := ParseStringFromFile(c, p.Name)
			if err != nil {
				return err
			}
			state.ParsedParameters.UpdateValue(p.Name, p, "form-parse-file", s)
		} else if p.Type == parameters.ParameterTypeObjectFromFile {
			obj, err := ParseObjectFromFile(c, p.Name)
			if err != nil {
				return err
			}
			state.ParsedParameters.UpdateValue(p.Name, p, "form-parse-file", obj)
		} else if p.Type == parameters.ParameterTypeStringListFromFile {
			// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
			// See: https://github.com/go-go-golems/parka/issues/23
			_ = state.ParsedParameters
		} else if p.Type == parameters.ParameterTypeObjectListFromFile {
			// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
			// See: https://github.com/go-go-golems/parka/issues/23
			_ = state.ParsedParameters
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (f *FormParseStep) Parse(c *gin.Context, state *LayerParseState) error {
	err := f.ParseLayerState(c, state)
	if err != nil {
		return err
	}

	return nil
}

func NewFormParseStep(onlyDefined bool) *FormParseStep {
	return &FormParseStep{
		onlyDefined: onlyDefined,
	}
}
