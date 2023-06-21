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
	for _, p := range state.ParameterDefinitions {
		value := c.DefaultPostForm(p.Name, state.Defaults[p.Name])
		// TODO(manuel, 2023-02-28) is this enough to check if a file is missing?
		if value == "" {
			if p.Required {
				return fmt.Errorf("required parameter '%s' is missing", p.Name)
			}
			if !f.onlyDefined {
				if p.Type == parameters.ParameterTypeDate {
					switch v := p.Default.(type) {
					case string:
						parsedDate, err := parameters.ParseDate(v)
						if err != nil {
							return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
						}

						state.Parameters[p.Name] = parsedDate
					case time.Time:
						state.Parameters[p.Name] = v

					}
				} else {
					state.Parameters[p.Name] = p.Default
				}
			}
		} else if !parameters.IsFileLoadingParameter(p.Type, value) {
			v := []string{value}
			if p.Type == parameters.ParameterTypeStringList ||
				p.Type == parameters.ParameterTypeIntegerList ||
				p.Type == parameters.ParameterTypeFloatList {
				v = strings.Split(value, ",")
			}
			pValue, err := p.ParseParameter(v)
			if err != nil {
				return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
			}
			state.Parameters[p.Name] = pValue
		} else if p.Type == parameters.ParameterTypeStringFromFile {
			s, err := ParseStringFromFile(c, p.Name)
			if err != nil {
				return err
			}
			state.Parameters[p.Name] = s
		} else if p.Type == parameters.ParameterTypeObjectFromFile {
			obj, err := ParseObjectFromFile(c, p.Name)
			if err != nil {
				return err
			}
			state.Parameters[p.Name] = obj
		} else if p.Type == parameters.ParameterTypeStringListFromFile {
			// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
			// See: https://github.com/go-go-golems/parka/issues/23
			_ = state.Parameters
		} else if p.Type == parameters.ParameterTypeObjectListFromFile {
			// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
			// See: https://github.com/go-go-golems/parka/issues/23
			_ = state.Parameters
		}
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
