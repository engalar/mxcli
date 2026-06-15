# mxcli widget — Improvement Design

**Date:** 2026-06-15  
**Scope:** `cmd/mxcli/widget_scaffold.go`, `cmd/mxcli/widget_build.go`, `cmd/mxcli/cmd_widget.go`

---

## Summary

Three focused improvements to `mxcli widget`:

1. **Scaffold completeness** — `widget new` generates `.gitignore`, `README.md`, supports `--description`, validates `--id` early
2. **Install integration** — new `widget install` command + `widget build --install` flag
3. **`widget list` clarity** — remove misleading "run extract" suggestion

No TypeScript changes; JSX stays as-is.

---

## Section 1: Scaffold improvements

### New generated files

`mxcli widget new <name>` adds two files at the project root alongside `package.json` / `package.xml`:

**`.gitignore`**
```
node_modules/
dist/
*.mpk
```

**`README.md`** — content adapts to declared properties and `--description`:
```markdown
# <Name>

<description or empty line>

## Build

```bash
mxcli widget build
```

## Install into a Mendix project

```bash
mxcli widget build --install -p /path/to/app.mpr
```

## Properties

| Property | Type | Required |
|----------|------|----------|
| value    | attribute (Decimal) | Yes |
| label    | string | No |
```

If no properties are declared the Properties table is omitted.

### New `--description` flag

Both `widget new` and `add-widget` accept `--description "text"`:
- Written into `<description>text</description>` in the widget XML (currently always empty)
- Written into the README opening paragraph
- Default: empty string (behaviour unchanged when flag is omitted)

### Early `--id` validation

`validateWidgetIDFormat(id string) error` is extracted from `validateWidgetInfo` and called in `runWidgetNew` immediately after flag parsing, before any files are written. A bad `--id` fails fast with the existing error message.

`add-widget` does not accept `--id` today; no change needed there.

---

## Section 2: `widget install` + `build --install`

### New command: `mxcli widget install`

```
mxcli widget install -p app.mpr [--mpk path.mpk]
```

**Auto-detect mode** (no `--mpk`): globs `*.mpk` in the current working directory.
- 0 found → error: `"no .mpk file found — run 'mxcli widget build' first"`
- 2+ found → error: `"multiple .mpk files found — specify one with --mpk"`
- 1 found → proceed

**Explicit mode** (`--mpk`): uses the given path directly (any directory).

**Behaviour:**
1. Resolve `<project-dir>/widgets/` from the `-p` MPR path; create it if absent (`os.MkdirAll`)
2. Copy MPK to `<project-dir>/widgets/<filename>.mpk` (overwrite if exists)
3. Print: `Installed <name>.mpk → <project-dir>/widgets/`

**Implementation:** shared `installMPK(mpkPath, projectPath string) error` function in `widget_build.go` (or a new `widget_install.go`).

### `widget build --install -p app.mpr`

`widgetBuildCmd` gains two flags:
- `--install` (bool) — trigger install after successful build
- `-p / --project` (string) — MPR path, required when `--install` is set

After `packageMPK` and `verifyMPK` succeed, if `--install` is set, call `installMPK(mpkPath, projectPath)`.

Output line appended on success:
```
Installed → <project-dir>/widgets/<name>.mpk
```

Error from install does not suppress the "Built …" line — build succeeded; install is a separate step.

---

## Section 3: `widget list` message fix

### Title change

```diff
---- Discovered in widgets/*.mpk (not yet extracted) ---
+--- Auto-discovered from widgets/*.mpk ---
```

### Footer change

```diff
-Run 'mxcli widget extract --mpk widgets/<file>.mpk' to generate .def.json with property names
+MPK widgets are auto-discovered — no extraction needed.
+To override property mappings: mxcli widget extract --mpk widgets/<file>.mpk
```

Change is in `runWidgetList` in `cmd_widget.go` — two `fmt.Fprintf` calls.

---

## Affected files

| File | Changes |
|------|---------|
| `cmd/mxcli/widget_scaffold.go` | `generateGitignore()`, `generateReadme()`, `--description` flag, early `--id` validation, `scaffoldWidget` writes 2 new files |
| `cmd/mxcli/widget_build.go` | `installMPK()` function (or new `widget_install.go`) |
| `cmd/mxcli/cmd_widget.go` | `widget install` command registration, `--install`/`-p` flags on build, `widget list` message fix |
| `cmd/mxcli/widget_scaffold_test.go` | Tests for `generateGitignore`, `generateReadme`, early ID validation |

---

## Out of scope

- TypeScript / `.tsx` support
- Watch mode
- `add-widget --description` propagating to an existing README (README only written by `widget new`)
- Changing the JSX stub content
