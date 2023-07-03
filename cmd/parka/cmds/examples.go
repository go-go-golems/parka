package cmds

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
)

type ExampleCommand struct {
	description *cmds.CommandDescription
}

func NewExampleCommand() *ExampleCommand {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	cobra.CheckErr(err)

	description := &cmds.CommandDescription{
		Name:  "example",
		Short: "Short parka example command",
		Long:  "",
		Flags: []*parameters.ParameterDefinition{
			// required string test argument
			{
				Name:      "test",
				ShortFlag: "t",
				Type:      parameters.ParameterTypeString,
				Help:      "Test string argument",
				Default:   "test",
			},
			{
				Name:      "string",
				ShortFlag: "s",
				Type:      parameters.ParameterTypeString,
				Help:      "Test string flag",
				Default:   "default",
				Required:  false,
			},
			{
				Name: "string_from_file",
				Type: parameters.ParameterTypeStringFromFile,
				Help: "Test string from file flag",
			},
			{
				Name: "object_from_file",
				Type: parameters.ParameterTypeObjectFromFile,
				Help: "Test object from file flag",
			},
			{
				Name:      "integer",
				ShortFlag: "i",
				Type:      parameters.ParameterTypeInteger,
				Help:      "Test integer flag",
				Default:   1,
			},
			{
				Name:      "float",
				ShortFlag: "f",
				Type:      parameters.ParameterTypeFloat,
				Help:      "Test float flag",
				Default:   1.0,
			},
			{
				Name:      "bool",
				ShortFlag: "b",
				Type:      parameters.ParameterTypeBool,
				Help:      "Test bool flag",
			},
			{
				Name:      "date",
				ShortFlag: "d",
				Type:      parameters.ParameterTypeDate,
				Help:      "Test date flag",
			},
			{
				Name:      "string_list",
				ShortFlag: "l",
				Type:      parameters.ParameterTypeStringList,
				Help:      "Test string list flag",
				Default:   []string{"default", "default2"},
			},
			{
				Name:    "integer_list",
				Type:    parameters.ParameterTypeIntegerList,
				Help:    "Test integer list flag",
				Default: []int{1, 2},
			},
			{
				Name:    "float_list",
				Type:    parameters.ParameterTypeFloatList,
				Help:    "Test float list flag",
				Default: []float64{1.0, 2.0},
			},
			{
				Name:      "choice",
				ShortFlag: "c",
				Type:      parameters.ParameterTypeChoice,
				Help:      "Test choice flag",
				Choices:   []string{"choice1", "choice2"},
				Default:   "choice1",
			},
		},
		Layers: []layers.ParameterLayer{
			glazedParameterLayer,
		},
	}
	return &ExampleCommand{
		description: description,
	}
}

func (e *ExampleCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	obj := types.NewRow(
		types.MRP("test", ps["test"]),
		types.MRP("string", ps["string"]),
		types.MRP("string_from_file", ps["string_from_file"]),
		types.MRP("object_from_file", ps["object_from_file"]),
		types.MRP("integer", ps["integer"]),

		types.MRP("float", ps["float"]),
		types.MRP("bool", ps["bool"]),
		types.MRP("date", ps["date"]),
		types.MRP("string_list", ps["string_list"]),
		types.MRP("integer_list", ps["integer_list"]),
		types.MRP("float_list", ps["float_list"]),
		types.MRP("choice", ps["choice"]),
	)
	err := gp.AddRow(ctx, obj)
	if err != nil {
		return err
	}

	err = gp.AddRow(ctx, types.NewRow(
		types.MRP("test", "test"),
		types.MRP("integer_list", []int{123, 123, 123, 123}),
		types.MRP("object_from_file", map[string]interface{}{
			"test":  "test",
			"test2": []int{123, 123, 123, 123},
		}),
	))
	if err != nil {
		return err
	}
	return nil
}

func (e *ExampleCommand) Description() *cmds.CommandDescription {
	return e.description
}

func (e *ExampleCommand) RunFromParka(
	c *gin.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	return e.Run(c, parsedLayers, ps, gp)
}
