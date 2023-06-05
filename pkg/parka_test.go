package pkg_test

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/parka/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestCommand struct{}

func (t *TestCommand) Description() *cmds.CommandDescription {
	return cmds.NewCommandDescription("test")
}

func (t *TestCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp processor.Processor,
) error {
	err := gp.ProcessInputObject(ctx, map[string]interface{}{
		"foo": 1,
		"bar": "baz",
	})

	if err != nil {
		return err
	}

	return nil
}

func TestRunGlazedCommand(t *testing.T) {
	tc := &TestCommand{}

	s, err := pkg.NewServer()
	require.NoError(t, err)

	handler := s.HandleSimpleQueryCommand(tc)

	gin.SetMode(gin.TestMode)

	s.Router.GET("/test", handler)

	server := httptest.NewServer(s.Router)
	defer server.Close()

	t.Run("test-simple-command", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// content type json
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		v := map[string]interface{}{}
		err = json.Unmarshal(body, &v)
		require.NoError(t, err)
		assert.Equal(t, float64(1), v["foo"])
		assert.Equal(t, "baz", v["bar"])
	})

}
