# mxcli - Mendix CLI for AI-Assisted Development

> **WARNING: Alpha-quality software.** This project is in early stages and has been largely vibe-engineered with AI coding assistants. Expect bugs, missing features, and rough edges. **mxcli can corrupt your Mendix project files** — always work on a copy or use version control. Use at your own risk. This has been developed and tested against Mendix 11.6, other versions are currently not validated.
>
> **Do not edit a project with mxcli while it is open in Studio Pro.** Studio Pro maintains in-memory caches that cannot be updated externally. Close the project in Studio Pro first, run mxcli, then re-open the project.

A command-line tool that enables AI coding assistants ([Claude Code](https://claude.ai/claude-code), GitHub Copilot, OpenCode, Cursor, Continue.dev, Windsurf, Aider, and others) to read, understand, and modify Mendix application projects.

**[Read the documentation](https://mendixlabs.github.io/mxcli/)** | **[Try it in the Playground](https://codespaces.new/mendixlabs/mxcli-playground)** — no install needed, runs in your browser

## Why mxcli?

Mendix projects are stored in binary `.mpr` files that AI agents can't read directly. `mxcli` bridges this gap:

- **MDL (Mendix Definition Language)** — SQL-like syntax for querying and modifying Mendix models
- **Multi-tool support** — Claude Code, GitHub Copilot, OpenCode, Cursor, Continue.dev, Windsurf, Aider, and more
- **Full-text search** — search across all strings, messages, and source definitions
- **Code navigation** — callers, callees, references, impact analysis
- **Linting and reporting** — 40+ built-in rules, scored best practices report
- **Local build and run** — build and run Mendix apps without Docker

## Installation

### Windows (recommended — avoids firewall issues)

Open **Git Bash** (from the Start menu, or via PowerShell):

```powershell
& 'C:\Program Files\Git\bin\sh.exe'
```

Then inside Git Bash:

```bash
curl -fsSL https://github.com/engalar/mxcli/raw/refs/heads/dev/install.sh | bash
```

The script installs `mxcli.exe` to `~/.local/bin/`, writes the path to `~/.bashrc`, and registers it in the Windows user PATH so PowerShell and CMD also find it after reopening.

To use `mxcli` in the **current** Git Bash session without reopening:

```bash
export PATH="$HOME/.local/bin:$PATH"
mxcli version
```

### Windows (alternative — PowerShell)

```powershell
irm https://github.com/engalar/mxcli/raw/refs/heads/dev/install.ps1 | iex
```

### Linux / macOS

```bash
curl -fsSL https://github.com/engalar/mxcli/raw/refs/heads/dev/install.sh | bash
```

The script installs to `/usr/local/bin` (if writable) or `~/.local/bin`, and updates `~/.bashrc` / `~/.zshrc` automatically.

### Upgrade

Re-run the install script — it replaces the binary in-place:

```bash
curl -fsSL https://github.com/engalar/mxcli/raw/refs/heads/dev/install.sh | bash
```

## Quick Start

### New project

Create a new Mendix project with everything configured in one command:

```bash
mxcli new MyApp --version 11.8.0
```

This downloads MxBuild, creates a blank Mendix project, sets up AI tooling and a Dev Container, and installs the correct mxcli binary. Open the resulting folder in VS Code and reopen in the Dev Container — you're ready to go.

### Existing project

Add AI tooling to an existing Mendix project:

```bash
mxcli init /path/to/my-mendix-project
```

Then open in VS Code, reopen in Dev Container, and start your AI assistant:

```bash
claude   # or Cursor, Continue.dev, Windsurf, etc.
```

### Supported AI tools

| Tool | Config file | Description |
|------|-------------|-------------|
| **Claude Code** | `.claude/`, `CLAUDE.md` | Full integration with skills and commands |
| **OpenCode** | `.opencode/`, `opencode.json` | Skills, commands, and lint rules |
| **Cursor** | `.cursorrules` | Compact MDL reference and command guide |
| **Continue.dev** | `.continue/config.json` | Custom commands and slash commands |
| **Windsurf** | `.windsurfrules` | MDL rules for Codeium |
| **Aider** | `.aider.conf.yml` | Terminal-based AI pair programming |
| **Universal** | `AGENTS.md` | Works with all tools |

```bash
mxcli init --list-tools   # list supported tools
mxcli add-tool cursor     # add a tool to an existing project
```

## First commands

```bash
# explore your project
mxcli -p app.mpr -c "show modules"
mxcli -p app.mpr -c "show entities in MyModule"
mxcli -p app.mpr -c "describe page MyModule.Home"

# search
mxcli search -p app.mpr "validation"

# lint
mxcli lint -p app.mpr
```

## Compatibility

- **Mendix versions**: Studio Pro 8.x, 9.x, 10.x, 11.x
- **MPR formats**: v1 and v2 (with mprcontents folder)
- **Platforms**: Linux, macOS, Windows (amd64 and arm64)

## Further reading

| Document | Contents |
|----------|----------|
| [CLI Reference](docs/10-user-docs/CLI_REFERENCE.md) | All commands — search, navigation, export/import, widgets, catalog, lint, testing |
| [MDL Quick Reference](docs/01-project/MDL_QUICK_REFERENCE.md) | Full MDL syntax tables |
| [AI Integration](docs/10-user-docs/AI_INTEGRATION.md) | `mxcli init` details, dev container, tool-specific setup |
| [Manual Install](docs/10-user-docs/MANUAL_INSTALL.md) | Fallback for restricted networks, corporate proxy |
| [Go Library](docs/GO_LIBRARY.md) | Using mxcli as a Go library |
| [Contributing](CONTRIBUTING.md) | Building from source, running tests, contribution workflow |

## License

Apache License 2.0 — See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) and open an issue before starting work.
