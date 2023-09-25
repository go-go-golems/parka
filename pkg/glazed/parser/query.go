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
	for _, p := range state.ParameterDefinitions {
		if parameters.IsListParameter(p.Type) {
			// check p.Name[] parameter
			values, ok := c.GetQueryArray(fmt.Sprintf("%s[]", p.Name))
			if ok {
				pValue, err := p.ParseParameter(values)
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, values, err.Error())
				}
				state.Parameters[p.Name] = pValue
				continue
			}
		}
		value := c.DefaultQuery(p.Name, state.Defaults[p.Name])
		if parameters.IsFileLoadingParameter(p.Type, value) {
			// if the parameter is supposed to be read from a file, we will just pass in the query parameters
			// as a placeholder here
			if value == "" {
				if p.Required {
					return errors.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !q.onlyDefined {
					if _, ok := state.Parameters[p.Name]; !ok {
						state.Parameters[p.Name] = p.Default
					}
				}
			} else {
				f := strings.NewReader(value)
				pValue, err := p.ParseFromReader(f, "")
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				state.Parameters[p.Name] = pValue
			}
		} else {
			if value == "" {
				if p.Required {
					return fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !q.onlyDefined {
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
						// only set default value if it is not already set
						if _, ok := state.Parameters[p.Name]; !ok {
							state.Parameters[p.Name] = p.Default
						}
					}
				}
			} else {
				var values []string
				if parameters.IsListParameter(p.Type) {
					values = strings.Split(value, ",")
				} else {
					values = []string{value}
				}
				pValue, err := p.ParseParameter(values)
				if err != nil {
					return fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				state.Parameters[p.Name] = pValue
			}
		}
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
