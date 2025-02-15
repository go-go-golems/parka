---
Title: Command Handler Parameter Filtering
Slug: command-handler-parameter-filtering
Short: Learn how to configure and use parameter filtering in Parka command handlers to control parameter handling, defaults, overrides, and filtering
Topics:
- handlers
- commands
- parameter filtering
- configuration
Commands:
- NewParameterFilter
- WithOverrideParameter
- WithDefaultParameter
- WithWhitelistParameters
- WithBlacklistParameters
Flags:
- WithParameterFilter
- WithOverrides
- WithDefaults
- WithWhitelist
- WithBlacklist
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Command Handler Parameter Filtering

This document describes the parameter filtering options available for command handlers in Parka. These options allow you to control how parameters are handled, including setting defaults, overrides, and filtering which parameters are exposed.

## Overview

Parameter filtering can be configured at three levels:
1. Generic Command Handler
2. Command Handler
3. Command Directory Handler

Each handler type supports the same parameter filtering options through the `ParameterFilter` configuration.

## Programmatic Configuration

### Required Imports

```go
import (
    "context"
    "os"

    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/go-go-golems/glazed/pkg/middlewares"
    "github.com/go-go-golems/parka/pkg/handlers"
    "github.com/go-go-golems/parka/pkg/handlers/command"
    "github.com/go-go-golems/parka/pkg/handlers/generic-command"
    "github.com/go-go-golems/parka/pkg/handlers/config"
    "github.com/go-go-golems/parka/pkg/repositories"
)
```

### Package Structure

The main packages you'll interact with are:

- `handlers/config` - Contains the parameter filtering configuration types and options
- `handlers/command` - Command handler implementation
- `handlers/generic-command` - Generic command handler base implementation
- `handlers/command-dir` - Command directory handler implementation
- `repositories` - Command repository management

Common import aliases:
```go
import (
    config "github.com/go-go-golems/parka/pkg/handlers/config"
    command "github.com/go-go-golems/parka/pkg/handlers/command"
    generic_command "github.com/go-go-golems/parka/pkg/handlers/generic-command"
)
```

### Basic Structure

The parameter filter is configured using the `ParameterFilter` struct and its associated types:

```go
type ParameterFilter struct {
    Overrides *LayerParameters
    Defaults  *LayerParameters
    Whitelist *ParameterFilterList
    Blacklist *ParameterFilterList
}

type LayerParameters struct {
    Parameters map[string]string
    Layers    map[string]map[string]interface{}
}
```

### Creating Parameter Filters

```go
// Basic filter creation using options
filter := config.NewParameterFilter(
    config.WithOverrideParameter("limit", "100"),
    config.WithMergeOverrideLayer("glazed", map[string]interface{}{
        "filter": []string{"id", "name"},
    }),
    config.WithDefaultParameter("format", "table"),
    config.WithMergeDefaultLayer("sql-connection", map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    }),
)

// Using replace options
filter := config.NewParameterFilter(
    config.WithReplaceOverrides(&config.LayerParameters{
        Parameters: map[string]string{
            "limit": "100",
        },
        Layers: map[string]map[string]interface{}{
            "glazed": {
                "filter": []string{"id", "name"},
            },
        },
    }),
)

// Merging configurations
filter := config.NewParameterFilter(
    config.WithMergeOverrides(&config.LayerParameters{
        Parameters: map[string]string{
            "limit": "100",
        },
    }),
    config.WithMergeDefaults(&config.LayerParameters{
        Layers: map[string]map[string]interface{}{
            "sql-connection": {
                "host": "localhost",
            },
        },
    }),
)
```

### Available Configuration Options

