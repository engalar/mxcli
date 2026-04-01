# Changelog

All notable changes to mxcli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.4.0] - 2026-03-31

### Added

- **SEND REST REQUEST** — Microflow action for consumed REST services with full BSON serialization roundtrip
- **Pluggable Image Widget** — Full roundtrip support for `com.mendix.widget.web.image.Image` with Studio Pro-extracted templates
- **ALTER PAGE SET Url** — Change page URLs via MDL
- **ALTER PAGE SET Layout** — Switch page layout via MDL
- **ALTER ENTITY SET POSITION** — Set entity position in domain model diagrams
- **VISIBLE IF / EDITABLE IF** — Conditional visibility and editability with XPath expressions, plus TabletWidth/PhoneWidth properties
- **EXECUTE DATABASE QUERY** — Microflow action for static, dynamic, and parameterized SQL with runtime connection override
- **Contract Browsing** — SHOW/DESCRIBE CONTRACT ENTITIES/ACTIONS from cached OData $metadata, CONTRACT CHANNELS/MESSAGES from AsyncAPI
- **Integration Catalog** — 7 new catalog tables (rest_clients, rest_operations, published_rest_services, external_entities, external_actions, business_events, contract tables)
- **SHOW EXTERNAL ACTIONS / PUBLISHED REST SERVICES** — Integration pane commands
- **SHOW CONSTANT VALUES** — Display constant values and catalog tables
- **CREATE/DROP CONFIGURATION** — Configuration management with constant overrides
- **JavaScript Actions** — NDSL/MDL support for JavaScript action definitions
- **DROP/MOVE FOLDER** — Remove empty folders and reorganize project structure
- **GALLERY Columns** — DesktopColumns/TabletColumns/PhoneColumns properties
- **Forward-Reference Hints** — Helpful error messages when exec fails on later-defined objects
- **IMAGE FROM FILE** — Image collection syntax for file-based images
- **OpenSSF Baseline Level 1** — Security foundations and CodeQL fixes
- **Multi-Agent Merge Proposal** — Design proposal for parallel agent work on Mendix projects
- **Documentation Site** — mdBook-based site with tutorials, language reference, migration guide, and internals
- **Tool Integrations** — Added support for OpenCode, Mistral Vibe, and GitHub Copilot in `mxcli init`
- **TUI Enhancements** — Agent channel (Unix socket), UX improvements, auto-create module support
- **Custom Widget AIGC Skill** — Skill for AI-generated custom pluggable widgets
- **AI Issue Triage** — GitHub Actions workflow for automated issue classification
- **Daily Project Digest** — Scheduled workflow for project activity summaries

### Fixed

- Skip null TextTemplate in opTextTemplate to avoid CE0463 widget definition errors
- Set Editable to Conditional and fix Visible XPath expression serialization
- REST client BSON serialization field ordering and roundtrip correctness
- Image widget template extraction (imageObject defaults, Parameters version marker, Texts$Translation)
- Escape single quotes in page DESCRIBE output via `mdlQuote()`
- Resolve association/attribute and entity/enumeration ambiguity in MDL parser
- LSP diagnostics for editable `mendix-mdl://` documents
- Gallery CE0463 by re-extracting template and fixing augmentation
- DataGrid2 column name derivation from attribute or caption
- ComboBox association EntityRef via IndirectEntityRef with association path
- XPath tokens written unquoted to prevent CE0161
- Long type written as `DataTypes$LongType` instead of IntegerType
- Date as distinct type from DateTime throughout the pipeline
- MPR version detection using DB schema and `_FormatVersion` field
- Recurse into loop bodies when extracting catalog references
- CodeQL symlink path traversal alerts in tar extraction
- Multiple TUI data races and agent channel stability fixes

### Changed

