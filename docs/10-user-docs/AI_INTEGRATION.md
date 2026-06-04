# AI Assistant Integration

How to set up a Mendix project for AI-assisted development with `mxcli init`.

---

## Setup

```bash
# default: Claude Code + universal docs
mxcli init /path/to/my-mendix-project

# specify tool(s)
mxcli init --tool cursor /path/to/my-mendix-project
mxcli init --tool claude --tool cursor /path/to/my-mendix-project

# all tools
mxcli init --all-tools /path/to/my-mendix-project

# add a tool to an already-initialized project
mxcli add-tool cursor

# list supported tools
mxcli init --list-tools
```

## What Gets Created

**Universal (all tools):**
- `AGENTS.md` — Comprehensive guide for AI assistants
- `.ai-context/skills/` — MDL pattern guides (write-microflows.md, create-page.md, etc.)
- `.ai-context/examples/` — Example MDL scripts

**Tool-specific:**

| Tool | Config | Description |
|------|--------|-------------|
| **Claude Code** | `.claude/settings.json`, `CLAUDE.md`, commands, lint-rules, skills | Full integration |
| **OpenCode** | `.opencode/`, `opencode.json` | Skills, commands, lint rules |
| **Cursor** | `.cursorrules` | Compact MDL reference |
| **Continue.dev** | `.continue/config.json` | Custom commands and slash commands |
| **Windsurf** | `.windsurfrules` | MDL rules for Codeium |
| **Aider** | `.aider.conf.yml` | YAML config for terminal-based AI |
| **Universal** | `AGENTS.md` | Works with all tools |

## Dev Container

`mxcli init` generates a `.devcontainer/` configuration that provides a sandboxed environment — the recommended way to run AI coding agents, since it limits their access to just the project files.

After initialization, open the project in VS Code or Cursor and **reopen in Dev Container**, then start your AI assistant:

```bash
claude   # or use Cursor, Continue.dev, Windsurf, etc.
```

**What's installed in the dev container:**

| Component | Purpose |
|-----------|---------|
| **mxcli** | Mendix CLI (copied into project as `./mxcli`) |
| **MxBuild / mx** | Project validation and building (`~/.mxcli/mxbuild/`) |
| **JDK 21** (Adoptium) | Required by MxBuild |
| **Docker-in-Docker** | Running Mendix apps with `mxcli docker` |
| **Node.js** | Playwright testing support |
| **PostgreSQL client** | Database connectivity |
| **Claude Code** | Auto-installed on container creation |

**Key paths inside the dev container:**

```
~/.mxcli/mxbuild/{version}/modeler/mx    # mx check / mx build
~/.mxcli/runtime/{version}/              # Mendix runtime (auto-downloaded)
./mxcli                                   # project-local mxcli binary
```

MxBuild is auto-downloaded on first use:

```bash
mxcli setup mxbuild -p app.mpr
~/.mxcli/mxbuild/*/modeler/mx check app.mpr

# or use the integrated command
mxcli docker check -p app.mpr
```

## What AI Assistants Can Do

Once initialized, the AI assistant has access to:
- MDL command reference in `AGENTS.md`
- Pattern guides in `.ai-context/skills/`
- Tool-specific configuration
- Full project context via `mxcli` commands

The skills teach the AI how to write Mendix projects using MDL — covering microflows, pages, domain models, security, and navigation patterns. You can add your own skills with project-specific design patterns and best practices.
