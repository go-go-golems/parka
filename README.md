# parka

Convert your CLI apps to APIs

This framework is part of the glazed ecosystem, and can be used to expose CLI / glazed applications
as APIs to the internet.

It uses the common concept of declaring flags and argument types
in a YAML file in order to generate different frontends for actual functionality,
as well as exposing a set of helpers for manipulating object and tabular data.

The long term goal of parka is to create not just API services (HTTP with REST, grpc, websockets, etc...),
but also provide graphical web user interfaces to interact with them (for example
by exposing the different flags and arguments as HTML forms).


## Steps

- [x] Serve templated file (test data)
- [x] Serve tailwind CSS
- [-] Integrate glazed.Command and expose as web form / API
- [ ] Integrate with htmx for dynamic webforms and dynamic apps]

### 2023-02-25 - Working on exposing glazed commands

Now that we have a proper generic framework for commands in glazed,
we can start wrapping and exposing individual commands as APIs.

We don't want to expose all arguments to a web api however, so the question is how to 
restrict parameters and arguments:

- add a new layer that specifies which arguments are exposed, how they get filtered
- annotate the original command with a `web` tag, which specifies how an argument should be handled?
- add additional section to command definition that can be loaded by parka (basically, the first option, but somehow
  maybe added as an extension layer to glazed commands themselves, instead of being directly parka specific)

In fact, we may want to add parka specific attributes to wrap the command,
so an additional layer makes sense. For example, a template argument might be used generically
to specify that different templates could be used. I don't know if this all makes sense declaratively,
so I will first focus on a code only API.

What I want to achieve today:

- [x] Reload HTML/JS from disk, not embed, to avoid having to recompile every 2 seconds
  - it seems like that's already how it works? odd
  - this might mean I have to add back the option to serve from embed
- [ ] Wrap a simple command as both REST+JSON and web form + htmx
  - does this mean we want to create a webform and go over every parameter?
  - this could be a template provided by parka already (just pass it a list of parameter definitions and it creates a form)
- [ ] Wrap a geppetto command to build the rewrite prompting
- [ ] Build and package the web application for easy deployment on DO, using parka as a library

#### Exposing commands as APIs

There is already a whole function for exposing commands as APIs, so I'm going to build upon that.
What I want to do is be able to specify a HTML template to render out HTML, when called in a certain 
form. In fact, because of htmx, it might make sense to call a single wrapped command
with different endpoints / additional parameters.

#### How much CSS / MD / HTML to bundle with the parka package itself

To make it easy to just bang out UIs, a fair amount of CSS and base HTML should already
be bundled with the package itself. It should be possible to just import parka, register
a command, and get going.

But, we should make all these things overridable too. But let's start with a single simple override.

### 2023-01-30 - Brainstorm parka structure

So now I want to make this a bit more useful.
I want to be able to serve markdown from multiple sources, so there should be
a mechanism to register sources of markdown to prefixes, the same way the fin router can do the
`r.Use()` stuff.

`r.Use()` takes a middleware handler function, so probably that's how things should be done,
along with helper methods (?).

The handlerFunc takes a `gin.Context`.

But also, it could all be much simpler, and we can just have the application register
a set of lookup functions that return markdown content for a certain prefix.

The server can then go through and take the first matching one.