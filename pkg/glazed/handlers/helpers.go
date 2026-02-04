package handlers

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

func CreateTableProcessorWithOutput(parsedValues *values.Values, outputType string, tableFormat string) (*middlewares.TableProcessor, error) {
	glazedLayer, ok := parsedValues.Get(settings.GlazedSlug)
	if !ok {
		return middlewares.NewTableProcessor(), nil
	}

	_, err := glazedLayer.Fields.UpdateExistingValue(
		"output", outputType,
		fields.WithSource("parka-handlers"),
	)
	if err != nil {
		return nil, err
	}
	_, err = glazedLayer.Fields.UpdateExistingValue("table-format", tableFormat,
		fields.WithSource("parka-handlers"),
	)
	if err != nil {
		return nil, err
	}
	gp, err := settings.SetupTableProcessor(glazedLayer)
	if err != nil {
		return nil, err
	}

	return gp, nil
}
