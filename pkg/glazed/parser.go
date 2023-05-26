package glazed

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io"
	"mime/multipart"
	"strings"
	"time"
)

// ParserFunc is used to parse parameters out of a gin.Context (meaning
// most certainly out of an incoming *http.Request). These parameters
// are stored in the hashmap `ps`, according to the parameter definitions
// in `pds`.
//
// If a parameter definition shouldn't be handled by a follow up step, return a new hashmap
// with the key deleted.
type ParserFunc func(
	c *gin.Context,
	// This is not pretty, but it's needed to handle aliases until https://github.com/go-go-golems/glazed/issues/287
	// is built.
	defaults map[string]string,
	ps map[string]interface{},
	pds map[string]*parameters.ParameterDefinition,
) (map[string]*parameters.ParameterDefinition, error)

// NewQueryParserFunc returns a ParserFunc that can handle an incoming GET query string.
// If the parameter is supposed to be read from a file, we will just pass in the query parameter's value.
func NewQueryParserFunc(onlyDefined bool) ParserFunc {
	return func(
		c *gin.Context,
		defaults map[string]string,
		ps map[string]interface{},
		pd map[string]*parameters.ParameterDefinition,
	) (map[string]*parameters.ParameterDefinition, error) {

		for _, p := range pd {
			value := c.DefaultQuery(p.Name, defaults[p.Name])
			if parameters.IsFileLoadingParameter(p.Type, value) {
				// if the parameter is supposed to be read from a file, we will just pass in the query parameters
				// as a placeholder here
				if value == "" {
					if p.Required {
						return nil, errors.Errorf("required parameter '%s' is missing", p.Name)
					}
					if !onlyDefined {
						ps[p.Name] = p.Default
					}
				} else {
					f := strings.NewReader(value)
					pValue, err := p.ParseFromReader(f, "")
					if err != nil {
						return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
					}
					ps[p.Name] = pValue
				}
			} else {
				if value == "" {
					if p.Required {
						return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
					}
					if !onlyDefined {
						if p.Type == parameters.ParameterTypeDate {
							switch v := p.Default.(type) {
							case string:
								parsedDate, err := parameters.ParseDate(v)
								if err != nil {
									return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
								}

								ps[p.Name] = parsedDate
							case time.Time:
								ps[p.Name] = v

							}
						} else {
							ps[p.Name] = p.Default
						}
					}
				} else {
					var values []string
					if parameters.IsListParameter(p.Type) {
						values = strings.Split(value, ",")
					} else {
						values = []string{value}
					}
					pValue, err := p.ParseParameter(values)
					if err != nil {
						return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
					}
					ps[p.Name] = pValue
				}
			}
		}

		return pd, nil
	}
}

