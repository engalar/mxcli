# Layer Diagram

Detailed description of each architectural layer in ModelSDK Go.

## 1. Command Layer (`cmd/`)

| Package | Purpose |
|---------|---------|
| `cmd/mxcli` | CLI entry point using Cobra framework; includes LSP server, Docker integration, diagnostics |
| `cmd/codegen` | Metamodel code generator from reflection data |

Key CLI subcommands:

| Subcommand | File | Purpose |
|------------|------|---------|
| `exec` | `cmd_exec.go` | Execute MDL script files |
| `check` | `cmd_check.go` | Syntax and reference validation |
| `lint` | `cmd_lint.go` | Run linting rules |
| `report` | `cmd_report.go` | Best practices report |
| `test` | `cmd_test_run.go` | Run `.test.mdl` / `.test.md` tests |
| `diff` | `cmd_diff.go` | Compare script against project |
| `sql` | `cmd_sql.go` | External SQL queries |
| `lsp` | `lsp.go` | Language Server Protocol server |
| `init` | `init.go` | Project initialization |
| `docker` | `docker.go` | Docker build/check/OQL integration |
| `diag` | `diag.go` | Session logs, bug report bundles |

## 2. MDL Layer (`mdl/`)

The MDL (Mendix Definition Language) layer provides a SQL-like interface for querying and modifying Mendix models.

```mermaid
sequenceDiagram
    participant User
    participant REPL
    participant Parser
    participant Visitor
    participant Executor
    participant SDK

    User->>REPL: "SHOW ENTITIES IN MyModule"
    REPL->>Parser: Parse MDL command
    Parser->>Visitor: Walk parse tree
    Visitor->>Executor: Build AST node
    Executor->>SDK: ListDomainModels()
    SDK-->>Executor: []*DomainModel
    Executor-->>REPL: Formatted output
    REPL-->>User: Display table
```

| Package | Purpose |
|---------|---------|
| `mdl/grammar` | ANTLR4 lexer/parser (generated from MDLLexer.g4 + MDLParser.g4) |
| `mdl/ast` | AST node types for MDL statements |
| `mdl/visitor` | ANTLR listener that builds AST from parse tree |
| `mdl/executor` | Executes AST nodes against the SDK (~45k lines across 40+ files) |
| `mdl/catalog` | SQLite-based catalog for querying project metadata |
| `mdl/linter` | Extensible linting framework with built-in rules and Starlark scripting |
| `mdl/repl` | Interactive REPL interface |

## 3. High-Level API Layer (`api/`)

The `api/` package provides a simplified, fluent builder API inspired by the Mendix Web Extensibility Model API.

```mermaid
classDiagram
    class ModelAPI {
        +writer *Writer
        +reader *Reader
        +currentModule *Module
        +DomainModels *DomainModelsAPI
        +Microflows *MicroflowsAPI
        +Pages *PagesAPI
        +Enumerations *EnumerationsAPI
        +Modules *ModulesAPI
        +New(writer) *ModelAPI
        +SetModule(module) *ModelAPI
    }

    class EntityBuilder {
        +Persistent() *EntityBuilder
        +NonPersistent() *EntityBuilder
        +WithStringAttribute(name, length) *EntityBuilder
        +WithIntegerAttribute(name) *EntityBuilder
        +Build() (*Entity, error)
    }

    class MicroflowBuilder {
        +WithParameter(name, entity) *MicroflowBuilder
        +WithStringParameter(name) *MicroflowBuilder
        +ReturnsBoolean() *MicroflowBuilder
        +Build() (*Microflow, error)
    }

    ModelAPI --> EntityBuilder : creates
    ModelAPI --> MicroflowBuilder : creates
```

| File | Purpose |
|------|---------|
| `api/api.go` | ModelAPI entry point with namespace access |
| `api/domainmodels.go` | EntityBuilder, AssociationBuilder, AttributeBuilder |
| `api/enumerations.go` | EnumerationBuilder, EnumValueBuilder |
| `api/microflows.go` | MicroflowBuilder with parameters and return types |
| `api/pages.go` | PageBuilder, widget builders |
| `api/modules.go` | ModulesAPI for module retrieval |

## 4. SDK Layer (`sdk/`)

The SDK layer provides Go types and APIs for Mendix model elements.

```mermaid
classDiagram
    class Reader {
        +Open(path) Reader
        +ListModules() []*Module
        +ListDomainModels() []*DomainModel
        +ListMicroflows() []*Microflow
        +ListPages() []*Page
        +FindCustomWidgetType(widgetID) *RawCustomWidgetType
        +Close()
    }

    class Writer {
        +OpenForWriting(path) Writer
        +AddEntity(dm, entity)
        +AddAssociation(dm, assoc)
        +Save()
        +Close()
    }

    class DomainModel {
        +ID
        +ContainerID
        +Entities []*Entity
        +Associations []*Association
    }

    class Entity {
        +ID
        +Name
        +Attributes []*Attribute
        +Source string
        +OqlQuery string
    }

    Reader --> DomainModel
    Writer --> DomainModel
    DomainModel --> Entity
```

