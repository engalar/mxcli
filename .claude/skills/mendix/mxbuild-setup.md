# mxbuild Setup & mx Tool Skill

This skill covers downloading mxbuild and using the `mx` tool for project validation and creation.

## When to Use This Skill

Use this when:
- The user needs to validate a project with `mx check`
- The user wants to create a blank Mendix project from scratch
- The user encounters "mx not found" or mxbuild-related errors
- You need to run `mx check` after modifying an MPR file via MDL

## Setup: Download mxbuild

mxbuild is downloaded once and cached at `~/.mxcli/mxbuild/{version}/`.

```bash
# Auto-detect version from project (recommended)
./mxcli setup mxbuild -p app.mpr

# Or specify version explicitly
./mxcli setup mxbuild --version 11.6.3
```

After download, the `mx` binary is at: `~/.mxcli/mxbuild/{version}/modeler/mx`

## Validate a Project

**Preferred** — use the mxcli wrapper (auto-resolves mx binary):

```bash
./mxcli docker check -p app.mpr
```

**Direct** — call mx binary explicitly:

```bash
~/.mxcli/mxbuild/11.6.3/modeler/mx check /path/to/app.mpr
```

Reports errors (CE-codes), warnings, and deprecations. Non-zero exit on errors.

## Create a Blank Project

**IMPORTANT:** `mx create-project` uses `--app-name` and outputs `App.mpr` in the **current directory**. It does NOT accept an output path argument.

```bash
# 1. Ensure mxbuild is available
./mxcli setup mxbuild --version 11.6.3

# 2. Create project directory and cd into it
mkdir /tmp/my-project && cd /tmp/my-project

# 3. Create blank project (outputs App.mpr in current directory)
~/.mxcli/mxbuild/11.6.3/modeler/mx create-project --app-name MyApp
```

Takes ~29 seconds. Creates an MPR v2 project with standard module structure.

**Caveat:** Blank projects have no demo users — login will fail until you configure security via MDL or Studio Pro. See `manage-security.md` for setting up demo users.

## Common Workflow: Modify and Validate

```bash
# 1. Download mxbuild (once)
./mxcli setup mxbuild -p app.mpr

# 2. Make changes via MDL
./mxcli exec changes.mdl -p app.mpr

# 3. Validate
./mxcli docker check -p app.mpr
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `mx: not found` | Run `./mxcli setup mxbuild -p app.mpr` |
| Wrong Mendix version | Use `--version` flag or check project version with `./mxcli -p app.mpr -c "SHOW VERSION"` |
| arm64 download fails | Some older versions lack arm64 builds; use Docker or Rosetta |
| `mx check` CE errors | See `cheatsheet-errors.md` for common error codes |
| `mx create-project` hangs | Ensure sufficient disk space; ~29s is normal |

## Related Skills

- [docker-workflow.md](./docker-workflow.md) — Full Docker build and run workflow
- [run-app.md](./run-app.md) — Running the app after validation
- [check-syntax.md](./check-syntax.md) — MDL syntax validation before `mx check`
- [manage-security.md](./manage-security.md) — Setting up demo users for blank projects
