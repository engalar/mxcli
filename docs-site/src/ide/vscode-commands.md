# Context Menu Commands

The VS Code extension adds three commands to the editor context menu (right-click) for `.mdl` files. These commands provide quick access to running and validating MDL without switching to a terminal.

## Run File

**Right-click > MDL: Run File**

Executes the entire `.mdl` file against the configured Mendix project. This is equivalent to:

```bash
mxcli exec script.mdl -p app.mpr
```

Output appears in the integrated terminal. Any errors from execution (missing references, invalid operations) are displayed inline.

## Check File

**Right-click > MDL: Check File**

Validates the file's syntax without executing it. This is equivalent to:

```bash
mxcli check script.mdl
```

If a project is configured, reference validation is also performed:

```bash
mxcli check script.mdl -p app.mpr --references
```

Check results appear in the Problems panel alongside the live diagnostics from the language server.

## Run Selection

**Right-click > MDL: Run Selection**

Executes only the currently selected text as MDL. This is useful for:

- Running a single `CREATE` statement from a larger script
- Testing a `SHOW` or `DESCRIBE` command interactively
- Executing a `REFRESH CATALOG` before running queries

The selection must contain one or more complete MDL statements (terminated by `;`).

## Keyboard Shortcuts

The commands do not have default keyboard shortcuts, but you can bind them in VS Code:

1. Open **Keyboard Shortcuts** (Ctrl+K Ctrl+S)
2. Search for "MDL"
3. Assign your preferred key bindings

## Terminal Integration

All commands use the VS Code integrated terminal. Output from `mxcli` includes clickable links for qualified names -- Ctrl+click on `MyModule.EntityName` in the terminal output to open the element's definition.
