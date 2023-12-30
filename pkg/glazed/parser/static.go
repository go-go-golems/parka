package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type StaticParseStep struct {
	Parameters map[string]interface{}
}

func NewStaticParseStep(parameters map[string]interface{}) *StaticParseStep {
	return &StaticParseStep{
		Parameters: parameters,
	}
}

func (s *StaticParseStep) Parse(_ *gin.Context, state *LayerParseState) error {
	for k, v := range s.Parameters {
		p, ok := state.ParameterDefinitions.Get(k)
		if !ok {
			return errors.Errorf("parameter '%s' is not defined", k)
		}
		state.ParsedParameters.UpdateValue(k, p, v)
	}

	return nil
}
