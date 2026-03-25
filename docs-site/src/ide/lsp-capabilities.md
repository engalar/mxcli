# Capabilities

The MDL language server supports the following LSP capabilities.

## Supported Methods

| LSP Method | Feature | Notes |
|-----------|---------|-------|
| `textDocument/publishDiagnostics` | Parse and semantic error reporting | Push-based; parse on change, semantic on save |
| `textDocument/completion` | Code completion | Keywords, snippets, and project references |
| `textDocument/hover` | Hover documentation | MDL definitions for qualified names |
| `textDocument/definition` | Go-to-definition | Opens element source as virtual document |
| `textDocument/documentSymbol` | Document outline | All MDL statements in the file |
| `textDocument/foldingRange` | Code folding | Statement blocks, widget trees, comments |

## Document Synchronization

| Method | Supported |
|--------|-----------|
| `textDocument/didOpen` | Yes |
| `textDocument/didChange` | Yes (full sync) |
| `textDocument/didSave` | Yes |
| `textDocument/didClose` | Yes |

## Completion Details

Trigger characters: `.` and ` ` (space).

Completion items include:

- **Keywords** -- `CREATE`, `SHOW`, `DESCRIBE`, `ALTER`, `DROP`, `GRANT`, etc.
- **Keyword sequences** -- `PERSISTENT ENTITY`, `MODULE ROLE`, `USER ROLE`
- **Data types** -- `String`, `Integer`, `Boolean`, `DateTime`, `Decimal`, etc.
- **Snippets** -- Multi-line templates for entity, microflow, page creation
- **Project references** -- Module names, entity names, microflow names (requires project)

## Diagnostics Details

### Parse Diagnostics (on change)

- Syntax errors from the ANTLR4 lexer and parser
- Missing tokens, unexpected tokens, mismatched brackets
- Reported with exact line/column positions

### Semantic Diagnostics (on save)

- Unresolved entity, microflow, or page references
- Invalid attribute names on known entities
- Module existence checks
- Requires a loaded Mendix project

## Not Yet Supported

The following LSP capabilities are not currently implemented:

| LSP Method | Status |
|-----------|--------|
| `textDocument/references` | Planned |
| `textDocument/rename` | Planned |
| `textDocument/formatting` | Planned |
| `textDocument/codeAction` | Planned |
| `textDocument/signatureHelp` | Not planned |
| `workspace/symbol` | Not planned |
