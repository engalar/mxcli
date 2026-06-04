# Language Describe + IR Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `describe settings` to output all registered languages as re-executable MDL, remove the lossy IR describe path from page/snippet describe, and update the helpdesk demo script to exercise all language operation patterns.

**Architecture:** Three independent changes applied in sequence: (1) fix the language output bug in `describeSettings()` with a focused unit test; (2) surgically remove the IR routing branches from `describePage()` / `describeSnippet()` and delete the now-dead `pageModelHasLossyWidgetReadOnly`; (3) rewrite the helpdesk i18n section and wire `describe settings` into the golden snapshot.

**Tech Stack:** Go (`mdl/executor`), MDL scripting (`mdl-examples/use-cases/helpdesk/`), golden MPR tests (`internal/goldenfs/`)

---

## File Map

| File | Change |
|------|--------|
| `mdl/executor/cmd_settings.go` | Fix language output in `describeSettings()` (lines 144–147) |
| `mdl/executor/cmd_settings_mock_test.go` | Add `TestDescribeSettings_Languages` unit test |
| `mdl/executor/cmd_pages_describe.go` | Remove IR branches from `describePage()` (lines 129–151) and `describeSnippet()` (lines 253–270) |
| `mdl/executor/cmd_pages_create_v3.go` | Delete `pageModelHasLossyWidgetReadOnly()` (lines 250–263) |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | Rewrite i18n section |
| `cmd/mxcli/examples/helpdesk-app.mdl` | Identical rewrite (kept in sync) |
| `mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl` | Append `describe settings;` |

---

## Task 1: Write failing test for language describe

**Files:**
- Modify: `mdl/executor/cmd_settings_mock_test.go`

- [ ] **Step 1.1: Add test after the existing `TestDescribeSettings_Mock`**

Open `mdl/executor/cmd_settings_mock_test.go` and add after line 49 (after the closing `}` of `TestDescribeSettings_Mock`):

```go
func TestDescribeSettings_Languages(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages: []model.Language{
						{Code: "en_US"},
						{Code: "zh_CN"},
						{Code: "nl_NL", CheckCompleteness: true},
					},
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	assertNoError(t, describeSettings(ctx))
	out := buf.String()

	// non-default languages must appear as alter statements
	assertContainsStr(t, out, "alter settings language add 'zh_CN';")
	assertContainsStr(t, out, "alter settings language add 'nl_NL' (checkCompleteness: true);")
	// default language (en_US) must NOT appear as an add statement
	if strings.Contains(out, "alter settings language add 'en_US'") {
		t.Errorf("default language must not appear as add statement, got:\n%s", out)
	}
	// DefaultLanguageCode line must still be present
	assertContainsStr(t, out, "DefaultLanguageCode = 'en_US'")
}
```

You also need to add `"strings"` to the import block at the top of the file if it's not already there. Check with `grep -n '"strings"' mdl/executor/cmd_settings_mock_test.go`.

- [ ] **Step 1.2: Run test — confirm it FAILS**

```bash
go test ./mdl/executor/... -run TestDescribeSettings_Languages -v
```

Expected: `FAIL — assertContainsStr: string not found: "alter settings language add 'zh_CN';"` (or similar)

---

## Task 2: Fix `describeSettings()` language output

**Files:**
- Modify: `mdl/executor/cmd_settings.go:144–147`

- [ ] **Step 2.1: Replace the language block**

Find this block (lines 144–147):

```go
	// Language settings
	if ps.Language != nil {
		fmt.Fprintf(ctx.Output, "alter settings LANGUAGE\n  DefaultLanguageCode = '%s';\n\n", ps.Language.DefaultLanguageCode)
	}
```

Replace with:

```go
	// Language settings
	if ps.Language != nil {
		for _, lang := range ps.Language.Languages {
			if lang.Code == ps.Language.DefaultLanguageCode {
				continue // default language is always present, no add needed
			}
			if lang.CheckCompleteness {
				fmt.Fprintf(ctx.Output, "alter settings language add '%s' (checkCompleteness: true);\n", lang.Code)
			} else {
				fmt.Fprintf(ctx.Output, "alter settings language add '%s';\n", lang.Code)
			}
		}
		fmt.Fprintf(ctx.Output, "alter settings language\n  DefaultLanguageCode = '%s';\n\n", ps.Language.DefaultLanguageCode)
	}
```

- [ ] **Step 2.2: Run test — confirm it PASSES**

```bash
go test ./mdl/executor/... -run TestDescribeSettings_Languages -v
```

Expected: `PASS`

- [ ] **Step 2.3: Run full executor tests — no regressions**

```bash
go test ./mdl/executor/... -timeout 60s
```

