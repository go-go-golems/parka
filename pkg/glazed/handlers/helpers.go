package handlers

import (
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed"
)

func CreateTableProcessor(pc *glazed.CommandContext, outputType string, tableType string) (*middlewares.TableProcessor, error) {
	var gp *middlewares.TableProcessor
	var err error

	glazedLayer := pc.ParsedLayers["glazed"]

	if glazedLayer != nil {
		glazedLayer.Parameters["output"] = outputType
		glazedLayer.Parameters["table"] = tableType
		gp, err = settings.SetupTableProcessor(glazedLayer.Parameters)
		if err != nil {
			return nil, err
		}
	} else {
		gp, err = settings.SetupTableProcessor(map[string]interface{}{
			"output": outputType,
			"table":  tableType,
		})
	}

	return gp, err
}
