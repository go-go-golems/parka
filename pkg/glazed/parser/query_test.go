package parser

import (
	_ "embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

//go:embed tests/query.yaml
var queryYAML []byte

// TODO(manuel, 2023-12-22) I need to write many many more tests here

func createHTTPRequestWithQueryValues(value map[string]string) *http.Request {
	req, _ := http.NewRequest("GET", "/test", nil)
	q := req.URL.Query()
	for k, v := range value {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req
}

func createGinContextWithQueryValues(value map[string]string) *gin.Context {
	req := createHTTPRequestWithQueryValues(value)
	return &gin.Context{
		Request: req,
	}
}

func TestEmptyQueryOnlyDefined(t *testing.T) {
	ps := NewQueryParseStep(true)

	parameterDefinitions := parameters.LoadParameterDefinitionsFromYAML(queryYAML)
	state := &LayerParseState{
		// can we parse that from yaml
		ParameterDefinitions: parameterDefinitions,
		Defaults:             map[string]string{},
		ParsedParameters:     parameters.NewParsedParameters(),
	}

	c := createGinContextWithQueryValues(map[string]string{})

	err := ps.Parse(c, state)
	require.Nil(t, err)

	require.Equal(t, 0, state.ParsedParameters.Len())
}

func TestEmptyQuery(t *testing.T) {
	ps := NewQueryParseStep(false)

	parameterDefinitions := parameters.LoadParameterDefinitionsFromYAML(queryYAML)
	state := NewLayerParseState(WithParameterDefinitions(parameterDefinitions))

	c := createGinContextWithQueryValues(map[string]string{})

	err := ps.Parse(c, state)
	require.Nil(t, err)

	require.Equal(t, 2, state.ParsedParameters.Len())
	require.Equal(t, "default", state.ParsedParameters.GetValue("testDefault"))
	require.Nil(t, state.ParsedParameters.GetValue("test"))
}
