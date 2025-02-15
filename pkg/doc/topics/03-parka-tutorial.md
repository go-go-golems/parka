---
Title: Building a Parka Server Tutorial
Slug: parka-tutorial
Short: A comprehensive tutorial on building a Parka server from scratch, covering basic setup to advanced features
Topics:
- tutorial
- server
- handlers
- templates
- commands
- web development
Commands:
- NewServer
- NewStaticDirHandler
- NewTemplateHandler
- NewCommandHandler
Flags:
- WithPort
- WithAddress
- WithGzip
- WithLocalPath
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Building a Parka Server: A Comprehensive Tutorial

This tutorial will guide you through building a Parka server from scratch, covering everything from basic setup to advanced features. We'll create a complete application that demonstrates the various capabilities of Parka.

## The Big Picture

Before diving into the implementation, let's understand how Parka works and how its components fit together.

### Core Concepts

Parka is built around several key concepts:

1. **Server Core**: The central server built on Echo, providing HTTP routing and middleware support
2. **Handlers**: Specialized components that serve different types of content:
   - Static handlers for files and directories
   - Template handlers for dynamic HTML content
   - Command handlers for exposing CLI tools as web services
3. **Commands**: CLI tools that can be exposed via HTTP endpoints with various output formats
4. **Repository**: A collection of commands that can be served together
5. **Middleware**: Components that process requests/responses (logging, security, etc.)

### Architecture Overview

```
┌─────────────────────────────────────────────────┐
│                  Parka Server                   │
├─────────────┬─────────────────┬────────────────┤
│  Static     │    Template     │    Command     │
│  Handlers   │    Handlers     │    Handlers    │
├─────────────┴─────────────────┴────────────────┤
│                Echo Framework                   │
├──────────────────────────────────────────────┬─┤
│              Command Repository              │M││
├──────────────────────────────────────────────┤i││
│                   Commands                   │d││
├──────────────────────────────────────────────┤d││
│                Glazed Framework              │w││
└──────────────────────────────────────────────┴─┘
```

### How It All Works Together

1. **Request Flow**:
   - HTTP request comes in
   - Echo routes it to the appropriate handler
   - Handler processes the request using its specific logic
   - Response is formatted and returned

2. **Handler Types**:
   - Static handlers serve files directly
   - Template handlers render dynamic content
   - Command handlers execute CLI tools and format their output

3. **Command Integration**:
   - Commands are defined using the Glazed framework
   - They can be exposed individually or through a repository
   - Output can be formatted as JSON, HTML tables, or downloadable files

4. **Configuration**:
   - Server settings control basic HTTP behavior
   - Handler configurations define content serving
   - Command settings control CLI tool behavior

Now that we understand the big picture, let's build our application step by step.

## Prerequisites

- Go 1.18 or later
- Basic understanding of Go and web development
- Familiarity with command-line applications

## Project Setup

First, let's create a new Go project. This structure will help us organize our code according to Parka's architecture:

```bash
mkdir my-parka-app
cd my-parka-app
go mod init my-parka-app
```

Add the required dependencies:

```bash
go get github.com/go-go-golems/parka      # Core Parka framework
go get github.com/labstack/echo/v4         # Web framework
go get github.com/spf13/cobra              # CLI framework
```

### Why These Dependencies?

- **Parka**: The main framework that ties everything together
- **Echo**: A high-performance web framework that provides routing and middleware
- **Cobra**: For building CLI applications that can be exposed via HTTP

## Basic Server Structure

Let's start with a basic server structure. This forms the foundation of our application:

```go
package main

import (
    "context"
    "github.com/go-go-golems/parka/pkg/server"
    "github.com/spf13/cobra"
    "os"
    "os/signal"
)

var rootCmd = &cobra.Command{
    Use:   "my-server",
    Short: "A Parka-based web server",
    Run: func(cmd *cobra.Command, args []string) {
        port, _ := cmd.Flags().GetUint16("port")
        host, _ := cmd.Flags().GetString("host")

        s, err := server.NewServer(
            server.WithPort(port),
            server.WithAddress(host),
            server.WithGzip(),
        )
        cobra.CheckErr(err)

        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
        defer stop()

        err = s.Run(ctx)
        cobra.CheckErr(err)
    },
}

func init() {
    rootCmd.Flags().Uint16("port", 8080, "Port to listen on")
    rootCmd.Flags().String("host", "localhost", "Host to listen on")
}

func main() {
    _ = rootCmd.Execute()
}
```

