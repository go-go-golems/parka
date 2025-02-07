# Template Rendering in Parka

Parka provides a flexible and powerful template rendering system that supports both HTML and Markdown templates, with features like template lookup, reloading, and directory-based serving. This document explains how the template system works and how to use it effectively.

## Overview

The template rendering system in Parka is built around several key concepts:

1. **Template Lookups**: Interfaces that define how templates are found and loaded
2. **Renderers**: Components that handle the actual template rendering process
3. **Handlers**: Web handlers that serve templates and template directories
4. **Template Types**: Support for both HTML and Markdown templates

## Template Lookup System

The template lookup system is the foundation of Parka's template rendering. It's designed to provide a flexible way to load templates from different sources while supporting development and production needs.

### The TemplateLookup Interface

```go
type TemplateLookup interface {
    // Lookup returns a template by name. If there are multiple names given,
    // implementations may choose how to handle them.
    Lookup(name ...string) (*template.Template, error)

    // Reload reloads all or partial templates. This is useful for development
    // where templates might change without server restart.
    Reload(name ...string) error
}
```

### Lookup Algorithm

The Renderer's lookup process follows these steps:

1. **Template Name Resolution**:
   ```go
   // From renderer.go
   t, err := r.LookupTemplate(templateName+".tmpl.md", templateName+".md", templateName)
   if err != nil {
       return errors.Wrap(err, "error looking up template")
   }
   ```
   - First tries `<name>.tmpl.md` for markdown templates
   - Then tries `<name>.md` for static markdown
   - Then tries `<name>` directly
   - If none found, tries `<name>.tmpl.html` and `<name>.html`

2. **Base Template Resolution** (for markdown):
   ```go
   baseTemplate, err := r.LookupTemplate(r.MarkdownBaseTemplateName)
   if err != nil {
       return errors.Wrap(err, "error looking up base template")
   }
   ```

3. **Template Chain Search**:
   - Iterates through all configured template lookups
   - Returns the first successful match
   - Logs debug information for failed lookups

### Template Lookup Implementations

#### 1. File-based Lookup (`LookupTemplateFromFile`)

Loads a single template file and optionally restricts it to a specific template name.

```go
// Basic usage
lookup := render.NewLookupTemplateFromFile("templates/index.tmpl.html", "")
tmpl, err := lookup.Lookup("index.tmpl.html")

// With specific template name
lookup := render.NewLookupTemplateFromFile("templates/base.tmpl.html", "base")
tmpl, err := lookup.Lookup("base") // Only responds to "base"
```

Example from tests:
```go
// Create a file-based lookup that always returns the same file
lookup := NewLookupTemplateFromFile("templates/tests/test.txt", "")
tmpl, err := lookup.Lookup("any-name.txt") // Will return test.txt content
if err != nil {
    log.Fatal(err)
}
```

#### 2. Directory-based Lookup (`LookupTemplateFromDirectory`)

Loads templates from a directory, reloading on every request.

```go
// Basic usage
lookup := render.NewLookupTemplateFromDirectory("./templates")
tmpl, err := lookup.Lookup("pages/index.html")

// With trailing slash handling
lookup := render.NewLookupTemplateFromDirectory("templates/")
tmpl, err := lookup.Lookup("index.html")
```

Example from tests:
```go
// Create a directory-based lookup
lookup := NewLookupTemplateFromDirectory("templates/tests")
tmpl, err := lookup.Lookup("test.txt")
if err != nil {
    log.Fatal(err)
}

// Execute the template
buf := new(bytes.Buffer)
err = tmpl.Execute(buf, nil)
fmt.Println(buf.String()) // Output: template content
```

#### 3. Filesystem-based Lookup (`LookupTemplateFromFS`)

Most flexible implementation, supporting embedded files and pattern matching.

```go
// Basic usage with default patterns (*.html)
lookup := render.NewLookupTemplateFromFS(
    render.WithFS(os.DirFS("./templates")),
)

// With custom patterns and base directory
lookup := render.NewLookupTemplateFromFS(
    render.WithFS(os.DirFS("./content")),
    render.WithBaseDir("pages"),
    render.WithPatterns("*.md", "*.html"),
    render.WithAlwaysReload(true),
)
```

Example from tests:
```go
// Create a filesystem-based lookup with specific patterns
lookup := NewLookupTemplateFromFS(
    WithFS(os.DirFS("templates/tests")),
    WithPatterns("*.html"),
)
tmpl, err := lookup.Lookup("test.html")
if err != nil {
    log.Fatal(err)
}

// With base directory
lookup := NewLookupTemplateFromFS(
    WithFS(os.DirFS("templates")),
    WithBaseDir("tests"),
    WithPatterns("*.html"),
)
```

### Template Reloading

Each implementation handles reloading differently:

1. **File-based**: Always reloads on lookup
   ```go
   func (l *LookupTemplateFromFile) Reload(name ...string) error {
       return nil // No need to reload, happens on every lookup
   }
   ```

2. **Directory-based**: Always reloads on lookup
   ```go
   func (l *LookupTemplateFromDirectory) Reload(_ ...string) error {
       return nil // No need to reload, happens on every lookup
   }
   ```

3. **Filesystem-based**: Configurable reloading
   ```go
   // Configure reloading
   lookup := NewLookupTemplateFromFS(
       WithAlwaysReload(true), // Reload on every lookup
   )

   // Manual reload
   err := lookup.Reload()
   if err != nil {
       log.Fatal(err)
   }
   ```

