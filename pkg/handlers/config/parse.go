package config

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

type Evaluator interface {
	// Evaluate takes a node and evaluates it.
	// If it succeeds, it returns the evaluated node, and true.
	// If it fails, it returns nil, false, and an error.
	// This would mean passing the value to the next evaluator down the chain.
	Evaluate(node interface{}) (interface{}, bool, error)
}

var evaluators = []Evaluator{}

func init() {
	evaluators = append(evaluators, &EnvEvaluator{})
	evaluator, err := NewSsmEvaluator(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize SSM evaluator")
	}
	if evaluator != nil {
		evaluators = append(evaluators, evaluator)
	}
}

type EnvEvaluator struct{}

func (e *EnvEvaluator) Evaluate(node interface{}) (interface{}, bool, error) {
	switch value := node.(type) {
	case map[string]interface{}:
		if len(value) == 1 && value["_env"] != nil {
			if envVar, ok := value["_env"]; ok {
				envVal, ok := envVar.(string)
				if !ok {
					return nil, false, fmt.Errorf("'_env' key must have a string value")
				}
				return os.Getenv(envVal), true, nil
			}
		}

		return nil, false, nil
	default:
		return nil, false, nil
	}
}

// EvaluateConfigEntry takes a node which can be either a string, or a map with a key `env` and a string value.
// If the node is a string, it is returned as is.
// If the node is a map, it is evaluated recursively. If the map has a single key named `_env`, the entire
// map is replaced with the environment value corresponding to `_env`.
// If it is a list, it is evaluated recursively.
func EvaluateConfigEntry(node interface{}) (interface{}, error) {
	switch value := node.(type) {
	case map[string]interface{}:
		for _, evaluator := range evaluators {
			evaluated, ok, err := evaluator.Evaluate(value)
			if err != nil {
				return nil, err
			}
			if ok {
				return evaluated, nil
			}
		}

		evaluated := make(map[string]interface{}, len(value))
		for k, v := range value {
			evalVal, err := EvaluateConfigEntry(v)
			if err != nil {
				return nil, err
			}
			evaluated[k] = evalVal
		}
		return evaluated, nil
	case []interface{}:
		evaluated := make([]interface{}, len(value))
		for i, v := range value {
			evalVal, err := EvaluateConfigEntry(v)
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
func evaluateLayerParams(params *LayerParameters) (*LayerParameters, error) {
	ret := &LayerParameters{
		Parameters: map[string]interface{}{},
		Layers:     map[string]map[string]interface{}{},
	}
	for slug, layer := range params.Layers {
		ret.Layers[slug] = map[string]interface{}{}
		for k, v := range layer {
			v_, err := EvaluateConfigEntry(v)
			if err != nil {
				return nil, err
			}
			ret.Layers[slug][k] = v_
		}
	}

	for name, v := range params.Parameters {
		v_, err := EvaluateConfigEntry(v)
		if err != nil {
			return nil, err
		}
		ret.Parameters[name] = v_
	}

	return ret, nil
}
