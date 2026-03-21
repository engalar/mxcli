# CLAUDE.md

This file provides guidance for Claude Code when working with this repository.

## Project Overview

**ModelSDK Go** is a Go library for reading and modifying Mendix application projects (`.mpr` files) stored locally on disk. It's a Go-native alternative to the TypeScript-based Mendix Model SDK, enabling programmatic access without cloud connectivity.

## Build & Test Commands

```bash
# Build the CLI (preferred - uses Makefile)
make build

# Run tests
make test

# Format and vet code
make fmt
make vet

# Run a specific example
go run ./examples/read_project/main.go /path/to/project.mpr
go run ./examples/modify_project/main.go /path/to/project.mpr

# Run the code generator
go run ./cmd/codegen/main.go -reflection-dir ./reference/mendixmodellib/reflection-data -version 10.0.0 -output ./generated/metamodel
```

**Note**: This project uses `modernc.org/sqlite` (pure Go) and does **not** require CGO. No C compiler is needed.

**Note**: The VS Code extension (`vscode-mdl/`) uses **bun**, not npm/node. Use `bun install`, `bun run compile`, etc. The Makefile targets (`make vscode-ext`, `make vscode-install`) already use bun.

## Mendix Tools

The `mx` command-line tool validates and builds Mendix projects. Location depends on environment:

| Environment | Path |
|-------------|------|
| Dev container | `~/.mxcli/mxbuild/{version}/modeler/mx` |
| This repo | `reference/mxbuild/modeler/mx` |

```bash
# Auto-download mxbuild for the project's Mendix version
mxcli setup mxbuild -p app.mpr

# Check/validate a Mendix project
~/.mxcli/mxbuild/*/modeler/mx check /path/to/app.mpr

# Or use the integrated command (auto-downloads mxbuild)
mxcli docker check -p app.mpr
```

## Project Architecture

```
ModelSDKGo/
├── modelsdk.go              # Main public API (Open, OpenForWriting, helpers)
├── model/                   # Core types: ID, QualifiedName, Module, Element interface
│
├── api/                     # High-level fluent API (inspired by Mendix Web Extensibility API)
│   ├── api.go               # ModelAPI entry point with namespace access
│   ├── domainmodels.go      # EntityBuilder, AssociationBuilder, AttributeBuilder
│   ├── enumerations.go      # EnumerationBuilder
│   ├── microflows.go        # MicroflowBuilder
│   ├── pages.go             # PageBuilder, widget builders
│   └── modules.go           # ModulesAPI
│
├── sdk/                     # SDK implementation packages
│   ├── domainmodel/         # Entity, Attribute, Association, DomainModel
│   ├── microflows/          # Microflow, Nanoflow, activities (60+ types)
│   ├── pages/               # Page, Layout, Widget types (50+ widgets)
│   ├── widgets/             # Embedded widget templates for pluggable widgets
│   │   ├── loader.go        # Template loading with go:embed
│   │   └── templates/       # JSON widget type definitions by Mendix version
│   └── mpr/                 # MPR file format handling
│       ├── reader.go        # Read-only MPR access
│       ├── writer.go        # Read-write MPR modification
│       ├── parser.go        # BSON parsing and deserialization
│       └── utils.go         # UUID generation utilities
│
├── mdl/                     # MDL (Mendix Definition Language) parser & CLI
│   ├── grammar/             # ANTLR4 grammar definition
│   │   ├── MDLLexer.g4      # ANTLR4 lexer grammar (tokens)
│   │   ├── MDLParser.g4     # ANTLR4 parser grammar (rules)
│   │   └── parser/          # Generated Go parser code
│   ├── ast/                 # AST node types for MDL statements
│   ├── visitor/             # ANTLR listener to build AST
│   ├── executor/            # Executes AST against modelsdk-go
│   ├── catalog/             # SQLite-based catalog for querying project metadata
│   ├── linter/              # Extensible linting framework
│   │   └── rules/           # Built-in lint rules (MDL001, MDL002, etc.)
│   └── repl/                # Interactive REPL interface
│
├── sql/                     # External database connectivity (PostgreSQL, Oracle, SQL Server)
│   ├── driver.go            # DriverName type, ParseDriver()
│   ├── connection.go        # Manager, Connection, credential isolation
│   ├── config.go            # DSN resolution (env vars, YAML config)
│   ├── query.go             # Execute() — query via database/sql
│   ├── meta.go              # ShowTables(), DescribeTable() via information_schema
│   ├── format.go            # Table and JSON output formatters
│   ├── mendix.go            # Mendix DB DSN builder, table/column name helpers
│   └── import.go            # IMPORT pipeline: batch insert, ID generation, sequence tracking
│
├── cmd/                     # Command-line tools
│   ├── mxcli/               # CLI entry point (Cobra-based)
│   └── codegen/             # Code generator CLI
│
├── internal/                # Internal packages (not exported)
│   └── codegen/             # Metamodel code generation system
│       ├── schema/          # JSON reflection data loading
│       ├── transform/       # Transform to Go types
│       └── emit/            # Go source code generation
│
├── generated/metamodel/     # Auto-generated type definitions
├── examples/                # Usage examples
│
└── reference/               # Reference materials (not Go code)
    ├── mendixmodellib/      # TypeScript library + reflection data
    ├── mendixmodelsdk/      # TypeScript SDK reference
    └── mdl-grammar/         # Comprehensive MDL grammar reference
```