```go
// Override options
config.WithReplaceOverrides(*LayerParameters)      // Replace all overrides
config.WithMergeOverrides(*LayerParameters)        // Merge with existing overrides
config.WithOverrideParameter(name string, value interface{})   // Set single parameter override
config.WithOverrideParameters(map[string]interface{})          // Set multiple parameter overrides
config.WithMergeOverrideLayer(name string, layer map[string]interface{})  // Merge layer overrides
config.WithReplaceOverrideLayer(name string, layer map[string]interface{}) // Replace layer overrides
config.WithOverrideLayers(map[string]map[string]interface{})   // Set multiple layer overrides

// Default options
config.WithReplaceDefaults(*LayerParameters)       // Replace all defaults
config.WithMergeDefaults(*LayerParameters)         // Merge with existing defaults
config.WithDefaultParameter(name string, value interface{})    // Set single parameter default
config.WithDefaultParameters(map[string]interface{})           // Set multiple parameter defaults
config.WithMergeDefaultLayer(name string, layer map[string]interface{})   // Merge layer defaults
config.WithReplaceDefaultLayer(name string, layer map[string]interface{}) // Replace layer defaults
config.WithDefaultLayers(map[string]map[string]interface{})    // Set multiple layer defaults

// Layer defaults helper
config.WithLayerDefaults(name string, layer map[string]interface{})       // Set defaults for layer, skip if exists

// Whitelist options
config.WithWhitelist(*ParameterFilterList)         // Replace entire whitelist
config.WithWhitelistParameters(...string)          // Add parameters to whitelist
config.WithWhitelistLayers(...string)             // Add layers to whitelist
config.WithWhitelistLayerParameters(layer string, ...string) // Add parameters for specific layer to whitelist

// Blacklist options
config.WithBlacklist(*ParameterFilterList)         // Replace entire blacklist
config.WithBlacklistParameters(...string)          // Add parameters to blacklist
config.WithBlacklistLayers(...string)             // Add layers to blacklist
config.WithBlacklistLayerParameters(layer string, ...string) // Add parameters for specific layer to blacklist
```

### Parameter Filter List Structure

The `ParameterFilterList` is used for both whitelists and blacklists:

```go
type ParameterFilterList struct {
    Layers          []string            // Layers to filter
    LayerParameters map[string][]string // Parameters to filter per layer
    Parameters      []string            // Parameters to filter in default layer
}
```

### Whitelist/Blacklist Examples

```go
// Creating a whitelist
filter := config.NewParameterFilter(
    // Whitelist specific parameters
    config.WithWhitelistParameters("limit", "offset", "format"),
    
    // Whitelist entire layers
    config.WithWhitelistLayers("sql-connection", "glazed"),
    
    // Whitelist specific parameters in a layer
    config.WithWhitelistLayerParameters("glazed", "filter", "output"),
)

// Creating a blacklist
filter := config.NewParameterFilter(
    // Blacklist debug parameters
    config.WithBlacklistParameters("debug", "verbose", "trace"),
    
    // Blacklist sensitive layers
    config.WithBlacklistLayers("secrets", "internal"),
    
    // Blacklist sensitive parameters in a layer
    config.WithBlacklistLayerParameters("sql-connection", "password", "api_key"),
)

// Combining whitelist and blacklist
filter := config.NewParameterFilter(
    // Whitelist allowed parameters
    config.WithWhitelistParameters("limit", "offset"),
    config.WithWhitelistLayerParameters("glazed", "filter"),
    
    // Blacklist sensitive information
    config.WithBlacklistParameters("debug"),
    config.WithBlacklistLayerParameters("sql-connection", "password"),
)

// Using complete filter lists
filter := config.NewParameterFilter(
    config.WithWhitelist(&config.ParameterFilterList{
        Layers: []string{"glazed", "sql-connection"},
        LayerParameters: map[string][]string{
            "glazed": {"filter", "output"},
            "sql-connection": {"host", "port", "database"},
        },
        Parameters: []string{"limit", "offset", "format"},
    }),
    config.WithBlacklist(&config.ParameterFilterList{
        Parameters: []string{"debug", "verbose"},
        LayerParameters: map[string][]string{
            "sql-connection": {"password", "api_key"},
        },
    }),
)
```

