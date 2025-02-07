# Parka Middlewares for Parameter Extraction

Parka provides powerful middlewares for extracting parameters from HTTP requests, specifically designed to work with Glazed commands. This guide explains how to use these middlewares to handle both URL query parameters and form data.

## Overview

The two main middlewares for parameter extraction are:

1. `UpdateFromQueryParameters` - Extracts parameters from URL query strings
2. `UpdateFromFormQuery` - Extracts parameters from form data, including file uploads

These middlewares are essential when exposing Glazed commands through HTTP endpoints, as they allow seamless translation of HTTP request data into Glazed command parameters.

## Using Query Parameter Middleware

The query parameter middleware extracts parameters from URL query strings and updates the Glazed command's parameter layers accordingly.

### Basic Usage

```go
import (
    "github.com/go-go-golems/parka/pkg/glazed/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// In your handler
middlewares_ := []middlewares.Middleware{
    parka_middlewares.UpdateFromQueryParameters(c,
        parameters.WithParseStepSource("query")),
}
```

### Example with a Glazed Command

Here's a complete example of using the query parameter middleware with a Glazed command:

```go
type MyCommand struct {
    *cmds.CommandDescription
}

func NewMyCommand() (*MyCommand, error) {
    return &MyCommand{
        CommandDescription: cmds.NewCommandDescription(
            "mycommand",
            cmds.WithShort("A command with query parameters"),
            cmds.WithFlags(
                parameters.NewParameterDefinition(
                    "limit",
                    parameters.ParameterTypeInteger,
                    parameters.WithHelp("Number of items to return"),
                    parameters.WithDefault(10),
                ),
                parameters.NewParameterDefinition(
                    "filter",
                    parameters.ParameterTypeString,
                    parameters.WithHelp("Filter results"),
                ),
            ),
        ),
    }, nil
}

// In your Echo handler
func HandleMyCommand(c echo.Context) error {
    cmd := NewMyCommand()
    parsedLayers := layers.NewParsedLayers()
    
    middlewares_ := []middlewares.Middleware{
        parka_middlewares.UpdateFromQueryParameters(c),
    }
    
    err := middlewares.ExecuteMiddlewares(
        cmd.Description().Layers.Clone(),
        parsedLayers,
        middlewares_...,
    )
    if err != nil {
        return err
    }
    
    // Now parsedLayers contains the parameters from the query string
    // e.g., /api/mycommand?limit=20&filter=active
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

## Using Form Data Middleware

The form data middleware handles both regular form fields and file uploads. It's particularly useful when your command needs to process uploaded files or handle complex form data.

### Basic Usage

```go
import (
    "github.com/go-go-golems/parka/pkg/glazed/middlewares"
)

