---
Title: Parka Middlewares for Parameter Extraction
Slug: parka-middlewares
Short: Learn about Parka's powerful middlewares for extracting parameters from HTTP requests, including query parameters, form data, and JSON POST requests
Topics:
- middlewares
- parameters
- http
- forms
- json
Commands:
- UpdateFromQueryParameters
- UpdateFromFormQuery
- NewJSONBodyMiddleware
Flags:
- WithParseStepSource
- WithRequired
- WithHelp
- WithDefault
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Parka Middlewares for Parameter Extraction

Parka provides powerful middlewares for extracting parameters from HTTP requests, specifically designed to work with Glazed commands. This guide explains how to use these middlewares to handle URL query parameters, form data, and JSON POST requests.

## Overview

The three main middlewares for parameter extraction are:

1. `UpdateFromQueryParameters` - Extracts parameters from URL query strings
2. `UpdateFromFormQuery` - Extracts parameters from form data, including file uploads
3. `JSONBodyMiddleware` - Extracts parameters from JSON POST request bodies

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

## Using JSON Body Middleware

The JSON body middleware handles parameters sent in JSON format via POST requests. It's particularly useful for complex parameter structures and file-like parameters that contain content directly in the request body.

### Basic Usage

```go
import (
    "github.com/go-go-golems/parka/pkg/glazed/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

// Create a new JSON middleware instance
jsonMiddleware := middlewares.NewJSONBodyMiddleware(c,
    parameters.WithParseStepSource("json"))
defer jsonMiddleware.Close() // Important: Always close to cleanup temporary files

// Use in your middleware chain
middlewares_ := []middlewares.Middleware{
    jsonMiddleware.Middleware(),
}
```

### Example with a Glazed Command

Here's a complete example of using the JSON body middleware with a Glazed command:

```go
type MyCommand struct {
    *cmds.CommandDescription
}

func NewMyCommand() (*MyCommand, error) {
    return &MyCommand{
        CommandDescription: cmds.NewCommandDescription(
            "mycommand",
            cmds.WithShort("A command with JSON parameters"),
            cmds.WithFlags(
                parameters.NewParameterDefinition(
                    "content",
                    parameters.ParameterTypeStringFromFile,
                    parameters.WithHelp("Content to process"),
                    parameters.WithRequired(true),
                ),
                parameters.NewParameterDefinition(
                    "options",
                    parameters.ParameterTypeObject,
                    parameters.WithHelp("Processing options"),
                ),
            ),
        ),
    }, nil
}

// In your Echo handler
func HandleMyCommand(c echo.Context) error {
    cmd := NewMyCommand()
    parsedLayers := layers.NewParsedLayers()
    
    jsonMiddleware := middlewares.NewJSONBodyMiddleware(c)
    defer jsonMiddleware.Close()
    
    middlewares_ := []middlewares.Middleware{
        jsonMiddleware.Middleware(),
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
    
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

### Using with File-Like Parameters

The JSON middleware can handle file-like parameters by creating temporary files from string content in the JSON:

```json
{
    "content": "This will be written to a temp file\nAnd processed as a file parameter",
    "options": {
        "format": "text",
        "encoding": "utf-8"
    }
}
```

The middleware will:
1. Create a temporary file with the content
2. Pass the file path to the parameter parser
3. Clean up the temporary file when the middleware is closed

### Using the JSON Handler

Parka provides a convenient JSON handler that can work with both query parameters and JSON body:

```go
import (
    "github.com/go-go-golems/parka/pkg/glazed/handlers/json"
)

// For query parameters (GET requests)
handler := json.CreateJSONQueryHandler(cmd,
    json.WithParseOptions(parameters.WithParseStepSource("query")))

// For JSON body (POST requests)
handler := json.CreateJSONBodyHandler(cmd,
    json.WithParseOptions(parameters.WithParseStepSource("json")))