Expected: `ok  github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 2.4: Commit**

```bash
git add mdl/executor/cmd_settings.go mdl/executor/cmd_settings_mock_test.go
git commit -m "fix(i18n): describe settings now outputs all registered languages"
```

---

## Task 3: Remove IR describe path from `describePage()`

**Files:**
- Modify: `mdl/executor/cmd_pages_describe.go:120–161`

The current `describePage()` end has three branches starting at line 129. The IR branch (the `else` at line 145–150) is the one to remove. The function must always use the legacy BSON path.

- [ ] **Step 3.1: Replace the three-branch block with a single legacy path**

Find this block (lines 120–151, from comment to closing brace of the if/else chain):

```go
	// Output widgets via the new PageModel IR path (Task 6 wire-up), but
	// fall back to the legacy raw BSON describe path when the IR cannot
	// fully represent the widget tree. The IR currently doesn't classify
	// pluggable widgets (DataGrid/Gallery/ComboBox/Image) beyond
	// WidgetUnknown, so describing pages that contain them would emit
	// `-- unsupported widget` lines and erase column/filter/datasource
	// information. The legacy parseRawWidget+outputWidgetMDLV3 path knows
	// how to extract those fields; keep it as fallback until Task 9
	// completes the IR coverage.
	pm, pmErr := ctx.Backend.GetPageModel(pageID)
	if pmErr != nil || pm == nil || len(pm.Widgets) == 0 {
		formatWidgetProps(ctx.Output, "", header, props, " {\n}")
	} else if pageModelHasLossyWidgetReadOnly(pm) {
		// Fallback: pluggable widget detected — use the legacy describe path
		// that has full DataGrid/Gallery/CustomWidget support.
		rawWidgets := getPageWidgetsFromRaw(ctx, pageID)
		if len(rawWidgets) > 0 {
			formatWidgetProps(ctx.Output, "", header, props, " {\n")
			for _, w := range rawWidgets {
				outputWidgetMDLV3(ctx, w, 1)
			}
			fmt.Fprint(ctx.Output, "}")
		} else {
			formatWidgetProps(ctx.Output, "", header, props, " {\n}")
		}
	} else {
		formatWidgetProps(ctx.Output, "", header, props, " {\n")
		for _, n := range pm.Widgets {
			renderWidget(ctx.Output, n, 1)
		}
		fmt.Fprint(ctx.Output, "}")
	}
```

Replace with:

```go
	// Always use the legacy raw-BSON describe path. The IR path was removed
	// because it produced lossy output (missing rendermode, contentparams,
	// column captions, etc.). The legacy path handles all widget types correctly.
	rawWidgets := getPageWidgetsFromRaw(ctx, pageID)
	if len(rawWidgets) > 0 {
		formatWidgetProps(ctx.Output, "", header, props, " {\n")
		for _, w := range rawWidgets {
			outputWidgetMDLV3(ctx, w, 1)
		}
		fmt.Fprint(ctx.Output, "}")
	} else {
		formatWidgetProps(ctx.Output, "", header, props, " {\n}")
	}
```

- [ ] **Step 3.2: Build to confirm no compile errors**

```bash
make build 2>&1 | tail -5
```

Expected: `Built bin/mxcli bin/mxcli-daemon bin/source_tree`

- [ ] **Step 3.3: Remove IR describe path from `describeSnippet()`**

Find this block in `describeSnippet()` (lines 251–271):

```go
	// Output widgets via the new PageModel IR path with legacy fallback for
	// pluggable widgets (mirrors describePage logic — see comment there).
	pm, pmErr := ctx.Backend.GetSnippetModel(snippetID)
	if pmErr == nil && pm != nil && len(pm.Widgets) > 0 {
		if pageModelHasLossyWidgetReadOnly(pm) {
			rawWidgets := getSnippetWidgetsFromRaw(ctx, snippetID)
			if len(rawWidgets) > 0 {
				fmt.Fprint(ctx.Output, " {\n")
				for _, w := range rawWidgets {
					outputWidgetMDLV3(ctx, w, 1)
				}
				fmt.Fprint(ctx.Output, "}")
			}
		} else {
			fmt.Fprint(ctx.Output, " {\n")
			for _, n := range pm.Widgets {
				renderWidget(ctx.Output, n, 1)
			}
			fmt.Fprint(ctx.Output, "}")
		}
	}
```

Replace with:

```go
	// Always use legacy raw-BSON path (IR path removed — see describePage).
	rawWidgets := getSnippetWidgetsFromRaw(ctx, snippetID)
	if len(rawWidgets) > 0 {
		fmt.Fprint(ctx.Output, " {\n")
		for _, w := range rawWidgets {
			outputWidgetMDLV3(ctx, w, 1)
		}
		fmt.Fprint(ctx.Output, "}")
	}
