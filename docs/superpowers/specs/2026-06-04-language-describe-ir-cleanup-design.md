# Language Describe + IR Cleanup Design

**Date**: 2026-06-04  
**Status**: Approved

## Problem

Two independent deficiencies in the describe pipeline:

1. **IR describe path is lossy.** `describePage()` / `describeSnippet()` route "non-lossy" widgets through an IR render path (`renderWidget`) that omits fields like `rendermode`, `contentparams`, and column `caption`. The IR approach is abandoned; describe must always use the raw-BSON legacy path.

2. **Language registration is invisible to describe.** `describeSettings()` only outputs `DefaultLanguageCode`. Registered languages added via `ALTER SETTINGS LANGUAGE ADD` are silently ignored. `describe settings` is therefore not idempotent — re-running its output on a fresh project does not reproduce the language state.

## Goals

- Describe for pages/snippets always produces full-fidelity output (no IR detour).
- `describe settings` outputs every registered non-default language as a re-executable `alter settings language add 'code';` statement.
- `helpdesk-app.mdl` demonstrates all language operation patterns in minimal lines.
- `describe-snapshot.mdl` captures the language state after the demo, exercising the fixed describe path.

## Non-Goals

- Adding a standalone `describe languages` command (settings already owns language state).
- Including `describe translations` output in the snapshot (mixed format, not re-executable MDL).
- Translating all pages/enumerations (only enough to make the demo meaningful).

---

## Design

### Part 1 — Remove IR describe path

**File**: `mdl/executor/cmd_pages_describe.go`

`describePage()` (lines 129–151) and `describeSnippet()` (lines 253–270) currently have three branches:

```
if GetPageModel fails        → legacy BSON path
else if hasLossyWidget(pm)   → legacy BSON path
else                         → IR renderWidget path   ← REMOVE
```

After change: both functions always use the legacy BSON path. Remove the `pm` variable, the `pageModelHasLossyWidgetReadOnly` call, and the IR rendering loop entirely.

**Dead code to delete** (only used for IR routing/rendering in describe):

| Symbol | File |
|--------|------|
| `pageModelHasLossyWidgetReadOnly()` | `cmd_pages_create_v3.go` |
| `widgetTreeHasLossyKind()` | `cmd_pages_create_v3.go` |
| `renderWidget()` | `cmd_pages_model_to_mdl.go` |
| `renderDataGridColumn()` | `cmd_pages_model_to_mdl.go` |

Before deleting, grep `renderWidget` and `renderDataGridColumn` across the executor package to confirm no other callers (CREATE/ALTER paths). If other callers exist, remove only the describe-side call sites; if none, delete the file.

Also remove the `"github.com/mendixlabs/mxcli/mdl/types"` import from `cmd_pages_describe.go` if it is only present for the IR path.

### Part 2 — Fix `describeSettings()` language output

**File**: `mdl/executor/cmd_settings.go`, function `describeSettings()`, lines 144–147.

Replace the current single-line language output with:

```go
if ps.Language != nil {
    for _, lang := range ps.Language.Languages {
        if lang.Code == ps.Language.DefaultLanguageCode {
            continue  // default language always present, no add needed
        }
        if lang.CheckCompleteness {
            fmt.Fprintf(ctx.Output,
                "alter settings language add '%s' (checkCompleteness: true);\n",
                lang.Code)
        } else {
            fmt.Fprintf(ctx.Output,
                "alter settings language add '%s';\n", lang.Code)
        }
    }
    fmt.Fprintf(ctx.Output,
        "alter settings language\n  DefaultLanguageCode = '%s';\n\n",
        ps.Language.DefaultLanguageCode)
}
```

**Output for golden state** (en_US default + zh_CN registered):

```
alter settings language add 'zh_CN';
alter settings language
  DefaultLanguageCode = 'en_US';
```

This output is idempotent: re-running it on a project that already has `zh_CN` is safe because `ALTER SETTINGS LANGUAGE ADD` is a no-op if the language is already registered.

### Part 3 — helpdesk-app.mdl i18n section

Replace the current i18n section (heavily commented-out) with a minimal, fully-executable demo. The section lives at the end of the file, after the folder organisation block.

**Operation sequence and pedagogical intent:**

| Step | Command | Demonstrates |
|------|---------|--------------|
| 1 | `alter settings language add 'zh_CN'` | plain add |
| 2 | `alter settings language add 'nl_NL' (checkCompleteness: true)` | add with option |
| 3 | `alter settings language add 'fr_FR'` | third language |
| 4 | `show languages` | inspect: 4 registered |
| 5 | `translate enumeration HD.TicketStatus in zh_CN set ...` | enum translation |
| 6 | `translate page HD.Ticket_Overview in zh_CN set ...` | page translation |
| 7 | `translate enumeration HD.TicketStatus in nl_NL set Draft... Open...` | partial translation (checkCompleteness matters) |
| 8 | `alter settings language drop 'fr_FR'` | **单个语言的删除** — no translations, drops registration only |
| 9 | `alter settings language drop 'nl_NL'` | **整个语言的删除** — had translations, drop removes all BSON text nodes |
| 10 | `show languages` | final state: en_US + zh_CN |

Final MPR state: `en_US` (default) + `zh_CN` with translations for `HD.TicketStatus` and `HD.Ticket_Overview`.

### Part 4 — Snapshot test

**File**: `internal/goldenfs/helpdesk_regression_test.go`, function `helpdeskParseableDescribeScript()`.

Append at the end of the returned script string:

```go
"describe settings;\n"
```

This causes `TestHelpdeskGolden_DescribeSnapshot` to capture the language state as re-executable MDL. The new content in `describe-snapshot.mdl`:

```
alter settings language add 'zh_CN';
alter settings language
  DefaultLanguageCode = 'en_US';
```

`describe translations` is **not** added to the snapshot script — its output contains informational table rows that are not valid MDL and would break re-execution.

---

## Sequence of implementation steps

1. Remove IR describe path and dead code (`cmd_pages_describe.go`, `cmd_pages_create_v3.go`, `cmd_pages_model_to_mdl.go`)
2. Fix `describeSettings()` language output (`cmd_settings.go`)
3. Write unit test for the new language output (mock backend, verify `alter settings language add` lines)
4. Rewrite i18n section in `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (both copies: use-cases and cmd/mxcli/examples)
5. Add `describe settings;\n` to `helpdeskParseableDescribeScript()`
6. Run `make update-helpdesk-golden` to rebuild golden MPR and snapshot
7. Run `make test` to confirm no regressions

## Test coverage

- Unit test: `describeSettings()` with Languages = [{Code: "zh_CN"}, {Code: "nl_NL", CheckCompleteness: true}], DefaultLanguageCode = "en_US" → assert output contains two `alter settings language add` lines and one `DefaultLanguageCode` line.
- Golden regression: `TestHelpdeskGolden_DescribeSnapshot` verifies the snapshot after golden rebuild.
- Existing page describe tests must continue to pass (legacy BSON path was already used for lossy pages; non-lossy pages now switch to it too — output may differ for non-lossy pages, requiring snapshot update).
