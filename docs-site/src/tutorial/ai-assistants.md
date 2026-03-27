# Working with AI Assistants

mxcli is built from the ground up to work with AI coding assistants. The `mxcli init` command sets up your Mendix project with skills, configuration files, and a dev container so that AI tools can read and modify your project using MDL.

## Why AI + MDL?

Mendix projects are stored in binary `.mpr` files. AI assistants cannot read or edit binary files directly. mxcli solves this by providing MDL -- a text-based, SQL-like language that describes Mendix model elements. An AI assistant uses mxcli commands to explore the project, writes MDL scripts to make changes, validates them, and executes them against the `.mpr` file.

The result: you describe what you want in natural language, and the AI builds it in your Mendix project.

## The token efficiency advantage

When working with AI APIs, context window size is a critical constraint. MDL's compact syntax provides a significant advantage over JSON model representations:

| Representation | Tokens for a 10-entity module |
|----------------|-------------------------------|
| JSON (raw model) | ~15,000--25,000 tokens |
| MDL | ~2,000--4,000 tokens |
| **Savings** | **5--10x fewer tokens** |

Fewer tokens means lower API costs, more of your application fits in a single prompt, and the AI produces better results because there is less noise in the context.

## Supported AI tools

mxcli supports six AI coding assistants out of the box, plus a universal format that works with any tool:

| Tool | Init Flag | Config File | Description |
|------|-----------|-------------|-------------|
| **Claude Code** | `--tool claude` (default) | `.claude/`, `CLAUDE.md` | Full integration with skills, commands, and lint rules |
| **OpenCode** | `--tool opencode` | `.opencode/`, `opencode.json` | Deep integration with skills, commands, and lint rules |
| **Cursor** | `--tool cursor` | `.cursorrules` | Compact MDL reference and command guide |
| **Continue.dev** | `--tool continue` | `.continue/config.json` | Custom commands and slash commands |
| **Windsurf** | `--tool windsurf` | `.windsurfrules` | Codeium's AI with MDL rules |
| **Aider** | `--tool aider` | `.aider.conf.yml` | Terminal-based AI pair programming |
| **Universal** | (always created) | `AGENTS.md`, `.ai-context/` | Works with all tools |

```bash
# List all supported tools
mxcli init --list-tools
```

## How it works at a glance

The workflow is the same regardless of which AI tool you use:

1. **Initialize** -- `mxcli init` creates config files, skills, and a dev container
2. **Open in dev container** -- sandboxes the AI so it only accesses your project
3. **Start the AI assistant** -- Claude Code, Cursor, Continue.dev, etc.
4. **Ask for changes in natural language** -- "Create a Customer entity with name and email"
5. **The AI explores** -- it runs SHOW, DESCRIBE, and SEARCH commands via mxcli
6. **The AI writes MDL** -- guided by the skill files installed in your project
7. **The AI validates** -- `mxcli check` catches syntax errors before anything is applied
8. **The AI executes** -- the MDL script is run against your `.mpr` file
9. **You review in Studio Pro** -- open the project and verify the result

The next pages cover each tool in detail, starting with [Claude Code](claude-code.md).
