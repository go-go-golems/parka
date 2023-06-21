package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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

	// NOTE(manuel, 2023-06-21) This could maybe better be done with a "StopParsingStep" that can be put
	// at the end of a chain.

	// no more parsing after this
	state.ParameterDefinitions = map[string]*parameters.ParameterDefinition{}

	return nil
}
