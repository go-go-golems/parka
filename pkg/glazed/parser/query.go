package parser

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// QueryParseStep parses parameters from the query string of a request.
type QueryParseStep struct {
	onlyDefined bool
}

func (q *QueryParseStep) ParseLayerState(c *gin.Context, state *LayerParseState) error {
	err := state.ParameterDefinitions.ForEachE(func(p *parameters.ParameterDefinition) error {
		if parameters.IsListParameter(p.Type) {
			// check p.Name[] parameter
			values, ok := c.GetQueryArray(fmt.Sprintf("%s[]", p.Name))
			if ok {
				// TODO(manuel, 2023-12-25) Need to pass in options to ParseParameter
				pValue, err := p.ParseParameter(values)
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, values, err.Error())
				}
				state.ParsedParameters.UpdateValue(p.Name, p, pValue)
				return nil
			}
		}
		value := c.DefaultQuery(p.Name, state.Defaults[p.Name])
		if parameters.IsFileLoadingParameter(p.Type, value) {
			// TODO(manuel, 2023-12-22) if the parameter is supposed to be read from a file,
			// we will just pass in the query parameters
			// as a placeholder here
			if value == "" {
				if p.Required {
					return errors.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !q.onlyDefined && p.Default != nil {
					state.ParsedParameters.SetAsDefault(p.Name, p, *p.Default)
				}
			} else {
				f := strings.NewReader(value)
				pValue, err := p.ParseFromReader(f, "")
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				state.ParsedParameters.UpdateValue(p.Name, p, pValue)
			}
		} else {
			if value == "" {
				if p.Required {
					return fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !q.onlyDefined && p.Default != nil {
					if p.Type == parameters.ParameterTypeDate {
						switch v := (*p.Default).(type) {
						case string:
							parsedDate, err := parameters.ParseDate(v)
							if err != nil {
								return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
							}

							state.ParsedParameters.SetAsDefault(p.Name, p, parsedDate)
						case time.Time:
							state.ParsedParameters.SetAsDefault(p.Name, p, v)

						}
					} else {
						state.ParsedParameters.SetAsDefault(p.Name, p, *p.Default)
					}
				}
			} else {
				var values []string
				if parameters.IsListParameter(p.Type) {
					values = strings.Split(value, ",")
				} else {
					values = []string{value}
				}
				// TODO(manuel, 2023-12-25) Need to pass in options to ParseParameter
				pValue, err := p.ParseParameter(values)
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				state.ParsedParameters.UpdateValue(p.Name, p, pValue)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (q *QueryParseStep) Parse(c *gin.Context, state *LayerParseState) error {
	err := q.ParseLayerState(c, state)
	if err != nil {
		return err
	}

	return nil
}

func NewQueryParseStep(onlyDefined bool) ParseStep {
	return &QueryParseStep{
		onlyDefined: onlyDefined,
	}
}