### Understanding the Server Structure

1. **Command-Line Interface**:
   - Uses Cobra for CLI functionality
   - Provides flags for configuration
   - Handles graceful shutdown

2. **Server Configuration**:
   - Port and host settings
   - Gzip compression for better performance
   - Context handling for clean shutdown

3. **Signal Handling**:
   - Captures interrupt signals
   - Ensures graceful shutdown
   - Prevents resource leaks

## Static File Serving

```bash
mkdir -p static/css static/js
```

Static file serving is essential for web applications. Here's how Parka handles it:

### Why Static File Serving?

- Serves CSS, JavaScript, images, and other assets
- Improves performance with proper caching
- Separates static content from dynamic rendering

### How It Works

1. **File System Abstraction**:
   - Uses Go's `fs.FS` interface
   - Supports both local and embedded files
   - Handles path resolution securely

2. **Handler Configuration**:
   - Maps URL paths to filesystem locations
   - Handles directory listings (optional)
   - Manages content types automatically

Let's add static file serving for CSS, JavaScript, and other assets:

```go
// In your Run function
staticHandler := static_dir.NewStaticDirHandler(
    static_dir.WithLocalPath("./static"),
)
err = staticHandler.Serve(s, "/assets")
if err != nil {
    return err
}
```

## Template Support

Templates are crucial for generating dynamic HTML content. Parka's template system provides:

Create a templates directory for your HTML templates:

```bash
mkdir -p templates/layouts templates/pages
```

Create a base layout (`templates/layouts/base.tmpl.html`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Title }}</title>
    <link rel="stylesheet" href="/assets/css/style.css">
</head>
<body>
    <header>
        <h1>{{ .Title }}</h1>
    </header>
    <main>
        {{ template "content" . }}
    </main>
    <footer>
        <p>&copy; 2024 My Parka App</p>
    </footer>
</body>
</html>
```

### Why Templates?
- Separates HTML structure from logic
- Enables dynamic content generation
- Supports reusable layouts and components

### Template System Features

1. **Layout System**:
   - Base templates for consistent structure
   - Content blocks for page-specific content
   - Partial templates for reusable components

2. **Development Support**:
   - Hot reloading in development mode
   - Clear error messages
   - Template debugging tools

```go
templateHandler := template_dir.NewTemplateDirHandler(
    template_dir.WithLocalDirectory("./templates"),
    template_dir.WithAlwaysReload(true), // For development
)
err = templateHandler.Serve(s, "/")
if err != nil {
    return err
}
```

## Command System

Commands are the core feature of Parka, allowing you to expose CLI tools as web services.

### Why Commands?

- Convert CLI tools to web services
- Provide multiple output formats
- Enable interactive web interfaces
- Support parameter validation and processing

### Command Architecture

1. **Command Definition**:
   - Parameters and flags
   - Input validation
   - Output formatting
   - Documentation

2. **Integration Points**:
   - HTTP endpoints
   - Web UI
   - File downloads
   - Streaming output

Let's create a simple command that we can expose via the web interface. Create `pkg/commands/hello.go`:

```go
package commands

import (
    "context"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/go-go-golems/glazed/pkg/types"
)

type HelloCommand struct {
    cmds.BaseCommand
    name string
}

func NewHelloCommand() *HelloCommand {
    return &HelloCommand{
        BaseCommand: cmds.BaseCommand{
            Name:        "hello",
            Short:       "A friendly greeting",
            Parameters: parameters.ParameterDefinitions{
                {
                    Name:        "name",
                    Type:        types.String,
                    Help:        "Name to greet",
                    Default:     "World",
                    Required:    false,
                },
            },
        },
    }
}

func (c *HelloCommand) Run(ctx context.Context, gp cmds.GlazeProcessor) error {
    return gp.AddRow(ctx, types.NewRowFromMap(map[string]interface{}{
        "message": fmt.Sprintf("Hello, %s!", c.name),
    }))
}
```

## Exposing Commands via HTTP

Now let's expose our command through various endpoints:

```go
// In your Run function
helloCmd := commands.NewHelloCommand()