## Key Concepts

### MPR File Formats
- **v1**: Single `.mpr` SQLite database file (Mendix < 10.18)
- **v2**: `.mpr` metadata + `mprcontents/` folder with individual documents (Mendix >= 10.18)
- Format detection is automatic

### BSON Storage Names vs Qualified Names

**CRITICAL**: Mendix uses different "storage names" in BSON `$Type` fields than the "qualified names" shown in the TypeScript SDK documentation. Using the wrong name causes `TypeCacheUnknownTypeException` when opening in Studio Pro.

| Qualified Name (SDK/docs) | Storage Name (BSON $Type) | Note |
|---------------------------|---------------------------|------|
| CreateObjectAction | CreateChangeAction | |
| ChangeObjectAction | ChangeAction | |
| DeleteObjectAction | DeleteAction | |
| CommitObjectsAction | CommitAction | |
| RollbackObjectAction | RollbackAction | |
| AggregateListAction | AggregateAction | |
| ListOperationAction | ListOperationsAction | |
| ShowPageAction | ShowFormAction | "Form" was original term for "Page" |
| ClosePageAction | CloseFormAction | "Form" was original term for "Page" |

When adding new types, always verify the storage name by:
1. Examining existing MPR files with the `mx` tool or SQLite browser
2. Checking the reflection data in `reference/mendixmodellib/reflection-data/`
3. Looking at the parser cases in `sdk/mpr/parser_microflow.go`

**IMPORTANT**: When unsure about the correct BSON structure for a new feature, **ask the user to create a working example in Mendix Studio Pro** so you can compare the generated BSON against a known-good reference.

### Pluggable Widget Templates

For pluggable widgets (DataGrid2, ComboBox, Gallery, etc.), templates must include **both** `type` AND `object` fields:
- `type`: Widget PropertyTypes schema (defines what properties exist)
- `object`: Default WidgetObject with all property values

**CE0463 "widget definition changed" error**: This error occurs when the Object's property structure doesn't match the Type's PropertyTypes. Always extract templates from Studio Pro-created widgets, not programmatically generated ones. See `sdk/widgets/templates/README.md` for details. For debugging CE0463 and other BSON issues, follow the workflow in `.claude/skills/debug-bson.md`.

### TypeEnumeration vs TypeEntity Ambiguity

The MDL visitor (`buildDataType` in `visitor_helpers.go`) cannot distinguish between entity types and enumeration types for bare qualified names like `Module.EntityName`. Both parse as `ast.TypeEnumeration` with `EnumRef` set. Code that consumes data types must handle `TypeEnumeration` alongside `TypeEntity` and use `EnumRef` as a fallback for the entity name.

### Mendix Expression String Escaping

