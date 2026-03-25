# What is mxcli?

**mxcli** is a command-line tool that enables developers and AI coding assistants to read, understand, and modify Mendix application projects. Mendix projects are stored in binary `.mpr` files that cannot be read or edited as text. mxcli bridges this gap by providing a text-based interface using MDL (Mendix Definition Language), a SQL-like syntax for querying and manipulating Mendix models.

## How It Works

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Developer     │     │    mxcli     │     │ Mendix Project  │
│   or AI Agent   │────>│  (MDL REPL)  │────>│   (.mpr file)   │
│                 │ MDL │              │     │                 │
│ "Create a       │     │ Parses MDL   │     │ Creates actual  │
│  Customer       │     │ Validates    │     │ entities,       │
│  entity..."     │     │ Executes     │     │ microflows,     │
└─────────────────┘     └──────────────┘     │ pages, etc.     │
                                             └─────────────────┘
```

A developer or AI agent writes MDL statements. mxcli parses and validates them, then applies the changes directly to the `.mpr` project file. The modified project can then be opened in Mendix Studio Pro as usual.

## Key Capabilities

- **Project exploration** -- list modules, entities, microflows, pages; describe any element in MDL; full-text search across all strings and source definitions.
- **Code navigation** -- find callers, callees, references, and impact analysis for any project element.
- **Model modification** -- create and alter entities, microflows, pages, security rules, navigation, workflows, and more using MDL scripts.
- **Catalog queries** -- SQL-based querying of project metadata (entity counts, microflow complexity, widget usage, cross-references).
- **Linting and reports** -- 40+ built-in lint rules with SARIF output for CI; scored best-practices reports.
- **Testing** -- test microflows using MDL syntax with javadoc-style annotations.
- **AI assistant integration** -- works with Claude Code, Cursor, Continue.dev, Windsurf, and Aider. `mxcli init` sets up project configuration, skills, and a Dev Container for sandboxed AI development.
- **VS Code extension** -- syntax highlighting, diagnostics, code completion, hover, go-to-definition, and context menu commands for `.mdl` files.
- **External SQL** -- connect to PostgreSQL, Oracle, or SQL Server; query external databases; import data into a running Mendix application.

## Supported Environments

| Item | Details |
|------|---------|
| Mendix versions | Studio Pro 8.x, 9.x, 10.x, 11.x |
| MPR formats | v1 (single `.mpr` file) and v2 (`.mpr` + `mprcontents/` folder) |
| Platforms | Linux, macOS, Windows (amd64 and arm64) |
| Dependencies | None -- mxcli is a single static binary with no runtime dependencies |
