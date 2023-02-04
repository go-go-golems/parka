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