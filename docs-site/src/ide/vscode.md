# VS Code Extension

The **MDL extension for VS Code** provides a rich editing experience for `.mdl` files, bringing IDE-level support for Mendix Definition Language. The extension communicates with the `mxcli lsp` language server to provide real-time diagnostics, code completion, hover documentation, and navigation.

## Feature Summary

| Feature | Description |
|---------|-------------|
| **Syntax highlighting** | MDL keyword, string, comment, and identifier coloring |
| **Parse diagnostics** | Real-time error reporting as you type |
| **Semantic diagnostics** | Reference validation on save (entities, microflows, pages) |
| **Code completion** | Context-aware keyword and snippet suggestions |
| **Hover** | View MDL definitions by hovering over `Module.Name` references |
| **Go-to-definition** | Ctrl+click to open element source as a virtual document |
| **Document outline** | Navigate MDL statements via the Outline panel |
| **Folding** | Collapse statement blocks and widget trees |
| **Context menu** | Run File, Run Selection, and Check File commands |
| **Project tree** | Browse modules and documents in a Mendix-specific tree view |

## How It Works

The extension spawns `mxcli lsp --stdio` as a background process and communicates via the Language Server Protocol over standard I/O. This means all analysis, completion, and navigation is performed by the same engine that powers the `mxcli` command line.

```
┌──────────────┐     stdio      ┌──────────────────┐     ┌────────────┐
│  VS Code     │◀──────────────▶│  mxcli lsp       │────▶│  .mpr file │
│  Extension   │   LSP JSON-RPC │  --stdio          │     │            │
└──────────────┘                └──────────────────┘     └────────────┘
```

## Requirements

- **VS Code** 1.80 or later (or any VS Code fork such as Cursor)
- **mxcli** binary on `PATH` or configured via the `mdl.mxcliPath` setting
- A Mendix project (`.mpr` file) for semantic features

## Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `mdl.mxcliPath` | `mxcli` | Path to the mxcli executable |
| `mdl.mprPath` | *(auto-discovered)* | Path to `.mpr` file; if empty, the extension searches the workspace |

## Getting Started

The fastest way to get started is `mxcli init`, which automatically installs the extension into your project. See [Installation](./vscode-installation.md) for details, or jump to specific features:

- [Syntax and Diagnostics](./vscode-syntax.md)
- [Completion, Hover, Go-to-Definition](./vscode-intellisense.md)
- [Project Tree](./vscode-project-tree.md)
- [Context Menu Commands](./vscode-commands.md)
