# Other AI tools

Claude Code is the default integration, but mxcli also supports OpenCode, Cursor, Continue.dev, Windsurf, and Aider. Each tool gets its own configuration file that teaches the AI about MDL syntax and mxcli commands.

## Initializing for a specific tool

Use the `--tool` flag to specify which AI tool you use:

```bash
# OpenCode
mxcli init --tool opencode /path/to/my-mendix-project

# Cursor
mxcli init --tool cursor /path/to/my-mendix-project

# Continue.dev
mxcli init --tool continue /path/to/my-mendix-project

# Windsurf
mxcli init --tool windsurf /path/to/my-mendix-project

# Aider
mxcli init --tool aider /path/to/my-mendix-project
```

## Setting up multiple tools

You can configure several tools at once. This is useful if different team members use different editors, or if you want to try several tools on the same project:

```bash
# Multiple tools
mxcli init --tool claude --tool cursor /path/to/my-mendix-project

# All supported tools at once
mxcli init --all-tools /path/to/my-mendix-project
```

## Adding a tool to an existing project

If you already ran `mxcli init` and want to add support for another tool without re-initializing:

```bash
mxcli add-tool cursor
mxcli add-tool windsurf
```

This creates the tool-specific config file without touching the existing setup.

## What each tool gets

Every tool gets the **universal** files that are always created:

| File | Purpose |
|------|---------|
| `AGENTS.md` | Comprehensive AI assistant guide (works with any tool) |
| `.ai-context/skills/` | MDL pattern guides shared by all tools |
| `.ai-context/examples/` | Example MDL scripts |
| `.devcontainer/` | Dev container configuration |
| `mxcli` | CLI binary |

On top of the universal files, each tool gets its own configuration:

| Tool | Config File | Contents |
|------|-------------|----------|
| **Claude Code** | `.claude/`, `CLAUDE.md` | Settings, skills, commands, lint rules, project context |
| **OpenCode** | `.opencode/`, `opencode.json` | Skills, commands, lint rules, project context |
| **Cursor** | `.cursorrules` | Compact MDL reference and mxcli command guide |
| **Continue.dev** | `.continue/config.json` | Custom commands and slash commands |
| **Windsurf** | `.windsurfrules` | MDL rules for Codeium's AI |
| **Aider** | `.aider.conf.yml` | YAML configuration for Aider |

## Tool details

### OpenCode

OpenCode receives full integration on par with Claude Code: dedicated skills in `.opencode/skills/`, slash commands in `.opencode/commands/`, and Starlark lint rules in `.claude/lint-rules/`. See the dedicated [OpenCode Integration](opencode.md) page for the complete walkthrough.

```bash
mxcli init --tool opencode /path/to/project
```

### Cursor

Cursor reads its instructions from `.cursorrules` in the project root. The file mxcli generates contains a compact MDL syntax reference and a list of mxcli commands the AI can use. Cursor's Composer and Chat features will reference this file automatically.

```bash
mxcli init --tool cursor /path/to/project
```

Created files:
- `.cursorrules` -- MDL syntax, command reference, and conventions
- `AGENTS.md` -- universal guide (Cursor also reads this)
- `.ai-context/skills/` -- shared skill files

To use: open the project in Cursor, then use Composer (Ctrl+I) or Chat (Ctrl+L) to ask for Mendix changes.

### Continue.dev

Continue.dev uses a JSON configuration file with custom commands. The generated config tells Continue about mxcli and provides slash commands for common MDL operations.

```bash
mxcli init --tool continue /path/to/project
```

Created files:
- `.continue/config.json` -- custom commands, slash command definitions
- `AGENTS.md` and `.ai-context/skills/` -- universal files

To use: open the project in VS Code with the Continue extension, then use the Continue sidebar to ask for changes.

### Windsurf

Windsurf reads `.windsurfrules` from the project root. The generated file contains MDL rules and mxcli command documentation tailored for Codeium's AI.

```bash
mxcli init --tool windsurf /path/to/project
```

Created files:
- `.windsurfrules` -- MDL rules and command reference
- `AGENTS.md` and `.ai-context/skills/` -- universal files

To use: open the project in Windsurf, then use Cascade to ask for Mendix changes.

### Aider

Aider is a terminal-based AI pair programming tool. The generated YAML config tells Aider about the project structure and available commands.

```bash
mxcli init --tool aider /path/to/project
```

Created files:
- `.aider.conf.yml` -- Aider configuration
- `AGENTS.md` and `.ai-context/skills/` -- universal files

To use: run `aider` in the project directory from the terminal.

## The universal format: AGENTS.md

Regardless of which tool you pick, mxcli always creates `AGENTS.md` and the `.ai-context/` directory. These use a universal format that most AI tools understand:

- **`AGENTS.md`** is a comprehensive guide placed in the project root. It describes mxcli, lists MDL commands, and explains the development workflow. Many AI tools (GitHub Copilot, OpenAI Codex, and others) automatically read markdown files in the project root.
- **`.ai-context/skills/`** contains the same skill files that Claude Code gets, but in the shared location. Any tool that can read project files can reference these.

This means even if your AI tool is not in the supported list, it can still benefit from `mxcli init`. The `AGENTS.md` file and skill directory provide enough context for any AI assistant to work with MDL.

## Listing supported tools

To see all tools mxcli knows about:

```bash
mxcli init --list-tools
```

## Next steps

To understand what the skill files contain and how they guide AI behavior, see [Skills and CLAUDE.md](skills.md).
