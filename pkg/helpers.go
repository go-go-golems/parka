package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io"
	"mime/multipart"
	"strings"
)

func parseQueryFromParameterDefinitions(
	c *gin.Context,
	pd map[string]*parameters.ParameterDefinition,
	onlyDefined bool,
) (map[string]interface{}, error) {
	ps := make(map[string]interface{})

	for _, p := range pd {
		if parameters.IsFileLoadingParameter(p.Type, c.Query(p.Name)) {
			// if the parameter is supposed to be read from a file, we will just pass in the query parameters
			// as a placeholder here
			value := c.Query(p.Name)
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
			value := c.Query(p.Name)
			if value == "" {
				if p.Required {
					return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
				}
				if !onlyDefined {
					ps[p.Name] = p.Default
				}
			} else {
				pValue, err := p.ParseParameter([]string{value})
				if err != nil {
					return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
				}
				ps[p.Name] = pValue
			}
		}
	}

	return ps, nil
}

func parseStringFromFile(c *gin.Context, name string) (string, error) {
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

func parseObjectFromFile(c *gin.Context, name string) (map[string]interface{}, error) {
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

func parseFormFromParameterDefinitions(
	c *gin.Context,
	ps map[string]*parameters.ParameterDefinition,
	onlyDefined bool,
) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for _, p := range ps {
		value := c.PostForm(p.Name)
		// TODO(manuel, 2023-02-28) is this enough to check if a file is missing?
		if value == "" {
			if p.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", p.Name)
			}
			if !onlyDefined {
				params[p.Name] = p.Default
			}
		} else if p.Type != parameters.ParameterTypeStringFromFile && p.Type != parameters.ParameterTypeObjectFromFile {
			pValue, err := p.ParseParameter([]string{value})
			if err != nil {
				return nil, fmt.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, value, err.Error())
			}
			params[p.Name] = pValue
		} else if p.Type == parameters.ParameterTypeStringFromFile {
			s, err := parseStringFromFile(c, p.Name)
			if err != nil {
				return nil, err
			}
			params[p.Name] = s
		} else if p.Type == parameters.ParameterTypeObjectFromFile {
			obj, err := parseObjectFromFile(c, p.Name)
			if err != nil {
				return nil, err
			}
			params[p.Name] = obj
		}
	}

	return params, nil
}