### Using Filters with Command Handlers

```go
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(
        config.NewParameterFilter(
            // Allow only specific parameters
            config.WithWhitelistParameters("limit", "offset", "format"),
            config.WithWhitelistLayerParameters("glazed", "filter", "output"),
            
            // Block sensitive parameters
            config.WithBlacklistLayerParameters("sql-connection", "password"),
            
            // Set defaults for allowed parameters
            config.WithDefaultParameter("limit", "100"),
            config.WithMergeDefaultLayer("glazed", map[string]interface{}{
                "filter": []string{"id", "name"},
            }),
        ),
    ),
)
```

### Configuring Command Handlers

#### Generic Command Handler

```go
handler := handlers.NewGenericCommandHandler(
    handlers.WithParameterFilter(
        config.NewParameterFilter(
            config.WithMergeDefaultLayer("glazed", map[string]interface{}{
                "filter": []string{"id", "name"},
            }),
            config.WithMergeOverrideLayer("sql-connection", map[string]interface{}{
                "host": os.Getenv("DB_HOST"),
                "port": os.Getenv("DB_PORT"),
            }),
        ),
    ),
)
```

#### Command Handler

```go
cmd := &MyCommand{}
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(filter),
    handlers.WithLayerDefaults("sql-connection", map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    }),
    handlers.WithLayerOverrides("glazed", map[string]interface{}{
        "filter": []string{"quantity_sold", "sales_usd"},
    }),
)
```

#### Command Directory Handler

```go
repo := repositories.NewRepository()
handler := handlers.NewCommandDirHandler(
    handlers.WithRepository(repo),
    handlers.WithParameterFilter(filter),
    handlers.WithLayerConfiguration("sql-connection", &SQLConnectionConfig{
        Host:     os.Getenv("DB_HOST"),
        Port:     os.Getenv("DB_PORT"),
        User:     "ttc_analytics",
        Password: os.Getenv("DB_PASSWORD"),
    }),
)
```

### Layer-Specific Configuration

```go
// Creating layer-specific defaults
layerDefaults := layers.NewParameterLayerDefaults().
    WithLayer("glazed", map[string]interface{}{
        "filter": []string{"id", "name"},
    }).
    WithLayer("sql-connection", map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    })

// Creating layer-specific overrides
layerOverrides := layers.NewParameterLayerOverrides().
    WithLayer("dbt", map[string]interface{}{
        "dbt-profile": "ttc.analytics",
    }).
    WithLayer("glazed", map[string]interface{}{
        "filter": []string{"quantity_sold", "sales_usd"},
    })

// Applying to a handler
handler := handlers.NewCommandHandler(cmd,
    handlers.WithLayerDefaults(layerDefaults),
    handlers.WithLayerOverrides(layerOverrides),
)
```

### Environment and External Values

```go
// Environment variables
envConfig := config.NewEnvironmentConfig().
    WithVariable("DB_HOST", "localhost").
    WithVariable("DB_PORT", "5432")

// AWS SSM parameters
ssmConfig := config.NewAWSSSMConfig().
    WithParameter("DB_PASSWORD", "/prod/db/password")

// Applying to a handler
handler := handlers.NewCommandHandler(cmd,
    handlers.WithEnvironmentConfig(envConfig),
    handlers.WithAWSSSMConfig(ssmConfig),
    handlers.WithLayerOverrides("sql-connection", map[string]interface{}{
        "host": config.EnvVar("DB_HOST"),
        "port": config.EnvVar("DB_PORT"),
        "password": config.SSMParam("DB_PASSWORD"),
    }),
)
```

### Dynamic Configuration