// NewFormParserFunc returns a ParserFunc that takes an incoming multipart Form, and can thus also handle uploaded files.
func NewFormParserFunc(onlyDefined bool) ParserFunc {
	return func(c *gin.Context,
		defaults map[string]string,
		ps map[string]interface{},
		pd map[string]*parameters.ParameterDefinition,
	) (map[string]*parameters.ParameterDefinition, error) {

		for _, p := range pd {
			value := c.DefaultPostForm(p.Name, defaults[p.Name])
			// TODO(manuel, 2023-02-28) is this enough to check if a file is missing?
			if value == "" {
				if p.Required {
					return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !onlyDefined {
					if p.Type == parameters.ParameterTypeDate {
						switch v := p.Default.(type) {
						case string:
							parsedDate, err := parameters.ParseDate(v)
							if err != nil {
								return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
							}

							ps[p.Name] = parsedDate
						case time.Time:
							ps[p.Name] = v

						}
					} else {
						ps[p.Name] = p.Default
					}
				}
			} else if !parameters.IsFileLoadingParameter(p.Type, value) {
				v := []string{value}
				if p.Type == parameters.ParameterTypeStringList ||
					p.Type == parameters.ParameterTypeIntegerList ||
					p.Type == parameters.ParameterTypeFloatList {
					v = strings.Split(value, ",")
				}
				pValue, err := p.ParseParameter(v)
				if err != nil {
					return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				ps[p.Name] = pValue
			} else if p.Type == parameters.ParameterTypeStringFromFile {
				s, err := ParseStringFromFile(c, p.Name)
				if err != nil {
					return nil, err
				}
				ps[p.Name] = s
			} else if p.Type == parameters.ParameterTypeObjectFromFile {
				obj, err := ParseObjectFromFile(c, p.Name)
				if err != nil {
					return nil, err
				}
				ps[p.Name] = obj
			} else if p.Type == parameters.ParameterTypeStringListFromFile {
				// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
				// See: https://github.com/go-go-golems/parka/issues/23
				_ = ps
			} else if p.Type == parameters.ParameterTypeObjectListFromFile {
				// TODO(manuel, 2023-04-16) Add support for StringListFromFile and ObjectListFromFile
				// See: https://github.com/go-go-golems/parka/issues/23
				_ = ps
			}
		}

		return pd, nil
	}
}

// ParseStringFromFile takes a multipart.File named `name` from the request and reads it into a string.
func ParseStringFromFile(c *gin.Context, name string) (string, error) {
	file, _, err := c.Request.FormFile(name)
	if err != nil {
		return "", fmt.Errorf("error retrieving file '%s': %v", name, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msgf("error closing file '%s'", name)
		}
	}(file)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading contents of file '%s': %v", name, err)
	}

	return string(fileBytes), nil
}

// ParseObjectFromFile takes a multipart.File named `name` from the request and reads it into a map[string]interface{}.
// If the file is a JSON file (ends with .json), it will be parsed as JSON,
// if it ends with .yaml or .yml, it will be parsed as YAML.
func ParseObjectFromFile(c *gin.Context, name string) (map[string]interface{}, error) {
	file, fileHeader, err := c.Request.FormFile(name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving file '%s': %v", name, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Msgf("error closing file '%s'", name)
		}
	}(file)

	// NOTE(manuel, 2023-04-16) We should actually look at the MIME type of the file instead of the file name
	// See https://github.com/go-go-golems/parka/issues/24

	// NOTE(manuel, 2023-04-16) We should support CSV and excel files here too. Excel might be more complicated because of multiple sheets and locations.
	// See https://github.com/go-go-golems/parka/issues/25

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading contents of file '%s': %v", name, err)
	}

	var obj map[string]interface{}
	if strings.HasSuffix(fileHeader.Filename, ".json") {
		err = json.Unmarshal(fileBytes, &obj)
		if err != nil {
			return nil, fmt.Errorf("error parsing contents of file '%s' as JSON: %v", fileHeader.Filename, err)
		}
	} else if strings.HasSuffix(fileHeader.Filename, ".yaml") || strings.HasSuffix(fileHeader.Filename, ".yml") {
		err = yaml.Unmarshal(fileBytes, &obj)
		if err != nil {
			return nil, fmt.Errorf("error parsing contents of file '%s' as YAML: %v", fileHeader.Filename, err)
		}
	} else {
		return nil, fmt.Errorf("unsupported file format for file '%s'", fileHeader.Filename)
	}

	return obj, nil
}

// NewStaticParserFunc returns a parser that adds the given static parameters to the parameter map.
func NewStaticParserFunc(ps map[string]interface{}) ParserFunc {
	return func(
		c *gin.Context,
		_ map[string]string,
		ps_ map[string]interface{},
		pds map[string]*parameters.ParameterDefinition,
	) (map[string]*parameters.ParameterDefinition, error) {
		// add the static parameters
		for k, v := range ps {
			ps_[k] = v
		}

		// no more parsing after this
		return map[string]*parameters.ParameterDefinition{}, nil
	}
}

// NewDefaultsFromLayerParserFunc returns a parser that adds the default values from the given
// layers.ParameterLayer to the parameter map (if they haven't been set in the `defaults` map first,
// which needs to be replaced anyway.
//
// NOTE(manuel, 2023-04-16) How this actually relate to the ParkaContext...
// NOTE(manuel, 2023-05-25) We shouldn't be dealing with the `defaults` map here
func NewDefaultsFromLayerParserFunc(l layers.ParameterLayer) ParserFunc {
	return func(
		c *gin.Context,
		defaults map[string]string,
		ps_ map[string]interface{},
		pds map[string]*parameters.ParameterDefinition,
	) (map[string]*parameters.ParameterDefinition, error) {
		// add the static parameters
		for _, pd := range l.GetParameterDefinitions() {
			// here we need to parse the default
			default_, ok := defaults[pd.Name]
			if ok {
				vs := []string{default_}
				if parameters.IsListParameter(pd.Type) {
					vs = strings.Split(default_, ",")
				}
				v, err := pd.ParseParameter(vs)
				if err != nil {
					return nil, fmt.Errorf("error parsing default value for parameter '%s': %v", pd.Name, err)
				}
				ps_[pd.Name] = v
			} else {
				if pd.Default != nil {
					ps_[pd.Name] = pd.Default
				}
			}
		}

		// no more parsing after this
		return map[string]*parameters.ParameterDefinition{}, nil
	}
}

