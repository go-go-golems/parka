package utils

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"io"
)

type TestGlazedCommand struct {
	description *cmds.CommandDescription
}

func NewTestGlazedCommand(options ...cmds.CommandDescriptionOption) (*TestGlazedCommand, error) {
	glazedLayer, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}

	options = append(options, cmds.WithSections(glazedLayer))

	description := cmds.NewCommandDescription("test-glazed-command", options...)
	return &TestGlazedCommand{
		description: description,
	}, nil
}

func (t *TestGlazedCommand) Description() *cmds.CommandDescription {
	return t.description
}

func (t *TestGlazedCommand) ToYAML(w io.Writer) error {
	return t.Description().ToYAML(w)
}

func (t *TestGlazedCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	var sectionValues *values.SectionValues
	if v, ok := parsedValues.Get(schema.DefaultSlug); ok {
		sectionValues = v
	}

	for i := 0; i < 3; i++ {
		row := types.NewRow(
			types.MRP("test", i),
			types.MRP("test2", fmt.Sprintf("test-%d", i)),
			types.MRP("test3", fmt.Sprintf("test3-%d", i)),
		)
		if sectionValues != nil {
			sectionValues.Fields.ForEach(func(_ string, f *fields.FieldValue) {
				row.Set(f.Definition.Name, f.Value)
			})
		}
		err := gp.AddRow(ctx,
			row,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ cmds.GlazeCommand = (*TestGlazedCommand)(nil)
