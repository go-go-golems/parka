package layout

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
)

// This section groups all the functionality related to laying out forms for input parameters
// for commands.

// Layout might look at first similar to glazed_layout.Layout, but it is actually
// the parsed and computed version used to render an HTML form.
type Layout struct {
	Sections []*Section
}

type Section struct {
	Title            string
	ShortDescription string
	LongDescription  string
	Style            string
	Classes          string
	Rows             []Row
}

type SectionOption func(*Section)

func WithSectionTitle(title string) SectionOption {
	return func(s *Section) {
		s.Title = title
	}
}

func WithSectionShortDescription(desc string) SectionOption {
	return func(s *Section) {
		s.ShortDescription = desc
	}
}

func WithSectionLongDescription(desc string) SectionOption {
	return func(s *Section) {
		s.LongDescription = desc
	}
}

type Row struct {
	Inputs  []Input
	Style   string
	Classes string
}

type Input struct {
	Name string

	Label   string
	Options []Option
	Default interface{}
	Help    string

	// this can be used to customizes the HTML output
	// see https://github.com/go-go-golems/parka/issues/28
	CSS      string
	Id       string
	Classes  string
	Template string
	Type     string

	ParameterDefinition *parameters.ParameterDefinition

	Value interface{}
}

type Option struct {
	Label string
	Value interface{}
}

func ComputeLayout(
	cmd cmds.Command,
	parsedLayers *layers.ParsedLayers,
) (*Layout, error) {
	description := cmd.Description()

	layout := description.Layout

	ret := &Layout{
		Sections: []*Section{},
	}

	defaultLayer := parsedLayers.GetDefaultParameterLayer()

	if len(layout) == 0 {
		pds := defaultLayer.Layer.GetParameterDefinitions()
		flagSection := NewSectionFromParameterDefinitions(
			pds, defaultLayer.Parameters.ToMap(),
			WithSectionTitle("All flags and arguments"),
		)
		ret.Sections = append(ret.Sections, flagSection)

		// This code would add a section for all layers, in the form.
		// I don't think this is super useful in the context of Parka,
		// and can be overriden with layouts if you really want.
		//
		//for _, l := range description.Layers {
		//	pds = l.GetParameterDefinitions()
		//	section := NewSectionFromParameterDefinitions(
		//		pds, values,
		//		WithSectionTitle(l.GetName()),
		//		WithSectionShortDescription(l.GetDescription()),
		//	)
		//	ret.Sections = append(ret.Sections, section)
		//}
	} else {
		allParameterDefinitions := cmd.Description().Layers.GetAllParameterDefinitions()
		values := parsedLayers.GetDataMap()

		for _, section_ := range layout {
			section := &Section{
				Title:            section_.Title,
				ShortDescription: section_.Description,
				Style:            section_.Style,
				Classes:          section_.Classes,
			}
			for _, row_ := range section_.Rows {
				row := Row{
					Inputs:  []Input{},
					Style:   row_.Style,
					Classes: row_.Classes,
				}

				for _, input_ := range row_.Inputs {
					pd, ok := allParameterDefinitions.Get(input_.Name)
					if !ok {
						return nil, errors.Errorf("parameter %s not found", input_.Name)
					}
					value, ok := values[input_.Name]
					if !ok {
						value = nil
					}

					var options []Option
					if len(input_.Options) > 0 {
						for _, option := range input_.Options {
							options = append(options, Option{
								Label: option.Label,
								Value: option.Value,
							})
						}
					} else {
						options = choicesToOptions(pd.Choices)
					}

					type_ := string(pd.Type)
					if input_.InputType != "" {
						type_ = input_.InputType
					}
					default_ := interface{}(nil)
					if pd.Default != nil {
						default_ = *pd.Default
					}
					if input_.DefaultValue != nil {
						default_ = input_.DefaultValue
					}

					help_ := pd.Help
					if input_.Help != "" {
						help_ = input_.Help
					}

					label_ := pd.Help
					if input_.Label != "" {
						label_ = input_.Label
					}

					row.Inputs = append(row.Inputs, Input{
						Name:                input_.Name,
						Label:               label_,
						Value:               value,
						Type:                type_,
						ParameterDefinition: pd,
						Options:             options,
						Default:             default_,
						Help:                help_,
						CSS:                 input_.CSS,
						Id:                  input_.Id,
						Classes:             input_.Classes,
						Template:            input_.Template,
					})
				}

				section.Rows = append(section.Rows, row)
			}

			ret.Sections = append(ret.Sections, section)
		}
	}

	return ret, nil
}

func choicesToOptions(choices []string) []Option {
	options := []Option{}
	for _, choice := range choices {
		options = append(options, Option{
			Label: choice,
			Value: choice,
		})
	}
	return options
}

func NewSectionFromParameterDefinitions(
	pds *parameters.ParameterDefinitions,
	values map[string]interface{},
	options ...SectionOption) *Section {
	section := &Section{
		Rows: []Row{},
	}

	for _, option := range options {
		option(section)
	}

	// if there is no layout, go through all flags and put 3 per row
	currentRow := Row{}
	pds.ForEach(func(pd *parameters.ParameterDefinition) {
		name := pd.Name
		value, ok := values[name]
		if !ok {
			value = nil
		}
		help := pd.Help
		if help == "" {
			help = pd.Name
		}
		input := Input{
			Name:    name,
			Value:   value,
			Type:    string(pd.Type),
			Help:    help,
			Options: choicesToOptions(pd.Choices),
		}
		if pd.Default != nil {
			input.Default = *pd.Default
		}
		currentRow.Inputs = append(currentRow.Inputs, input)
		if len(currentRow.Inputs) == 3 {
			section.Rows = append(section.Rows, currentRow)
			currentRow = Row{}
		}
	})

	if len(currentRow.Inputs) > 0 {
		section.Rows = append(section.Rows, currentRow)
	}

	return section
}

type Link struct {
	Href  string
	Text  string
	Class string
}
