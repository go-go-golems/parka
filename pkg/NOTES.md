package pkg

# Brainstorming

- 2023-02-28 - Manuel Odendahl
   - Initial writeup
- 2023-04-16 - Manuel Odendahl
   - Cleaning things up
   - Reflecting on builder API

## Problem 1: parsing incoming parameters into something to call a glaze.Command

So what would a parsing middleware for REST and co look like. It should
parse incoming parameters and properly call a given command. Some of the layers
of the command should be statically set, and not accessible to the outside,
while other parameters could be transformed before being passed downstreams.

We have, on the input side:
- query parameters
- form parameters
- REST JSON body

We need to map this to:
- command flags and arguments
- parameter layers

We want to be able to:
- parse the incoming parameters into a map of parameters
- parse the incoming parameters into layers
- provide filtering functions to modify incoming parameters before passing them to the command
- validate incoming parameters
- hide / make immutable a set of command parameters (for example, connection settings)

### API design: CommandHandlerFunc

Ultimately, when a request comes in to the API, we want to go through all the
"middlewares" and ultimately get a map of parsed layers and a map of parameters,
and call Run on the underlying glaze.Command.

#### CommandHandlerFunc - parsing incoming requests into a command's parameters and layers

This is done through the `CommandHandlerFunc` type, which is a function that
takes a `*gin.Context` and a `*CommandContext` and is allowed to modify both as
it does its thing.

A `CommandContext` is a struct that has a reference to the `GlazeCommand` and keeps
track of the parsed layers and parameters.

Basically, by iterating through the `CommandHandlerFunc` stored in a `HandleOptions`
struct, the gin handler slowly builds up all the necessary parameters and parsed layers
necessary to run the glaze command.

#### CreateProcessorFunc - creating the output formatter

We also have something called a `CreateProcessorFunc`, which is a function that
takes a `*gin.Context` and a `*CommandContext` and returns a `glaze.Processor`.
This allows us to override what output formatter is created, depending on the request.
The default handler will process the parsed parameters for the `glazed` layer, just 
as it would on the command line. If nothing is set, it would create a JSON output formatter.

### API design: Parsing parameters


### Filtering parameters

One very important part of parka is that it should allow for filtering and validating
in addition to what the actual command layers give us.

I'm not entirely sure how this fits in with the current ParkaContext that I build,
but I think it might be possible by defining somethings that's a "wrapper layer".
It takes another layer, potentially a set of defaults (although it might be easiest to
just pass that straight to the underlying layer anyway), and it does then
expose its own set of flags, and then once parsed, update the lower layer.

The command itself would actually just know its own layers, but when exposed over REST,
it will be wrapped in whatever the developer specifies.

The problem with this approach however is that we might need to dissect the lower layers of the
command, which is maybe not so elegant. But let's try that.

While this seems like something that could be useful in glazed itself, I'm going to
start building it only in parka for now, and then see how far it goes.

### Comparison to Cobra commands

On the cobra side, when we have a cobra command, we register a layer and this
returns a parser function that can be called later on. This is done because
we have a cobra *cmd object that we need to create, and only later
actually parse. This doesn't really apply to the web scenario, I think.

## Problem 2: expose parameters as HTML Form elements

We also want a way for these layers and parameters to provide HTML form elements
or a way to render themselves in response to a request.

### Rendering parameter widgets

One really important part of parka is being able to render exposed parameters
as HTML widgets. This might be possible to achieve by the handler chain by providing
renderers for either a layer or individual flags or by default.

When a request comes in to the HTML side, we want to render out the form elements
for each layer, but ultimately it comes down to the command itself to render the final picture.

It can of course call on the HTML forms of the individual layers. Of course,
we provide standard form elements for each parameter type as well, so that there is a
quick way to render the different parameters.

### Using templates for the rendering of parameters

We should capitalize on the heave use of templates through the go go golems ecosystem
and make it easy for people to provide overrides for parameter types, individual flags
in the form of reusable templates that take metadata describing the parameter.

Maybe this kind of metadata could be part of the original parameter definitions / command description,
and there is a way to create an extensible mechanism right from the start?

Let's not worry too much about that for now, and stay typed and potentially verbose and
not that generic for parka, not hesitating to create our own structs and parallel constructions
until we better understand what is different about the parka use cases.

## Problem 2a: Exposing the flag metadata

This is marked as problem 2a because it is basically the API equivalent of exposing
parameters as HTML elements.

Another interesting to do would be to recreate voodoo in golang. For that, it should be
possible to query the metadata about the arguments that a HTTP API command accepts, so that
they can be used to configure a client side command line app dynamically.

- [ ] Build voodoo on top of parka, see https://github.com/go-go-golems/parka/issues/15

## Problem 3: Rendering the glazed output

The response is then rendered by the command / command wrapper, potentially rendering
out the individual layers to give some feedback on the request.

- [ ] Parse the content-type of an incoming request to configure the output type for glazed
- [ ] Allow some parameters of the incoming request to configure the glazed output
 - this might not be good for SimpleCommand (but maybe?)
 - there might be something there to be done with the layer wrapper discussed in the section
   below

### Using CreateProcessorFunc to create the output formatter

2023-04-17 

I am building a simple CreateProcessorFunc that returns an OutputFormatter() that
uses the standard formatter to output HTML (but I might change that to a template entirely,
it depends on being able to access the column headers) and have it output some HTML that renders
the response to HTML using DataTables.

## Problem 4: Easily configuring parka handlers (declaratively?)

If we want to follow the pattern of declaratively based commands,
we could make a folder that specifies the templates to be loaded,
and the parameters and their types to be exposed, in a yaml.

This could be loaded, and then used to wrap *another* command (which might or might not
be declarative). Probably the best is to be a bit less bullish on the yaml flag declarations,
but maybe just leverage the recursive template loading that I now have. This makes it possible
to load plenty of smaller fragments for reuse.
