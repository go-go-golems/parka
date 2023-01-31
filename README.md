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

- Serve templated file (test data)
- Serve tailwind CSS