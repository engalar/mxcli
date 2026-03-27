# Vision

mxcli and MDL exist to make Mendix development accessible to coding agents, automation pipelines, and developers who prefer text-based tools. The long-term goal is a complete alternative to Mendix Studio Pro that runs headless, integrates with AI workflows, and never requires leaving VS Code.

## Coding Agents First

The primary audience for mxcli is not a human typing commands -- it is a coding agent (Claude Code, Cursor, Windsurf, or similar) that reads, reasons about, and modifies Mendix projects autonomously. Everything in mxcli is designed with this in mind:

- **MDL as machine-readable output.** Every `DESCRIBE` command produces valid MDL that can be fed back as input. Agents can read a microflow, modify it, and write it back without lossy format conversions.
- **Skills and context files.** `mxcli init` generates structured skill files that teach agents MDL syntax, common patterns, and project conventions -- reducing hallucination and retry loops.
- **Headless operation.** mxcli is a single static binary with no GUI, no daemon, and no login. It runs in CI pipelines, containers, and sandboxed agent environments without setup friction.

## QA Feedback in the Agentic Loop

Agents produce better results when they can evaluate their own output. mxcli provides the tooling to close this feedback loop:

- **`mxcli check`** validates MDL syntax and catches anti-patterns before changes are applied.
- **`mxcli lint`** runs 40+ rules across the full project, surfacing issues an agent can fix in the same session.
- **`mxcli docker build` and `mx check`** compile and validate the project against the real Mendix runtime, catching structural errors that static analysis misses.
- **`mxcli test`** runs microflow tests inside a live Mendix container, giving agents pass/fail signals on functional correctness.

Together, these tools let an agent write MDL, validate it, fix issues, and verify the result -- all without human intervention.

## VS Code as the Review Surface

While agents do the building, humans need to review and approve the result. The VS Code extension (`vscode-mdl`) provides the visual tools to do this without leaving the editor:

- **Syntax highlighting, diagnostics, and completion** for `.mdl` files.
- **Hover and go-to-definition** for navigating the Mendix model from code.
- **Context menu commands** to run, check, and lint MDL directly from the editor.
- **Terminal link integration** so that entity names, microflow references, and error locations in terminal output are clickable.

The goal is that a developer can review an agent's changes, run the app, inspect the result, and approve -- all within VS Code.

## Long-Term Roadmap

The end state is a complete headless development environment for Mendix:

| Capability | Status | Description |
|------------|--------|-------------|
| Model read/write | Shipped | Full domain model, microflow, page, security, navigation, workflow support |
| Validation and linting | Shipped | `mxcli check`, `mxcli lint`, 40+ rules, SARIF output for CI |
| Build and run | Shipped | `mxcli docker build/run` with hot reload |
| Testing | Shipped | `.test.mdl` microflow tests with Docker runtime |
| VS Code extension | Shipped | LSP, diagnostics, completion, hover, go-to-definition |
| External SQL | Shipped | Query PostgreSQL, Oracle, SQL Server; import data |
| Mendix Marketplace | Planned | Install and manage Marketplace modules from the CLI |
| Mendix Cloud deployment | Planned | Deploy to Mendix Cloud environments without Studio Pro |
| Mendix Catalog | Planned | Publish and consume APIs via the Mendix Catalog |
| Visual page preview | Planned | Preview pages in VS Code without running the app |
| Full Studio Pro parity | Long-term | Cover remaining metamodel domains (REST, OData, consumed services, etc.) |

The measure of success is simple: a team should be able to develop, test, review, and deploy a Mendix application without ever opening Studio Pro.