// JSON API endpoint
s.Router.GET("/api/hello", json.CreateJSONQueryHandler(helloCmd))

// Interactive DataTables UI
s.Router.GET("/hello", datatables.CreateDataTablesHandler(
    helloCmd,
    "hello.tmpl.html",
    "hello",
))

// File download endpoint
s.Router.GET("/download/hello.csv", output_file.CreateGlazedFileHandler(
    helloCmd,
    "hello.csv",
))
```

## Command Directory Handler

The Command Directory Handler manages multiple commands in a structured way.

### Why Use Command Directory?

- Organize multiple commands
- Automatic routing and discovery
- Consistent interface across commands
- Centralized configuration

### Key Features

1. **Repository Management**:
   - Command discovery
   - Version control
   - Documentation generation
   - Parameter handling

2. **Output Formats**:
   - JSON API endpoints
   - Interactive DataTables
   - File downloads
   - Streaming data

```go
repo := repositories.NewRepository()
repo.AddCommand(commands.NewHelloCommand())
// Add more commands...

cmdHandler, err := command_dir.NewCommandDirHandler(
    command_dir.WithRepository(repo),
    command_dir.WithDevMode(true),
    command_dir.WithGenericCommandHandlerOptions(
        generic_command.WithIndexTemplateName("commands/index.tmpl.html"),
        generic_command.WithTemplateName("commands/view.tmpl.html"),
    ),
)
cobra.CheckErr(err)

err = cmdHandler.Serve(s, "/commands")
if err != nil {
    return err
}
```

## Configuration System

A flexible configuration system is essential for managing complex applications.

### Why Configuration Files?

- Separate configuration from code
- Environment-specific settings
- Easy deployment management
- Runtime configuration changes

### Configuration Features

1. **Server Settings**:
   - Network configuration
   - Handler setup
   - Middleware configuration
   - Development options

2. **Route Configuration**:
   - Path mapping
   - Handler selection
   - Command settings
   - Template configuration

Create a configuration file structure (`config/config.yaml`):

```yaml
server:
  port: 8080
  host: localhost

routes:
  - path: /static
    staticDir:
      localPath: ./static

  - path: /
    templateDir:
      localDirectory: ./templates
      indexTemplateName: index.tmpl.html

  - path: /commands
    commandDirectory:
      includeDefaultRepositories: true
      repositories:
        - ./pkg/commands
      templateLookup:
        directories:
          - ./templates/commands
      indexTemplateName: commands/index.tmpl.html
      defaults:
        flags:
          limit: 100
```

Add configuration loading to your server:

```go
type Config struct {
    Server struct {
        Port uint16 `yaml:"port"`
        Host string `yaml:"host"`
    } `yaml:"server"`
    Routes []config.Route `yaml:"routes"`
}

func loadConfig(file string) (*Config, error) {
    data, err := os.ReadFile(file)
    if err != nil {
        return nil, err
    }

    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    return &cfg, err
}

// In your Run function
cfg, err := loadConfig("config/config.yaml")
cobra.CheckErr(err)

s, err := server.NewServer(
    server.WithPort(cfg.Server.Port),
    server.WithAddress(cfg.Server.Host),
)
cobra.CheckErr(err)

for _, route := range cfg.Routes {
    handler, err := route.CreateHandler()
    cobra.CheckErr(err)
    err = handler.Serve(s, route.Path)
    cobra.CheckErr(err)
}
```

## Development Mode

Development mode enhances the development experience with useful features.

### Why Development Mode?

- Faster development cycle
- Better debugging information
- Hot reloading support
- Detailed error messages

### Development Features

1. **Hot Reloading**:
   - Template changes
   - Static files
   - Configuration updates

2. **Debugging**:
   - Detailed logging
   - Stack traces
   - Performance metrics
   - Request inspection

```go
func init() {
    rootCmd.Flags().Bool("dev", false, "Enable development mode")
}

// In your Run function
dev, _ := cmd.Flags().GetBool("dev")