```

- [ ] **Step 3.4: Build again**

```bash
make build 2>&1 | tail -5
```

Expected: `Built bin/mxcli bin/mxcli-daemon bin/source_tree`

- [ ] **Step 3.5: Run executor tests**

```bash
go test ./mdl/executor/... -timeout 60s
```

Expected: `ok  github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 3.6: Commit**

```bash
git add mdl/executor/cmd_pages_describe.go
git commit -m "refactor(pages): remove IR describe path — always use legacy BSON"
```

---

## Task 4: Delete `pageModelHasLossyWidgetReadOnly`

**Files:**
- Modify: `mdl/executor/cmd_pages_create_v3.go`

`pageModelHasLossyWidget` (no suffix) is still used in the write path and must be kept. Only `pageModelHasLossyWidgetReadOnly` (with `ReadOnly`) is now dead.

- [ ] **Step 4.1: Locate the function**

```bash
grep -n "pageModelHasLossyWidgetReadOnly" mdl/executor/cmd_pages_create_v3.go
```

Expected output example: lines 250–263 define the function.

- [ ] **Step 4.2: Delete `pageModelHasLossyWidgetReadOnly`**

Delete the entire function body. It looks like:

```go
// pageModelHasLossyWidgetReadOnly is the read-side gate that still walks
// ... (comment lines)
func pageModelHasLossyWidgetReadOnly(pm *types.PageModel) bool {
	for _, n := range pm.Widgets {
		if widgetTreeHasLossyKind(n) {
			return true
		}
	}
	return false
}
```

Delete those lines entirely. Do not touch `pageModelHasLossyWidget` or `widgetTreeHasLossyKind`.

- [ ] **Step 4.3: Confirm `pageModelHasLossyWidgetReadOnly` has no remaining callers**

```bash
grep -rn "pageModelHasLossyWidgetReadOnly" mdl/
```

Expected: no output (function is gone and no callers remain).

- [ ] **Step 4.4: Build and test**

```bash
make build 2>&1 | tail -5 && go test ./mdl/executor/... -timeout 60s
```

Expected: build succeeds, tests pass.

- [ ] **Step 4.5: Commit**

```bash
git add mdl/executor/cmd_pages_create_v3.go
git commit -m "refactor(pages): delete pageModelHasLossyWidgetReadOnly — dead code after IR removal"
```

---

## Task 5: Rewrite i18n section in helpdesk-app.mdl

Both copies of `helpdesk-app.mdl` must be **identical** after this change.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
- Modify: `cmd/mxcli/examples/helpdesk-app.mdl`

- [ ] **Step 5.1: Find the i18n section start in both files**

```bash
grep -n "MARK: Multilingual" mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
```

This gives the line number where the section begins (the `-- ====` separator before the MARK comment).

- [ ] **Step 5.2: Replace everything from that separator to the end of the file**

In both files, replace from the `-- ===...=== MARK: Multilingual` separator to EOF with:

```mdl
-- ============================================================================
-- MARK: Multilingual / i18n Demo
-- ============================================================================
-- Patterns covered in order:
--   1. alter settings language add 'code'               plain add
--   2. alter settings language add 'code' (checkCompleteness: true)
--   3. show languages                                   inspect all registered
--   4. translate enumeration … in lang set …            enum caption translation
--   5. translate page … in lang set …                   page widget translation
--   6. alter settings language drop 'code'              单个语言的删除 (no translations)
--   7. alter settings language drop 'code'              整个语言的删除 (with translations)
--   8. describe settings                                 re-executable language state
-- Final MPR state: en_US (default) + zh_CN with translations.
-- ============================================================================

-- Step 1: Register three additional languages.
-- en_US is always the Mendix default — registering it again is a no-op.
-- checkCompleteness: true tells Studio Pro to flag untranslated texts for nl_NL.
alter settings language add 'zh_CN';
alter settings language add 'nl_NL' (checkCompleteness: true);
alter settings language add 'fr_FR';

-- Step 2: Inspect — four languages are now registered.
show languages;

-- Step 3: Apply zh_CN translations.
-- Enum captions use path format: <ValueName>.caption
translate enumeration HD.TicketStatus in zh_CN
  set Draft.caption      = '草稿'
  set Open.caption       = '待处理'
  set InProgress.caption = '处理中'
  set Resolved.caption   = '已解决'
  set Closed.caption     = '已关闭';

translate enumeration HD.TicketPriority in zh_CN
  set Low.caption      = '低'
  set Normal.caption   = '普通'
  set High.caption     = '高'
  set Critical.caption = '紧急';

-- Page captions use path format: <WidgetName>.<property>
-- Special path "title" translates the page title bar.
translate page HD.Ticket_Overview in zh_CN
  set title               = '工单列表'
  set btnNew.caption      = '新建工单'
  set btnEdit.caption     = '编辑'
  set colSubject.caption  = '主题'
  set colStatus.caption   = '状态'
  set colPriority.caption = '优先级'
  set colSLADue.caption   = 'SLA 截止'
  set colIsOver.caption   = '超时'
  set colActions.caption  = '操作';

-- Step 4: Apply nl_NL translations (partial — checkCompleteness will flag the rest).
translate enumeration HD.TicketStatus in nl_NL
  set Draft.caption = 'Concept'
  set Open.caption  = 'Open';

-- Step 5: 单个语言的删除 — fr_FR was never translated; drop removes the registration only.
alter settings language drop 'fr_FR';

-- Step 6: 整个语言的删除 — nl_NL had partial translations; drop removes both the
-- registration and all nl_NL Texts$Translation entries from the BSON.
alter settings language drop 'nl_NL';

-- Step 7: Final state — en_US (default) + zh_CN with translations.
show languages;

-- ============================================================================
-- End of i18n Demo
-- ============================================================================
```

