package glazed

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/formatters"
	"github.com/go-go-golems/glazed/pkg/formatters/json"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/parka/pkg/glazed/parser"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
)

type GinOutputFormatter interface {
	Output(w io.Writer) error
	RegisterMiddlewares(p *middlewares.TableProcessor) error
}

type GinOutputFormatterFactory interface {
	CreateOutputFormatter(c *gin.Context, pc *CommandContext) (GinOutputFormatter, error)
}

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

	// CreateProcessor takes a gin.Context and a CommandContext and returns a processor.TableProcessor (and a content-type)
	OutputFormatterFactory GinOutputFormatterFactory

	// This is the actual gin output writer
	Writer io.Writer
}

type HandleOption func(*HandleOptions)

func (h *HandleOptions) Copy(options ...HandleOption) *HandleOptions {
	ret := &HandleOptions{
		ParserOptions:          h.ParserOptions,
		Handlers:               h.Handlers,
		OutputFormatterFactory: h.OutputFormatterFactory,
		Writer:                 h.Writer,
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

func CreateJSONProcessor(_ *gin.Context, pc *CommandContext) (
	*middlewares.TableProcessor,
	error,
) {
	l, ok := pc.ParsedLayers["glazed"]
	l.Parameters["output"] = "json"

	var gp *middlewares.TableProcessor
	var err error

	if ok {
		gp, err = settings.SetupTableProcessor(l.Parameters)
	} else {
		gp, err = settings.SetupTableProcessor(map[string]interface{}{
			"output": "json",
		})
	}

	if err != nil {
		return nil, err
	}

	return gp, nil
}

// GinHandleGlazedCommand returns a gin.HandlerFunc that runs a glazed.Command and writes
// the results to the gin.Context ResponseWriter.
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
//
// TODO(manuel, 2023-06-22) Now that TableOutputFormatter renders directly into a io.Writer,
// I don't think we need all this anymore, we just need to set the relevant header.
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

	var gp *middlewares.TableProcessor
	var err error

	glazedLayer := pc.ParsedLayers["glazed"]

	if glazedLayer != nil {
		gp, err = settings.SetupTableProcessor(glazedLayer.Parameters)
		if err != nil {
			return err
		}
	} else {
		gp = middlewares.NewTableProcessor()
	}

	var writer io.Writer = c.Writer
	if opts.Writer != nil {
		writer = opts.Writer
	}

	if opts.OutputFormatterFactory != nil {
		of, err := opts.OutputFormatterFactory.CreateOutputFormatter(c, pc)
		if err != nil {
			return err
		}
		// remove table middlewares to do streaming rows
		gp.ReplaceTableMiddleware()

		// create rowOutputChannelMiddleware here? But that's actually a responsibility of the OutputFormatterFactory.
		// we need to create these before running the command, and we need to figure out a way to get the Columns.

		err = of.RegisterMiddlewares(gp)
		if err != nil {
			return err
		}

		eg := &errgroup.Group{}
		eg.Go(func() error {
			return cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
		})

		eg.Go(func() error {
			// we somehow need to pass the channels to the OutputFormatterFactory
			return of.Output(writer)
		})

		// no cancellation on error?

		return eg.Wait()
	}

	// here we run a normal full table render
	var of formatters.TableOutputFormatter
	if glazedLayer != nil {
		of, err = settings.SetupTableOutputFormatter(glazedLayer.Parameters)
		if err != nil {
			return err
		}
	} else {
		of = json.NewOutputFormatter(
			json.WithOutputIndividualRows(true),
		)
	}
	if opts.Writer == nil {
		c.Writer.Header().Set("Content-Type", of.ContentType())
	}

	gp.AddTableMiddleware(table.NewOutputMiddleware(of, writer))

	err = cmd.Run(c, pc.ParsedLayers, pc.ParsedParameters, gp)
	if err != nil {
		return err
	}

	err = gp.RunTableMiddlewares(c)
	if err != nil {
		return err
	}

	return nil
}