Example of dynamic template reloading from tests:
```go
func TestLookupTemplateFromFS_Reload(t *testing.T) {
    l := NewLookupTemplateFromFS()
    
    // Copy a template file
    err := copyFile("templates/tests/test.html", "templates/tests/test_tmpl.html")
    require.NoError(t, err)
    
    // Reload to pick up the new file
    err = l.Reload()
    require.NoError(t, err)
    
    // Lookup should now find the new template
    tmpl, err := l.Lookup("templates/tests/test_tmpl.html")
    require.NoError(t, err)
    
    // Clean up
    os.Remove("templates/tests/test_tmpl.html")
}
```

## Template Rendering Process

The rendering process in Parka follows these steps:

1. **Template Resolution**:
   - Looks for templates in the following order:
     1. `<name>.tmpl.md`
     2. `<name>.md`
     3. `<name>.tmpl.html`
     4. `<name>.html`

2. **Template Processing**:
   - For Markdown templates:
     1. Renders the markdown template with data
     2. Converts markdown to HTML
     3. Wraps the result in a base template (if configured)
   - For HTML templates:
     1. Renders directly with the provided data

3. **Data Injection**:
   - Merges global renderer data with request-specific data
   - Makes data available to templates during rendering

## Template Handlers

Parka provides two main types of template handlers for serving templates over HTTP:

1. **Single Template Handler** (`TemplateHandler`): For serving individual template files
2. **Template Directory Handler** (`TemplateDirHandler`): For serving multiple templates from a directory

For detailed information about these handlers, including their structure, configuration options, and usage examples, see the [Template Handlers Documentation](./02-handlers.md#template-handlers).

## Development Mode Features

When developing with Parka templates, you can enable several features to make development easier:

1. **Always Reload**: Templates are reloaded on every request
2. **Local Directory Override**: Use local files instead of embedded ones
3. **Base Template Override**: Customize the base template for markdown rendering

## Best Practices

1. **Template Organization**:
   - Use `.tmpl.html` for HTML templates that need processing
   - Use `.html` for static HTML files
   - Use `.tmpl.md` for markdown templates that need processing
   - Use `.md` for static markdown files

2. **Template Lookup Configuration**:
   - Use `LookupTemplateFromFS` for production
   - Use `LookupTemplateFromDirectory` for development
   - Use `LookupTemplateFromFile` for single-file cases

3. **Performance Optimization**:
   - Disable `alwaysReload` in production
   - Use pattern matching to limit template scanning
   - Consider using embedded filesystems in production

4. **Development Workflow**:
   - Enable `alwaysReload` during development
   - Use local directories for easy template editing
   - Utilize the markdown base template for consistent styling

## Configuration File Examples

This section shows how to configure template handlers using Parka's configuration file system. For detailed information about the handlers themselves, refer to the [Template Handlers Documentation](./02-handlers.md#template-handlers).

### Single Template Configuration

```yaml
routes:
  - path: "/about"
    template:
      templateFile: "about.tmpl.html"
      alwaysReload: true
```

### Template Directory Configuration

```yaml
routes:
  - path: "/docs"
    templateDirectory:
      localDirectory: "./docs"
      indexTemplateName: "index.tmpl.html"
      markdownBaseTemplateName: "base.tmpl.html"
      alwaysReload: true
```

### Complete Configuration Example

Here's a complete example showing various template configurations:

```yaml
defaults:
  renderer:
    useDefaultParkaRenderer: true
    templateDirectory: "./templates"
    markdownBaseTemplateName: "base.tmpl.html"

routes:
  # Serve a single template
  - path: "/"
    template:
      templateFile: "index.tmpl.html"
      alwaysReload: true

  # Serve a documentation directory
  - path: "/docs"
    templateDirectory:
      localDirectory: "./docs"
      indexTemplateName: "index.tmpl.html"
      markdownBaseTemplateName: "docs-base.tmpl.html"
      alwaysReload: true

  # Serve API documentation
  - path: "/api"
    template:
      templateFile: "api.tmpl.html"
      data:
        title: "API Documentation"
        version: "1.0"
```

And here's the equivalent programmatic setup:

```go
package main

import (
    "github.com/go-go-golems/parka/pkg/render"
    "github.com/go-go-golems/parka/pkg/handlers/template"
    "github.com/go-go-golems/parka/pkg/handlers/template-dir"
    "github.com/go-go-golems/parka/pkg/server"
)

func main() {
    // Create a new server
    server, err := server.NewServer()
    if err != nil {
        panic(err)
    }

    // Set up default renderer
    defaultRenderer, err := render.NewRenderer(
        render.WithMarkdownBaseTemplateName("base.tmpl.html"),
    )
    if err != nil {
        panic(err)
    }

    // Set up index page
    indexHandler := template.NewTemplateHandler(
        "index.tmpl.html",
        template.WithAlwaysReload(true),
    )
    indexHandler.Serve(server, "/")

    // Set up documentation pages
    docsHandler := template_dir.NewTemplateDirHandler(
        template_dir.WithLocalDirectory("./docs"),
        template_dir.WithAlwaysReload(true),
        template_dir.WithAppendRendererOptions(
            render.WithMarkdownBaseTemplateName("docs-base.tmpl.html"),
            render.WithIndexTemplateName("index.tmpl.html"),
        ),
    )
    docsHandler.Serve(server, "/docs")

    // Set up API documentation
    apiHandler := template.NewTemplateHandler(
        "api.tmpl.html",
        template.WithAppendRendererOptions(
            render.WithMergeData(map[string]interface{}{
                "title":   "API Documentation",
                "version": "1.0",
            }),
        ),
    )
    apiHandler.Serve(server, "/api")

    // Start the server
    if err := server.Start(":8080"); err != nil {
        panic(err)
    }
}
```

## Conclusion

Parka's template rendering system provides a flexible and powerful way to serve both HTML and Markdown content. By understanding the different components and their interactions, you can effectively use templates in your Parka applications while maintaining good development practices and performance considerations. 