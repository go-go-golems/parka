package config

import (
	"fmt"
	"os"
)

// evaluateEnv takes a node which can be either a string, or a map with a key `env` and a string value.
// If the node is a string, it is returned as is.
// If the node is a map, it is evaluated recursively. If the map has a single key named `_env`, the entire
// map is replaced with the environment value corresponding to `_env`.
// If it is a list, it is evaluated recursively.
func evaluateEnv(node interface{}) (interface{}, error) {
	switch value := node.(type) {
	case map[string]interface{}:
		if len(value) == 1 && value["_env"] != nil {
			if envVar, ok := value["_env"]; ok {
				envVal, ok := envVar.(string)
				if !ok {
					return nil, fmt.Errorf("'_env' key must have a string value")
				}
				return os.Getenv(envVal), nil
			}
		}

		evaluated := make(map[string]interface{}, len(value))
		for k, v := range value {
			evalVal, err := evaluateEnv(v)
			if err != nil {
				return nil, err
			}
			evaluated[k] = evalVal
		}
		return evaluated, nil
	case []interface{}:
		evaluated := make([]interface{}, len(value))
		for i, v := range value {
			evalVal, err := evaluateEnv(v)
			if err != nil {
				return nil, err
			}
			evaluated[i] = evalVal
		}
		return evaluated, nil
	default:
		return value, nil
	}
}

// evaluateLayerParams goes over the layer params and evaluates the environment variables.
func evaluateLayerParams(params *LayerParams) (*LayerParams, error) {
	ret := &LayerParams{}
	evaluatedLayers, err := evaluateEnv(params.Layers)
	if err != nil {
		return nil, err
	}
	ret.Layers = evaluatedLayers.(map[string]map[string]interface{})

	evaluatedFlags, err := evaluateEnv(params.Flags)
	if err != nil {
		return nil, err
	}
	ret.Flags = evaluatedFlags.(map[string]interface{})

	evaluatedArguments, err := evaluateEnv(params.Arguments)
	if err != nil {
		return nil, err
	}
	ret.Arguments = evaluatedArguments.(map[string]interface{})

	return ret, nil
}
