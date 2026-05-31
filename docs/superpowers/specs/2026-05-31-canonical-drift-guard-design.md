# Canonical Model Drift Guard — Design Spec

**Date:** 2026-05-31
**Status:** Approved

## Problem

`cmd_diff_mdl.go` contains two serialization paths per document type:
- `*StmtToMDL` — AST → MDL text
- `*ToMDLGen` — gen (BSON) → MDL text

Phase 1 migrated `entity` to the canonical model layer (`mdl/model/entity/`), where both paths converge on a single `ToMDL()` method. Domains not yet migrated (enumeration, association, microflow, nanoflow, page, …) still use inline serialization. Edits to these functions risk divergence between the diff and describe paths.

## Goal

A non-blocking pre-commit check that warns developers when staged changes touch non-migrated serialization functions, and also catches new functions added with the old pattern.

## Scope

- **Input:** `git diff --cached` output, `mdl/executor/` Go source files
- **Out of scope:** blocking commits, detecting drift in already-committed code, covering non-executor packages

---

## Architecture

```
.githooks/checks/06-canonical-model-drift.sh     ← shell entry point, always exit 0
        │
        │  (skipped if no mdl/executor/*.go staged)
        ▼
tools/check-canonical-drift/main.go              ← AST analyzer
        │
        ├─ reads stdin: git diff --cached --unified=0
        │       → parses changed line ranges per file
        │
        ├─ go/parser.ParseDir("mdl/executor/")
        │       → finds all FuncDecl matching *StmtToMDL or *ToMDLGen
        │       → AST walk: has .ToMDL() call? → migrated (skip)
        │       → no .ToMDL() call → unmigtated { file, name, startLine, endLine }
        │
        └─ cross-match changed lines ∩ unmigtated function ranges → violations
                → print WARNING to stderr, exit 0
```

Two trigger scenarios handled inside the analyzer:

1. **Modify existing unmigtated function** — changed lines intersect `[startLine, endLine]`
2. **Add new old-pattern function** — diff contains `+func .*StmtToMDL` or `+func .*ToMDLGen`; analyzer checks new function body for `.ToMDL()` call

---

## Detection Logic

### Identifying unmigtated functions (AST)

```
for each FuncDecl in mdl/executor/**/*.go (non-test):
    if funcName does not match .*StmtToMDL$ or .*ToMDLGen$: skip
    walk function body AST:
        if any SelectorExpr.Sel.Name == "ToMDL": mark as migrated → skip
    else: record as unmigtated { file, funcName, startLine, endLine }
```

### Diff cross-match

Analyzer reads `git diff --cached --unified=0` from stdin.
Parses `@@` hunk headers into `{ file → [(startLine, count)] }`.

```
for each unmigtated function f:
    if staged_lines[f.file] ∩ [f.startLine, f.endLine] ≠ ∅:
        add to violations
```

### New function detection

```
for each +func line in diff matching .*StmtToMDL or .*ToMDLGen:
    collect the entire added function block from diff
    if no ".ToMDL()" in block:
        add to violations (no line-range cross-match needed)
```

### Automatic convergence

As domains migrate, their `*StmtToMDL` / `*ToMDLGen` functions gain a `.ToMDL()` call. The analyzer detects this automatically — no allowlist to maintain.

---

## Warning Output Format

```
⚠ CANONICAL MODEL DRIFT WARNING (non-blocking)

  mdl/executor/cmd_diff_mdl.go: associationStmtToMDL
    Not yet migrated to the canonical model layer.
    Editing risks divergence between diff and describe paths.

  mdl/executor/cmd_diff_mdl.go: microflowStmtToMDL
    Not yet migrated to the canonical model layer.
    Editing risks divergence between diff and describe paths.

Migration plan: docs/superpowers/plans/2026-05-23-canonical-model-layer-phase1.md
Silence this warning by migrating the domain (add .ToMDL() call in function body).
```

The warning prints to stderr. The check never prints to stdout. Exit code is always 0.

---

## Shell Wrapper

**`.githooks/checks/06-canonical-model-drift.sh`**

```sh
#!/bin/sh
# Warn when staged changes touch non-migrated canonical model functions.
# Non-blocking: always exit 0.

STAGED=$(git diff --cached --name-only | grep "^mdl/executor/.*\.go$" | grep -v "_test\.go")
[ -z "$STAGED" ] && exit 0

git diff --cached --unified=0 | go run ./tools/check-canonical-drift/ >&2
exit 0
```

- Skips entirely when no executor Go files are staged (fast path)
- Passes diff via stdin to avoid temp files
- `go run` with module cache runs in ~200ms after first build

---

## File Map

| File | Purpose |
|------|---------|
| `.githooks/checks/06-canonical-model-drift.sh` | Shell entry point |
| `tools/check-canonical-drift/main.go` | AST analyzer |
| `tools/check-canonical-drift/main_test.go` | Unit tests with Go fixture strings |

---

## Testing

`main_test.go` covers three golden cases using in-memory Go source strings (no file I/O needed):

| Case | Input | Expected output |
|------|-------|-----------------|
| Migrated function | `func entityStmtToMDL` body calls `m.ToMDL()` | no warning |
| Unmigtated function touched | `func associationStmtToMDL` body has no `.ToMDL()`, diff touches it | WARNING printed |
| New old-pattern function added | diff adds `+func fooStmtToMDL` without `.ToMDL()` | WARNING printed |

---

## Integration

| Aspect | Decision |
|--------|----------|
| Check number | `06-` — follows `05-mx-check-golden.sh` |
| Exit code | Always `0` — warn, never block |
| Test files | Excluded via `grep -v "_test\.go"` in shell and `go/build` tag filter in analyzer |
| Grandfathered files | None needed — detection is semantic (`.ToMDL()` presence), not file-based |
| Maintenance | Zero — converges to silence automatically as migration progresses |