```go
// Creating a dynamic parameter filter
filter := config.NewDynamicParameterFilter(
    config.WithFilterFunc(func(ctx context.Context, name string, value interface{}) (interface{}, error) {
        // Custom filtering logic
        return value, nil
    }),
    config.WithValidationFunc(func(ctx context.Context, name string, value interface{}) error {
        // Custom validation logic
        return nil
    }),
)

// Adding middleware for parameter processing
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(filter),
    handlers.WithMiddleware(
        middlewares.NewParameterProcessingMiddleware().
            WithPreProcessor(func(ctx context.Context, params map[string]interface{}) error {
                // Pre-processing logic
                return nil
            }).
            WithPostProcessor(func(ctx context.Context, params map[string]interface{}) error {
                // Post-processing logic
                return nil
            }),
    ),
)
```

### Complete Example

Here's a comprehensive example showing various programmatic configuration options:

```go
func NewAnalyticsHandler(ctx context.Context) (*handlers.CommandHandler, error) {
    // Create base parameter filter
    filter := config.NewParameterFilter().
        WithDefaults(map[string]interface{}{
            "flags": map[string]interface{}{
                "limit": 100,
                "format": "table",
            },
        }).
        WithWhitelist("limit", "offset", "format").
        WithBlacklist("debug", "verbose")

    // Configure layers
    layerConfig := layers.NewParameterLayers(
        layers.WithDefaults(map[string]interface{}{
            "glazed": map[string]interface{}{
                "filter": []string{"id", "timestamp"},
            },
        }),
        layers.WithOverrides(map[string]interface{}{
            "sql-connection": map[string]interface{}{
                "host": config.EnvVar("SQLETON_HOST"),
                "port": config.EnvVar("SQLETON_PORT"),
                "user": "ttc_analytics",
                "password": config.SSMParam("DB_PASSWORD_SSM_KEY"),
                "schema": "ttc_analytics",
                "database": "ttc_analytics",
                "db-type": "mysql",
            },
            "dbt": map[string]interface{}{
                "dbt-profile": "ttc.analytics",
            },
            "glazed": map[string]interface{}{
                "filter": []string{"quantity_sold", "sales_usd"},
            },
        }),
    )

    // Create command
    cmd := &AnalyticsCommand{}

    // Create and configure handler
    handler := handlers.NewCommandHandler(cmd,
        handlers.WithParameterFilter(filter),
        handlers.WithLayers(layerConfig),
        handlers.WithMiddleware(
            middlewares.NewLoggingMiddleware(),
            middlewares.NewValidationMiddleware(),
            middlewares.NewParameterProcessingMiddleware(),
        ),
        handlers.WithErrorHandler(func(ctx context.Context, err error) error {
            log.Error().Err(err).Msg("Command execution failed")
            return err
        }),
    )

    return handler, nil
}
```

### Best Practices for Programmatic Configuration

1. **Type Safety**
   - Use strongly typed configuration structs where possible
   - Implement validation for parameter values
   - Use constants for parameter names

2. **Error Handling**
   - Always check errors from configuration methods
   - Provide meaningful error messages
   - Implement proper cleanup in error cases

3. **Testing**
   - Create test configurations
   - Mock external dependencies
   - Verify parameter filtering behavior

4. **Middleware**
   - Use middleware for cross-cutting concerns
   - Implement proper middleware ordering
   - Keep middleware focused and composable

## Parameter Filter Configuration

The parameter filter configuration can include:

- `defaults`: Default values for parameters
- `overrides`: Values that override any user input
- `whitelist`: List of parameters that are allowed
- `blacklist`: List of parameters that are blocked

### Structure

```yaml
commandDirectory:
  defaults:
    flags:
      paramName: value
    layers:
      layerName:
        paramName: value
  overrides:
    layers:
      layerName:
        paramName: value
  whitelist:
    - param1
    - param2
  blacklist:
    - param3
    - param4
```

## Default Values

Default values are applied when a parameter is not provided by the user. They can be specified for both flags and layer parameters.

### Flag Defaults

```yaml
defaults:
  flags:
    limit: 1337
    offset: 0
    format: "table"
```

### Layer Defaults

