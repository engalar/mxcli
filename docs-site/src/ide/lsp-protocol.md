# Protocol

The MDL language server uses the standard LSP stdio transport: JSON-RPC messages are exchanged over the process's standard input and standard output.

## Starting the Server

```bash
mxcli lsp --stdio
```

The `--stdio` flag is required. The server reads LSP requests from stdin and writes responses to stdout. Diagnostic logging goes to stderr.

## Initialization Handshake

The client initiates the connection with an `initialize` request. The server responds with its capabilities:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "capabilities": {},
    "rootUri": "file:///path/to/project",
    "initializationOptions": {
      "mprPath": "app.mpr"
    }
  }
}
```

The `initializationOptions.mprPath` is optional. If provided, the server uses it to locate the Mendix project. Otherwise, it searches the `rootUri` for `.mpr` files.

The server responds with:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "capabilities": {
      "textDocumentSync": 1,
      "completionProvider": { "triggerCharacters": [".", " "] },
      "hoverProvider": true,
      "definitionProvider": true,
      "documentSymbolProvider": true,
      "foldingRangeProvider": true,
      "diagnosticProvider": {
        "interFileDependencies": false,
        "workspaceDiagnostics": false
      }
    }
  }
}
```

After receiving the response, the client sends `initialized` to complete the handshake.

## Document Synchronization

The server uses **full document sync** (`textDocumentSync: 1`). On every change, the client sends the complete document content:

- `textDocument/didOpen` -- file opened, triggers initial parse
- `textDocument/didChange` -- content changed, triggers re-parse and diagnostics
- `textDocument/didSave` -- file saved, triggers semantic validation
- `textDocument/didClose` -- file closed, clears diagnostics

## Diagnostics

Diagnostics are pushed from the server to the client via `textDocument/publishDiagnostics` notifications. Two levels of diagnostics are reported:

1. **Parse diagnostics** -- sent after every `didChange`, based on ANTLR4 parser errors
2. **Semantic diagnostics** -- sent after `didSave`, based on reference validation against the project

## Shutdown

The client sends `shutdown` followed by `exit` to terminate the server cleanly.