if dev {
    // Enable template reloading
    templateOptions = append(templateOptions,
        render.WithAlwaysReload(true),
    )

    // Use local assets
    serverOptions = append(serverOptions,
        server.WithStaticPaths(fs.NewStaticPath(os.DirFS("static"), "/assets")),
    )

    // Enable detailed logging
    log.Logger = log.Logger.Level(zerolog.DebugLevel)
}
```

## Error Handling and Logging

Proper error handling and logging are crucial for maintaining and debugging applications.

```go
// Create a custom error handler
s.Router.HTTPErrorHandler = func(err error, c echo.Context) {
    code := http.StatusInternalServerError
    if he, ok := err.(*echo.HTTPError); ok {
        code = he.Code
    }

    log.Error().
        Err(err).
        Str("path", c.Path()).
        Int("status", code).
        Msg("Request error")

    if dev {
        // Show detailed error in development
        err = c.JSON(code, map[string]interface{}{
            "error": err.Error(),
            "stack": fmt.Sprintf("%+v", err),
        })
    } else {
        // Show generic error in production
        err = c.JSON(code, map[string]interface{}{
            "error": http.StatusText(code),
        })
    }

    if err != nil {
        log.Error().Err(err).Msg("Error sending error response")
    }
}
```

## Security Considerations

Security is a critical aspect of any web application.

### Why Security Middleware?

- Protect against common attacks
- Control access to resources
- Monitor application usage
- Ensure data integrity

### Security Features

1. **Protection Layers**:
   - CORS configuration
   - Rate limiting
   - Request validation
   - Error sanitization

2. **Monitoring**:
   - Request tracking
   - Error logging
   - Access patterns
   - Security events

```go
// Add security middleware
s.Router.Use(middleware.Secure())
s.Router.Use(middleware.CORS())
s.Router.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

// Add request ID tracking
s.Router.Use(middleware.RequestID())

// Add recovery middleware
s.Router.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
    StackSize:  1 << 10, // 1 KB
    LogLevel:   log.ERROR,
    LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
        log.Error().
            Err(err).
            Str("stack", string(stack)).
            Msg("Panic recovered")
        return nil
    },
}))
```

## Complete Project Structure

Your final project structure should look like this:

```
my-parka-app/
├── cmd/
│   └── server/
│       └── main.go
├── config/
│   └── config.yaml
├── pkg/
│   ├── commands/
│   │   └── hello.go
│   └── handlers/
│       └── custom_handlers.go
├── static/
│   ├── css/
│   │   └── style.css
│   └── js/
│       └── main.js
├── templates/
│   ├── layouts/
│   │   └── base.tmpl.html
│   ├── pages/
│   │   └── index.tmpl.html
│   └── commands/
│       ├── index.tmpl.html
│       └── view.tmpl.html
├── go.mod
└── go.sum
```

## Running the Server

Start the server in development mode:

```bash
go run cmd/server/main.go --dev
```

For production:

```bash
go build -o my-server cmd/server/main.go
./my-server
```

## Further Enhancements

1. Add authentication and authorization
2. Implement WebSocket support for real-time updates
3. Add database integration
4. Create a custom UI theme
5. Add monitoring and metrics
6. Implement caching
7. Add API documentation using Swagger/OpenAPI

## Troubleshooting

Common issues and solutions:

1. Template not found
   - Check template paths
   - Verify template lookup configuration
   - Enable development mode for detailed errors

2. Static files not serving
   - Verify file permissions
   - Check static path configuration
   - Ensure files exist in the correct location

3. Command not found
   - Verify command registration
   - Check repository configuration
   - Enable debug logging

## Further Reading

- [Parka Server Documentation](./01-parka-server.md)
- [Handlers Documentation](./02-handlers.md)
- [Echo Framework Documentation](https://echo.labstack.com/)
- [Glazed Command Documentation](https://github.com/go-go-golems/glazed)

## Putting It All Together

Now that we understand each component, here's how they work together in a typical request:

1. **HTTP Request Arrives**:
   ```
   GET /commands/hello?name=World
   ```

2. **Request Processing**:
   - Echo routes to correct handler
   - Command handler parses parameters
   - Command executes with parameters
   - Output is formatted according to endpoint

3. **Response Generation**:
   - Handler formats command output
   - Middleware processes response
   - Content is sent to client

This flow combines all the components we've built into a cohesive system. 