- [ ] **Step 5.3: Verify both files are identical**

```bash
diff mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
```

Expected: no output (files are identical).

- [ ] **Step 5.4: Syntax-check the script**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exit 0, no errors.

- [ ] **Step 5.5: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
git commit -m "docs(i18n): rewrite helpdesk i18n demo — all language operation patterns"
```

---

## Task 6: Add `describe settings` to the snapshot script

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl`

- [ ] **Step 6.1: Append `describe settings;` at the end of the file**

```bash
echo "describe settings;" >> mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl
```

- [ ] **Step 6.2: Verify the last two lines**

```bash
tail -3 mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl
```

Expected:
```
describe page HD.EscalationWorkflow_Overview;
describe page HD.EscalationStart_Form;
describe settings;
```

- [ ] **Step 6.3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl
git commit -m "test(golden): add describe settings to helpdesk snapshot script"
```

---

## Task 7: Rebuild golden MPR and update snapshot

- [ ] **Step 7.1: Rebuild both golden versions**

```bash
make update-helpdesk-golden 2>&1 | tail -20
```

This takes ~3–4 minutes. Expected: both versions print `Golden updated: testdata/helpdesk-golden-{version}` and finish with `PASS`.

- [ ] **Step 7.2: Verify snapshot now contains language output**

```bash
grep "alter settings language" testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl
```

Expected:
```
alter settings language add 'zh_CN';
alter settings language
```

```bash
grep "alter settings language" testdata/helpdesk-golden-11.10.0/describe-snapshot.mdl
```

Expected: same output.

- [ ] **Step 7.3: Run `make test` — no failures**

```bash
make test 2>&1 | grep -E "^FAIL|^ok.*goldenfs|^ok.*executor" | head -20
```

Expected: no `FAIL` lines, both packages show `ok`.

- [ ] **Step 7.4: Commit the golden files**

```bash
git add testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/ testdata/helpdesk-clean-11.6.6/ testdata/helpdesk-clean-11.10.0/
git commit -m "test(golden): rebuild helpdesk golden with i18n demo + describe settings"
```

---

## Task 8: Full regression check

- [ ] **Step 8.1: Run full test suite**

```bash
make test 2>&1 | grep -E "^FAIL" | head -20
```

Expected: no output (no failures).

- [ ] **Step 8.2: Syntax-check all doctype examples**

```bash
for f in mdl-examples/doctype-tests/*.mdl; do
  case "$f" in *.test.mdl) continue ;; esac
  ./bin/mxcli check "$f" || echo "FAIL: $f"
done
```

Expected: no `FAIL:` lines.

---

## Self-Review

### Spec coverage

| Spec section | Covered by task |
|---|---|
| Remove IR describe path from `describePage()` | Task 3 |
| Remove IR describe path from `describeSnippet()` | Task 3 |
| Delete `pageModelHasLossyWidgetReadOnly` | Task 4 |
| Fix `describeSettings()` language loop | Task 2 |
| Unit test for language describe | Task 1 |
| Rewrite helpdesk-app.mdl i18n section | Task 5 |
| Keep both helpdesk-app.mdl copies in sync | Task 5 step 3 |
| Add `describe settings;` to snapshot script | Task 6 |
| Rebuild golden, verify output | Task 7 |
| `describe translations` NOT in snapshot | ✓ not added |

### Type consistency

- `model.Language` struct fields used: `Code` (string), `CheckCompleteness` (bool) — confirmed from `model/types.go:819–825`.
- `model.LanguageSettings` fields used: `DefaultLanguageCode` (string), `Languages` ([]Language) — confirmed from `model/types.go:810–814`.
- `describeSettings` function signature unchanged: `func describeSettings(ctx *ExecContext) error`.

### No placeholders

All code blocks are complete. All commands have expected output. No TBD/TODO/similar.
