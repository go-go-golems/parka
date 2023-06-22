package parser

import (
	"github.com/gin-gonic/gin"
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
		if _, ok := state.ParameterDefinitions[k]; ok {
			// then, check if the parameter is not set yet
			if _, ok = state.Parameters[k]; !ok {
				state.Parameters[k] = v
			}
		}
	}

	return nil
}
