# ANTLR4 Grammar Design

The MDL parser uses ANTLR4 with separate lexer and parser grammars. This page covers the design patterns and key decisions in the grammar.

## Grammar Files

| File | Purpose |
|------|---------|
| `mdl/grammar/MDLLexer.g4` | Token definitions (keywords, operators, literals) |
| `mdl/grammar/MDLParser.g4` | Parser rules (statements, expressions, blocks) |
| `mdl/grammar/parser/` | Generated Go code (do not edit) |

## Regenerating the Parser

After modifying either `.g4` file:

```bash
make grammar
```

This runs the ANTLR4 tool to regenerate the Go parser code in `mdl/grammar/parser/`.

## Key Design Patterns

### Case-Insensitive Keywords

MDL keywords are case-insensitive. This is implemented using fragment rules in the lexer:

```antlr
// Lexer fragments for case-insensitive matching
fragment A : [aA] ;
fragment B : [bB] ;
// ...

// Keywords use fragments
SHOW    : S H O W ;
CREATE  : C R E A T E ;
ENTITY  : E N T I T Y ;
```

This means `SHOW`, `show`, `Show`, and `sHoW` are all valid.

### Separate Lexer and Parser

The grammar uses split lexer/parser grammars (not a combined grammar). This provides:

- Clearer separation between tokenization and parsing
- Better control over token modes
- Easier maintenance of complex grammars

### Statement Structure

Top-level statements follow a consistent pattern:

```antlr
statement
    : showStatement
    | describeStatement
    | createStatement
    | alterStatement
    | dropStatement
    | grantStatement
    | revokeStatement
    | searchStatement
    | refreshStatement
    | sqlStatement
    | importStatement
    // ...
    ;
```

Each statement type has its own rule, keeping the grammar modular.

### Qualified Names

Module-qualified names (`Module.Entity`) are a fundamental pattern:

```antlr
qualifiedName
    : IDENTIFIER ('.' IDENTIFIER)*
    ;
```

This matches `Customer`, `Sales.Customer`, and `Sales.Customer.Name`.

### Widget Trees

Page definitions use recursive widget nesting with curly braces:

```antlr
widgetDecl
    : widgetType IDENTIFIER widgetProperties? widgetBody?
    ;

widgetBody
    : '{' widgetDecl* '}'
    ;
```

This allows arbitrary nesting depth for layout grids, containers, and data views.

## Architecture Pipeline

```
MDL Input String
       │
       ▼
┌──────────────┐
│ ANTLR4 Lexer │  Tokenizes into SHOW, IDENTIFIER, etc.
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ ANTLR4 Parser│  Builds parse tree from grammar rules
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Visitor    │  Walks parse tree, builds typed AST nodes
│  (Listener)  │  (mdl/visitor/visitor.go)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│     AST      │  Strongly-typed statement nodes
│              │  (mdl/ast/ast.go)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Executor   │  Executes AST against modelsdk-go
│              │  (mdl/executor/executor.go)
└──────────────┘
```

## Visitor (Listener) Pattern

ANTLR4 generates a listener interface. The MDL visitor (`mdl/visitor/visitor.go`) implements this interface to build AST nodes:

- `EnterCreateEntityStatement` -> creates `*ast.CreateEntityStmt`
- `EnterCreateMicroflowStatement` -> creates `*ast.CreateMicroflowStmt`
- `EnterShowStatement` -> creates `*ast.ShowStmt`

Each listener method extracts data from the parse tree context and constructs the corresponding AST node.

## Error Recovery

The ANTLR4 parser provides built-in error recovery. When a syntax error is encountered:

1. The parser reports the error with line/column position
2. It attempts to recover by consuming or inserting tokens
3. Parsing continues to find additional errors

This enables the language server to report multiple syntax errors in a single pass.

## Adding New Syntax

See [Adding New Statements](./adding-statements.md) for the step-by-step process of extending the grammar.
