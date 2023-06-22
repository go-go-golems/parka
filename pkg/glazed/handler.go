package glazed

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"io"
	"net/http"
	"os"
)

// CreateProcessorFunc is a simple func type to create a cmds.GlazeProcessor
// and formatters.OutputFormatter out of a CommandContext.
//
// This is so that we can create a processor that is configured based on the input
// data provided in CommandContext. For example, the user might want to request a specific response
// format through a query argument or through a header.
type CreateProcessorFunc func(c *gin.Context, pc *CommandContext) (
	processor.Processor,
	error,
)

// HandleOptions groups all the settings for a gin handler that handles a glazed command.
type HandleOptions struct {
	// ParserOptions are passed to the given parser (the thing that gathers the glazed.Command
	// flags and arguments.
	ParserOptions []parser.ParserOption

	// Handlers are run right at the start of the gin.Handler to build up the CommandContext based on the
	// gin.Context. They can be chained because they get passed the previous CommandContext.
	//
	// NOTE(manuel, 2023-06-22) We currently use a single CommandHandler, which is created with NewParserCommandHandlerFunc.
	// This creates a command handler that uses a parser.Parser to parse the gin.Context and return a CommandContext.
	// For example, the FormParser will parse command parameters passed as a HTML form.
	//
	// While we currently only use a single handler, the current setup allows us to chain a middleware of handlers.
	// This would potentially allow us to catch parse errors and return an appropriate error template
	// I'm not entirely sure if this all makes sense.
	Handlers []CommandHandlerFunc

	// CreateProcessor takes a gin.Context and a CommandContext and returns a processor.Processor (and a content-type)
	CreateProcessor CreateProcessorFunc

	// This is the actual gin output writer
	Writer io.Writer
}

type HandleOption func(*HandleOptions)

func (h *HandleOptions) Copy(options ...HandleOption) *HandleOptions {
	ret := &HandleOptions{
		ParserOptions:   h.ParserOptions,
		Handlers:        h.Handlers,
		CreateProcessor: h.CreateProcessor,
		Writer:          h.Writer,
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

func NewHandleOptions(options []HandleOption) *HandleOptions {
	opts := &HandleOptions{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

func WithParserOptions(parserOptions ...parser.ParserOption) HandleOption {
	return func(o *HandleOptions) {
		o.ParserOptions = parserOptions
	}
}

func WithHandlers(handlers ...CommandHandlerFunc) HandleOption {
	return func(o *HandleOptions) {
		o.Handlers = handlers
	}
}

func WithWriter(w io.Writer) HandleOption {
	return func(o *HandleOptions) {
		o.Writer = w
	}
}

func WithCreateProcessor(createProcessor CreateProcessorFunc) HandleOption {
	return func(o *HandleOptions) {
		o.CreateProcessor = createProcessor
	}
}

func CreateJSONProcessor(_ *gin.Context, pc *CommandContext) (
	processor.Processor,
	error,
) {
	l, ok := pc.ParsedLayers["glazed"]
	l.Parameters["output"] = "json"

	var gp *processor.GlazeProcessor
	var err error

	if ok {
		gp, err = settings.SetupProcessor(l.Parameters)
	} else {
		gp, err = settings.SetupProcessor(map[string]interface{}{
			"output": "json",
		})
	}

	if err != nil {
		return nil, err
	}

	return gp, nil
}

// GinHandleGlazedCommand returns a gin.HandlerFunc that is responsible for
// running the provided command, parsing the necessary context from the provided handlers.
// This context is then used to create a cmds.Processor and to provide
// the necessary parameters and layers to the command, calling Run.
//
// TODO(manuel, 2023-04-16) Here we want to pass handlers that can modify the resulting output
// For example, take the HTML and add a page around it. Take the response and render it into a template
// (although that might be able to get done with the standard setup).
//
// NOTE(manuel, 2023-04-16)
func GinHandleGlazedCommand(
	cmd cmds.GlazeCommand,
	opts *HandleOptions,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := runGlazeCommand(c, cmd, opts)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.Status(200)
	}
}

// GinHandleGlazedCommandWithOutputFile returns a gin.HandlerFunc that is responsible for
// running the GlazeCommand, and then returning the output file as an attachment.
// This usually requires the caller to provide a temporary file path.
func GinHandleGlazedCommandWithOutputFile(
	cmd cmds.GlazeCommand,
	outputFile string,
	fileName string,
	opts *HandleOptions,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		buf := &bytes.Buffer{}
		opts_ := opts.Copy(WithWriter(buf))
		err := runGlazeCommand(c, cmd, opts_)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.Status(200)

		f, err := os.Open(outputFile)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		c.Writer.Header().Set("Content-Disposition", "attachment; filename="+fileName)

		_, err = io.Copy(c.Writer, f)
		if err != nil {
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}
	}
}

func runGlazeCommand(c *gin.Context, cmd cmds.GlazeCommand, opts *HandleOptions) error {
	pc := NewCommandContext(cmd)

	for _, h := range opts.Handlers {
		err := h(c, pc)
		if err != nil {
			return err
		}
	}

	var gp processor.Processor

	var err error
	if opts.CreateProcessor != nil {
		// TODO(manuel, 2023-03-02) We might want to switch on the requested content type here too
		// This would be done by passing in a handler that configures the glazed layer accordingly.
		gp, err = opts.CreateProcessor(c, pc)
	} else {
		gp, err = SetupProcessor(pc)
	}
	if err != nil {
		return err
	}

	contentType := gp.OutputFormatter().ContentType()

	err = cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
	if err != nil {
		return err
	}

	of := gp.OutputFormatter()

	if opts.Writer == nil {
		c.Writer.Header().Set("Content-Type", contentType)
	}

	var writer io.Writer = c.Writer
	if opts.Writer != nil {
		writer = opts.Writer
	}
	err = of.Output(c, writer)
	if err != nil {
		return err
	}

	return err
}

// SetupProcessor creates a new cmds.GlazeProcessor. It uses the parsed layer glazed if present, and return
// a simple JsonOutputFormatter and standard glazed processor otherwise.
func SetupProcessor(pc *CommandContext, options ...processor.GlazeProcessorOption) (*processor.GlazeProcessor, error) {
	l, ok := pc.ParsedLayers["glazed"]
	if ok {
		gp, err := settings.SetupProcessor(l.Parameters)
		return gp, err
	}

	of := json.NewOutputFormatter(
		json.WithOutputIndividualRows(true),
	)
	gp := processor.NewGlazeProcessor(of, options...)

	return gp, nil
}
