# Syncing with Updates

When you upgrade `mxcli` to a newer version, the skills, commands, and VS Code extension bundled with it may have changed. This page explains how to keep your project's files in sync.

## What Gets Updated

| Component | Source of Truth | Sync Target |
|-----------|----------------|-------------|
| Skills | `reference/mendix-repl/templates/.claude/skills/` | `.ai-context/skills/` |
| Commands | `.claude/commands/mendix/` | `.claude/commands/mendix/` |
| VS Code extension | `vscode-mdl/vscode-mdl-*.vsix` | Installed extension |
| Lint rules | Bundled Starlark rules | `.claude/lint-rules/` |

## Re-running mxcli init

The simplest way to sync is to re-run `mxcli init`:

```bash
mxcli init /path/to/my-mendix-project
```

This overwrites the generated skill files with the latest versions. **Custom modifications to built-in skill files will be lost.** To preserve customizations:

1. Keep custom skills in separate files (e.g., `my-custom-pattern.md`)
2. Or use version control to merge changes

## Build-Time Sync (for mxcli developers)

When building mxcli from source, `make build` automatically syncs all assets:

```bash
make build          # Syncs everything
make sync-skills    # Skills only
make sync-commands  # Commands only
make sync-vsix      # VS Code extension only
```

## Checking Versions

To see which version of mxcli and its bundled assets you have:

```bash
mxcli version
```

## Recommended Workflow

1. **Keep custom skills separate** from built-in skills so re-syncing does not overwrite them
2. **Use version control** for your `.ai-context/` and `.claude/` directories
3. **Re-run `mxcli init`** after upgrading mxcli to pick up new skills and bug fixes
4. **Review the diff** after syncing to see what changed in the skill files
