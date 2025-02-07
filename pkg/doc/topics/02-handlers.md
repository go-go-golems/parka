# Parka Static Handlers Documentation

Parka provides two specialized handlers for serving static content: `StaticDirHandler` and `StaticFileHandler`. These handlers are designed to serve static files from either the filesystem or embedded files, with different strategies for path handling and file organization.

## StaticDirHandler

The `StaticDirHandler` is designed to serve an entire directory of static files, maintaining the directory structure when serving the files over HTTP.

### Structure

```go
type StaticDirHandler struct {
    fs        fs.FS
    localPath string
}
```

- `fs`: The filesystem interface that provides access to the static files
- `localPath`: The base path within the filesystem where the static files are located

### Configuration Options

The handler can be configured using functional options:

1. `WithDefaultFS(fs fs.FS, localPath string)`: Sets a default filesystem and local path
   ```go
   handler := NewStaticDirHandler(
       WithDefaultFS(embeddedFS, "static"),
   )
   ```

2. `WithLocalPath(localPath string)`: Sets up the handler to serve files from a local directory
   ```go
   handler := NewStaticDirHandler(
       WithLocalPath("/path/to/static/files"),
   )
   ```

### Creation Methods

1. Basic creation with options:
   ```go
   handler := NewStaticDirHandler(options...)
   ```

2. Creation from configuration:
   ```go
   handler := NewStaticDirHandlerFromConfig(staticConfig, options...)
   ```

### Usage

The handler is registered with a Parka server using the `Serve` method:

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create and configure the handler
handler := NewStaticDirHandler(
    WithLocalPath("./static"),
)

// Register the handler with a base path
err = handler.Serve(server, "/static")
if err != nil {
    return err
}
```

This will serve all files from the configured directory under the `/static` URL path.

### Path Handling

- If serving from a local path, the handler automatically creates a directory filesystem
- Trailing slashes are automatically handled
- The local path is prefixed to filesystem paths when necessary

## StaticFileHandler

The `StaticFileHandler` is designed to serve individual files or specific subdirectories, with more precise control over the served paths.

### Structure

```go
type StaticFileHandler struct {
    fs        fs.FS
    localPath string
}
```

- `fs`: The filesystem interface that provides access to the static files
- `localPath`: The specific path to the file or subdirectory to serve

### Configuration Options

1. `WithDefaultFS(fs fs.FS, localPath string)`: Sets a default filesystem and local path
   ```go
   handler := NewStaticFileHandler(
       WithDefaultFS(embeddedFS, "assets/file.css"),
   )
   ```

2. `WithLocalPath(localPath string)`: Sets up the handler to serve from a local path
   ```go
   handler := NewStaticFileHandler(
       WithLocalPath("/path/to/specific/file.js"),
   )
   ```

### Creation Methods

1. Basic creation with options:
   ```go
   handler := NewStaticFileHandler(options...)
   ```

2. Creation from configuration:
   ```go
   handler := NewStaticFileHandlerFromConfig(staticFileConfig, options...)
   ```

### Usage

The handler is registered with a Parka server using the `Serve` method:

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create and configure the handler
handler := NewStaticFileHandler(
    WithLocalPath("/path/to/specific/file.js"),
)

// Register the handler with a specific URL path
err = handler.Serve(server, "/assets/js/script.js")
if err != nil {
    return err
}
```

### Path Handling

- Leading slashes in local paths are automatically handled
- Uses Echo's `MustSubFS` for safe subpath handling
- Maintains exact path mapping between filesystem and URL paths

## Differences Between Handlers

1. **Scope**:
   - `StaticDirHandler`: Serves entire directories with their structure
   - `StaticFileHandler`: Serves specific files or subdirectories with precise path control

2. **Path Handling**:
   - `StaticDirHandler`: Automatically handles directory structure and trailing slashes
   - `StaticFileHandler`: Provides exact path mapping and uses Echo's subfilesystem functionality

3. **Use Cases**:
   - `StaticDirHandler`: Best for serving static assets like images, CSS, and JavaScript files in their directory structure
   - `StaticFileHandler`: Best for serving individual files or when precise control over URL paths is needed