When generating Mendix expression strings (e.g., in `expressionToString()`), single quotes within string literals must be escaped by doubling them: `'it''s here'`. Do NOT use backslash escaping (`\'`). This matches Mendix Studio Pro's expression syntax.

### Public API Pattern
```go
// Read-only access
reader, err := modelsdk.Open("/path/to/project.mpr")
defer reader.Close()

// Read-write access
writer, err := modelsdk.OpenForWriting("/path/to/project.mpr")
defer writer.Close()
```

### High-Level Fluent API (in api/)
The `api/` package provides a simplified, fluent API inspired by Mendix Web Extensibility Model API:

```go
modelAPI := api.New(writer)
module, _ := modelAPI.Modules.GetModule("MyModule")
modelAPI.SetModule(module)

entity, _ := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    WithStringAttribute("Name", 100).
    WithIntegerAttribute("Age").
    Build()
```

Available namespaces: `DomainModels`, `Enumerations`, `Microflows`, `Pages`, `Modules`

## Code Style Guidelines

- Follow standard Go conventions (`go fmt`, `go vet`)
- Use descriptive names matching Mendix terminology
- Keep BSON/JSON tags consistent with Mendix serialization format
- Export types that should be part of the public API
- Use interfaces for polymorphic types (e.g., `Element`, `MicroflowObject`)

## Dependencies

- `modernc.org/sqlite` - Pure Go SQLite driver (no CGO required)
- `go.mongodb.org/mongo-driver` - BSON parsing for Mendix document format
- `github.com/jackc/pgx/v5` - PostgreSQL driver for external SQL connectivity
- `github.com/sijms/go-ora/v2` - Oracle driver for external SQL connectivity
- `github.com/microsoft/go-mssqldb` - SQL Server driver for external SQL connectivity

## MDL CLI (mxcli)

The `mxcli` command-line tool allows reading and modifying Mendix projects using MDL (Mendix Definition Language), a SQL-like syntax.

```bash
# Build the CLI
go build -o bin/mxcli ./cmd/mxcli

# Run interactive REPL
./bin/mxcli

# Execute commands directly
./bin/mxcli -p /path/to/app.mpr -c "SHOW ENTITIES"

# Execute MDL script file
./bin/mxcli exec script.mdl -p /path/to/app.mpr

# Check MDL syntax (no project needed)
./bin/mxcli check script.mdl

# Check syntax and validate references
./bin/mxcli check script.mdl -p app.mpr --references
```

### Key CLI Features

| Feature | Commands | Details |
|---------|----------|---------|
| **Project structure** | `SHOW STRUCTURE [DEPTH 1\|2\|3] [IN Module] [ALL]` | Compact overview at 3 depth levels |
| **Catalog queries** | `SHOW CATALOG TABLES`, `SELECT ... FROM CATALOG.table` | SQL querying of project metadata |
| **Code search** | `SHOW CALLERS\|CALLEES\|REFERENCES\|IMPACT\|CONTEXT OF ...` | Cross-reference navigation (requires `REFRESH CATALOG FULL`) |
| **Full-text search** | `SEARCH 'keyword'` | Search across all strings and source |
| **Linting** | `mxcli lint -p app.mpr [--format json\|sarif]` | 14 built-in rules + 27 Starlark rules (MDL, SEC, QUAL, ARCH, DESIGN, CONV) |
| **Report** | `mxcli report -p app.mpr [--format markdown\|json\|html]` | Scored best practices report with category breakdown |
| **Testing** | `mxcli test tests/ -p app.mpr` | `.test.mdl` / `.test.md` files, requires Docker |
| **Diff** | `mxcli diff -p app.mpr changes.mdl` | Compare script against project state |
| **Diff local** | `mxcli diff-local -p app.mpr --ref HEAD` | Git diff for MPR v2 projects |
| **OQL** | `mxcli oql -p app.mpr "SELECT ..."` | Query running Mendix runtime |
| **Widgets** | `SHOW WIDGETS`, `UPDATE WIDGETS SET ...` | Widget discovery and bulk updates (experimental) |
| **External SQL** | `SQL CONNECT`, `SQL <alias> SELECT ...`, `mxcli sql` | Direct SQL queries against PostgreSQL, Oracle, SQL Server (credential isolation) |
| **Data import** | `IMPORT FROM <alias> QUERY '...' INTO Module.Entity MAP (...)` | Import from external DB into Mendix app PostgreSQL (batch insert with ID generation) |
| **Connector gen** | `SQL <alias> GENERATE CONNECTOR INTO <module> [TABLES (...)] [VIEWS (...)] [EXEC]` | Auto-generate Database Connector MDL from discovered schema |
| **Diagnostics** | `mxcli diag [--bundle]` | Session logs, version info, bug report bundles |

