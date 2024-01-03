package utils

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"io"
)

type TestGlazedCommand struct {
	description *cmds.CommandDescription
}

func NewTestGlazedCommand(options ...cmds.CommandDescriptionOption) (*TestGlazedCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	options = append(options, cmds.WithLayersList(glazedLayer))

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
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	m := parameters.NewParsedParameters()

	v, ok := parsedLayers.Get(layers.DefaultSlug)
	if ok {
		m = v.Parameters
	}

	for i := 0; i < 3; i++ {
		row := types.NewRow(
			types.MRP("test", i),
			types.MRP("test2", fmt.Sprintf("test-%d", i)),
			types.MRP("test3", fmt.Sprintf("test3-%d", i)),
		)
		m.ForEach(func(_ string, p *parameters.ParsedParameter) {
			row.Set(p.ParameterDefinition.Name, p.Value)

		})
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
