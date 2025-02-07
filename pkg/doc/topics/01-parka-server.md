# Parka Server Documentation

The Parka server is a flexible HTTP server built on top of the Echo web framework that provides both static file serving and dynamic template rendering capabilities. This document explains how the server works and how to extend it.

## Core Concepts

The Parka server is built around these main concepts:

1. Static File Serving
2. Template Rendering
3. Custom Route Handlers
4. Middleware Support

## Server Configuration

### Creating a New Server

To create a new Parka server, use the `NewServer` function with desired options:

```go
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
    server.WithGzip(),
    server.WithDefaultParkaRenderer(),
    server.WithDefaultParkaStaticPaths(),
)
```

### Server Options

The server can be configured using various options:

- `WithPort(port uint16)` - Sets the listening port
- `WithAddress(address string)` - Sets the listening address
- `WithGzip()` - Enables Gzip compression
- `WithDefaultParkaRenderer()` - Sets up the default template renderer
- `WithDefaultParkaStaticPaths()` - Configures default static file paths
- `WithStaticPaths(paths ...utils_fs.StaticPath)` - Adds custom static file paths
- `WithDefaultRenderer(r *render.Renderer)` - Sets a custom renderer

## Static File Serving

Static files can be served using the `StaticPaths` configuration. Each static path consists of:

1. A filesystem implementation (can be embed.FS or os.FS)
2. A URL path where the files will be served

Example:

```go
staticPath := utils_fs.NewStaticPath(myFS, "/static")
server, err := server.NewServer(
    server.WithStaticPaths(staticPath),
)
```

## Template Rendering

Parka uses a flexible template rendering system that supports:

1. Multiple template lookups
2. Markdown rendering with Tailwind CSS support
3. Custom base templates
4. Template directory handling

The default renderer can be configured using:

```go
options, err := server.GetDefaultParkaRendererOptions()
renderer, err := render.NewRenderer(options...)
server, err := server.NewServer(
    server.WithDefaultRenderer(renderer),
)
```

## Adding Custom Routes

Since Parka is built on Echo, you can add custom routes using the standard Echo routing system:

```go
s.Router.GET("/api/hello", func(c echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{
        "message": "Hello, World!",
    })
})
```

### Route Groups

You can organize routes using Echo's group feature:

```go
api := s.Router.Group("/api")
api.GET("/users", handleUsers)
api.POST("/users", createUser)
```

## Middleware

Parka comes with some default middleware:

1. Recovery middleware
2. Request logging using zerolog
3. Optional Gzip compression

Adding custom middleware:

```go
s.Router.Use(myCustomMiddleware)
```

## Running the Server

To start the server:

```go
ctx := context.Background()
err := server.Run(ctx)
```

The server supports graceful shutdown through context cancellation.

## Error Handling

Parka uses Echo's error handling system. You can customize error handling by implementing custom error handlers:

```go
s.Router.HTTPErrorHandler = func(err error, c echo.Context) {
    // Custom error handling logic
}
```


## Examples

Here's a complete example of setting up a Parka server with custom routes and middleware:

```go
server, err := server.NewServer(
    server.WithPort(8080),
    server.WithAddress("localhost"),
    server.WithGzip(),
    server.WithDefaultParkaRenderer(),
    server.WithDefaultParkaStaticPaths(),
)
if err != nil {
    log.Fatal(err)
}

// Add custom routes
server.Router.GET("/api/status", func(c echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{
        "status": "healthy",
    })
})

// Add custom middleware
server.Router.Use(middleware.CORS())

// Start the server
ctx := context.Background()
if err := server.Run(ctx); err != nil {
    log.Fatal(err)
}
```

## Further Reading

- [Echo Framework Documentation](https://echo.labstack.com/)