// Register with Echo
e.GET("/api/command", handler)
e.POST("/api/command", handler)
```

The handler supports configuration through options:
- `WithJSONBody()` - Use JSON body parsing instead of query parameters
- `WithParseOptions()` - Add parameter parse options
- `WithMiddlewares()` - Add additional middlewares to the chain

### Best Practices

1. **Always Close the Middleware**: Use `defer middleware.Close()` to ensure temporary files are cleaned up.

2. **Error Handling**: The middleware provides detailed error messages for:
   - Missing required parameters
   - Invalid parameter types
   - JSON parsing errors
   - File handling errors

3. **Parameter Types**: The middleware supports:
   - Basic types (string, number, boolean)
   - Arrays (for list parameters)
   - File-like parameters (content provided as strings)
   - Object parameters (nested JSON structures)

4. **Thread Safety**: The middleware is thread-safe for temporary file management.

## Combining Middlewares

You can combine different middlewares to handle various parameter sources:

```go
middlewares_ := []middlewares.Middleware{
    // For GET requests
    parka_middlewares.UpdateFromQueryParameters(c),
    
    // For POST with form data
    parka_middlewares.UpdateFromFormQuery(c),
    
    // For POST with JSON
    jsonMiddleware.Middleware(),
    
    // Always set defaults last
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

### API Endpoint with JSON Body

```go
func HandleAPIEndpoint(c echo.Context) error {
    cmd := NewAPICommand()
    parsedLayers := layers.NewParsedLayers()
    
    jsonMiddleware := middlewares.NewJSONBodyMiddleware(c,
        parameters.WithParseStepSource("json"))
    defer jsonMiddleware.Close()
    
    err := middlewares.ExecuteMiddlewares(
        cmd.Description().Layers.Clone(),
        parsedLayers,
        jsonMiddleware.Middleware(),
        middlewares.SetFromDefaults(),
    )
    if err != nil {
        return err
    }
    
    // Process the command
    return cmd.RunIntoGlazeProcessor(c.Request().Context(), parsedLayers, processor)
}
```

### Flexible API Endpoint

```go
func HandleFlexibleEndpoint(c echo.Context) error {
    cmd := NewFlexibleCommand()
    
    // Use the JSON handler which can handle both query and body
    handler := json.NewQueryHandler(cmd,
        json.WithParseOptions(parameters.WithParseStepSource("auto")))
    
    if c.Request().Method == "POST" {
        handler.UseJSONBody = true
    }
    
    return handler.Handle(c)
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

## Understanding the JSON Middleware Internals

The JSON middleware (`JSONBodyMiddleware`) is designed to handle JSON POST requests and manage temporary files. Here's a detailed look at its internal workings:

### Middleware Structure

```go
type JSONBodyMiddleware struct {
    c       echo.Context
    options []parameters.ParseStepOption
    files   []string
    mu      sync.Mutex
}
```

1. **Context**: Stores the Echo context for accessing the request body
2. **Options**: Parse step options for parameter processing
3. **Files**: List of temporary files to clean up
4. **Mutex**: Ensures thread-safe file management

### Parameter Processing Flow

1. **Body Reading**:
   ```go
   body, err := io.ReadAll(m.c.Request().Body)
   var jsonData map[string]interface{}
   if err := json.Unmarshal(body, &jsonData); err != nil {
       return errors.Wrap(err, "could not parse JSON body")
   }
   ```
   - Reads the entire request body
   - Parses it as a JSON object

2. **Parameter Extraction**:
   ```go
   value, exists := jsonData[p.Name]
   if !exists {
       if p.Required {
           return errors.Errorf("required parameter '%s' is missing", p.Name)
       }
       return nil
   }
   ```
   - Checks for parameter existence
   - Handles required parameters

3. **File Parameter Handling**:
   ```go
   if p.Type.NeedsFileContent("") {
       switch v := value.(type) {
       case string:
           tmpPath, err := m.createTempFileFromString(v)
           // ... process file ...
       }
   }
   ```
   - Creates temporary files for file-like parameters
   - Manages file cleanup through the Close method

4. **Type Conversion**:
   ```go
   switch v := value.(type) {
   case string:
       stringValue = v
   case float64:
       stringValue = fmt.Sprintf("%v", v)
   case bool:
       stringValue = fmt.Sprintf("%v", v)
   case []interface{}:
       // Handle arrays
   }
   ```
   - Converts JSON values to appropriate parameter types
   - Handles arrays and primitive types

### Temporary File Management

The middleware uses a thread-safe approach to manage temporary files:

```go
func (m *JSONBodyMiddleware) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    var errs []error
    for _, f := range m.files {
        if err := os.Remove(f); err != nil {
            errs = append(errs, errors.Wrapf(err, "failed to remove temporary file %s", f))
        }
    }
    m.files = m.files[:0]

    if len(errs) > 0 {
        return errors.Errorf("failed to clean up some temporary files: %v", errs)
    }
    return nil
}
```

This ensures that:
- All temporary files are properly tracked
- Cleanup is thread-safe
- Errors during cleanup are collected and reported
- The files list is cleared after cleanup

## Further Reading

- [Glazed Commands Documentation](15-using-commands.md)
- [Parka Server Documentation](01-parka-server.md)
- [Echo Framework Middleware Guide](https://echo.labstack.com/middleware) 