middlewares_ := []middlewares.Middleware{
    parka_middlewares.UpdateFromFormQuery(c),
}
```

### Handling File Uploads

The form middleware can handle file parameters defined in your Glazed command:

```go
func NewFileUploadCommand() (*FileUploadCommand, error) {
    return &FileUploadCommand{
        CommandDescription: cmds.NewCommandDescription(
            "upload",
            cmds.WithShort("Upload and process files"),
            cmds.WithFlags(
                parameters.NewParameterDefinition(
                    "file",
                    parameters.ParameterTypeStringFromFile,
                    parameters.WithHelp("File to process"),
                    parameters.WithRequired(true),
                ),
                parameters.NewParameterDefinition(
                    "description",
                    parameters.ParameterTypeString,
                    parameters.WithHelp("File description"),
                ),
            ),
        ),
    }, nil
}
```

### Example with File Upload and Form Data

Here's a complete example showing how to handle both file uploads and regular form fields:

```go
func HandleFileUpload(c echo.Context) error {
    cmd := NewFileUploadCommand()
    parsedLayers := layers.NewParsedLayers()
    
    middlewares_ := []middlewares.Middleware{
        parka_middlewares.UpdateFromFormQuery(c),
    }
    
    err := middlewares.ExecuteMiddlewares(
        cmd.Description().Layers.Clone(),
        parsedLayers,
        middlewares_...,
    )
    if err != nil {
        return err
    }
    
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

## Combining Middlewares

You can combine both middlewares to handle both query parameters and form data:

```go
middlewares_ := []middlewares.Middleware{
    parka_middlewares.UpdateFromQueryParameters(c),
    parka_middlewares.UpdateFromFormQuery(c),
    middlewares.SetFromDefaults(),
}
```

## Best Practices

1. **Order Matters**: Place the middlewares in the order you want them to process. Later middlewares can override values set by earlier ones.

2. **Default Values**: Always use `middlewares.SetFromDefaults()` as the last middleware to ensure default values are set for unspecified parameters.

3. **Error Handling**: Always check for middleware execution errors before proceeding with command execution.

4. **Parameter Types**: Be mindful of parameter types when defining your command. The middlewares will attempt to parse the input according to the parameter type.

5. **File Handling**: When dealing with file uploads:
   - Use appropriate parameter types (`ParameterTypeStringFromFile`, `ParameterTypeStringFromFiles`)
   - Consider file size limits
   - Handle cleanup of temporary files

## Common Patterns

### API Endpoint with Query Parameters

```go
func HandleAPIEndpoint(c echo.Context) error {
    cmd := NewAPICommand()
    parsedLayers := layers.NewParsedLayers()
    
    err := middlewares.ExecuteMiddlewares(
        cmd.Description().Layers.Clone(),
        parsedLayers,
        parka_middlewares.UpdateFromQueryParameters(c),
        middlewares.SetFromDefaults(),
    )
    if err != nil {
        return err
    }
    
    // Process the command
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

### Form Submission Handler

```go
func HandleFormSubmission(c echo.Context) error {
    cmd := NewFormCommand()
    parsedLayers := layers.NewParsedLayers()
    
    err := middlewares.ExecuteMiddlewares(
        cmd.Description().Layers.Clone(),
        parsedLayers,
        parka_middlewares.UpdateFromFormQuery(c),
        middlewares.SetFromDefaults(),
    )
    if err != nil {
        return err
    }
    
    // Process the form submission
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

## Integration with DataTables Handler

The DataTables handler in Parka provides a good example of how these middlewares are used in practice:

```go
func CreateDataTablesHandler(cmd cmds.GlazeCommand, options ...QueryHandlerOption) echo.HandlerFunc {
    return func(c echo.Context) error {
        parsedLayers := layers.NewParsedLayers()
        
        middlewares_ := []middlewares.Middleware{
            parka_middlewares.UpdateFromQueryParameters(c),
            // Add custom middlewares
            middlewares.SetFromDefaults(),
        }
        
        err := middlewares.ExecuteMiddlewares(
            cmd.Description().Layers.Clone(),
            parsedLayers,
            middlewares_...,
        )
        if err != nil {
            return err
        }
        
        // Process the command and render the DataTables view
        return nil
    }
}
```

## Understanding the Query Parameter Middleware Internals

The query parameter middleware (`UpdateFromQueryParameters`) is designed to extract and parse URL query parameters into Glazed command parameters. Here's a detailed look at how it works internally:

### Parameter Extraction Process

1. **Context Wrapping**
```go
func UpdateFromQueryParameters(c echo.Context, options ...parameters.ParseStepOption) middlewares.Middleware {
    return func(next middlewares.HandlerFunc) middlewares.HandlerFunc {
        return func(layers_ *layers.ParameterLayers, parsedLayers *layers.ParsedLayers) error {
            // ... middleware implementation
        }
    }
}
```

The middleware is structured as a closure that takes an Echo context and returns a Glazed middleware function. This pattern allows it to access both the HTTP context and the Glazed parameter system.

### Parameter Processing Flow

1. **Layer Iteration**:
   ```go
   err := layers_.ForEachE(func(_ string, l layers.ParameterLayer) error {
       parsedLayer := parsedLayers.GetOrCreate(l)
       // ... process parameters
   })
   ```
   - Iterates through each parameter layer defined in the command
   - Creates or retrieves corresponding parsed layers for storing values

2. **Parameter Extraction**:
   ```go
   value := c.QueryParam(p.Name)
   if value == "" {
       if p.Required {
           return errors.Errorf("required parameter '%s' is missing", p.Name)
       }
       return nil
   }
   ```
   - Extracts parameter values using Echo's `QueryParam` method
   - Handles required parameters by returning errors if missing

3. **Type Conversion**:
   ```go
   parsedParameter, err := p.ParseParameter([]string{value}, options...)
   if err != nil {
       return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, value)
   }
   ```
   - Converts string values to the appropriate parameter type
   - Uses Glazed's parameter parsing system for type conversion
   - Handles validation and error cases

4. **Array Parameter Handling**:
   ```go
   if p.Type.IsList() {
       values := c.QueryParams()[p.Name]
       if len(values) > 0 {
           parsedParameter, err := p.ParseParameter(values, options...)
           // ... error handling
           parsedLayer.Parameters.Update(p.Name, parsedParameter)
       }
   }
   ```
   - Special handling for array parameters
   - Uses `QueryParams()` to get all values for a parameter name
   - Supports multiple values for the same parameter name

## Understanding the Form Middleware Internals

The form middleware (`UpdateFromFormQuery`) handles both regular form fields and file uploads. Here's a detailed look at its internal workings:

### File Upload Processing

1. **Multipart Form Handling**:
```go
func getFileParameterFromForm(c echo.Context, p *parameters.ParameterDefinition) (interface{}, error) {
    form, err := c.MultipartForm()
    if err != nil {
        return nil, err
    }
    headers := form.File[p.Name]
    // ... process files
}
```
- Accesses the multipart form data
- Retrieves file headers for the specified parameter name

2. **File Content Processing**:
```go
for _, h := range headers {
    err = func() error {
        f, err := h.Open()
        if err != nil {
            return err
        }
        defer func() {
            _ = f.Close()
        }()

        v, err := p.ParseFromReader(f, h.Filename)
        if err != nil {
            return errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, h.Filename)
        }

        values = append(values, v.Value)
        return nil
    }()
}
```
- Opens each uploaded file
- Uses Glazed's `ParseFromReader` for content processing
- Handles proper file cleanup with deferred close
- Accumulates processed values

### List Parameter Processing

1. **Array Field Detection**:
```go
func getListParameterFromForm(c echo.Context, p *parameters.ParameterDefinition, options ...parameters.ParseStepOption) (*parameters.ParsedParameter, error) {
    if p.Type.IsList() {
        values_, err := c.FormParams()
        if err != nil {
            return nil, err
        }
        values, ok := values_[fmt.Sprintf("%s[]", p.Name)]
        // ... process array values
    }
}
```
- Detects array parameters using the `[]` suffix convention
- Extracts all values for the array parameter

2. **Value Processing**:
```go
if ok {
    pValue, err := p.ParseParameter(values, options...)
    if err != nil {
        return nil, errors.Wrapf(err, "invalid value for parameter '%s': %s", p.Name, values)
    }
    return pValue, nil
}
```
- Parses array values using Glazed's parameter parsing system
- Handles type conversion and validation

### Type-Specific Processing

```go
switch {
case p.Type.IsList():
    vs := []interface{}{}
    for _, v_ := range values {
        vss, err := cast.CastListToInterfaceList(v_)
        if err != nil {
            return nil, err
        }
        vs = append(vs, vss...)
    }
    v = vs

case p.Type == parameters.ParameterTypeStringFromFile,
    p.Type == parameters.ParameterTypeStringFromFiles:
    s := ""
    for _, v_ := range values {
        ss, ok := v_.(string)
        if !ok {
            return nil, errors.Errorf("invalid value for parameter '%s': (%v) %s", p.Name, v_, "expected string")
        }
        s += ss
    }
    v = s
}
```
- Handles different parameter types differently
- Special processing for lists and file content
- Type-specific validation and conversion

### Error Handling and Validation

Both middlewares implement comprehensive error handling:

1. **Required Parameters**:
   - Check for presence of required parameters
   - Return descriptive error messages for missing values

2. **Type Validation**:
   - Validate parameter types during parsing
   - Handle conversion errors gracefully

3. **File Processing Errors**:
   - Handle file open/read errors
   - Manage temporary file cleanup
   - Report file processing errors with context

## Further Reading

- [Glazed Commands Documentation](15-using-commands.md)
- [Parka Server Documentation](01-parka-server.md)
- [Echo Framework Middleware Guide](https://echo.labstack.com/middleware) 