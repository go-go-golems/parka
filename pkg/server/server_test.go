package server_test

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	json2 "github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestCommand struct{}

var _ cmds.GlazeCommand = &TestCommand{}

func (t *TestCommand) Description() *cmds.CommandDescription {
	return cmds.NewCommandDescription("test")
}

func (t *TestCommand) ToYAML(w io.Writer) error {
	return t.Description().ToYAML(w)
}

func (t *TestCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
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

	handler := json2.CreateJSONQueryHandler(tc)

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
		v := []map[string]interface{}{}
		err = json.Unmarshal(body, &v)
		require.NoError(t, err)

		require.Len(t, v, 1)
		v_ := v[0]
		assert.Equal(t, float64(1), v_["foo"])
		assert.Equal(t, "baz", v_["bar"])
	})
}