### mxcli init

`mxcli init` creates a `.claude/` folder with skills, commands, CLAUDE.md, and VS Code MDL extension in a target Mendix project. Source of truth for synced assets:
- Skills: `reference/mendix-repl/templates/.claude/skills/`
- Commands: `.claude/commands/mendix/`
- VS Code extension: `vscode-mdl/vscode-mdl-*.vsix`

Build-time sync: `make build` syncs everything automatically. Individual targets: `make sync-skills`, `make sync-commands`, `make sync-vsix`.

### VS Code Extension

The `vscode-mdl` extension provides MDL language support: syntax highlighting, parse/semantic diagnostics, completion, symbols, folding, hover, go-to-definition, clickable terminal links, and context menu commands. The extension spawns `mxcli lsp --stdio` as the language server. Build with `make vscode-ext` (requires bun).

### ANTLR4 Parser

Regenerate after modifying `MDLLexer.g4` or `MDLParser.g4`: `make grammar`. See `docs/03-development/MDL_PARSER_ARCHITECTURE.md` for design details.

## IMPORTANT: Before Writing MDL Scripts or Working with Data

**Read the relevant skill files FIRST before writing any MDL, seeding data, or doing database/import work:**
- `.claude/skills/write-microflows.md` - Microflow syntax, common mistakes, validation checklist
- `.claude/skills/create-page.md` - Page/widget syntax reference
- `.claude/skills/alter-page.md` - ALTER PAGE/SNIPPET in-place modifications (SET, INSERT, DROP, REPLACE)
- `.claude/skills/overview-pages.md` - CRUD page patterns
- `.claude/skills/master-detail-pages.md` - Master-detail page patterns
- `.claude/skills/generate-domain-model.md` - Entity/Association syntax
- `.claude/skills/check-syntax.md` - Pre-flight validation checklist
- `.claude/skills/organize-project.md` - Folders, MOVE command, project structure conventions
- `.claude/skills/manage-security.md` - Security roles, access control, GRANT/REVOKE patterns
- `.claude/skills/manage-navigation.md` - Navigation profiles, home pages, menus, login pages
- `.claude/skills/demo-data.md` - **READ for any database/import work** - Mendix ID system, association storage, demo data insertion
- `.claude/skills/xpath-constraints.md` - XPath syntax in WHERE clauses, association paths, nested predicates, functions
- `.claude/skills/database-connections.md` - External database connections from microflows
- `.claude/skills/test-microflows.md` - **READ for testing work** - Test annotations, file formats, Docker setup requirement

**Always validate before presenting to user:**
```bash
./bin/mxcli check script.mdl                    # Syntax check
./bin/mxcli check script.mdl -p app.mpr --references  # With reference validation
```

## MDL Syntax Quick Reference

Full syntax tables for all MDL statements (microflows, pages, security, navigation, settings, business events, ALTER PAGE, reserved words) are in **[docs/01-project/MDL_QUICK_REFERENCE.md](docs/01-project/MDL_QUICK_REFERENCE.md)**.

## Current Implementation Status

