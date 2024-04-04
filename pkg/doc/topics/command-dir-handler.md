---
Title: Command Directory Handler 
Slug: command-directory-handler
Short: |
  Learn how to configure and use the Command Directory Handler to expose commands as API endpoints, text, downloads, and datatables UI.
Topics:
- api
- configuration
- parka
Commands:
- serve
IsTopLevel: false
ShowPerDefault: false
SectionType: GeneralTopic
---

# Command Directory Handler Configuration

The Command Directory Handler allows you to expose commands as API endpoints, text
outputs, file downloads, and interactive datatables UI. This document will guide you through configuring the Command
Directory Handler using a configuration file and explain how to use it effectively.

## Configuration Using a Configuration File

To configure the Command Directory Handler, you need to create a YAML configuration file that specifies various settings and options. Below is an example of a configuration file and an explanation of its components:

```yaml
routes:
  - path: /
    commandDirectory:
      includeDefaultRepositories: true
      repositories:
        - ~/code/ttc/ttc/sql/sqleton
      templateLookup:
        directories:
          - ~/code/wesen/corporate-headquarters/parka/pkg/glazed/handlers/datatables/templates
      indexTemplateName: index.tmpl.html
      defaults:
        flags:
          limit: 1337
        layers:
          glazed:
            filter:
              - id
      overrides:
        layers:
          dbt:
            dbt-profile: ttc.analytics
          glazed:
            filter:
              - quantity_sold
              - sales_usd
      additionalData:
        foobar: baz
  - path: /analytics
    commandDirectory:
      includeDefaultRepositories: false
      repositories:
        - ~/code/ttc/ttc/sql/sqleton
```

### Key Configuration Options:

- `path`: The URL path where the commands will be exposed.
- `commandDirectory`: A section that defines the settings for the command directory.
    - `includeDefaultRepositories`: A boolean that indicates whether to include default repositories.
    - `repositories`: A list of paths to the command repositories.
    - `templateLookup`: Specifies the directories where templates are located.
    - `indexTemplateName`: The name of the template file for rendering index pages.
    - `defaults`: Default values for flags and other parameters.
    - `overrides`: Overrides for specific parameters.
    - `additionalData`: Additional data to be passed to the templates.

## Using the Command Directory Handler

Once configured, the Command Directory Handler exposes different types of endpoints:

- `/data/*path`: Returns command output in JSON format.
- `/text/*path`: Returns command output as plain text.
- `/streaming/*path`: Streams command output using Server-Sent Events (SSE).
- `/datatables/*path`: Displays command output in an interactive datatable UI.
- `/download/*path.[json|csv|txt|md|...]`: Allows downloading command output as a file.
- `/commands/*path`: Renders a page with available commands and their documentation.
- `/commands`: Renders an index page with links to available commands.

### Example Command Line Usage

To use the Command Directory Handler, you would typically make HTTP requests to the endpoints it exposes. Here's an example of how you might use `curl` to interact with these endpoints:

```sh
# Get JSON data
curl http://localhost:8080/data/my-command

# Get plain text output
curl http://localhost:8080/text/my-command

# Stream output
curl http://localhost:8080/streaming/my-command

# Interact with the datatable UI in a web browser
open http://localhost:8080/datatables/my-command

# Download output as a file
curl -OJ http://localhost:8080/download/my-command/output.txt

# View available commands
open http://localhost:8080/commands
```

Remember to replace `http://localhost:8080` with the actual URL where your application is hosted and `my-command` with
the specific command you want to interact with.