```yaml
defaults:
  layers:
    glazed:
      filter:
        - id
        - name
    sql-connection:
      host: "localhost"
      port: 5432
```

## Parameter Overrides

Overrides force specific parameter values regardless of user input. This is useful for enforcing security policies or ensuring consistent configuration.

```yaml
overrides:
  layers:
    dbt:
      dbt-profile: "ttc.analytics"
    glazed:
      filter:
        - quantity_sold
        - sales_usd
    sql-connection:
      schema: "ttc_analytics"
      database: "ttc_analytics"
      user: "ttc_analytics"
```

## Environment Variables and External Values

Parameters can reference environment variables and external sources:

```yaml
overrides:
  layers:
    sql-connection:
      host:
        _env: SQLETON_HOST
      port:
        _env: SQLETON_PORT
      password:
        _aws_ssm:
          _env: SSM_KEY_DB_PASSWORD
```

## Layer Inheritance

You can use YAML anchors and aliases to share configuration between routes:

```yaml
overrides:
  layers:
    sql-connection: &sql-connection
      host:
        _env: SQLETON_HOST
      port:
        _env: SQLETON_PORT
      user: "ttc_analytics"

# Later in the configuration
- path: /reports/
  commandDirectory:
    overrides:
      layers:
        sql-connection:
          <<: *sql-connection
          schema: "ttc_prod"
          database: "ttc_prod"
```

## Complete Example

Here's a comprehensive example showing various parameter filtering options:

```yaml
routes:
  - path: /analytics/
    commandDirectory:
      includeDefaultRepositories: false
      repositories:
        - /queries/ttc
      
      # Default values for parameters
      defaults:
        flags:
          limit: 100
          format: "table"
        layers:
          glazed:
            filter:
              - id
              - timestamp
      
      # Override specific parameters
      overrides:
        layers:
          sql-connection:
            host:
              _env: SQLETON_HOST
            port:
              _env: SQLETON_PORT
            user: "ttc_analytics"
            password:
              _aws_ssm:
                _env: SSM_KEY_DB_PASSWORD
            schema: "ttc_analytics"
            database: "ttc_analytics"
            db-type: "mysql"
          
          dbt:
            dbt-profile: "ttc.analytics"
          
          glazed:
            filter:
              - quantity_sold
              - sales_usd
      
      # Optional whitelist/blacklist
      whitelist:
        - limit
        - offset
        - format
      blacklist:
        - debug
        - verbose
```

## Best Practices

1. **Security**
   - Use overrides for sensitive configuration like database credentials
   - Blacklist debug or verbose flags in production
   - Use environment variables for configuration that changes between environments

2. **Defaults**
   - Set reasonable defaults for pagination (limit, offset)
   - Configure default output formats
   - Provide sensible filter columns

3. **Layer Configuration**
   - Group related parameters in layers
   - Use YAML anchors for shared configurations
   - Override only necessary parameters in derived configurations

4. **Parameter Filtering**
   - Whitelist parameters that should be exposed to users
   - Blacklist sensitive or dangerous parameters
   - Document which parameters are available/blocked

## Common Use Cases

### Database Connection Configuration

```yaml
overrides:
  layers:
    sql-connection:
      host:
        _env: DB_HOST
      port:
        _env: DB_PORT
      user:
        _env: DB_USER
      password:
        _aws_ssm:
          _env: DB_PASSWORD_SSM_KEY
```

### Output Formatting

```yaml
defaults:
  flags:
    format: "table"
    limit: 1000
  layers:
    glazed:
      filter:
        - id
        - name
        - timestamp
```

### Environment-Specific Settings

```yaml
- path: /prod/
  commandDirectory:
    overrides:
      layers:
        dbt:
          dbt-profile: "prod"
        sql-connection:
          schema: "prod"
          database: "prod"
```

## Troubleshooting

1. **Parameter Not Available**
   - Check if parameter is blacklisted
   - Verify parameter isn't overridden
   - Ensure parameter is whitelisted if whitelist is used

