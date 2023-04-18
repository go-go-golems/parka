package glazed

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters/csv"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/formatters/table"
	"github.com/go-go-golems/glazed/pkg/formatters/template"
	"github.com/go-go-golems/glazed/pkg/formatters/yaml"
	"net/http"
)

// CreateProcessorFunc is a simple func type to create a cmds.GlazeProcessor and formatters.OutputFormatter out of a CommandContext.
type CreateProcessorFunc func(c *gin.Context, pc *CommandContext) (
	cmds.Processor,
	string, // content type
	error,
)

type HandleOptions struct {
	ParserOptions   []ParserOption
	Handlers        []CommandHandlerFunc
	CreateProcessor CreateProcessorFunc
}

type HandleOption func(*HandleOptions)

func NewHandleOptions(options []HandleOption) *HandleOptions {
	opts := &HandleOptions{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

func WithParserOptions(parserOptions ...ParserOption) HandleOption {
	return func(o *HandleOptions) {
		o.ParserOptions = parserOptions
	}
}

func WithHandlers(handlers ...CommandHandlerFunc) HandleOption {
	return func(o *HandleOptions) {
		o.Handlers = handlers
	}
}

func WithCreateProcessor(createProcessor CreateProcessorFunc) HandleOption {
	return func(o *HandleOptions) {
		o.CreateProcessor = createProcessor
	}
}

// NewGinHandlerFromCommandHandlers returns a gin.HandlerFunc that is responsible for
// running the provided command, parsing the necessary context from the provided handlers.
// This context is then used to create a cmds.Processor and to provide
// the necessary parameters and layers to the command, calling Run.
//
// TODO(manuel, 2023-04-16) Here we want to pass handlers that can modify the resulting output
// For example, take the HTML and add a page around it. Take the response and render it into a template
// (although that might be able to get done with the standard setup).
//
// NOTE(manuel, 2023-04-16)
func NewGinHandlerFromCommandHandlers(
	cmd cmds.GlazeCommand,
	opts *HandleOptions,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		pc := NewCommandContext(cmd)

		for _, h := range opts.Handlers {
			err := h(c, pc)
			if err != nil {
				_ = c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		var gp cmds.Processor

		var contentType string

		var err error
		if opts.CreateProcessor != nil {
			// TODO(manuel, 2023-03-02) We might want to switch on the requested content type here too
			// This would be done by passing in a handler that configures the glazed layer accordingly.
			gp, contentType, err = opts.CreateProcessor(c, pc)
		} else {
			gp, err = SetupProcessor(pc)
		}
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		err = cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// NOTE(manuel, 2023-04-16) API design wise, we might want to reuse gin.HandlerFunc here for lower processing
		// For example, computing the response (?) I'm not sure this makes sense

		of := gp.OutputFormatter()

		if contentType == "" {
			switch of_ := of.(type) {
			case *json.OutputFormatter:
				contentType = "application/json"
			case *csv.OutputFormatter:
				contentType = "text/csv"
			case *table.OutputFormatter:
				//exhaustive:ignore
				switch of_.TableFormat {
				case "html":
					contentType = "text/html"
				case "markdown":
					contentType = "text/markdown"
				default:
					contentType = "text/plain"
				}
			case *yaml.OutputFormatter:
				contentType = "application/x-yaml"
			case *template.OutputFormatter:
				// TODO(manuel, 2023-03-02) Unclear how to render HTML templates or text templates here
				// probably the best idea is to have the formatter return a content type anyway
				contentType = "text/html"
			}
		}

		s, err := of.Output()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Status(200)
		c.Writer.Header().Set("Content-Type", contentType)
		_, err = c.Writer.Write([]byte(s))
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

// SetupProcessor creates a new cmds.GlazeProcessor. It uses the parsed layer glazed if present, and return
// a simple JsonOutputFormatter and standard glazed processor otherwise.
func SetupProcessor(pc *CommandContext) (*cmds.GlazeProcessor, error) {
	l, ok := pc.ParsedLayers["glazed"]
	if ok {
		gp, err := cli.SetupProcessor(l.Parameters)
		return gp, err
	}

	of := json.NewOutputFormatter(
		json.WithOutputIndividualRows(true),
	)
	gp := cmds.NewGlazeProcessor(of)

	return gp, nil
}
