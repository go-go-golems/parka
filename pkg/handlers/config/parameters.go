package config

import (
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
)

type HandlerParameters struct {
	Layers    map[string]map[string]interface{}
	Flags     map[string]interface{}
	Arguments map[string]interface{}
}

func NewHandlerParameters() *HandlerParameters {
	return &HandlerParameters{
		Layers:    map[string]map[string]interface{}{},
		Flags:     map[string]interface{}{},
		Arguments: map[string]interface{}{},
	}
}

// NewHandlerParametersFromLayerParams creates a new HandlerParameters from the config file.
// It currently requires a list of layerDefinitions in order to lookup the correct
// layers to stored as ParsedParameterLayer. It doesn't fail if configured layers don't exist.
//
// TODO(manuel, 2023-05-31) Add a way to validate the fact that overrides in a config file might
// have a typo and don't correspond to existing layer definitions in the application.
func NewHandlerParametersFromLayerParams(p *LayerParams) {
	ret := NewHandlerParameters()
	for name, l := range p.Layers {
		ret.Layers[name] = map[string]interface{}{}
		for k, v := range l {
			ret.Layers[name][k] = v
		}
	}

	for name, v := range p.Flags {
		ret.Flags[name] = v
	}

	for name, v := range p.Arguments {
		ret.Arguments[name] = v
	}
}

// Merge merges the given overrides into this one.
// If a layer is already present, it is merged with the given one.
// Flags and arguments are merged, overrides taking precedence.
func (ho *HandlerParameters) Merge(other *HandlerParameters) {
	for k, v := range other.Layers {
		if _, ok := ho.Layers[k]; !ok {
			ho.Layers[k] = map[string]interface{}{}
		}
		for k2, v2 := range v {
			ho.Layers[k][k2] = v2
		}
	}
	for k, v := range other.Flags {
		ho.Flags[k] = v
	}
	for k, v := range other.Arguments {
		ho.Arguments[k] = v
	}
}

type OverridesAndDefaults struct {
	Overrides *HandlerParameters
	Defaults  *HandlerParameters
}

type OverridesAndDefaultsOption func(*OverridesAndDefaults)

func WithReplaceOverrides(overrides *HandlerParameters) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		handler.Overrides = overrides
	}
}

func WithMergeOverrides(overrides *HandlerParameters) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = overrides
		} else {
			handler.Overrides.Merge(overrides)
		}
	}
}

func WithOverrideFlag(name string, value string) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Flags[name] = value
	}
}

func WithOverrideArgument(name string, value string) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Arguments[name] = value
	}
}

func WithMergeOverrideLayer(name string, layer map[string]interface{}) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Overrides.Layers[name]; !ok {
				handler.Overrides.Layers[name] = map[string]interface{}{}
			}
			handler.Overrides.Layers[name][k] = v
		}
	}
}

// WithLayerDefaults populates the defaults for the given layer. If a value is already set, the value is skipped.
func WithLayerDefaults(name string, layer map[string]interface{}) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Overrides.Layers[name]; !ok {
				handler.Overrides.Layers[name] = map[string]interface{}{}
			}
			if _, ok := handler.Overrides.Layers[name][k]; !ok {
				handler.Overrides.Layers[name][k] = v
			}
		}
	}
}

func WithReplaceOverrideLayer(name string, layer map[string]interface{}) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Overrides == nil {
			handler.Overrides = NewHandlerParameters()
		}
		handler.Overrides.Layers[name] = layer
	}
}

// TODO(manuel, 2023-05-25) We can't currently override defaults, since they are parsed up front.
// For that we would need https://github.com/go-go-golems/glazed/issues/239
// So for now, we only deal with overrides.
//
// Handling all the way to configure defaults.

func WithReplaceDefaults(defaults *HandlerParameters) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		handler.Defaults = defaults
	}
}

func WithMergeDefaults(defaults *HandlerParameters) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Defaults == nil {
			handler.Defaults = defaults
		} else {
			handler.Defaults.Merge(defaults)
		}
	}
}

func WithDefaultFlag(name string, value string) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Flags[name] = value
	}
}

func WithDefaultArgument(name string, value string) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Arguments[name] = value
	}
}

func WithMergeDefaultLayer(name string, layer map[string]interface{}) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Defaults.Layers[name]; !ok {
				handler.Defaults.Layers[name] = map[string]interface{}{}
			}
			handler.Defaults.Layers[name][k] = v
		}
	}
}

func WithReplaceDefaultLayer(name string, layer map[string]interface{}) OverridesAndDefaultsOption {
	return func(handler *OverridesAndDefaults) {
		if handler.Defaults == nil {
			handler.Defaults = NewHandlerParameters()
		}
		handler.Defaults.Layers[name] = layer
	}
}

func (od *OverridesAndDefaults) ComputeParserOptions(stream bool) []parser.ParserOption {
	parserOptions := []parser.ParserOption{}

	if stream {
		// if the config file says to use stream (which is the default), override the stream glazed flag,
		// which will make it prefer the row output when possible
		parserOptions = append(parserOptions,
			parser.WithAppendOverrides("glazed", map[string]interface{}{
				"stream": true,
			}))
	}

	// TODO(manuel, 2023-06-21) This needs to be handled for each backend, not just the HTML one
	if od.Overrides != nil {
		parserOptions = append(parserOptions,
			parser.WithAppendOverrides(layers.DefaultSlug, od.Overrides.Flags),
		)
		parserOptions = append(parserOptions,
			parser.WithAppendOverrides(layers.DefaultSlug, od.Overrides.Arguments),
		)
		for slug, layer := range od.Overrides.Layers {
			parserOptions = append(parserOptions, parser.WithAppendOverrides(slug, layer))
		}
	}

	if od.Defaults != nil {
		parserOptions = append(parserOptions,
			parser.WithPrependDefaults(layers.DefaultSlug, od.Defaults.Flags),
		)
		parserOptions = append(parserOptions,
			parser.WithPrependDefaults(layers.DefaultSlug, od.Defaults.Arguments),
		)
		for slug, layer := range od.Defaults.Layers {
			// we use prepend because that way, later options will actually override earlier flag values,
			// since they will be applied earlier.
			parserOptions = append(parserOptions, parser.WithPrependDefaults(slug, layer))
		}
	}

	return parserOptions
}