2. **Default Not Applied**
   - Confirm default is in correct section (flags vs layers)
   - Check for overrides that might supersede default
   - Verify parameter name matches exactly

3. **Override Not Working**
   - Verify override is in correct layer
   - Check environment variables are set if used
   - Confirm YAML syntax for anchors and aliases 

## Parameter Filter Middleware Implementation

The parameter filter system in Parka is implemented through a series of middlewares that modify the parameter layers before and after command execution. Here's a detailed look at how these middlewares work and how they can be used effectively.

### Middleware Chain Execution

The middleware system follows these key principles:

1. Middlewares are executed in reverse order of their definition
2. Each middleware can modify both the parameter layers and parsed layers
3. The execution order matters for different types of operations:
   - For modifying parsed layers (e.g., setting values): Call `next` first
   - For modifying parameter layers (e.g., filtering): Call `next` last
4. Middlewares can be added before or after the parameter filter middlewares using `WithPreMiddlewares` and `WithPostMiddlewares`

Example of middleware execution order:

```go
// This chain with pre and post middlewares:
handler := NewGenericCommandHandler(
    WithPreMiddlewares(
        LoggingMiddleware(),      // Run first
        ValidationMiddleware(),    // Run second
    ),
    WithParameterFilter(filter),  // Parameter filter middlewares run third
    WithPostMiddlewares(
        MetricsMiddleware(),      // Run fourth
        AuditMiddleware(),        // Run last
    ),
)

// Executes as:
LoggingMiddleware(
    ValidationMiddleware(
        ParameterFilterMiddlewares(
            MetricsMiddleware(
                AuditMiddleware(
                    Identity
                )
            )
        )
    )
)
```

### Core Middleware Types

#### 1. Whitelist Middlewares

Whitelist middlewares restrict which layers and parameters are available:

```go
// Whitelist entire layers
filter := config.NewParameterFilter(
    config.WithWhitelistLayers("sql", "http"),
)

// Whitelist specific parameters in layers
filter := config.NewParameterFilter(
    config.WithWhitelistLayerParameters("sql", "host", "port", "database"),
    config.WithWhitelistLayerParameters("http", "timeout", "retries"),
)
```

Implementation details:
- `WhitelistLayers`: Removes any layers not in the whitelist
- `WhitelistLayerParameters`: Removes parameters not in the whitelist for each layer
- Both can be applied before or after other middlewares using `First` variants

#### 2. Blacklist Middlewares

Blacklist middlewares exclude specific layers and parameters:

```go
// Blacklist sensitive layers
filter := config.NewParameterFilter(
    config.WithBlacklistLayers("secrets", "internal"),
)

// Blacklist sensitive parameters
filter := config.NewParameterFilter(
    config.WithBlacklistLayerParameters("database", "password", "api_key"),
    config.WithBlacklistLayerParameters("auth", "token", "secret"),
)
```

Implementation details:
- `BlacklistLayers`: Removes specified layers
- `BlacklistLayerParameters`: Removes specified parameters from layers
- Both support `First` variants for execution order control

#### 3. Parameter Update Middlewares

Update middlewares modify parameter values:

```go
// Set overrides
filter := config.NewParameterFilter(
    config.WithOverrideParameter("debug", "false"),
    config.WithMergeOverrideLayer("database", map[string]interface{}{
        "max_connections": 100,
        "timeout": "30s",
    }),
)

// Set defaults
filter := config.NewParameterFilter(
    config.WithDefaultParameter("page_size", "50"),
    config.WithMergeDefaultLayer("http", map[string]interface{}{
        "timeout": "5s",
        "retries": 3,
    }),
)
```

### Advanced Middleware Patterns

#### 1. Conditional Parameter Filtering

Apply different filters based on conditions:

```go
// Production environment filter
if env == "production" {
    filter := config.NewParameterFilter(
        // Strict security in production
        config.WithBlacklistParameters("debug", "trace"),
        config.WithWhitelistLayerParameters("database", 
            "host", "port", "name", "pool_size"),
        config.WithOverrideLayer("logging", map[string]interface{}{
            "level": "error",
            "format": "json",
        }),
    )
} else {
    // Development environment - more permissive
    filter := config.NewParameterFilter(
        config.WithDefaultParameter("debug", "true"),
        config.WithMergeDefaultLayer("database", map[string]interface{}{
            "host": "localhost",
            "port": 5432,
        }),
    )
}
```

#### 2. Layer-Specific Middleware Chains

Apply different middleware chains to different layers:

```go
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(
        config.NewParameterFilter(
            // Database layer gets special treatment
            config.WithWrapWithWhitelistedLayers(
                []string{"database"},
                config.WithOverrideParameter("ssl", "true"),
                config.WithBlacklistParameters("password"),
            ),
            // HTTP layer gets different treatment
            config.WithWrapWithWhitelistedLayers(
                []string{"http"},
                config.WithDefaultParameter("timeout", "10s"),
                config.WithWhitelistParameters("method", "url", "headers"),
            ),
        ),
    ),
)
```

#### 3. Pre and Post Middleware Configuration

Configure different middleware chains for different stages:

```go
// Create middleware chains
preMiddlewares := []middlewares.Middleware{
    // Run before parameter filtering
    middlewares.NewLoggingMiddleware(),
    middlewares.NewValidationMiddleware(),
}

postMiddlewares := []middlewares.Middleware{
    // Run after parameter filtering
    middlewares.NewMetricsMiddleware(),
    middlewares.NewAuditMiddleware(),
}

// Apply to handler
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(filter),
    handlers.WithPreMiddlewares(preMiddlewares...),
    handlers.WithPostMiddlewares(postMiddlewares...),
)
```

#### 4. Comprehensive Middleware Configuration

```go
func NewSecureHandlerWithMiddlewares() *handlers.CommandHandler {
    // Create pre-middlewares for validation and logging
    preMiddlewares := []middlewares.Middleware{
        middlewares.NewValidationMiddleware(
            middlewares.WithRequiredParameters("user_id", "action"),
            middlewares.WithParameterValidation(func(name string, value interface{}) error {
                // Custom validation logic
                return nil
            }),
        ),
        middlewares.NewLoggingMiddleware(
            middlewares.WithLogLevel("debug"),
            middlewares.WithLogFormat("json"),
        ),
    }

    // Create post-middlewares for metrics and auditing
    postMiddlewares := []middlewares.Middleware{
        middlewares.NewMetricsMiddleware(
            middlewares.WithMetricsPrefix("api"),
            middlewares.WithLabels(map[string]string{
                "service": "command-handler",
            }),
        ),
        middlewares.NewAuditMiddleware(
            middlewares.WithAuditLogger(auditLogger),
            middlewares.WithAuditLevel("info"),
        ),
    }

    // Create parameter filter
    filter := config.NewParameterFilter(
        config.WithWhitelistParameters(
            "user_id", "resource_id", "action",
        ),
        config.WithBlacklistParameters(
            "password", "token", "api_key",
        ),
    )

    // Create handler with all middleware chains
    return handlers.NewCommandHandler(cmd,
        handlers.WithParameterFilter(filter),
        handlers.WithPreMiddlewares(preMiddlewares...),
        handlers.WithPostMiddlewares(postMiddlewares...),
    )
}
```

### Best Practices for Parameter Filtering

1. **Security First**
   - Always blacklist sensitive parameters
   - Use whitelists in production environments
   - Override security-critical settings

2. **Configuration Layering**
   - Use defaults for development-friendly values
   - Apply environment-specific overrides
   - Keep security parameters separate from functional ones

3. **Middleware Ordering**
   - Apply blacklists before whitelists
   - Set defaults before overrides
   - Consider using `First` variants for critical filters

4. **Error Handling**
   - Validate parameter values after filtering
   - Provide clear error messages for filtered parameters
   - Log attempted access to restricted parameters