## Best Practices

1. **Directory Structure**:
   - Keep static files organized in a clear directory structure
   - Use `StaticDirHandler` for serving multiple related files
   - Use `StaticFileHandler` for specific files that need custom URL paths

2. **Security**:
   - Always validate and sanitize paths
   - Be careful with directory traversal attacks
   - Use embedded filesystems when possible for better security

3. **Performance**:
   - Consider using a CDN for large static assets
   - Enable compression middleware for text-based files
   - Use caching headers appropriately

## Examples

### Serving an Embedded Directory

```go
//go:embed static/*
var staticFS embed.FS

handler := NewStaticDirHandler(
    WithDefaultFS(staticFS, "static"),
)
err = handler.Serve(server, "/assets")
if err != nil {
    return err
}
```

### Serving a Local Directory

```go
handler := NewStaticDirHandler(
    WithLocalPath("./static"),
)
err = handler.Serve(server, "/static")
if err != nil {
    return err
}
```

### Serving a Specific File

```go
handler := NewStaticFileHandler(
    WithLocalPath("./assets/main.js"),
)
err = handler.Serve(server, "/js/main.js")
if err != nil {
    return err
}
```

### Configuration-based Setup

```go
config := &config.Static{
    LocalPath: "./static",
}
handler := NewStaticDirHandlerFromConfig(config)
err = handler.Serve(server, "/assets")
if err != nil {
    return err
}
```

## Error Handling

Both handlers handle errors gracefully:
- Invalid paths return appropriate HTTP errors
- Missing files return 404 Not Found
- Permission issues return 403 Forbidden

## Integration with Echo

Both handlers integrate seamlessly with Echo's static file serving capabilities:
- Use Echo's `StaticFS` method internally
- Support Echo's middleware stack
- Compatible with Echo's error handling

## Further Reading

