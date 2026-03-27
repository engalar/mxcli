# mxcli init

The `mxcli init` command prepares a Mendix project for AI-assisted development. It creates configuration files, installs skills, sets up a dev container, and optionally installs the VS Code extension.

## Basic Usage

```bash
# Initialize with Claude Code (default)
mxcli init /path/to/my-mendix-project

# Initialize for a specific AI tool
mxcli init --tool cursor /path/to/my-mendix-project

# Initialize for multiple tools
mxcli init --tool claude --tool cursor /path/to/my-mendix-project

# Initialize for all supported tools
mxcli init --all-tools /path/to/my-mendix-project

# List supported tools
mxcli init --list-tools
```

## Supported AI Tools

| Tool | Flag | Config Files Created |
|------|------|---------------------|
| **Claude Code** (default) | `--tool claude` | `.claude/settings.json`, `CLAUDE.md`, commands, lint rules |
| **OpenCode** | `--tool opencode` | `.opencode/`, `opencode.json`, commands, skills, lint rules |
| **Cursor** | `--tool cursor` | `.cursorrules` |
| **Continue.dev** | `--tool continue` | `.continue/config.json` |
| **Windsurf** | `--tool windsurf` | `.windsurfrules` |
| **Aider** | `--tool aider` | `.aider.conf.yml` |

All tools also receive the universal files (`AGENTS.md`, `.ai-context/`).

## What Happens During Init

1. **Detect the Mendix project** -- finds the `.mpr` file in the target directory
2. **Create universal AI context** -- `AGENTS.md` and `.ai-context/skills/` with MDL pattern guides
3. **Create tool-specific configuration** -- based on the selected `--tool` flag(s)
4. **Set up dev container** -- `.devcontainer/` with Dockerfile and configuration
5. **Copy mxcli binary** -- places the mxcli executable in the project root
6. **Install VS Code extension** -- copies and installs the bundled `.vsix` file

## Adding a Tool Later

To add support for an additional AI tool to an existing project:

```bash
mxcli add-tool cursor
```

This creates only the tool-specific files without overwriting existing configuration.

## Related Pages

- [What Gets Created](./init-output.md) -- detailed directory listing
- [Customizing Skills](./customizing-skills.md) -- modifying generated skill files
- [Syncing with Updates](./syncing.md) -- keeping files current after mxcli upgrades