// Parser is contains a list of ParserFunc that are used to parse an incoming
// request into a proper CommandContext, and ultimately be used to Run a glazed Command.
//
// These ParserFunc can be operating on the general parameters as well as per layer.
// The flexibility is there so that more complicated commands can ultimately be built that leverage
// different validations and rewrite rules.
//
// NOTE(manuel, 2023-04-16) I wonder when I will queue multiple ParserFunc and LayerParser Func.
// We might actually already do this by leveraging it to overwrite layer parameters (say, sqleton
// connection parameters).
type Parser struct {
	Parsers            []ParserFunc
	LayerParsersBySlug map[string][]ParserFunc
}

type ParserOption func(*Parser)

func NewParser(options ...ParserOption) *Parser {
	ph := &Parser{
		Parsers:            []ParserFunc{},
		LayerParsersBySlug: map[string][]ParserFunc{},
	}

	for _, option := range options {
		option(ph)
	}

	return ph
}

// NOTE(manuel, 2023-04-16) This might be better called WithPrependParserFunc ? What is a better name for ParserFunc.

// WithPrependParser adds the given ParserFunc to the beginning of the list of parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithPrependParser(ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = append(ps, ph.Parsers...)
	}
}

// WithAppendParser adds the given ParserFunc to the end of the list of parsers.
// Be mindful that this can later on be overwritten by a WithReplaceParser.
func WithAppendParser(ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = append(ph.Parsers, ps...)
	}
}

// WithReplaceParser replaces the list of parsers with the given ParserFunc.
// This will remove all previously added prepend, replace, append parsers.
func WithReplaceParser(ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		ph.Parsers = ps
	}
}

// WithPrependLayerParser adds the given ParserFunc to the beginning of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceLayerParser.
func WithPrependLayerParser(slug string, ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParserFunc{}
		}
		ph.LayerParsersBySlug[slug] = append(ps, ph.LayerParsersBySlug[slug]...)
	}
}

// WithAppendLayerParser adds the given ParserFunc to the end of the list of layer parsers.
// Be mindful that this can later on be overwritten by a WithReplaceLayerParser.
func WithAppendLayerParser(slug string, ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		if _, ok := ph.LayerParsersBySlug[slug]; !ok {
			ph.LayerParsersBySlug[slug] = []ParserFunc{}
		}
		ph.LayerParsersBySlug[slug] = append(ph.LayerParsersBySlug[slug], ps...)
	}
}

// WithReplaceLayerParser replaces the list of layer parsers with the given ParserFunc.
func WithReplaceLayerParser(slug string, ps ...ParserFunc) ParserOption {
	return func(ph *Parser) {
		ph.LayerParsersBySlug[slug] = ps
	}
}

// WithReplaceCustomizedParameterLayerParser adds a DefaultFromLayerParserFunc tweaked by the given
// `overrides` map. This entirely replaces the current layer parser, but can later on be amended
// with other parsers, for example with WithAppendOverrideLayer.
func WithReplaceCustomizedParameterLayerParser(l layers.ParameterLayer, overrides map[string]interface{}) ParserOption {
	slug := l.GetSlug()
	return WithReplaceLayerParser(
		slug,
		NewDefaultsFromLayerParserFunc(l),
		NewStaticParserFunc(overrides),
	)
}

// WithGlazeOutputParserOption is a convenience function to override the output and table format glazed settings.
func WithGlazeOutputParserOption(output string, tableFormat string) ParserOption {
	return WithAppendLayerParser(
		"glazed",
		NewStaticParserFunc(map[string]interface{}{
			"output":       output,
			"table-format": tableFormat,
		}),
	)
}

// WithReplaceStaticLayer is a convenience function to use static layer parsing.
// This entirely replaces current layer parsers, but can later on be amended with other parsers,
// for example with WithAppendOverrideLayer.
func WithReplaceStaticLayer(slug string, overrides map[string]interface{}) ParserOption {
	return WithReplaceLayerParser(
		slug,
		NewStaticParserFunc(overrides),
	)
}

// WithAppendOverrideLayer is a convenience function to override the parameters of a layer.
// The overrides are appended past currently present parser functions.
func WithAppendOverrideLayer(slug string, overrides map[string]interface{}) ParserOption {
	return WithAppendLayerParser(
		slug,
		NewStaticParserFunc(overrides),
	)
}
