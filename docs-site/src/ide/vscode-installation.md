# Installation

There are three ways to install the VS Code MDL extension.

## Option 1: mxcli init (Recommended)

Running `mxcli init` on a Mendix project automatically installs the VS Code extension along with skills, commands, and configuration files:

```bash
mxcli init /path/to/my-mendix-project
```

The command copies the bundled `.vsix` file into the project and installs it into VS Code. This is the recommended approach because it also sets up the full AI-assisted development environment.

## Option 2: Manual VSIX Install

If you already have the VSIX file (e.g., from the mxcli build artifacts), install it directly:

```bash
code --install-extension vscode-mdl/vscode-mdl-*.vsix
```

Or from the VS Code Command Palette:

1. Open **Extensions: Install from VSIX...** (Ctrl+Shift+P)
2. Navigate to the `.vsix` file
3. Click **Install**

## Option 3: Build from Source

If you are developing or modifying the extension:

```bash
# Requires bun (not npm/node)
cd vscode-mdl
bun install
bun run compile

# Package as VSIX
make vscode-ext

# Install the built VSIX
make vscode-install
```

> **Note:** The VS Code extension build toolchain uses **bun**, not npm or node. The Makefile targets `make vscode-ext` and `make vscode-install` handle this automatically.

## Verifying the Installation

After installation, open any `.mdl` file in VS Code. You should see:

1. **Syntax highlighting** -- MDL keywords like `CREATE`, `SHOW`, and `DESCRIBE` are colored
2. **Parse diagnostics** -- Syntax errors appear as red underlines immediately
3. **Status bar** -- The MDL language server status appears in the bottom bar

## Configuring the Extension

Open VS Code settings (Ctrl+,) and search for "MDL":

| Setting | Default | Description |
|---------|---------|-------------|
| `mdl.mxcliPath` | `mxcli` | Path to the mxcli binary. Set this if `mxcli` is not on your `PATH`. |
| `mdl.mprPath` | *(empty)* | Path to the `.mpr` file. If empty, the extension auto-discovers it by searching the workspace root. |

Example `settings.json`:

```json
{
  "mdl.mxcliPath": "./mxcli",
  "mdl.mprPath": "./app.mpr"
}
```

## Dev Container Setup

When using `mxcli init`, a `.devcontainer/` configuration is also created. Opening the project in VS Code and choosing **Reopen in Container** gives you a sandboxed environment with:

| Component | Purpose |
|-----------|---------|
| **mxcli** | Mendix CLI binary (copied into the project) |
| **MxBuild / mx** | Project validation and building |
| **JDK 21** | Required by MxBuild |
| **Docker-in-Docker** | Running Mendix apps locally |
| **Node.js** | Playwright testing support |
| **PostgreSQL client** | Database connectivity |
| **Claude Code** | AI coding assistant (auto-installed) |

The VS Code extension is automatically available inside the dev container.