- Bumped dependencies: pgx v5.9.1, zap v1.27.1, go-runewidth v0.0.21, cobra v1.10.2, mongo-driver v1.17.9, sqlite v1.48.0
- Refactored Visible/Editable syntax to `Visible: [xpath]` and `Editable: [xpath]`
- Used dedicated CWTest module in custom widget examples
- Always-quoted identifiers in MDL to prevent reserved keyword conflicts
- Added scope & atomicity and documentation sections to PR review checklist

## [0.3.0] - 2026-03-26

### Added

- **TUI** — Interactive terminal UI (`mxcli tui`) with yazi-style Miller columns, BSON/MDL preview, search, tabs, command palette (`:` key), session restore (`-c`), and mouse support
- **Workflows** — Full CREATE/DESCRIBE WORKFLOW support with activities (UserTask, Decision, CallMicroflow, CallWorkflow, Jump, WaitForTimer, ParallelSplit, BoundaryEvent), BSON round-trip, and ANNOTATION statements
- **Consumed REST Clients** — SHOW/DESCRIBE/CREATE consumed REST services with BSON writer and mx check validation
- **Image Collections** — SHOW/DESCRIBE/CREATE/DROP IMAGE COLLECTION with BSON writer and Kitty/iTerm2/Sixel inline image rendering in TUI
- **WHILE Loops** — WHILE loop support in microflows with examples
- **ALTER PAGE Variables** — ALTER PAGE ADD/DROP VARIABLE support (Phase 3)
- **XPath** — Dedicated XPath expression grammar, catalog table population, and skills reference
- **BSON Tools** — `bson dump --format ndsl`, `bson compare` with smart array matching, `bson discover` for field coverage analysis
- **Documentation Site** — mdBook-based site with full language reference, tutorials, and internals documentation
- **Anti-pattern Detection** — `mxcli check` detects nested loops and empty list anti-patterns (issue #21)
- **CREATE OR MODIFY** — Additive upsert for USER ROLE and DEMO USER
- **AI PR Review** — GitHub Actions workflow using GitHub Models API for automated pull request review
- **RETRIEVE FROM $Variable** — Support for in-memory and NPE list association traversal (issue #22)
- **Constants** — Constant syntax help topic, LSP snippet, and CREATE OR MODIFY examples
- **UnknownElement Fallback** — Table-driven parser registries with graceful fallback for unrecognized BSON types (issue #19)

### Fixed

- MPR corruption from dangling GUIDs after attribute drop/add (#4)
- BSON field ordering loss in ALTER PAGE operations (#3)
- ALTER PAGE SET Attribute property support (issue #10)
- ALTER PAGE REPLACE deep GUID regeneration for stale $ID fields (issue #9)
- Quoted identifiers not resolved in page widget references (issue #8)
- DATAGRID placeholder ID leak during template augmentation (issue #6)
- COMBOBOX association EntityRef via IndirectEntityRef with association path
- Page/layout unit type mismatch (Forms$ vs Pages$ prefix)
- VIEW entity types, constant value BSON, and test error detection
- False positive OQL type inference for CASE expressions
- RETRIEVE using DatabaseRetrieveSource for reverse Reference association traversal
- RETURNS Void treated as void return type like Nothing
- ANNOTATION keyword added to annotationName grammar rule
- System entity types and RETURN keyword formatting in microflows
- 10 CodeQL security alerts
- XPath token quoting for `[%CurrentDateTime%]` (#1)
- DROP MODULE/ROLE cascade-removes module roles from user roles
- Security script CE0066 entity access out-of-date errors
- Slow integration tests with build tags and TestMain (issue #16)
- Docker run failing on fresh projects (issue #13)

### Changed

- Aligned `mxcli check` and `mxcli lint` reporting with shared Violation format (issue #10)
- Promoted BSON commands from debug-only to release build
- Auto-discover `.mpr` file when `-p` is omitted
- Moved `bson/` and `tui/` packages under `cmd/mxcli/` for better encapsulation
- Consolidated show-describe proposals into `docs/11-proposals/` with archive
- Documented association ParentPointer/ChildPointer semantics in CLAUDE.md
- Normalized CRLF to LF in bug reports via `.gitattributes`

## [0.2.0] - 2026-03-15

### Added

- **CI/CD** — GitHub Actions workflow for build, test, and lint on push; release workflow for tagged versions
- **Makefile Lint Targets** — `make lint`, `make lint-go` (fmt + vet), `make lint-ts` (tsc --noEmit)
- **Playwright Testing** — Browser name config support, port-offset fixes, project directory CWD for session discovery
- **VS Code Extension** — Project tree auto-refresh via file watchers, association cardinality label fix

### Fixed

- Enum truncation, DROP+CREATE cache invalidation, duplicate variable detection, subfolder enum resolution
- IMPORT FK column NULL fallback and entity attribute validation
- Docker exec using host port instead of container-internal port
- AGGREGATE syntax in skills docs
- Association cardinality labels in domain model diagrams
- 3 MDL bugs and standardized enum DEFAULT syntax

### Changed

- Default to always-quoted identifiers in MDL to prevent reserved keyword conflicts
- Communication Style section in generated CLAUDE.md for human-readable change descriptions
- Shortened mxcli startup warning to single line
- Chromium system dependencies added to devcontainer Dockerfile

## [0.1.0] - 2026-03-13

First public release.

### Added

- **MDL Language** — SQL-like syntax (Mendix Definition Language) for querying and modifying Mendix projects
- **Domain Model** — CREATE/ALTER/DROP ENTITY, CREATE ASSOCIATION, attribute types, indexes, validation rules
- **Microflows & Nanoflows** — 60+ activity types, loops, error handling, expressions, parameters
- **Pages** — 50+ widget types, CREATE/ALTER PAGE/SNIPPET, DataGrid, DataView, ListView, pluggable widgets
- **Page Variables** — `Variables: { $name: Type = 'expression' }` in page/snippet headers for column visibility and conditional logic
- **Security** — Module roles, entity access rules, GRANT/REVOKE, UPDATE SECURITY reconciliation
- **Navigation** — Navigation profiles, menu items, home pages, login pages
- **Enumerations** — CREATE/ALTER/DROP ENUMERATION with localized values
- **Business Events** — CREATE/DROP business event services
- **Project Settings** — SHOW/DESCRIBE/ALTER for runtime, language, and theme settings
- **Database Connections** — CREATE/DESCRIBE DATABASE CONNECTION for Database Connector module
- **Full-text Search** — SEARCH across all strings, messages, captions, labels, and MDL source
- **Code Navigation** — SHOW CALLERS/CALLEES/REFERENCES/IMPACT/CONTEXT for cross-reference analysis
- **Catalog Queries** — SQL-based querying of project metadata via CATALOG tables
- **Linting** — 14 built-in rules + 27 Starlark rules across MDL, SEC, QUAL, ARCH, DESIGN, CONV categories
- **Report** — Scored best practices report with category breakdown (`mxcli report`)
- **Testing** — `.test.mdl` / `.test.md` test files with Docker-based runtime validation
- **Diff** — Compare MDL scripts against project state, git diff for MPR v2 projects
- **External SQL** — Direct queries against PostgreSQL, Oracle, SQL Server with credential isolation
- **Data Import** — IMPORT FROM external DB into Mendix app PostgreSQL with batch insert and ID generation
- **Connector Generation** — Auto-generate Database Connector MDL from external schema discovery
- **OQL** — Query running Mendix runtime via admin API
- **Docker Build** — `mxcli docker build` with PAD patching
- **VS Code Extension** — Syntax highlighting, diagnostics, completion, hover, go-to-definition, symbols, folding
- **LSP Server** — `mxcli lsp --stdio` for editor integration
- **Multi-tool Init** — `mxcli init` with support for Claude Code, Cursor, Continue.dev, Windsurf, Aider
- **Dev Container** — `mxcli init` generates `.devcontainer/` configuration for sandboxed AI agent development
- **MPR v1/v2** — Automatic format detection, read/write support for both formats
- **Fluent API** — High-level Go API (`api/` package) for programmatic model manipulation
