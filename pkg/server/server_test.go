package server_test

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/server"
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
	gp middlewares.Processor,
) error {
	err := gp.AddRow(ctx, types.NewRow(
		types.MRP("foo", 1),
		types.MRP("bar", "baz"),
	))

	if err != nil {
		return err
	}

	return nil
}

func TestRunGlazedCommand(t *testing.T) {
	tc := &TestCommand{}

	s, err := server.NewServer()
	require.NoError(t, err)

	handler := s.HandleSimpleQueryCommand(tc)

	gin.SetMode(gin.TestMode)

	s.Router.GET("/test", handler)

	server := httptest.NewServer(s.Router)
	defer server.Close()

	t.Run("test-simple-command", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/test")
		require.NoError(t, err)
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

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
