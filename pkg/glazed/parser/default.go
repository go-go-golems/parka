package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

type DefaultParseStep struct {
	Parameters map[string]interface{}
}

func NewDefaultParseStep(parameters map[string]interface{}) *DefaultParseStep {
	return &DefaultParseStep{
		Parameters: parameters,
	}
}

func (s *DefaultParseStep) Parse(_ *gin.Context, state *LayerParseState) error {
	for k, v := range s.Parameters {
		// first, check that we are supposed to parsed that parameter
		if pd, ok := state.ParameterDefinitions.Get(k); ok {
			// then, check if the parameter is not set yet
			if _, ok = state.ParsedParameters.Get(k); !ok {
				parsedParameter := &parameters.ParsedParameter{
					ParameterDefinition: pd,
				}
				parsedParameter.Set("default-parse", v)
			}

		}
	}

	return nil
}