5. **Documentation**
   - Document which parameters are available/restricted
   - Explain the rationale for parameter filtering
   - Provide examples for common use cases 

### Parameter Configuration Examples

```go
// Setting individual parameters
filter := config.NewParameterFilter(
    // Override single parameter
    config.WithOverrideParameter("debug", false),
    
    // Override multiple parameters
    config.WithOverrideParameters(map[string]interface{}{
        "limit": 100,
        "format": "json",
    }),
    
    // Default single parameter
    config.WithDefaultParameter("timeout", "30s"),
    
    // Default multiple parameters
    config.WithDefaultParameters(map[string]interface{}{
        "page_size": 50,
        "sort_order": "desc",
    }),
)

// Configuring layers
filter := config.NewParameterFilter(
    // Merge layer overrides
    config.WithMergeOverrideLayer("sql-connection", map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    }),
    
    // Replace layer overrides
    config.WithReplaceOverrideLayer("http", map[string]interface{}{
        "timeout": "5s",
        "retries": 3,
    }),
    
    // Set multiple layer overrides
    config.WithOverrideLayers(map[string]map[string]interface{}{
        "sql-connection": {
            "host": "localhost",
            "port": 5432,
        },
        "http": {
            "timeout": "5s",
            "retries": 3,
        },
    }),
)

// Combining defaults and overrides
filter := config.NewParameterFilter(
    // Set defaults
    config.WithDefaultParameters(map[string]interface{}{
        "limit": 50,
        "format": "table",
    }),
    config.WithDefaultLayers(map[string]map[string]interface{}{
        "sql-connection": {
            "host": "localhost",
            "port": 5432,
        },
    }),
    
    // Override specific values
    config.WithOverrideParameters(map[string]interface{}{
        "format": "json",
    }),
    config.WithMergeOverrideLayer("sql-connection", map[string]interface{}{
        "ssl": true,
    }),
)

// Complete example with all options
filter := config.NewParameterFilter(
    // Defaults
    config.WithDefaultParameters(map[string]interface{}{
        "limit": 50,
        "format": "table",
    }),
    config.WithDefaultLayers(map[string]map[string]interface{}{
        "sql-connection": {
            "host": "localhost",
            "port": 5432,
        },
        "http": {
            "timeout": "5s",
        },
    }),
    
    // Overrides
    config.WithOverrideParameters(map[string]interface{}{
        "debug": false,
    }),
    config.WithOverrideLayers(map[string]map[string]interface{}{
        "sql-connection": {
            "ssl": true,
            "max_connections": 100,
        },
    }),
    
    // Whitelist
    config.WithWhitelistParameters("limit", "format"),
    config.WithWhitelistLayerParameters("sql-connection", "host", "port", "ssl"),
    
    // Blacklist
    config.WithBlacklistParameters("verbose"),
    config.WithBlacklistLayerParameters("sql-connection", "password"),
)
```

### Using with Command Handlers

```go
handler := handlers.NewCommandHandler(cmd,
    handlers.WithParameterFilter(
        config.NewParameterFilter(
            // Set defaults
            config.WithDefaultParameters(map[string]interface{}{
                "limit": 50,
                "format": "table",
            }),
            config.WithDefaultLayers(map[string]map[string]interface{}{
                "sql-connection": {
                    "host": "localhost",
                    "port": 5432,
                },
            }),
            
            // Override production settings
            config.WithOverrideLayers(map[string]map[string]interface{}{
                "sql-connection": {
                    "host": os.Getenv("DB_HOST"),
                    "port": os.Getenv("DB_PORT"),
                    "ssl": true,
                },
                "logging": {
                    "level": "error",
                    "format": "json",
                },
            }),
            
            // Security filters
            config.WithWhitelistParameters("limit", "format"),
            config.WithBlacklistLayerParameters("sql-connection", "password"),
        ),
    ),
)
``` 