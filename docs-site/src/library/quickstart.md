# Quick Start

The `modelsdk-go` library provides programmatic access to Mendix projects from Go code. It is the underlying library that powers the `mxcli` CLI tool.

## Overview

- **Read** Mendix project files (`.mpr`) to inspect modules, entities, microflows, pages, and more
- **Write** changes back to projects: create entities, attributes, associations, microflows, pages
- **No cloud connectivity required** -- works directly with local `.mpr` files
- **No CGO required** -- uses a pure Go SQLite driver (`modernc.org/sqlite`)
- **Automatic format detection** -- handles both MPR v1 (single file) and v2 (directory-based) formats

## Minimal Example

```go
package main

import (
    "fmt"
    "github.com/mendixlabs/mxcli"
)

func main() {
    reader, err := modelsdk.Open("/path/to/MyApp.mpr")
    if err != nil {
        panic(err)
    }
    defer reader.Close()

    modules, _ := reader.ListModules()
    for _, m := range modules {
        fmt.Printf("Module: %s\n", m.Name)
    }
}
```

## Two Access Modes

| Mode | Function | Use Case |
|------|----------|----------|
| Read-only | `modelsdk.Open(path)` | Inspecting project structure, generating reports |
| Read-write | `modelsdk.OpenForWriting(path)` | Creating entities, microflows, pages |

## Two API Levels

| Level | Package | Style |
|-------|---------|-------|
| Low-level SDK | `modelsdk` (root) | Direct type construction and writer calls |
| High-level Fluent API | `api/` | Builder pattern with method chaining |

The low-level SDK gives full control over model elements. The fluent API provides a simplified interface for common operations.

## MPR File Format

Mendix projects are stored in `.mpr` files which are SQLite databases containing BSON-encoded model elements.

- **MPR v1** (Mendix < 10.18): Single `.mpr` file containing all model data
- **MPR v2** (Mendix >= 10.18): `.mpr` metadata file + `mprcontents/` folder with individual documents

The library automatically detects and handles both formats.

## Model Structure

```
Project
├── Modules
│   ├── Domain Model
│   │   ├── Entities
│   │   │   ├── Attributes
│   │   │   ├── Indexes
│   │   │   ├── Access Rules
│   │   │   ├── Validation Rules
│   │   │   └── Event Handlers
│   │   ├── Associations
│   │   └── Annotations
│   ├── Microflows
│   │   ├── Parameters
│   │   └── Activities & Flows
│   ├── Nanoflows
│   ├── Pages
│   │   ├── Widgets
│   │   └── Data Sources
│   ├── Layouts
│   ├── Snippets
│   ├── Enumerations
│   ├── Constants
│   ├── Scheduled Events
│   └── Java Actions
└── Project Documents
```

## Comparison with Official SDK

| Feature | Mendix Model SDK (TypeScript) | modelsdk-go |
|---------|-------------------------------|-------------|
| Language | TypeScript/JavaScript | Go |
| Runtime | Node.js | Native binary |
| Cloud Required | Yes (Platform API) | No |
| Local Files | No | Yes |
| Real-time Collaboration | Yes | No |
| Read Operations | Yes | Yes |
| Write Operations | Yes | Yes |
| Type Safety | Yes (TypeScript) | Yes (Go) |
| CLI Tool | No | Yes (mxcli) |
| SQL-like DSL | No | Yes (MDL) |
