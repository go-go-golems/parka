package config

import (
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// ParameterFilterList are used to configure whitelists and blacklists.
// Entire layers as well as individual flags and arguments can be whitelisted or blacklisted.
// Params is used for the default layer.
type ParameterFilterList struct {
	Layers          []string            `yaml:"layers,omitempty"`
	LayerParameters map[string][]string `yaml:"layerParameters,omitempty"`
	Parameters      []string            `yaml:"parameters,omitempty"`
}

func (p *ParameterFilterList) GetAllLayerParameters() map[string][]string {
	ret := map[string][]string{}
	for layer, params := range p.LayerParameters {
		ret[layer] = params
	}
	if _, ok := ret[layers.DefaultSlug]; !ok {
		ret[layers.DefaultSlug] = []string{}
	}
	ret[layers.DefaultSlug] = append(ret[layers.DefaultSlug], p.Parameters...)
	return ret
}

type LayerParameters struct {
	Layers     map[string]map[string]interface{} `yaml:"layers,omitempty"`
	Parameters map[string]interface{}            `yaml:"parameters,omitempty"`
}

func NewLayerParameters() *LayerParameters {
	return &LayerParameters{
		Layers:     map[string]map[string]interface{}{},
		Parameters: map[string]interface{}{},
	}
}

// Merge merges the two LayerParameters, with the overrides taking precedence.
// It merges all the layers, flags, and arguments. For each layer, the layer flags are merged as well,
// overrides taking precedence.
func (p *LayerParameters) Merge(overrides *LayerParameters) {
	for k, v := range overrides.Layers {
		if _, ok := p.Layers[k]; !ok {
			p.Layers[k] = map[string]interface{}{}
		}
		for k2, v2 := range v {
			p.Layers[k][k2] = v2
		}
	}

	for k, v := range overrides.Parameters {
		p.Parameters[k] = v
	}
}

func (p *LayerParameters) Clone() *LayerParameters {
	ret := NewLayerParameters()
	ret.Merge(p)
	return ret
}

func (p *LayerParameters) GetParameterMap() map[string]map[string]interface{} {
	r := p.Clone()
	ret := r.Layers
	if _, ok := ret[layers.DefaultSlug]; !ok {
		ret[layers.DefaultSlug] = map[string]interface{}{}
	}
	for k, v := range r.Parameters {
		ret[layers.DefaultSlug][k] = v
	}

	return ret
}

type ParameterFilter struct {
	Overrides *LayerParameters
	Defaults  *LayerParameters
	Whitelist *ParameterFilterList
	Blacklist *ParameterFilterList
}

type ParameterFilterOption func(*ParameterFilter)

// Override options
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

func WithOverrideParameter(name string, value interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		handler.Overrides.Parameters[name] = value
	}
}

func WithOverrideParameters(params map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		for k, v := range params {
			handler.Overrides.Parameters[k] = v
		}
	}
}

func WithMergeOverrideLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		if _, ok := handler.Overrides.Layers[name]; !ok {
			handler.Overrides.Layers[name] = map[string]interface{}{}
		}
		for k, v := range layer {
			handler.Overrides.Layers[name][k] = v
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

func WithOverrideLayers(layers map[string]map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Overrides == nil {
			handler.Overrides = NewLayerParameters()
		}
		for name, layer := range layers {
			handler.Overrides.Layers[name] = layer
		}
	}
}

// Default options
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

func WithDefaultParameter(name string, value interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		handler.Defaults.Parameters[name] = value
	}
}

func WithDefaultParameters(params map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		for k, v := range params {
			handler.Defaults.Parameters[k] = v
		}
	}
}

func WithMergeDefaultLayer(name string, layer map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		if _, ok := handler.Defaults.Layers[name]; !ok {
			handler.Defaults.Layers[name] = map[string]interface{}{}
		}
		for k, v := range layer {
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

func WithDefaultLayers(layers map[string]map[string]interface{}) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Defaults == nil {
			handler.Defaults = NewLayerParameters()
		}
		for name, layer := range layers {
			handler.Defaults.Layers[name] = layer
		}
	}
}

// Whitelist options
func WithWhitelist(whitelist *ParameterFilterList) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		handler.Whitelist = whitelist
	}
}

func WithWhitelistParameters(params ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Whitelist == nil {
			handler.Whitelist = &ParameterFilterList{}
		}
		handler.Whitelist.Parameters = append(handler.Whitelist.Parameters, params...)
	}
}

func WithWhitelistLayers(layers ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Whitelist == nil {
			handler.Whitelist = &ParameterFilterList{}
		}
		handler.Whitelist.Layers = append(handler.Whitelist.Layers, layers...)
	}
}

func WithWhitelistLayerParameters(layer string, params ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Whitelist == nil {
			handler.Whitelist = &ParameterFilterList{}
		}
		if handler.Whitelist.LayerParameters == nil {
			handler.Whitelist.LayerParameters = map[string][]string{}
		}
		handler.Whitelist.LayerParameters[layer] = append(handler.Whitelist.LayerParameters[layer], params...)
	}
}

// Blacklist options
func WithBlacklist(blacklist *ParameterFilterList) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		handler.Blacklist = blacklist
	}
}

func WithBlacklistParameters(params ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Blacklist == nil {
			handler.Blacklist = &ParameterFilterList{}
		}
		handler.Blacklist.Parameters = append(handler.Blacklist.Parameters, params...)
	}
}

func WithBlacklistLayers(layers ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Blacklist == nil {
			handler.Blacklist = &ParameterFilterList{}
		}
		handler.Blacklist.Layers = append(handler.Blacklist.Layers, layers...)
	}
}

func WithBlacklistLayerParameters(layer string, params ...string) ParameterFilterOption {
	return func(handler *ParameterFilter) {
		if handler.Blacklist == nil {
			handler.Blacklist = &ParameterFilterList{}
		}
		if handler.Blacklist.LayerParameters == nil {
			handler.Blacklist.LayerParameters = map[string][]string{}
		}
		handler.Blacklist.LayerParameters[layer] = append(handler.Blacklist.LayerParameters[layer], params...)
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

	// in reverse order of applications. This means that ultimately, the defaults are run first,
	// then overrides, then whitelist, then blacklist, and then finally the query handlers.

	if od.Blacklist != nil {
		ret = append(ret, middlewares.BlacklistLayers(od.Blacklist.Layers))
		ret = append(ret, middlewares.BlacklistLayerParameters(od.Blacklist.GetAllLayerParameters()))
	}

	if od.Whitelist != nil {
		ret = append(ret, middlewares.WhitelistLayers(od.Whitelist.Layers))
		ret = append(ret, middlewares.WhitelistLayerParameters(od.Whitelist.GetAllLayerParameters()))
	}

	if od.Overrides != nil {
		// TODO(manuel, 2024-05-14) Here we would ideally parse potential strings that map to non strings (for example when using _env: SQLETON_PORT where the result is a string, not an int)
		// Currently, we migrated this to UpdateFromMap but it's not a great look
		ret = append(ret, middlewares.UpdateFromMap(
			od.Overrides.GetParameterMap(),
			parameters.WithParseStepSource("overrides")),
		)
	}

	if od.Defaults != nil {
		// this needs to override the defaults set by the underlying handler...
		ret = append(ret, middlewares.UpdateFromMapAsDefaultFirst(od.Defaults.GetParameterMap(), parameters.WithParseStepSource("defaults")))
	}

	return ret
}
