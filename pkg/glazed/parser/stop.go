package parser

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// StopParseStep is a step that stops the parsing process.
// No more parameter definitions will be parsed after this step.
type StopParseStep struct{}

func NewStopParseStep() *StopParseStep {
	return &StopParseStep{}
}

func (s *StopParseStep) Parse(_ *gin.Context, state *LayerParseState) error {
	// no more parsing after this
	state.ParameterDefinitions = map[string]*parameters.ParameterDefinition{}

	return nil
}
