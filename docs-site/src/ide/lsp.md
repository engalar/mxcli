# LSP Server

The MDL language server implements the [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) (LSP), enabling any editor with LSP support to provide rich MDL editing features. The server is built into the `mxcli` binary -- no separate installation is needed.

## Overview

The language server is started with:

```bash
mxcli lsp --stdio
```

It communicates over standard I/O using JSON-RPC, the standard LSP transport. The server provides:

- **Real-time diagnostics** from the ANTLR4 parser
- **Semantic diagnostics** validated against the Mendix project
- **Code completion** with keywords, snippets, and project references
- **Hover** showing MDL definitions for qualified names
- **Go-to-definition** opening element source as virtual documents
- **Document symbols** for outline and breadcrumb navigation
- **Folding ranges** for collapsible statement blocks

## Architecture

```
┌──────────────┐                    ┌─────────────────────┐
│    Editor     │◀── LSP JSON-RPC ─▶│   mxcli lsp         │
│  (VS Code,   │     over stdio     │                     │
│   Neovim,    │                    │  ┌───────────────┐  │
│   Emacs)     │                    │  │ ANTLR4 Parser │  │
└──────────────┘                    │  └───────────────┘  │
                                    │  ┌───────────────┐  │
                                    │  │ MPR Reader    │  │
                                    │  └───────────────┘  │
                                    │  ┌───────────────┐  │
                                    │  │ Catalog       │  │
                                    │  └───────────────┘  │
                                    └─────────────────────┘
```

The server is implemented in `cmd/mxcli/lsp.go`. It reuses the same ANTLR4 parser, MPR reader, and catalog components that power the `mxcli` CLI.

## Project Discovery

On startup, the server auto-discovers the Mendix project:

1. Check the `mdl.mprPath` client configuration
2. Search the workspace root for `*.mpr` files
3. If found, load the project for semantic features

If no project is found, the server still provides syntax highlighting and parse diagnostics, but semantic features (reference validation, hover definitions, completion from project) are disabled.

## Related Pages

- [Protocol Details](./lsp-protocol.md) -- stdio transport and initialization
- [Capabilities](./lsp-capabilities.md) -- full list of supported LSP methods
- [Other Editors](./lsp-editors.md) -- Neovim, Emacs, and generic LSP client setup