- [Echo Static File Serving](https://echo.labstack.com/guide/static-files)
- [Go Filesystem Interface](https://golang.org/pkg/io/fs/)
- [Embedding Static Files](https://golang.org/pkg/embed/)

# Template Handlers

Parka provides two specialized handlers for serving templated content: `TemplateHandler` and `TemplateDirHandler`. These handlers enable dynamic content rendering using Go templates, with support for both HTML and Markdown files.

## TemplateHandler

The `TemplateHandler` is designed to serve a single template file, rendering it with optional data and supporting both HTML and Markdown content.

### Structure

```go
type TemplateHandler struct {
    fs              fs.FS
    TemplateFile    string
    rendererOptions []render.RendererOption
    renderer        *render.Renderer
    alwaysReload    bool
}
```

- `fs`: The filesystem interface that provides access to the template files
- `TemplateFile`: The path to the template file to be rendered
- `rendererOptions`: Additional options for configuring the template renderer
- `renderer`: The renderer instance used to process templates
- `alwaysReload`: Whether to reload templates on every request (useful for development)

### Configuration Options

The handler can be configured using functional options:

1. `WithDefaultFS(fs fs.FS)`: Sets a default filesystem for template loading
   ```go
   handler := NewTemplateHandler("index.tmpl.html",
       WithDefaultFS(embeddedFS),
   )
   ```

2. `WithAlwaysReload(alwaysReload bool)`: Enables template reloading for development
   ```go
   handler := NewTemplateHandler("index.tmpl.html",
       WithAlwaysReload(true),
   )
   ```

### Usage

The handler is registered with a Parka server using the `Serve` method:

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create and configure the handler
handler := NewTemplateHandler("index.tmpl.html",
    WithDefaultFS(embeddedFS),
    WithAlwaysReload(true),
)

// Register the handler with a URL path
err = handler.Serve(server, "/")
if err != nil {
    return err
}
```

## TemplateDirHandler

The `TemplateDirHandler` is designed to serve an entire directory of templates, supporting both HTML and Markdown files with automatic routing based on file paths.

### Structure

```go
type TemplateDirHandler struct {
    fs                       fs.FS
    LocalDirectory           string
    IndexTemplateName        string
    MarkdownBaseTemplateName string
    rendererOptions          []render.RendererOption
    renderer                 *render.Renderer
    alwaysReload            bool
}
```

- `fs`: The filesystem interface that provides access to the template files
- `LocalDirectory`: The base directory containing templates
- `IndexTemplateName`: The template to use for directory index pages
- `MarkdownBaseTemplateName`: The base template for rendering Markdown files
- `rendererOptions`: Additional options for configuring the template renderer
- `renderer`: The renderer instance used to process templates
- `alwaysReload`: Whether to reload templates on every request

### Configuration Options

1. `WithDefaultFS(fs fs.FS, localPath string)`: Sets a default filesystem and local path
   ```go
   handler, err := NewTemplateDirHandler(
       WithDefaultFS(embeddedFS, "templates"),
   )
   ```

2. `WithLocalDirectory(localPath string)`: Sets up the handler to serve from a local directory
   ```go
   handler, err := NewTemplateDirHandler(
       WithLocalDirectory("./templates"),
   )
   ```

### Usage

The handler is registered with a Parka server using the `Serve` method:

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create and configure the handler
handler, err := NewTemplateDirHandler(
    WithLocalDirectory("./templates"),
    WithAlwaysReload(true),
)
if err != nil {
    return err
}

// Register the handler with a base path
err = handler.Serve(server, "/")
if err != nil {
    return err
}
```

### Template Discovery

The TemplateDirHandler automatically discovers and serves:
- `*.tmpl.md` - Markdown templates
- `*.md` - Plain Markdown files
- `*.tmpl.html` - HTML templates
- `*.html` - Plain HTML files

## Differences Between Handlers

1. **Scope**:
   - `TemplateHandler`: Serves a single template file
   - `TemplateDirHandler`: Serves an entire directory of templates with automatic routing

2. **File Support**:
   - `TemplateHandler`: Focused on single template rendering
   - `TemplateDirHandler`: Supports multiple template types and automatic discovery

3. **Use Cases**:
   - `TemplateHandler`: Best for single pages or specific templates
   - `TemplateDirHandler`: Best for documentation sites, multi-page applications, or content-heavy sites

## Best Practices

1. **Template Organization**:
   - Use clear naming conventions for templates
   - Separate content from layout templates
   - Use base templates for consistent styling
   - Keep templates modular and reusable

2. **Development Workflow**:
   - Use `WithAlwaysReload(true)` during development
   - Create a base template for consistent layouts
   - Use partials for reusable components
   - Implement proper error handling in templates

3. **Performance**:
   - Disable template reloading in production
   - Use caching headers appropriately
   - Minimize template complexity
   - Consider precompiling templates

## Examples

### Serving a Single Template

```go
handler := NewTemplateHandler("index.tmpl.html",
    WithDefaultFS(embeddedFS),
    WithAlwaysReload(true),
)
server.AddHandler(handler, "/")
```

### Serving a Documentation Site

```go
handler, err := NewTemplateDirHandler(
    WithLocalDirectory("./docs"),
    WithAlwaysReload(true),
)
server.AddHandler(handler, "/docs")
```

### Configuration-based Setup

```go
config := &config.TemplateDir{
    LocalDirectory: "./templates",
    IndexTemplateName: "index.tmpl.html",
}
handler, err := NewTemplateDirHandlerFromConfig(config)
server.AddHandler(handler, "/content")
```

## Error Handling

All handlers handle errors gracefully and return appropriate error types that should be checked:
- Invalid configuration returns initialization errors
- Registration errors are returned by the Serve method
- Runtime errors are handled through Echo's error handling system

## Integration with Echo

The handlers integrate with Echo's routing system:
- Use Echo's context for request handling
- Support middleware for authentication and logging
- Compatible with Echo's error handling
- Support streaming responses

## Further Reading

- [Go Template Documentation](https://golang.org/pkg/text/template/)
- [Echo Template Guide](https://echo.labstack.com/guide/templates)
- [Markdown Processing](https://github.com/gomarkdown/markdown)

# Command Handlers

Parka provides three specialized handlers for serving commands: `CommandHandler`, `CommandDirHandler`, and `GenericCommandHandler`. These handlers enable exposing commands as HTTP endpoints with various output formats and interactive UIs.

## GenericCommandHandler

The `GenericCommandHandler` is the base handler that provides core functionality for serving commands over HTTP. It's used internally by both `CommandHandler` and `CommandDirHandler`.

### Structure

```go
type GenericCommandHandler struct {
    Stream          bool
    AdditionalData  map[string]interface{}
    ParameterFilter *config.ParameterFilter
    TemplateName    string
    IndexTemplateName string
    TemplateLookup   render.TemplateLookup
    BasePath         string
    preMiddlewares   []middlewares.Middleware
    postMiddlewares  []middlewares.Middleware
    middlewares      []middlewares.Middleware
}
```

- `Stream`: Whether to use row-based streaming output (true by default)
- `AdditionalData`: Extra data passed to templates
- `ParameterFilter`: Configuration for parameter filtering, defaults, and overrides
- `TemplateName`: Template for rendering command output
- `IndexTemplateName`: Template for rendering command indexes
- `TemplateLookup`: Interface for finding templates
- `BasePath`: Base URL path for the handler
- `preMiddlewares`: Middleware chain to run before parameter filter middlewares
- `postMiddlewares`: Middleware chain to run after parameter filter middlewares

### Endpoints

The handler provides several endpoints for different output formats:

1. `/data/*`: Returns command output in JSON format
2. `/text/*`: Returns command output as plain text
3. `/streaming/*`: Streams command output using Server-Sent Events (SSE)
4. `/datatables/*`: Displays command output in an interactive DataTables UI
5. `/download/*`: Allows downloading command output in various formats

### Configuration Options

1. `WithTemplateName(name string)`: Sets the template for command output
2. `WithParameterFilter(filter *config.ParameterFilter)`: Configures parameter handling
3. `WithMergeAdditionalData(data map[string]interface{}, override bool)`: Adds template data
4. `WithPreMiddlewares(middlewares ...middlewares.Middleware)`: Add middlewares to run before parameter filter middlewares
5. `WithPostMiddlewares(middlewares ...middlewares.Middleware)`: Add middlewares to run after parameter filter middlewares

Example:

```go
handler := NewGenericCommandHandler(
    WithTemplateName("command.tmpl.html"),
    WithParameterFilter(filter),
    WithPreMiddlewares(myPreMiddleware1, myPreMiddleware2),
    WithPostMiddlewares(myPostMiddleware1, myPostMiddleware2),
)
```

The middlewares will be executed in this order:
1. Pre-middlewares (in the order they were added)
2. Parameter filter middlewares
3. Post-middlewares (in the order they were added)

## CommandHandler

The `CommandHandler` is designed to serve a single command with multiple output formats.

### Structure

```go
type CommandHandler struct {
    GenericCommandHandler
    DevMode bool
    Command cmds.Command
}
```

- Inherits all functionality from `GenericCommandHandler`
- `DevMode`: Enables development features like template reloading
- `Command`: The command to be served

### Configuration Options

1. `WithDevMode(devMode bool)`: Enables development mode
   ```go
   handler := NewCommandHandler(cmd,
       WithDevMode(true),
   )
   ```

2. `WithGenericCommandHandlerOptions(options ...GenericCommandHandlerOption)`: Adds generic options
   ```go
   handler := NewCommandHandler(cmd,
       WithGenericCommandHandlerOptions(
           WithTemplateName("command.tmpl.html"),
           WithParameterFilter(filter),
       ),
   )
   ```

### Creation Methods

1. Basic creation with options:
   ```go
   handler := NewCommandHandler(myCommand, options...)
   ```

2. Creation from configuration:
   ```go
   handler, err := NewCommandHandlerFromConfig(config, loader, options...)
   ```

### Usage

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create your command
cmd := &MyCommand{}

// Create and configure the handler
handler := NewCommandHandler(cmd,
    WithDevMode(true),
    WithGenericCommandHandlerOptions(
        WithTemplateName("command.tmpl.html"),
        WithParameterFilter(filter),
    ),
)

// Register the handler with a URL path
err = handler.Serve(server, "/my-command")
if err != nil {
    return err
}
```

## CommandDirHandler

The `CommandDirHandler` serves multiple commands from a repository, providing automatic routing and discovery.

### Structure

```go
type CommandDirHandler struct {
    GenericCommandHandler
    DevMode    bool
    Repository *repositories.Repository
}
```

- Inherits all functionality from `GenericCommandHandler`
- `DevMode`: Enables development features
- `Repository`: The command repository to serve

### Configuration Options

1. `WithDevMode(devMode bool)`: Enables development mode
   ```go
   handler := NewCommandDirHandler(
       WithDevMode(true),
   )
   ```

2. `WithRepository(r *repositories.Repository)`: Sets the command repository
   ```go
   handler := NewCommandDirHandler(
       WithRepository(myRepo),
   )
   ```

### Configuration File Example

```yaml
routes:
  - path: /commands
    commandDirectory:
      includeDefaultRepositories: true
      repositories:
        - ~/code/my-commands
      templateLookup:
        directories:
          - ~/templates
      indexTemplateName: index.tmpl.html
      defaults:
        flags:
          limit: 100
      overrides:
        layers:
          glazed:
            filter:
              - id
              - name
      additionalData:
        title: "My Commands"
```

### Usage

```go
// Create a new Parka server
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
)
if err != nil {
    return err
}

// Create and configure your repository
repo := repositories.NewRepository()
repo.AddCommand(commands.NewHelloCommand())
// Add more commands...

// Create and configure the handler
handler, err := NewCommandDirHandler(
    WithRepository(repo),
    WithDevMode(true),
    WithGenericCommandHandlerOptions(
        WithIndexTemplateName("commands/index.tmpl.html"),
        WithTemplateName("commands/view.tmpl.html"),
    ),
)
if err != nil {
    return err
}

// Register the handler with a base path
err = handler.Serve(server, "/commands")
if err != nil {
    return err
}
```

## Best Practices

1. **Command Organization**:
   - Group related commands in repositories
   - Use clear naming conventions
   - Provide comprehensive command documentation
   - Use appropriate output formats for different use cases

2. **Security**:
   - Validate command parameters
   - Use parameter filters to restrict access
   - Consider authentication for sensitive commands
   - Implement proper error handling

3. **Performance**:
   - Use streaming for large outputs
   - Enable caching when appropriate
   - Consider rate limiting for heavy commands
   - Monitor command execution times

## Examples

### Serving a Single Command

```go
cmd := &MyCommand{}
handler := NewCommandHandler(cmd,
    WithDevMode(true),
    WithGenericCommandHandlerOptions(
        WithTemplateName("command.tmpl.html"),
        WithParameterFilter(filter),
    ),
)
server.AddHandler(handler, "/my-command")
```

### Serving a Command Repository

```go
repo := repositories.NewRepository()
repo.AddCommandFromFile("./commands/my-command.yaml")

handler, err := NewCommandDirHandler(
    WithRepository(repo),
    WithDevMode(true),
    WithGenericCommandHandlerOptions(
        WithIndexTemplateName("index.tmpl.html"),
        WithTemplateName("command.tmpl.html"),
    ),
)
server.AddHandler(handler, "/commands")
```

### Configuration-based Setup

```go
config := &config.CommandDir{
    IncludeDefaultRepositories: true,
    Repositories: []string{"./commands"},
    IndexTemplateName: "index.tmpl.html",
}
handler, err := NewCommandDirHandlerFromConfig(config)
server.AddHandler(handler, "/api")
```

## Error Handling

All handlers handle errors gracefully and return appropriate error types that should be checked:
- Invalid configuration returns initialization errors
- Registration errors are returned by the Serve method
- Runtime errors are handled through Echo's error handling system

## Integration with Echo

The handlers integrate with Echo's routing system:
- Use Echo's context for request handling
- Support middleware for authentication and logging
- Compatible with Echo's error handling
- Support streaming responses

## Further Reading

- [Command Repository Documentation](./command-repository.md)
- [Parameter Filtering](./parameter-filtering.md)
- [DataTables Integration](./datatables.md)
- [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) 