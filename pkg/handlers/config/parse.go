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
	ret := &LayerParams{
		Flags:     map[string]interface{}{},
		Arguments: map[string]interface{}{},
		Layers:    map[string]map[string]interface{}{},
	}
	for slug, layer := range params.Layers {
		ret.Layers[slug] = map[string]interface{}{}
		for k, v := range layer {
			v_, err := evaluateEnv(v)
			if err != nil {
				return nil, err
			}
			ret.Layers[slug][k] = v_
		}
	}

	for name, v := range params.Flags {
		v_, err := evaluateEnv(v)
		if err != nil {
			return nil, err
		}
		ret.Flags[name] = v_
	}

	for name, v := range params.Arguments {
		v_, err := evaluateEnv(v)
		if err != nil {
			return nil, err
		}
		ret.Arguments[name] = v_
	}

	return ret, nil
}
