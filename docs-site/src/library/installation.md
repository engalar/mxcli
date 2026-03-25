# Installation

## Go Module

Install the library with `go get`:

```bash
go get github.com/mendixlabs/mxcli
```

## Requirements

- **Go 1.21+**
- **No CGO required** -- the project uses `modernc.org/sqlite` (pure Go SQLite) and does not require a C compiler

## Dependencies

The library pulls in the following key dependencies:

| Dependency | Purpose |
|------------|---------|
| `modernc.org/sqlite` | Pure Go SQLite driver for reading `.mpr` files |
| `go.mongodb.org/mongo-driver` | BSON parsing for Mendix document format |

These are resolved automatically by `go get`.

## Verify Installation

Create a simple Go program to verify the library is working:

```go
package main

import (
    "fmt"
    "github.com/mendixlabs/mxcli"
)

func main() {
    reader, err := modelsdk.Open("/path/to/MyApp.mpr")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer reader.Close()

    version := reader.GetMendixVersion()
    fmt.Printf("Mendix version: %s\n", version)
    fmt.Printf("MPR format: v%d\n", reader.Version())
}
```

Run it:

```bash
go run main.go
```

## Building from Source

If you want to build the `mxcli` CLI tool from source:

```bash
git clone https://github.com/mendixlabs/mxcli.git
cd mxcli
make build
```

The binary will be placed in `bin/mxcli`.
