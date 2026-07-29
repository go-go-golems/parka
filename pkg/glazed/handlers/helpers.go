package handlers

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

func CreateTableProcessorWithOutput(parsedValues *values.Values, outputType string, tableFormat string) (*middlewares.TableProcessor, error) {
	glazedLayer, ok := parsedValues.Get(settings.StructuredOutputSlug)
	if !ok {
		return middlewares.NewTableProcessor(), nil
	}

	// Override the structured-output format for this handler (e.g. force JSON).
	// The table-format flag was removed in the structured-output cleanup; the
	// table formatter now uses "ascii" unconditionally.
	if outputType != "" {
		_, err := glazedLayer.Fields.UpdateExistingValue(
			"format", outputType,
			fields.WithSource("parka-handlers"),
		)
		if err != nil {
			return nil, err
		}
	}
	gp, _, err := settings.SetupStructuredProcessor(glazedLayer)
	if err != nil {
		return nil, err
	}

	return gp, nil
}
