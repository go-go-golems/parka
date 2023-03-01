package cmds

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

type ExampleCommand struct {
	description *cmds.CommandDescription
}

func NewExampleCommand() *ExampleCommand {
	return &ExampleCommand{
		description: &cmds.CommandDescription{
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
					Required:  true,
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
		},
	}
}

func (e *ExampleCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	obj := map[string]interface{}{
		"test":             ps["test"],
		"string":           ps["string"],
		"string_from_file": ps["string_from_file"],
		"object_from_file": ps["object_from_file"],
		"integer":          ps["integer"],
		"float":            ps["float"],
		"bool":             ps["bool"],
		"date":             ps["date"],
		"string_list":      ps["string_list"],
		"integer_list":     ps["integer_list"],
		"float_list":       ps["float_list"],
		"choice":           ps["choice"],
	}
	err := gp.ProcessInputObject(obj)
	if err != nil {
		return err
	}

	err = gp.ProcessInputObject(map[string]interface{}{
		"test":  "test",
		"test2": []int{123, 123, 123, 123},
		"test3": map[string]interface{}{
			"test":  "test",
			"test2": []int{123, 123, 123, 123},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (e *ExampleCommand) Description() *cmds.CommandDescription {
	return e.description
}

func (e *ExampleCommand) RunFromParka(c *gin.Context, parsedLayers map[string]*layers.ParsedParameterLayer, ps map[string]interface{}, gp *cmds.GlazeProcessor) error {
	return e.Run(c, parsedLayers, ps, gp)
}