**Implemented:**
- MPR v1/v2 reading and writing
- Domain model (entities, attributes, associations)
- ALTER ENTITY (add/rename/modify/drop attributes, indexes, documentation)
- Microflows/Nanoflows with 60+ activity types
- Pages with 50+ widget types
- ALTER PAGE/SNIPPET (SET, INSERT, DROP, REPLACE operations on widget trees)
- Image widgets (IMAGE, STATICIMAGE, DYNAMICIMAGE) with Width/Height properties
- Code generator for metamodel types
- MDL CLI (`mxcli`) with ANTLR4 parser
- MDL support for domain model, microflows, pages, and security
- Security management (module roles, user roles, access control, demo users)
- High-level fluent API (`api/` package) for simplified model manipulation
- LSP server with hover, go-to-definition, completion, diagnostics, symbols, folding
- VS Code extension (`vscode-mdl`) with context menu commands (Run/Check/Selection)
- Docker build integration (`mxcli docker build`) with PAD patching (Phase 1)
- OQL query execution against running runtime (`mxcli oql`)
- Business event services (SHOW/DESCRIBE/CREATE/DROP)
- Project settings (SHOW/DESCRIBE/ALTER)
- External SQL query execution against PostgreSQL, Oracle, SQL Server (`mxcli sql`, MDL `SQL CONNECT/QUERY`)
- Data import from external databases into Mendix app DB (`IMPORT FROM ... INTO ... MAP ...`)
- Database Connector generation from external schema (`SQL <alias> GENERATE CONNECTOR INTO <module>`)
- EXECUTE DATABASE QUERY microflow action (static, dynamic SQL, parameterized, runtime connection override)
- CREATE/DROP WORKFLOW with user tasks, decisions, parallel splits, and other activity types
- CALCULATED BY microflow syntax for calculated attributes

**Not Yet Implemented:**
- 47 of 52 metamodel domains (REST, etc.)
- Delta/change tracking system
- Runtime type reflection

## Useful Files for Context

- `README.md` - User documentation and API reference
- `api/api.go` - High-level fluent API entry point
- `api/domainmodels.go` - Entity/Association/Attribute builders
- `docs/SDK_EQUIVALENCE.md` - Detailed comparison with TypeScript SDK, gap analysis
- `sdk/mpr/parser.go` - BSON parsing logic (complex, handles polymorphic types)
- `sdk/mpr/writer_widgets.go` - Widget BSON serialization
- `sdk/widgets/templates/` - Embedded widget templates for pluggable widgets (ComboBox, DataGrid2, etc.)
- `sdk/widgets/templates/README.md` - **Critical**: Template extraction requirements (must include both `type` AND `object`)
- `generated/metamodel/enums.go` - All Mendix enumeration types
- `mdl/grammar/MDL.g4` - ANTLR4 grammar for MDL syntax (production)
- `mdl/executor/executor.go` - MDL statement execution logic
- `reference/mdl-grammar/` - Comprehensive MDL grammar reference
- `reference/mendixmodellib/reflection-data/` - Type definitions with storage names and default values
- `docs/03-development/MDL_PARSER_ARCHITECTURE.md` - ANTLR4 parser design documentation
- `docs/03-development/PAGE_BSON_SERIALIZATION.md` - Page/widget BSON format, type mappings, required defaults
- `.claude/skills/debug-bson.md` - Workflow for debugging BSON serialization issues with `mx` tool
- `cmd/mxcli/lsp.go` - LSP server implementation (hover, definition, diagnostics, completion, symbols)
- `cmd/mxcli/init.go` - `mxcli init` command (project initialization + VS Code extension install)
- `cmd/mxcli/docker/oql.go` - OQL query execution against running Mendix runtime via M2EE admin API
- `sql/connection.go` - External SQL connection manager (credential isolation)
- `sql/config.go` - DSN resolution (env vars, YAML config)
- `sql/import.go` - IMPORT pipeline (batch insert, Mendix ID generation, sequence tracking)
- `sql/generate.go` - Database Connector MDL generation from external schema
- `sql/typemap.go` - SQL → Mendix type mapping, DSN → JDBC URL conversion
- `sql/mendix.go` - Mendix DB helpers (DSN builder, table/column name conversion)
- `cmd/mxcli/cmd_sql.go` - `mxcli sql` CLI subcommand
- `mdl/executor/cmd_sql.go` - SQL statement executor handlers
- `mdl/executor/cmd_import.go` - IMPORT statement executor (auto-connects to Mendix DB)
- `vscode-mdl/src/extension.ts` - VS Code extension entry point
- `vscode-mdl/package.json` - VS Code extension manifest (commands, menus, settings)
