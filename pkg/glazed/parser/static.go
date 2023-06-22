package parser

import (
	"github.com/gin-gonic/gin"
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
		state.Parameters[k] = v
	}

	return nil
}
