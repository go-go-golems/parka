package handlers

import (
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

func CreateTableProcessorWithOutput(parsedLayers *layers.ParsedLayers, outputType string, tableFormat string) (*middlewares.TableProcessor, error) {
	glazedLayer, ok := parsedLayers.Get("glazed")
	if !ok {
		return middlewares.NewTableProcessor(), nil
	}

	glazedLayer.Parameters.UpdateExistingValue(
		"output", outputType,
		parameters.WithParseStepSource("parka-handlers"),
	)
	glazedLayer.Parameters.UpdateExistingValue("table-format", tableFormat,
		parameters.WithParseStepSource("parka-handlers"),
	)
	gp, err := settings.SetupTableProcessor(glazedLayer)
	if err != nil {
		return nil, err
	}

	return gp, nil
}
