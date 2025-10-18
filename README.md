# Govnilo

A Go library for testing checkers on services.

## Project Structure

This project follows the standard Go project layout:

- `cmd/` - Command-line applications
- `internal/` - Private application code (not importable by other projects)
- `pkg/` - Public packages (importable by other projects)
- `proto/` - Protocol buffer definitions
- `example/` - Example usage of the library

## Building

### Main Binary

```bash
go build -o bin/govnilo ./cmd/govnilo
```

### Example Binary

```bash
cd _examples
go build -o bin/example .
```

## Usage

### Run the main binary

```bash
./bin/govnilo --help
./bin/govnilo list --help
./bin/govnilo do --help
./bin/govnilo run --help
```

### Run the example

```bash
cd _examples
./bin/example --help
```

## Development

All binaries are built into `*/bin/` directories and these directories are ignored by git.

There is a vscode run spec to debug the **_example** binary. 

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch example",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "cwd": "${workspaceFolder}/_example/",
            "program": "${workspaceFolder}/_example/cmd/main/",
            "args": ["run", "--config", "./conf.yaml"]
        }
    ]
}
```