| Package | Purpose |
|---------|---------|
| `sdk/mpr/` | MPR file format handling (~18k lines across reader, writer, parser files split by domain) |
| `sdk/domainmodel` | Entity, Attribute, Association types |
| `sdk/microflows` | Microflow, Activity types (60+ types) |
| `sdk/pages` | Page, Widget types (50+ types) |
| `sdk/widgets` | Embedded widget templates for pluggable widgets |

The `sdk/mpr/` package is split by domain for maintainability:

| File Pattern | Purpose |
|--------------|---------|
| `reader.go`, `reader_*.go` | Read-only MPR access, split by element type |
| `writer.go`, `writer_*.go` | Read-write MPR modification (domainmodel, microflow, security, widgets, etc.) |
| `parser.go`, `parser_*.go` | BSON parsing and deserialization (domainmodel, microflow, etc.) |
| `utils.go` | UUID generation utilities |

## 5. Model Layer (`model/`)

Core types shared across the SDK:

```mermaid
classDiagram
    class ID {
        <<type alias>>
        string
    }

    class BaseElement {
        +ID ID
        +TypeName string
    }

    class QualifiedName {
        +Module string
        +Name string
    }

    class Module {
        +BaseElement
        +Name string
        +FromAppStore bool
    }

    class Text {
        +Translations map
        +GetTranslation(lang) string
    }

    BaseElement <|-- Module
```

## 6. External SQL Layer (`sql/`)

The `sql/` package provides external database connectivity for querying PostgreSQL, Oracle, and SQL Server databases.

```mermaid
flowchart LR
    subgraph "MDL Commands"
        CONNECT["SQL CONNECT"]
        QUERY["SQL alias SELECT ..."]
        IMPORT["IMPORT FROM alias ..."]
        GENERATE["SQL alias GENERATE CONNECTOR"]
    end

    subgraph "sql/ Package"
        CONN[Connection Manager]
        QE[Query Engine]
        IMP[Import Pipeline]
        GEN[Connector Generator]
        META[Schema Introspection]
        FMT[Output Formatters]
    end

    subgraph "Databases"
        PG[(PostgreSQL)]
        ORA[(Oracle)]
        MSSQL[(SQL Server)]
    end

    CONNECT --> CONN
    QUERY --> QE
    IMPORT --> IMP
    GENERATE --> GEN

    CONN --> PG
    CONN --> ORA
    CONN --> MSSQL
    QE --> CONN
    IMP --> CONN
    GEN --> META
    META --> CONN
```

| File | Purpose |
|------|---------|
| `driver.go` | DriverName type, ParseDriver() |
| `connection.go` | Manager, Connection, credential isolation |
| `config.go` | DSN resolution (env vars, YAML config) |
| `query.go` | Execute() -- query via database/sql |
| `meta.go` | ShowTables(), DescribeTable() via information_schema |
| `import.go` | IMPORT pipeline: batch insert, ID generation, sequence tracking |
| `generate.go` | Database Connector MDL generation from external schema |
| `typemap.go` | SQL to Mendix type mapping, DSN to JDBC URL conversion |
| `mendix.go` | Mendix DB DSN builder, table/column name helpers |
| `format.go` | Table and JSON output formatters |

## 7. VS Code Extension (`vscode-mdl/`)

The VS Code extension provides MDL language support via an LSP client that communicates with `mxcli lsp --stdio`.

```mermaid
graph TB
    subgraph "VS Code"
        EXT[extension.ts]
        TREE[Project Tree Provider]
        LINK[Terminal Link Provider]
        CONTENT[MDL Content Provider]
        PREVIEW[Preview Provider]
    end

    subgraph "Preview Renderers"
        DMRENDER[Domain Model]
        MFRENDER[Microflow]
        PGRENDER[Page Wireframe]
        MODRENDER[Module Overview]
        QPRENDER[Query Plan]
    end

    subgraph "mxcli"
        LSP[LSP Server]
    end

    EXT --> TREE
    EXT --> LINK
    EXT --> CONTENT
    EXT --> PREVIEW
    PREVIEW --> DMRENDER
    PREVIEW --> MFRENDER
    PREVIEW --> PGRENDER
    PREVIEW --> MODRENDER
    PREVIEW --> QPRENDER
    EXT -->|stdio| LSP
```

LSP features include syntax highlighting, parse/semantic diagnostics, completion, symbols, folding, hover, go-to-definition, clickable terminal links, and context menu commands.

## 8. LSP Server (`cmd/mxcli/lsp*.go`)

The LSP server is embedded in the `mxcli` binary:

| File | Purpose |
|------|---------|
| `lsp.go` | Main LSP server, hover, go-to-definition |
| `lsp_diagnostics.go` | Parse and semantic error reporting |
| `lsp_completion.go` | Context-aware completions |
| `lsp_completions_gen.go` | Generated completion data |
| `lsp_symbols.go` | Document symbols |
| `lsp_folding.go` | Code folding ranges |
| `lsp_hover.go` | Hover information |
| `lsp_helpers.go` | Shared utilities |
