package handlers

import (
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
)

func CreateTableProcessorWithOutput(pc *glazed.CommandContext, outputType string, tableFormat string) (*middlewares.TableProcessor, error) {
	glazedLayer, ok := pc.ParsedLayers.Get("glazed")
	if !ok {
		return middlewares.NewTableProcessor(), nil
	}

	glazedLayer.Parameters.UpdateExistingValue("output", "parka-handlers", outputType)
	glazedLayer.Parameters.UpdateExistingValue("table-format", "parka-handlers", tableFormat)
	gp, err := settings.SetupTableProcessor(glazedLayer)
	if err != nil {
		return nil, err
	}

	return gp, nil
}
