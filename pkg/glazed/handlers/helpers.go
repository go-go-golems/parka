package handlers

import (
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
)

func CreateTableProcessorWithOutput(pc *glazed.CommandContext, outputType string, tableFormat string) (*middlewares.TableProcessor, error) {
	var gp *middlewares.TableProcessor
	var err error

	glazedLayer := pc.ParsedLayers["glazed"]

	if glazedLayer != nil {
		glazedLayer.Parameters["output"] = outputType
		glazedLayer.Parameters["table"] = tableFormat
		gp, err = settings.SetupTableProcessor(glazedLayer.Parameters)
		if err != nil {
			return nil, err
		}
	} else {
		gp, err = settings.SetupTableProcessor(map[string]interface{}{
			"output":       outputType,
			"table-format": tableFormat,
		})
	}

	return gp, err
}

func CreateTableProcessor(pc *glazed.CommandContext) (*middlewares.TableProcessor, error) {
	var gp *middlewares.TableProcessor
	var err error

	glazedLayer := pc.ParsedLayers["glazed"]

	if glazedLayer != nil {
		gp, err = settings.SetupTableProcessor(glazedLayer.Parameters)
		if err != nil {
			return nil, err
		}
	} else {
		gp, err = settings.SetupTableProcessor(map[string]interface{}{})
	}

	return gp, err
}
