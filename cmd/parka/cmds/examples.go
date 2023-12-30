package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
)

type ExampleCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &ExampleCommand{}

func NewExampleCommand() *ExampleCommand {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	cobra.CheckErr(err)

	defaultLayer, err := layers.NewParameterLayer(
		"default",
		"Default layer",
		layers.WithParameterDefinitions(
			[]*parameters.ParameterDefinition{
				// required string test argument
				{
					Name:      "test",
					ShortFlag: "t",
					Type:      parameters.ParameterTypeString,
					Help:      "Test string argument",
					Default:   cast.InterfaceAddr("test"),
				},
				{
					Name:      "string",
					ShortFlag: "s",
					Type:      parameters.ParameterTypeString,
					Help:      "Test string flag",
					Default:   cast.InterfaceAddr("default"),
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
					Default:   cast.InterfaceAddr(1),
				},
				{
					Name:      "float",
					ShortFlag: "f",
					Type:      parameters.ParameterTypeFloat,
					Help:      "Test float flag",
					Default:   cast.InterfaceAddr(1.0),
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
					Default:   cast.InterfaceAddr([]string{"default", "default2"}),
				},
				{
					Name:    "integer_list",
					Type:    parameters.ParameterTypeIntegerList,
					Help:    "Test integer list flag",
					Default: cast.InterfaceAddr([]int{1, 2}),
				},
				{
					Name:    "float_list",
					Type:    parameters.ParameterTypeFloatList,
					Help:    "Test float list flag",
					Default: cast.InterfaceAddr([]float64{1.0, 2.0}),
				},
				{
					Name:      "choice",
					ShortFlag: "c",
					Type:      parameters.ParameterTypeChoice,
					Help:      "Test choice flag",
					Choices:   []string{"choice1", "choice2"},
					Default:   cast.InterfaceAddr("choice1"),
				},
			}...))
	if err != nil {
		panic(err)
	}

	description := &cmds.CommandDescription{
		Name:  "example",
		Short: "Short parka example command",
		Long:  "",
		Layers: layers.NewParameterLayers(layers.WithLayers(
			defaultLayer,
			glazedParameterLayer,
		)),
	}
	return &ExampleCommand{
		CommandDescription: description,
	}
}

func (e *ExampleCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	d := parsedLayers.GetDefaultParameterLayer()

	fields := []string{
		"test",
		"string",
		"string_from_file",
		"object_from_file",
		"integer",
		"float",
		"bool",
		"date",
		"string_list",
		"integer_list",
		"float_list",
		"choice",
	}
	mrps := []types.MapRowPair{}
	for _, f := range fields {
		mrp := types.MRP(f, d.Parameters.GetValue(f))
		mrps = append(mrps, mrp)
	}

	obj := types.NewRow(mrps...)
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
