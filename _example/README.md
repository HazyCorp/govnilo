# Govnilo Example

This is an example application that demonstrates how to use the govnilo library as a dependency. This example is located in the `_examples/` directory following Go's underscore convention for non-importable packages.

## Structure

- `main.go` - Simple main function that calls `cmd.Execute()` from the govnilo library
- `go.mod` - Module definition with replace directive to use local govnilo library

## Usage

### Build the example

```bash
go build -o bin/example .
```

### Run the example

```bash
./bin/example --help
./bin/example list --help
./bin/example do --help
./bin/example run --help
```

## How it works

The example uses a `replace` directive in `go.mod` to use the local govnilo library instead of a published version:

```go
replace github.com/HazyCorp/govnilo => ../
```

This allows you to test changes to the library locally before publishing it.
