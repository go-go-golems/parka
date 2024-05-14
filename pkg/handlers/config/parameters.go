package config

import (
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

type ParameterFilter struct {
	Overrides *LayerParameters
	Defaults  *LayerParameters
	Whitelist *ParameterFilterList
	Blacklist *ParameterFilterList
}

type ParameterFilterOption func(*ParameterFilter)

func WithReplaceOverrides(overrides *LayerParameters) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		handler.Overrides = overrides
	}
}

func WithMergeOverrides(overrides *LayerParameters) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = overrides
		} else {
			handler.Overrides.Merge(overrides)
		}
	}
}

func WithOverrideParameter(name string, value string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		handler.Overrides.Parameters[name] = value
	}
}

func WithMergeOverrideLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
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
func WithLayerDefaults(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
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

func WithReplaceOverrideLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		handler.Overrides.Layers[name] = layer
	}
}

// TODO(manuel, 2023-05-25) We can't currently override defaults, since they are parsed up front.
// For that we would need https://github.com/go-go-golems/glazed/issues/239
// So for now, we only deal with overrides.
//
// Handling all the way to configure defaults.

func WithReplaceDefaults(defaults *LayerParameters) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		handler.Defaults = defaults
	}
}

func WithMergeDefaults(defaults *LayerParameters) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = defaults
		} else {
			handler.Defaults.Merge(defaults)
		}
	}
}

func WithDefaultParameter(name string, value string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		handler.Defaults.Parameters[name] = value
	}
}

func WithMergeDefaultLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		for k, v := range layer {
			if _, ok := handler.Defaults.Layers[name]; !ok {
				handler.Defaults.Layers[name] = map[string]interface{}{}
			}
			handler.Defaults.Layers[name][k] = v
		}
	}
}

func WithReplaceDefaultLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		handler.Defaults.Layers[name] = layer
	}
}

func NewParameterFilter(options ...ParameterFilterOption) *ParameterFilter {
	ret := &ParameterFilter{}
	for _, opt := range options {
		opt(ret)
	}
	return ret
}

func (od *ParameterFilter) ComputeMiddlewares(stream bool) []middlewares.Middleware {
	ret := []middlewares.Middleware{}

	if od.Defaults != nil {
		// this needs to override the defaults set by the underlying handler...
		ret = append(ret, middlewares.UpdateFromMapAsDefaultFirst(od.Defaults.GetParameterMap(), parameters.WithParseStepSource("defaults")))
	}

	if od.Overrides != nil {
		// TODO(manuel, 2024-05-14) Here we would ideally parse potential strings that map to non strings (for example when using _env: SQLETON_PORT where the result is a string, not an int)
		// Currently, we migrated this to UpdateFromMap but it's not a great look
		ret = append(ret, middlewares.UpdateFromMap(
			od.Overrides.GetParameterMap(),
			parameters.WithParseStepSource("overrides")),
		)
	}

	if od.Whitelist != nil {
		ret = append(ret, middlewares.WhitelistLayers(od.Whitelist.Layers))
		ret = append(ret, middlewares.WhitelistLayerParameters(od.Whitelist.GetAllLayerParameters()))
	}

	if od.Blacklist != nil {
		ret = append(ret, middlewares.BlacklistLayers(od.Blacklist.Layers))
		ret = append(ret, middlewares.BlacklistLayerParameters(od.Blacklist.GetAllLayerParameters()))
	}

	return ret
}
