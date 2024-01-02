package handlers

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/helpers"
)

//go:embed text/test-data/text-handler.yaml

// JsonHandlerTest describes a test for the `json` handler. This handler
// takes a GET HTTP query and parses the url parameters to render the given
// template rows into a resulting json array.
type JsonHandlerTest struct {
	Name            string                       `yaml:"name"`
	Description     string                       `yaml:"description"`
	ParameterLayers []helpers.TestParameterLayer `yaml:"parameterLayers"`
	Template        []map[string]string          `yaml:"template"`
	ExpectedOutput  []map[string]string          `yaml:"expectedOutput"`
	ExpectedError   bool                         `yaml:"expectedError"`
}

// GlazedHandlerTest describes a test for the `glazed` handler, which can
// render the input object list templates into a variety of formats (yaml, csv, json).
// The rendered output should matched the given expected output after parsing it
// according to the specified type.
type GlazedHandlerTest struct {
	Name                     string                       `yaml:"name"`
	Description              string                       `yaml:"description"`
	ParameterLayers          []helpers.TestParameterLayer `yaml:"parameterLayers"`
	Type                     string                       `yamt:"type"`
	Template                 []map[string]string          `yaml:"template"`
	ExpectedMarshalledOutput string                       `yaml:"expectedMarshalledOutput"`
	ExpectedError            bool                         `yaml:"expectedError"`
}

// OutputFileHandlerTest describes a test sfor the `output-file`l handler, which is
// similar to the glazed handler, except that the result is returned as a file to be downloaded.
// The type of the output file is given by the passed filename.
type OutputFileHandlerTest struct {
	Name                     string                       `yaml:"name"`
	Description              string                       `yaml:"description"`
	ParameterLayers          []helpers.TestParameterLayer `yaml:"parameterLayers"`
	Filename                 string                       `yamt:"filename"`
	Template                 []map[string]string          `yaml:"template"`
	ExpectedMarshalledOutput string                       `yaml:"expectedMarshalledOutput"`
	ExpectedError            bool                         `yaml:"expectedError"`
}
