package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
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
	glazedSection, err := settings.NewGlazedSection()
	cobra.CheckErr(err)

	defaultSection, err := schema.NewSection(
		schema.DefaultSlug,
		"Default section",
		schema.WithFields(
			// required string test argument
			fields.New(
				"test",
				fields.TypeString,
				fields.WithShortFlag("t"),
				fields.WithHelp("Test string argument"),
				fields.WithDefault("test"),
			),
			fields.New(
				"string",
				fields.TypeString,
				fields.WithShortFlag("s"),
				fields.WithHelp("Test string flag"),
				fields.WithDefault("default"),
			),
			fields.New(
				"string_from_file",
				fields.TypeStringFromFile,
				fields.WithHelp("Test string from file flag"),
			),
			fields.New(
				"object_from_file",
				fields.TypeObjectFromFile,
				fields.WithHelp("Test object from file flag"),
			),
			fields.New(
				"integer",
				fields.TypeInteger,
				fields.WithShortFlag("i"),
				fields.WithHelp("Test integer flag"),
				fields.WithDefault(1),
			),
			fields.New(
				"float",
				fields.TypeFloat,
				fields.WithShortFlag("f"),
				fields.WithHelp("Test float flag"),
				fields.WithDefault(1.0),
			),
			fields.New(
				"bool",
				fields.TypeBool,
				fields.WithShortFlag("b"),
				fields.WithHelp("Test bool flag"),
			),
			fields.New(
				"date",
				fields.TypeDate,
				fields.WithShortFlag("d"),
				fields.WithHelp("Test date flag"),
			),
			fields.New(
				"string_list",
				fields.TypeStringList,
				fields.WithShortFlag("l"),
				fields.WithHelp("Test string list flag"),
				fields.WithDefault([]string{"default", "default2"}),
			),
			fields.New(
				"integer_list",
				fields.TypeIntegerList,
				fields.WithHelp("Test integer list flag"),
				fields.WithDefault([]int{1, 2}),
			),
			fields.New(
				"float_list",
				fields.TypeFloatList,
				fields.WithHelp("Test float list flag"),
				fields.WithDefault([]float64{1.0, 2.0}),
			),
			fields.New(
				"choice",
				fields.TypeChoice,
				fields.WithShortFlag("c"),
				fields.WithHelp("Test choice flag"),
				fields.WithChoices("choice1", "choice2"),
				fields.WithDefault("choice1"),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	description := cmds.NewCommandDescription(
		"example",
		cmds.WithShort("Short parka example command"),
		cmds.WithSections(
			defaultSection,
			glazedSection,
		),
	)
	return &ExampleCommand{
		CommandDescription: description,
	}
}

func (e *ExampleCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	d := parsedValues.DefaultSectionValues()

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
		mrp := types.MRP(f, d.Fields.GetValue(f))
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
