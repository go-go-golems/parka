package layout

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/parka/pkg/glazed"
)

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

	Value interface{}
}

type Option struct {
	Label string
	Value interface{}
}

func ComputeLayout(
	pc *glazed.CommandContext,
) (*Layout, error) {
	description := pc.Cmd.Description()

	layout := description.Layout

	ret := &Layout{
		Sections: []*Section{},
	}

	values := pc.GetAllParameterValues()

	if layout == nil || len(layout.Sections) == 0 {
		pds := pc.GetFlagsAndArgumentsParameterDefinitions()
		flagSection := NewSectionFromParameterDefinitions(
			pds, values,
			WithSectionTitle("All flags and arguments"),
		)
		ret.Sections = append(ret.Sections, flagSection)

		// This code would add a section for all layers, in the form.
		// I don't think this is super useful in the context of Parka,
		// and can be overriden with layouts if you really want.
		//
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
		allParameterDefinitions := pc.GetAllParameterDefinitions()
		allParameterDefinitionsByName := map[string]*parameters.ParameterDefinition{}

		for _, pd := range allParameterDefinitions {
			allParameterDefinitionsByName[pd.Name] = pd
		}

		for _, section_ := range layout.Sections {
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
					pd, ok := allParameterDefinitionsByName[input_.Name]
					if !ok {
						return nil, fmt.Errorf("parameter %s not found", input_.Name)
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
					default_ := pd.Default
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
						Name:     input_.Name,
						Label:    label_,
						Value:    value,
						Type:     type_,
						Options:  options,
						Default:  default_,
						Help:     help_,
						CSS:      input_.CSS,
						Id:       input_.Id,
						Classes:  input_.Classes,
						Template: input_.Template,
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
	pds []*parameters.ParameterDefinition,
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
	for _, pd := range pds {
		name := pd.Name
		value, ok := values[name]
		if !ok {
			value = nil
		}
		help := pd.Help
		if help == "" {
			help = pd.Name
		}
		currentRow.Inputs = append(currentRow.Inputs, Input{
			Name:    name,
			Value:   value,
			Type:    string(pd.Type),
			Default: pd.Default,
			Help:    help,
			Options: choicesToOptions(pd.Choices),
		})
		if len(currentRow.Inputs) == 3 {
			section.Rows = append(section.Rows, currentRow)
			currentRow = Row{}
		}
	}
	if len(currentRow.Inputs) > 0 {
		section.Rows = append(section.Rows, currentRow)
	}

	return section
}
