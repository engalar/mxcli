# MDL Parser

The MDL parser translates SQL-like MDL (Mendix Definition Language) syntax into executable operations against Mendix project files. It uses ANTLR4 for grammar definition, enabling cross-language grammar sharing with other implementations (TypeScript, Java, Python).

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MDL Input String                            │
│              "SHOW ENTITIES IN MyModule"                            │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    ANTLR4 Lexer (mdl_lexer.go)                      │
│    Generated from MDLLexer.g4 - Tokenizes input into SHOW, ENTITIES,│
│    IN, IDENTIFIER tokens                                            │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    ANTLR4 Parser (mdl_parser.go)                    │
│    Generated from MDLParser.g4 - Builds parse tree according to     │
│    grammar rules                                                    │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    ANTLR Listener (visitor/visitor.go)              │
│    Walks parse tree and builds strongly-typed AST nodes            │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         AST (ast/ast.go)                            │
│    *ast.ShowStmt{Type: "ENTITIES", Module: "MyModule"}             │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Executor (executor/executor.go)                  │
│    Executes AST against modelsdk-go API                            │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      modelsdk-go Library                            │
│    mpr.Writer, domainmodel.Entity, etc.                            │
└─────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
mdl/
├── grammar/
│   ├── MDLLexer.g4         # ANTLR4 lexer grammar (tokens)
│   ├── MDLParser.g4        # ANTLR4 parser grammar (rules)
│   └── parser/             # Generated parser code (DO NOT EDIT)
│       ├── mdl_lexer.go
│       ├── mdl_parser.go
│       ├── mdlparser_listener.go
│       └── mdlparser_base_listener.go
├── ast/
│   └── ast.go, ast_microflow.go, ast_expression.go, ast_datatype.go, ...
├── visitor/
│   └── visitor.go          # ANTLR listener implementation
├── executor/
│   ├── executor.go              # AST execution logic
│   ├── cmd_microflows_builder.go  # Microflow builder (variable tracking)
│   └── validate_microflow.go     # AST-level semantic checks (mxcli check)
├── catalog/
│   └── catalog.go          # SQLite-based project metadata catalog
├── linter/
│   ├── linter.go           # Linting framework
│   └── rules/              # Built-in lint rules
└── repl/
    └── repl.go             # Interactive REPL interface
```

## Why ANTLR4?

| Consideration | ANTLR4 | Parser Combinators |
|---------------|--------|-------------------|
| Cross-language | Same grammar for Go, TS, Java | Rewrite per language |
| Grammar docs | EBNF-like, readable | Code is the doc |
| Error messages | Built-in recovery | Custom implementation |
| Performance | Optimized lexer/parser | Comparable |
| Tooling | ANTLR Lab, IDE plugins | Limited |

## Why Listener Pattern (not Visitor)?

MDL uses the **listener** pattern rather than the visitor pattern:

- **Listener**: Callbacks fired during tree walk, simpler for AST building
- **Visitor**: Returns values from each node, better for expression evaluation

For MDL, statements are independent and do not need return value propagation, making the listener pattern more appropriate.

## Regenerating the Parser

After modifying `MDLLexer.g4` or `MDLParser.g4`:

```bash
make grammar
```

Or manually:

```bash
cd mdl/grammar
antlr4 -Dlanguage=Go -package parser -o parser MDLLexer.g4 MDLParser.g4
```

Requirements:
- ANTLR4 tool (`antlr4` command or Java JAR)
- Go target runtime (`github.com/antlr4-go/antlr/v4`)
