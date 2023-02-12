package cmds

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
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
			Flags: []*cmds.ParameterDefinition{
				// required string test argument
				{
					Name:      "test",
					ShortFlag: "t",
					Type:      cmds.ParameterTypeString,
					Help:      "Test string argument",
					Required:  true,
				},
				{
					Name:      "string",
					ShortFlag: "s",
					Type:      cmds.ParameterTypeString,
					Help:      "Test string flag",
					Default:   "default",
					Required:  false,
				},
				{
					Name: "string_from_file",
					Type: cmds.ParameterTypeStringFromFile,
					Help: "Test string from file flag",
				},
				{
					Name: "object_from_file",
					Type: cmds.ParameterTypeObjectFromFile,
					Help: "Test object from file flag",
				},
				{
					Name:      "integer",
					ShortFlag: "i",
					Type:      cmds.ParameterTypeInteger,
					Help:      "Test integer flag",
					Default:   1,
				},
				{
					Name:      "float",
					ShortFlag: "f",
					Type:      cmds.ParameterTypeFloat,
					Help:      "Test float flag",
					Default:   1.0,
				},
				{
					Name:      "bool",
					ShortFlag: "b",
					Type:      cmds.ParameterTypeBool,
					Help:      "Test bool flag",
				},
				{
					Name:      "date",
					ShortFlag: "d",
					Type:      cmds.ParameterTypeDate,
					Help:      "Test date flag",
				},
				{
					Name:      "string_list",
					ShortFlag: "l",
					Type:      cmds.ParameterTypeStringList,
					Help:      "Test string list flag",
					Default:   []string{"default", "default2"},
				},
				{
					Name:    "integer_list",
					Type:    cmds.ParameterTypeIntegerList,
					Help:    "Test integer list flag",
					Default: []int{1, 2},
				},
				{
					Name:    "float_list",
					Type:    cmds.ParameterTypeFloatList,
					Help:    "Test float list flag",
					Default: []float64{1.0, 2.0},
				},
				{
					Name:      "choice",
					ShortFlag: "c",
					Type:      cmds.ParameterTypeChoice,
					Help:      "Test choice flag",
					Choices:   []string{"choice1", "choice2"},
					Default:   "choice1",
				},
			},
		},
	}
}

func (e *ExampleCommand) Run(parameters map[string]interface{}, gp *cmds.GlazeProcessor) error {
	obj := map[string]interface{}{
		"test":             parameters["test"],
		"string":           parameters["string"],
		"string_from_file": parameters["string_from_file"],
		"object_from_file": parameters["object_from_file"],
		"integer":          parameters["integer"],
		"float":            parameters["float"],
		"bool":             parameters["bool"],
		"date":             parameters["date"],
		"string_list":      parameters["string_list"],
		"integer_list":     parameters["integer_list"],
		"float_list":       parameters["float_list"],
		"choice":           parameters["choice"],
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
	return nil
}

func (e *ExampleCommand) Description() *cmds.CommandDescription {
	return e.description
}

func (e *ExampleCommand) RunFromParka(_ *gin.Context, parameters map[string]interface{}, gp *cmds.GlazeProcessor) error {
	return e.Run(parameters, gp)
}
