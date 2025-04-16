package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	json2 "github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	"github.com/go-go-golems/parka/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunGlazedCommand(t *testing.T) {
	tc, err := utils.NewTestGlazedCommand()
	require.NoError(t, err)

	s, err := NewServer()
	require.NoError(t, err)

	handler := json2.CreateJSONQueryHandler(tc)

	s.Group.GET("/test", handler)

	server := httptest.NewServer(s.router)
	defer server.Close()

	t.Run("test-simple-command", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/test")
		require.NoError(t, err)
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		// content type json
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		v := []map[string]interface{}{}
		err = json.Unmarshal(body, &v)
		require.NoError(t, err)

		require.Len(t, v, 3)
		v_ := v[0]
		assert.Equal(t, float64(0), v_["test"])
		assert.Equal(t, "test-0", v_["test2"])
		assert.Equal(t, "test3-0", v_["test3"])
	})
}
