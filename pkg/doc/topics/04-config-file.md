---
Title: Configuring Parka Servers with Config Files
Slug: config-file
Short: Learn how to configure Parka servers using YAML configuration files to define routes, handlers, and their settings
Topics:
- configuration
- yaml
- routes
- handlers
- server
Commands:
- NewConfigFileHandler
- ParseConfig
Flags:
- WithDevMode
- WithRepositoryFactory
- WithAppendCommandDirHandlerOptions
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Configuring Parka Servers with Config Files

Parka servers can be configured using YAML configuration files that define routes, handlers, and their settings. This document explains how to use config files to set up your Parka server, with a focus on integrating Glazed commands and other handlers.

## Overview

A Parka config file allows you to:
- Define multiple routes with different handlers
- Configure static file serving
- Set up template rendering
- Register Glazed commands and command directories
- Configure parameter filters and defaults
- Set up development mode options

## Basic Structure

The configuration file has this basic structure:

```yaml
defaults:
  useParkaStaticFiles: true
  renderer:
    useDefaultParkaRenderer: true
    templateDirectory: "./templates"
    markdownBaseTemplateName: "base.tmpl.html"

routes:
  - path: "/api"
    commandDirectory:
      repositories:
        - "./commands"
      includeDefaultRepositories: true
      templateLookup:
        directories:
          - "./templates"
      indexTemplateName: "commands/index.tmpl.html"
      defaults:
        flags:
          limit: 100
```

## Configuration Sections

### Global Defaults

The `defaults` section configures global settings for the server:

```yaml
defaults:
  # Whether to use Parka's built-in static files (CSS, JS, etc.)
  useParkaStaticFiles: true
  
  # Renderer configuration for templates
  renderer:
    # Use Parka's default renderer (includes markdown support)
    useDefaultParkaRenderer: true
    # Directory containing templates
    templateDirectory: "./templates"
    # Base template for markdown rendering
    markdownBaseTemplateName: "base.tmpl.html"
```

### Routes

Routes define the URL paths and their corresponding handlers. Each route can use one of several handler types:

#### 1. Command Directory Handler

The Command Directory handler serves multiple Glazed commands from a repository:

```yaml
routes:
  - path: "/commands"
    commandDirectory:
      # List of directories containing command definitions
      repositories:
        - "./commands"
        - "./more-commands"
      
      # Include repositories from environment variables
      includeDefaultRepositories: true
      
      # Template configuration
      templateLookup:
        directories:
          - "./templates/commands"
      indexTemplateName: "index.tmpl.html"
      
      # Default parameter values
      defaults:
        flags:
          limit: 100
          format: "table"
      
      # Parameter overrides
      overrides:
        layers:
          glazed:
            filter:
              - id
              - name
      
      # Additional data passed to templates
      additionalData:
        title: "Command Repository"
```

#### 2. Single Command Handler

For serving individual Glazed commands:

```yaml
routes:
  - path: "/hello"
    command:
      name: "hello"
      templateName: "command.tmpl.html"
      defaults:
        flags:
          greeting: "Hello"
```

#### 3. Template Directory Handler

Serves a directory of templates with support for both HTML and Markdown:

```yaml
routes:
  - path: "/docs"
    templateDirectory:
      localDirectory: "./templates"
      indexTemplateName: "index.tmpl.html"
      markdownBaseTemplateName: "base.tmpl.html"
      alwaysReload: true
```

#### 4. Single Template Handler

For serving a single template:

```yaml
routes:
  - path: "/"
    template:
      templateFile: "index.tmpl.html"
      alwaysReload: true
```

#### 5. Static Directory Handler

Serves static files from a directory:

```yaml
routes:
  - path: "/static"
    static:
      localPath: "./static"
```

#### 6. Static File Handler

Serves a single static file:

```yaml
routes:
  - path: "/favicon.ico"
    staticFile:
      localPath: "./static/favicon.ico"
```

## Integration with Glazed Commands

When integrating Glazed commands, you can configure various aspects of their behavior through the config file:

### Parameter Filtering

Control which parameters are exposed and their default values:

```yaml
commandDirectory:
  defaults:
    flags:
      limit: 100
      format: "json"
    layers:
      sql-connection:
        host: "localhost"
        port: 5432
  
  overrides:
    layers:
      glazed:
        filter:
          - id
          - name
```

### Template Configuration

Configure how commands are rendered in the web interface:

```yaml
commandDirectory:
  templateLookup:
    directories:
      - "./templates/commands"
  indexTemplateName: "commands/index.tmpl.html"
  defaultTemplateName: "commands/view.tmpl.html"
```

## Development Mode

Development mode can be enabled through the configuration file or programmatically. It affects various aspects of the server:

- Template reloading
- Static file serving from local directories
- Detailed error messages
- Debug endpoints

Example configuration with development settings:

```yaml
defaults:
  renderer:
    alwaysReload: true

routes:
  - path: "/api"
    commandDirectory:
      devMode: true
      alwaysReload: true
```

## Example Implementation

Here's an example of how to use a config file in your Parka server:

```go
func main() {
    // Read config file
    configData, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Parse config
    configFile, err := config.ParseConfig(configData)
    if err != nil {
        log.Fatal(err)
    }

    // Create server
    server, err := server.NewServer(
        server.WithPort(8080),
        server.WithAddress("localhost"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create config file handler
    cfh := handlers.NewConfigFileHandler(
        configFile,
        handlers.WithDevMode(true),
        handlers.WithRepositoryFactory(myRepositoryFactory),
        handlers.WithAppendCommandDirHandlerOptions(
            command_dir.WithDevMode(true),
        ),
    )

    // Serve
    if err := cfh.Serve(server); err != nil {
        log.Fatal(err)
    }

    // Run server with config file watching
    ctx := context.Background()
    if err := runConfigFileHandler(ctx, server, cfh); err != nil {
        log.Fatal(err)
    }
}
```


## Further Reading

- [Parka Server Documentation](./01-parka-server.md)
- [Handlers Documentation](./02-handlers.md)
- [Glazed Command Tutorial](../../../glazed/prompto/glazed/create-command-tutorial.md) 