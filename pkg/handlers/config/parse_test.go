package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestEvaluateEnvSimpleString(t *testing.T) {
	node := "test"
	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, "test", evaluated)
}

func TestEvaluateEnvSimpleMap(t *testing.T) {
	node := map[string]interface{}{
		"test": "test",
		"foo":  "bar",
	}
	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"test": "test",
		"foo":  "bar",
	}, evaluated)
}

func TestEvaluateEnvNestedMap(t *testing.T) {
	node := map[string]interface{}{
		"test": "test",
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}
	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"test": "test",
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}, evaluated)
}

func TestEvaluateEnvSimpleStringList(t *testing.T) {
	node := []interface{}{
		"test",
		"foo",
	}
	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, []interface{}{
		"test",
		"foo",
	}, evaluated)
}

func TestEvaluateEnvString(t *testing.T) {
	node := map[string]interface{}{
		"_env": "VARIABLE",
	}
	err := os.Setenv("VARIABLE", "test")
	require.Nil(t, err)

	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, "test", evaluated)
}

func TestEvaluateEnvMap(t *testing.T) {
	node := map[string]interface{}{
		"test": map[string]interface{}{
			"_env": "VARIABLE",
		},
		"foo": map[string]interface{}{
			"_env": "VARIABLE2",
		},
		"bar": "baz",
	}
	err := os.Setenv("VARIABLE", "test")
	require.Nil(t, err)
	err = os.Setenv("VARIABLE2", "test2")
	require.Nil(t, err)

	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"test": "test",
		"foo":  "test2",
		"bar":  "baz",
	}, evaluated)
}

func TestEvaluateEnvStringList(t *testing.T) {
	node := []interface{}{
		map[string]interface{}{
			"_env": "VARIABLE",
		},
		map[string]interface{}{
			"_env": "VARIABLE2",
		},
		"bar",
	}
	err := os.Setenv("VARIABLE", "test")
	require.Nil(t, err)
	err = os.Setenv("VARIABLE2", "test2")
	require.Nil(t, err)

	evaluated, err := EvaluateConfigEntry(node)
	require.Nil(t, err)
	assert.Equal(t, []interface{}{
		"test",
		"test2",
		"bar",
	}, evaluated)
}
