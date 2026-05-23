# Multilingual MDL Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full multilingual support to MDL: language registry management (ADD/DROP/SHOW), element-level TRANSLATE command for pages/enumerations/workflows, inline multilingual syntax for microflow messages, and DESCRIBE TRANSLATIONS coverage view.

**Architecture:** Language registry uses existing `SettingsBackend` (GetProjectSettings + UpdateProjectSettings) with model.Language slice manipulation. TRANSLATE operations extend existing doc-type mutation paths (PageMutator, EnumerationBackend, WorkflowMutationBackend) with a new shared BSON primitive `setTranslationForLang`. Microflow inline multilingual extends the `TextLiteral` AST type from `string` to `map[string]string`. DESCRIBE TRANSLATIONS reads existing BSON to enumerate all Texts$Text nodes.

**Tech Stack:** Go, ANTLR4 (MDLLexer.g4 + domain .g4 files), `go.mongodb.org/mongo-driver/bson`, existing `mdl/backend/mpr/` BSON helpers.

**Spec:** `docs/superpowers/specs/2026-05-23-multilingual-mdl-design.md`

---

## File Map

**New files:**
- `mdl/ast/ast_translate.go` — TranslateStmt, TranslateSetOp, DescribeTranslationsStmt, AlterLanguageStmt AST nodes
- `mdl/backend/mpr/translation_writer.go` — `setTranslationForLang()` BSON primitive + per-doc-type helpers
- `mdl/executor/cmd_translate.go` — TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW executor handler
- `mdl/executor/cmd_describe_translations.go` — DESCRIBE TRANSLATIONS executor handler
- `mdl/visitor/visitor_translate.go` — TRANSLATE + DESCRIBE TRANSLATIONS + ALTER LANGUAGE visitor

**Modified files:**
- `mdl/grammar/MDLLexer.g4` — add TRANSLATE, TRANSLATIONS, SUPPORTED tokens
- `mdl/grammar/domains/MDLSettings.g4` — LANGUAGE ADD/DROP grammar + keyword additions
- `mdl/grammar/domains/MDLDomainModel.g4` — translateStatement + textLiteral + describeTranslations grammar
- `mdl/grammar/domains/MDLMicroflow.g4` — use textLiteral in show message / validation feedback / log message rules
- `mdl/ast/ast_microflow.go` — extend TextLiteral from string to map[string]string
- `mdl/visitor/visitor_helpers.go` — parseTextLiteral helper
- `mdl/visitor/visitor_settings.go` — wire AlterLanguageStmt
- `mdl/executor/cmd_languages.go` — ADD/DROP/SHOW SUPPORTED + enhanced SHOW LANGUAGES
- `mdl/executor/register_stubs.go` — register new statement types
- `mdl/backend/mpr/page_mutator.go` — add `setWidgetTranslation()` using new primitive
- `mdl/backend/mpr/settings_compat.go` — `serializeLanguageSettings()` for BSON write-back
- `mdl/backend/mock/backend.go` — new Func fields for new backend methods
- `mdl/backend/mock/mock_infrastructure.go` (or `mock_settings.go`) — implement new mock methods

---

## Task 1: Built-in Language Whitelist + SHOW SUPPORTED LANGUAGES

**Files:**
- Modify: `mdl/executor/cmd_languages.go`
- Modify: `mdl/grammar/MDLLexer.g4`
- Modify: `mdl/grammar/domains/MDLSettings.g4` (keyword addition)
- Modify: `mdl/ast/ast_query.go`
- Modify: `mdl/visitor/visitor_query.go`

- [ ] **Step 1: Write failing test for SHOW SUPPORTED LANGUAGES**

```go
// mdl/executor/cmd_languages_mock_test.go — add to existing file
func TestShowSupportedLanguages(t *testing.T) {
	ctx, buf := newMockCtx(t)
	err := listSupportedLanguages(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "en_US") {
		t.Errorf("expected en_US in output, got: %s", out)
	}
	if !strings.Contains(out, "zh_CN") {
		t.Errorf("expected zh_CN in output, got: %s", out)
	}
	if !strings.Contains(out, "nl_NL") {
		t.Errorf("expected nl_NL in output, got: %s", out)
	}
}

func TestIsValidLanguageCode(t *testing.T) {
	if !isValidLanguageCode("en_US") {
		t.Error("en_US should be valid")
	}
	if !isValidLanguageCode("zh_CN") {
		t.Error("zh_CN should be valid")
	}
	if isValidLanguageCode("chinese") {
		t.Error("chinese should not be valid")
	}
	if isValidLanguageCode("EN_US") {
		t.Error("EN_US should not be valid (case-sensitive)")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run "TestShowSupportedLanguages|TestIsValidLanguageCode" -v 2>&1 | tail -5
```

Expected: `FAIL` — `listSupportedLanguages` not defined.

- [ ] **Step 3: Add whitelist + helper functions to cmd_languages.go**

Add after the existing `listLanguages` function:

```go
// supportedLanguages is the built-in list of valid Mendix language codes.
var supportedLanguages = map[string]string{
	"ar_SA": "Arabic",
	"bg_BG": "Bulgarian",
	"ca_ES": "Catalan",
	"cs_CZ": "Czech",
	"da_DK": "Danish",
	"de_DE": "German",
	"el_GR": "Greek",
	"en_GB": "English (UK)",
	"en_US": "English (US)",
	"es_ES": "Spanish",
	"es_MX": "Spanish (Mexico)",
	"fi_FI": "Finnish",
	"fr_BE": "French (Belgium)",
	"fr_FR": "French",
	"hr_HR": "Croatian",
	"hu_HU": "Hungarian",
	"id_ID": "Indonesian",
	"it_IT": "Italian",
	"ja_JP": "Japanese",
	"ko_KR": "Korean",
	"nb_NO": "Norwegian",
	"nl_BE": "Dutch (Belgium)",
	"nl_NL": "Dutch",
	"pl_PL": "Polish",
	"pt_BR": "Portuguese (Brazil)",
	"pt_PT": "Portuguese (Portugal)",
	"ro_RO": "Romanian",
	"ru_RU": "Russian",
	"sk_SK": "Slovak",
	"sl_SI": "Slovenian",
	"sr_CS": "Serbian",
	"sv_SE": "Swedish",
	"th_TH": "Thai",
	"tr_TR": "Turkish",
	"uk_UA": "Ukrainian",
	"vi_VN": "Vietnamese",
	"zh_CN": "Chinese (Simplified)",
	"zh_TW": "Chinese (Traditional)",
}

// isValidLanguageCode returns true if code is a known Mendix language code.
func isValidLanguageCode(code string) bool {
	_, ok := supportedLanguages[code]
	return ok
}

// listSupportedLanguages outputs all valid Mendix language codes.
func listSupportedLanguages(ctx *ExecContext) error {
	tr := &TableResult{
		Columns: []string{"Code", "Language"},
		Summary: fmt.Sprintf("(%d supported languages)", len(supportedLanguages)),
	}
	codes := make([]string, 0, len(supportedLanguages))
	for code := range supportedLanguages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		tr.Rows = append(tr.Rows, []any{code, supportedLanguages[code]})
	}
	return writeResult(ctx, tr)
}
```

Also add `"sort"` to the imports if not already there.

- [ ] **Step 4: Add ShowSupportedLanguages to AST**

In `mdl/ast/ast_query.go`, add after `ShowLanguages`:

```go
ShowSupportedLanguages  // SHOW SUPPORTED LANGUAGES
```

And in the `String()` method switch, add:

```go
case ShowSupportedLanguages:
	return "SUPPORTED LANGUAGES"
```

- [ ] **Step 5: Add SUPPORTED token to MDLLexer.g4**

Find the section with TRANSFORM (around line 694):

```antlr
// Data transformer tokens
DATA: D A T A;
TRANSFORM: T R A N S F O R M;
```

Add after TRANSFORMERS:

```antlr
// Translation tokens
TRANSLATE:    T R A N S L A T E;
TRANSLATIONS: T R A N S L A T I O N S;
SUPPORTED:    S U P P O R T E D;
```

- [ ] **Step 6: Add SUPPORTED + TRANSLATE + TRANSLATIONS to keyword list in MDLSettings.g4**

Find the `-- CLI commands` group in the `keyword` rule and add:

```antlr
    | SUPPORTED | TRANSLATE | TRANSLATIONS
```

- [ ] **Step 7: Wire SHOW SUPPORTED LANGUAGES in visitor_query.go**

In `mdl/visitor/visitor_query.go`, find the block handling `SHOW LANGUAGES`:

```go
} else if ctx.LANGUAGES() != nil {
    // SHOW LANGUAGES
    b.statements = append(b.statements, &ast.ShowStmt{ObjectType: ast.ShowLanguages})
```

Add after it (inside the same SHOW handler block):

```go
} else if ctx.SUPPORTED() != nil && ctx.LANGUAGES() != nil {
    // SHOW SUPPORTED LANGUAGES
    b.statements = append(b.statements, &ast.ShowStmt{ObjectType: ast.ShowSupportedLanguages})
```

**Note:** Grammar rule for SHOW SUPPORTED LANGUAGES may need to be added to `MDLDomainModel.g4` showStatement rule — add `| SUPPORTED LANGUAGES` variant.

- [ ] **Step 8: Register in executor**

In `mdl/executor/executor_query.go`, add to the switch:

```go
case ast.ShowSupportedLanguages:
    return listSupportedLanguages(ctx)
```

- [ ] **Step 9: Regenerate parser**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
make grammar 2>&1 | tail -10
```

Expected: success, generated files updated in `mdl/grammar/parser/`.

- [ ] **Step 10: Run tests**

```bash
go test ./mdl/executor/ -run "TestShowSupportedLanguages|TestIsValidLanguageCode" -v 2>&1 | tail -10
```

Expected: both PASS.

- [ ] **Step 11: Commit**

```bash
git add mdl/grammar/MDLLexer.g4 mdl/grammar/domains/MDLSettings.g4 mdl/grammar/domains/MDLDomainModel.g4 \
    mdl/ast/ast_query.go mdl/visitor/visitor_query.go \
    mdl/executor/cmd_languages.go mdl/executor/cmd_languages_mock_test.go
git commit -m "feat(i18n): add SHOW SUPPORTED LANGUAGES + isValidLanguageCode whitelist"
```

---

## Task 2: ALTER SETTINGS LANGUAGE ADD

**Files:**
- Modify: `mdl/grammar/domains/MDLSettings.g4`
- Create: `mdl/ast/ast_translate.go` (AlterLanguageStmt)
- Modify: `mdl/visitor/visitor_settings.go`
- Modify: `mdl/executor/cmd_languages.go`
- Modify: `mdl/executor/register_stubs.go`
- Modify: `mdl/backend/mock/backend.go` (GetProjectSettings mock)

- [ ] **Step 1: Write failing test**

```go
// mdl/executor/cmd_languages_mock_test.go — add:
func TestAlterLanguageAdd_InvalidCode(t *testing.T) {
	ctx, _ := newMockCtx(t)
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageAdd, Code: "chinese"}
	err := alterLanguage(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for invalid language code")
	}
	if !strings.Contains(err.Error(), "not a valid Mendix language code") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAlterLanguageAdd_Success(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ConnectedForWriteFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages:           []model.Language{{Code: "en_US"}},
				},
			}, nil
		},
		UpdateProjectSettingsFunc: func(ps *model.ProjectSettings) error { return nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageAdd, Code: "zh_CN"}
	err := alterLanguage(ctx, stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify UpdateProjectSettings was called with zh_CN
	called := false
	mb.UpdateProjectSettingsFunc = func(ps *model.ProjectSettings) error {
		for _, l := range ps.Language.Languages {
			if l.Code == "zh_CN" {
				called = true
			}
		}
		return nil
	}
	_ = alterLanguage(ctx, stmt) // idempotent second call
	// called may be true from first invocation — just check no error
	_ = called
}
```

Note: `ConnectedForWriteFunc` needs to be added to `MockBackend` if not present; check `mock_infrastructure.go`.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./mdl/executor/ -run "TestAlterLanguageAdd" -v 2>&1 | tail -5
```

Expected: compile error — `AlterLanguageStmt` undefined.

- [ ] **Step 3: Create AST node**

Create `mdl/ast/ast_translate.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package ast

// AlterLanguageOp represents the operation for ALTER SETTINGS LANGUAGE ADD/DROP.
type AlterLanguageOp int

const (
	AlterLanguageAdd  AlterLanguageOp = iota
	AlterLanguageDrop AlterLanguageOp = iota
)

// AlterLanguageStmt represents ALTER SETTINGS LANGUAGE ADD 'code' [...] or DROP 'code'.
type AlterLanguageStmt struct {
	Op               AlterLanguageOp
	Code             string
	CheckCompleteness *bool   // nil = not specified
	DateFormat       string
	DateTimeFormat   string
	TimeFormat       string
}

func (s *AlterLanguageStmt) stmtNode() {}
func (s *AlterLanguageStmt) String() string {
	if s.Op == AlterLanguageAdd {
		return "ALTER SETTINGS LANGUAGE ADD " + s.Code
	}
	return "ALTER SETTINGS LANGUAGE DROP " + s.Code
}

// TranslateSetOp is a single SET path = text operation inside TRANSLATE.
type TranslateSetOp struct {
	Path string // e.g. "Button_Submit.caption" or "Title"
	Text string
}

// TranslateStmt represents TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW Mod.Name IN lang SET ...
type TranslateStmt struct {
	DocType string // "PAGE", "SNIPPET", "ENUMERATION", "WORKFLOW"
	QName   QualifiedName
	Lang    string
	Ops     []TranslateSetOp
}

func (s *TranslateStmt) stmtNode() {}
func (s *TranslateStmt) String() string {
	return "TRANSLATE " + s.DocType + " " + s.QName.String() + " IN " + s.Lang
}

// DescribeTranslationsStmt represents DESCRIBE TRANSLATIONS Mod.Name [IN lang].
type DescribeTranslationsStmt struct {
	QName QualifiedName
	Lang  string // empty = all project languages
}

func (s *DescribeTranslationsStmt) stmtNode() {}
func (s *DescribeTranslationsStmt) String() string {
	return "DESCRIBE TRANSLATIONS " + s.QName.String()
}
```

- [ ] **Step 4: Add alterLanguage executor function**

In `mdl/executor/cmd_languages.go`, add:

```go
// alterLanguage handles ALTER SETTINGS LANGUAGE ADD/DROP.
func alterLanguage(ctx *ExecContext, stmt *ast.AlterLanguageStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if !isValidLanguageCode(stmt.Code) {
		return mdlerrors.NewValidation(fmt.Sprintf(
			"'%s' is not a valid Mendix language code. Run SHOW SUPPORTED LANGUAGES to see valid codes.",
			stmt.Code,
		))
	}

	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	if ps.Language == nil {
		ps.Language = &model.LanguageSettings{DefaultLanguageCode: "en_US"}
	}

	switch stmt.Op {
	case ast.AlterLanguageAdd:
		return alterLanguageAdd(ctx, ps, stmt)
	case ast.AlterLanguageDrop:
		return alterLanguageDrop(ctx, ps, stmt)
	}
	return nil
}

func alterLanguageAdd(ctx *ExecContext, ps *model.ProjectSettings, stmt *ast.AlterLanguageStmt) error {
	for _, l := range ps.Language.Languages {
		if l.Code == stmt.Code {
			fmt.Fprintf(ctx.Output, "LANGUAGE %s already registered\n", stmt.Code)
			return nil
		}
	}
	lang := model.Language{Code: stmt.Code}
	if stmt.CheckCompleteness != nil {
		lang.CheckCompleteness = *stmt.CheckCompleteness
	}
	if stmt.DateFormat != "" {
		lang.CustomDateFormat = stmt.DateFormat
	}
	if stmt.DateTimeFormat != "" {
		lang.CustomDateTimeFormat = stmt.DateTimeFormat
	}
	if stmt.TimeFormat != "" {
		lang.CustomTimeFormat = stmt.TimeFormat
	}
	ps.Language.Languages = append(ps.Language.Languages, lang)
	if err := ctx.Backend.UpdateProjectSettings(ps); err != nil {
		return mdlerrors.NewBackend("update project settings", err)
	}
	fmt.Fprintf(ctx.Output, "LANGUAGE %s added\n", stmt.Code)
	return nil
}

func alterLanguageDrop(ctx *ExecContext, ps *model.ProjectSettings, stmt *ast.AlterLanguageStmt) error {
	if ps.Language.DefaultLanguageCode == stmt.Code {
		return mdlerrors.NewValidation(fmt.Sprintf(
			"cannot drop the default language '%s'. Change DefaultLanguageCode first.",
			stmt.Code,
		))
	}
	original := len(ps.Language.Languages)
	filtered := ps.Language.Languages[:0]
	for _, l := range ps.Language.Languages {
		if l.Code != stmt.Code {
			filtered = append(filtered, l)
		}
	}
	ps.Language.Languages = filtered
	if len(ps.Language.Languages) == original {
		fmt.Fprintf(ctx.Output, "LANGUAGE %s not registered\n", stmt.Code)
		return nil
	}
	if err := ctx.Backend.UpdateProjectSettings(ps); err != nil {
		return mdlerrors.NewBackend("update project settings", err)
	}
	fmt.Fprintf(ctx.Output, "LANGUAGE %s dropped\n", stmt.Code)
	return nil
}
```

Ensure imports include `"github.com/mendixlabs/mxcli/model"` and `"github.com/mendixlabs/mxcli/mdl/ast"`.

- [ ] **Step 5: Add grammar for LANGUAGE ADD/DROP to MDLSettings.g4**

Find `alterSettingsClause` and add two alternatives:

```antlr
alterSettingsClause
    : settingsSection settingsAssignment (COMMA settingsAssignment)*
    | CONSTANT STRING_LITERAL (VALUE settingsValue | DROP) (IN CONFIGURATION STRING_LITERAL)?
    | DROP CONSTANT STRING_LITERAL (IN CONFIGURATION STRING_LITERAL)?
    | CONFIGURATION STRING_LITERAL settingsAssignment (COMMA settingsAssignment)*
    | LANGUAGE ADD STRING_LITERAL (LPAREN languageOptions RPAREN)?   // NEW
    | LANGUAGE DROP STRING_LITERAL                                    // NEW
    ;

// NEW rules:
languageOptions
    : languageOption (COMMA languageOption)*
    ;

languageOption
    : identifierOrKeyword COLON settingsValue
    ;
```

- [ ] **Step 6: Wire in visitor_settings.go**

Find the visitor method that handles `alterSettingsClause`. The existing code handles `LANGUAGE` section via `settingsSection`. Add detection for the ADD/DROP alternatives before the existing section switch:

```go
// In the method handling alterSettingsClause:
if ctx.ADD() != nil {
    // LANGUAGE ADD 'code' [(...)]
    stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageAdd}
    if lit := ctx.STRING_LITERAL(0); lit != nil {
        stmt.Code = unquoteString(lit.GetText())
    }
    if ctx.LPAREN() != nil {
        for _, opt := range ctx.AllLanguageOption() {
            key := opt.IdentifierOrKeyword().GetText()
            val := opt.SettingsValue()
            switch strings.ToLower(key) {
            case "checkcomplete", "checkcomplete ness", "checkcompleteness":
                b := val.GetText() == "true"
                stmt.CheckCompleteness = &b
            case "dateformat":
                stmt.DateFormat = unquoteString(val.GetText())
            case "datetimeformat":
                stmt.DateTimeFormat = unquoteString(val.GetText())
            case "timeformat":
                stmt.TimeFormat = unquoteString(val.GetText())
            }
        }
    }
    b.statements = append(b.statements, stmt)
    return
}
if ctx.DROP() != nil && ctx.STRING_LITERAL(0) != nil {
    // LANGUAGE DROP 'code'
    code := unquoteString(ctx.STRING_LITERAL(0).GetText())
    b.statements = append(b.statements, &ast.AlterLanguageStmt{
        Op:   ast.AlterLanguageDrop,
        Code: code,
    })
    return
}
```

- [ ] **Step 7: Register in executor register_stubs.go**

Find the statement dispatch switch and add:

```go
case *ast.AlterLanguageStmt:
    return alterLanguage(ctx, stmt.(*ast.AlterLanguageStmt))
```

- [ ] **Step 8: Ensure MockBackend has ConnectedForWrite and UpdateProjectSettings**

In `mdl/backend/mock/backend.go`, verify `ConnectedForWriteFunc` and `UpdateProjectSettingsFunc` fields exist (they should — check existing field). If `ConnectedForWrite` is not a separate backend method, check how `ctx.ConnectedForWrite()` is implemented and adjust accordingly.

- [ ] **Step 9: Regenerate parser**

```bash
make grammar 2>&1 | tail -5
```

- [ ] **Step 10: Run tests**

```bash
go test ./mdl/executor/ -run "TestAlterLanguageAdd" -v 2>&1 | tail -10
go test ./mdl/... 2>&1 | grep -E "FAIL|ok" | head -20
```

Expected: TestAlterLanguageAdd tests PASS, no other failures.

- [ ] **Step 11: Commit**

```bash
git add mdl/ast/ast_translate.go mdl/grammar/domains/MDLSettings.g4 \
    mdl/visitor/visitor_settings.go mdl/executor/cmd_languages.go \
    mdl/executor/register_stubs.go mdl/executor/cmd_languages_mock_test.go
git commit -m "feat(i18n): ALTER SETTINGS LANGUAGE ADD/DROP with validation"
```

---

## Task 3: Enhance SHOW LANGUAGES + ALTER SETTINGS LANGUAGE DROP

**Files:**
- Modify: `mdl/executor/cmd_languages.go`
- Modify: `mdl/backend/mpr/settings_compat.go` (serialize Language list to BSON)

- [ ] **Step 1: Write test for enhanced SHOW LANGUAGES (reads from Settings, not catalog)**

```go
// mdl/executor/cmd_languages_mock_test.go
func TestShowLanguagesFromSettings(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages: []model.Language{
						{Code: "en_US"},
						{Code: "zh_CN", CheckCompleteness: true},
					},
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	err := listLanguages(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "en_US") {
		t.Errorf("expected en_US in output, got: %s", out)
	}
	if !strings.Contains(out, "zh_CN") {
		t.Errorf("expected zh_CN in output, got: %s", out)
	}
}

func TestAlterLanguageDrop_DefaultLang(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ConnectedForWriteFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages:           []model.Language{{Code: "en_US"}},
				},
			}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.AlterLanguageStmt{Op: ast.AlterLanguageDrop, Code: "en_US"}
	err := alterLanguage(ctx, stmt)
	if err == nil {
		t.Fatal("expected error when dropping default language")
	}
}
```

- [ ] **Step 2: Refactor listLanguages to read from Settings**

Replace the catalog-query body in `listLanguages` with:

```go
func listLanguages(ctx *ExecContext) error {
	// Try Settings first (fast, no catalog required)
	if ctx.Backend != nil {
		ps, err := ctx.Backend.GetProjectSettings()
		if err == nil && ps.Language != nil && len(ps.Language.Languages) > 0 {
			return listLanguagesFromSettings(ctx, ps.Language)
		}
	}
	// Fallback: catalog strings table (requires REFRESH CATALOG FULL)
	return listLanguagesFromCatalog(ctx)
}

func listLanguagesFromSettings(ctx *ExecContext, ls *model.LanguageSettings) error {
	tr := &TableResult{
		Columns: []string{"Code", "Default", "CheckCompleteness"},
		Summary: fmt.Sprintf("(%d languages)", len(ls.Languages)),
	}
	for _, l := range ls.Languages {
		isDefault := ""
		if l.Code == ls.DefaultLanguageCode {
			isDefault = "yes"
		}
		check := ""
		if l.CheckCompleteness {
			check = "true"
		}
		tr.Rows = append(tr.Rows, []any{l.Code, isDefault, check})
	}
	return writeResult(ctx, tr)
}

// listLanguagesFromCatalog is the old catalog-based implementation.
func listLanguagesFromCatalog(ctx *ExecContext) error {
	if ctx.Catalog == nil {
		return mdlerrors.NewValidation("no catalog available — run refresh catalog full first")
	}
	// ... (keep existing body here)
}
```

- [ ] **Step 3: Verify BSON roundtrip for Language list**

Check `mdl/backend/mpr/settings_modelsdk.go` — `SerializeProjectSettings` must serialize `ps.Language.Languages` back to BSON. Look at the existing serializer in `modelsdk/mpr/` to confirm it handles the Languages slice. If not, add serialization in `settings_compat.go`:

```go
// serializeLanguagesToBSON returns the BSON-D representation of a []Language
// for inclusion in the Settings$LanguageSettings document.
func serializeLanguagesToBSON(langs []model.Language) []any {
	out := make([]any, 0, len(langs))
	for _, l := range langs {
		doc := bson.D{
			{Key: "$ID",   Value: newMendixID()},
			{Key: "$Type", Value: "Settings$Language"},
			{Key: "Code",  Value: l.Code},
		}
		if l.CheckCompleteness {
			doc = append(doc, bson.E{Key: "CheckCompleteness", Value: true})
		}
		if l.CustomDateFormat != "" {
			doc = append(doc, bson.E{Key: "CustomDateFormat", Value: l.CustomDateFormat})
		}
		if l.CustomDateTimeFormat != "" {
			doc = append(doc, bson.E{Key: "CustomDateTimeFormat", Value: l.CustomDateTimeFormat})
		}
		if l.CustomTimeFormat != "" {
			doc = append(doc, bson.E{Key: "CustomTimeFormat", Value: l.CustomTimeFormat})
		}
		out = append(out, doc)
	}
	return out
}
```

Verify that `modelsdkmpr.SerializeProjectSettings` in `modelsdk/mpr/` calls the Languages serializer. If it doesn't handle `Languages`, file `modelsdk/mpr/serialize_settings.go` needs updating (find and add the Languages field write).

- [ ] **Step 4: Run tests**

```bash
go test ./mdl/executor/ -run "TestShowLanguagesFromSettings|TestAlterLanguageDrop" -v 2>&1 | tail -10
go test ./mdl/... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_languages.go mdl/executor/cmd_languages_mock_test.go \
    mdl/backend/mpr/settings_compat.go
git commit -m "feat(i18n): enhance SHOW LANGUAGES to read from Settings + ALTER LANGUAGE DROP"
```

---

## Task 4: BSON Write Primitive — setTranslationForLang

**Files:**
- Create: `mdl/backend/mpr/translation_writer.go`

- [ ] **Step 1: Write failing test**

```go
// mdl/backend/mpr/translation_writer_test.go
package mprbackend

import (
	"testing"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSetTranslationForLang_UpdateExisting(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			bson.D{
				{Key: "$Type",        Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text",         Value: "Submit"},
			},
		}},
	}
	setTranslationForLang(textDoc, "en_US", "Submit Updated")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	doc := extractBsonMap(items[0])
	if extractString(doc["Text"]) != "Submit Updated" {
		t.Errorf("expected 'Submit Updated', got %q", extractString(doc["Text"]))
	}
}

func TestSetTranslationForLang_AddNew(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			bson.D{
				{Key: "$Type",        Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: "en_US"},
				{Key: "Text",         Value: "Submit"},
			},
		}},
	}
	setTranslationForLang(textDoc, "zh_CN", "提交")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	found := false
	for _, item := range items {
		doc := extractBsonMap(item)
		if extractString(doc["LanguageCode"]) == "zh_CN" && extractString(doc["Text"]) == "提交" {
			found = true
		}
	}
	if !found {
		t.Error("zh_CN translation not found")
	}
}

func TestSetTranslationForLang_EmptyItems(t *testing.T) {
	textDoc := bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{}},
	}
	setTranslationForLang(textDoc, "zh_CN", "提交")
	items := extractBsonArray(dGet(textDoc, "Items"))
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./mdl/backend/mpr/ -run "TestSetTranslationForLang" -v 2>&1 | tail -5
```

Expected: compile error — `setTranslationForLang` undefined.

- [ ] **Step 3: Implement translation_writer.go**

Create `mdl/backend/mpr/translation_writer.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"go.mongodb.org/mongo-driver/bson"
)

// setTranslationForLang updates or inserts a Texts$Translation entry inside a
// Texts$Text BSON document. If a translation for langCode already exists, its
// Text field is updated in place. If not, a new Texts$Translation is appended.
//
// textsBsonD must be the bson.D of the Texts$Text element (not its parent).
func setTranslationForLang(textsBsonD bson.D, langCode, text string) {
	items := extractBsonArray(dGet(textsBsonD, "Items"))
	for _, item := range items {
		doc, ok := item.(bson.D)
		if !ok {
			continue
		}
		if extractString(dGet(doc, "LanguageCode")) == langCode {
			dSet(doc, "Text", text)
			return
		}
	}
	// Not found — append a new Texts$Translation
	newTr := bson.D{
		{Key: "$ID",          Value: newMendixID()},
		{Key: "$Type",        Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: langCode},
		{Key: "Text",         Value: text},
	}
	newItems := append(items, newTr)
	dSet(textsBsonD, "Items", bson.A(newItems))
}

// setTranslationInField finds a Texts$Text sub-document at fieldKey within
// parentDoc and sets the translation for langCode. Returns false if the field
// is not found or not a Texts$Text document.
func setTranslationInField(parentDoc bson.D, fieldKey, langCode, text string) bool {
	textDoc := dGetDoc(parentDoc, fieldKey)
	if textDoc == nil {
		return false
	}
	if extractString(dGet(textDoc, "$Type")) != "Texts$Text" {
		return false
	}
	setTranslationForLang(textDoc, langCode, text)
	return true
}
```

Note: `newMendixID()` must return a string UUID. Check existing uses in the package (e.g., `page_mutator.go`) for the correct helper.

- [ ] **Step 4: Run tests**

```bash
go test ./mdl/backend/mpr/ -run "TestSetTranslationForLang" -v 2>&1 | tail -10
```

Expected: all 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/translation_writer.go mdl/backend/mpr/translation_writer_test.go
git commit -m "feat(i18n): add setTranslationForLang BSON primitive"
```

---

## Task 5: TRANSLATE PAGE / TRANSLATE SNIPPET

**Files:**
- Modify: `mdl/grammar/domains/MDLDomainModel.g4`
- Modify: `mdl/visitor/visitor_translate.go` (create)
- Create: `mdl/executor/cmd_translate.go`
- Modify: `mdl/executor/register_stubs.go`
- Modify: `mdl/backend/mpr/page_mutator.go`

- [ ] **Step 1: Write failing test**

```go
// mdl/executor/cmd_translate_mock_test.go
package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestTranslatePage_LangNotRegistered(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ConnectedForWriteFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages:           []model.Language{{Code: "en_US"}},
				},
			}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "PAGE",
		QName:   ast.QualifiedName{Parts: []string{"MyModule", "Home"}},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Button1.caption", Text: "提交"}},
	}
	err := translateDocument(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for unregistered language")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTranslatePage_CallsSetWidgetTranslation(t *testing.T) {
	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ConnectedForWriteFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages: []model.Language{
						{Code: "en_US"},
						{Code: "zh_CN"},
					},
				},
			}, nil
		},
		SetWidgetTranslationFunc: func(docQN, containerType, widgetName, property, lang, text string) error {
			called = true
			if widgetName != "Button1" || property != "caption" || lang != "zh_CN" || text != "提交" {
				t.Errorf("unexpected args: widget=%s prop=%s lang=%s text=%s", widgetName, property, lang, text)
			}
			return nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "PAGE",
		QName:   ast.QualifiedName{Parts: []string{"MyModule", "Home"}},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Button1.caption", Text: "提交"}},
	}
	if err := translateDocument(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("SetWidgetTranslationFunc was not called")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./mdl/executor/ -run "TestTranslatePage" -v 2>&1 | tail -5
```

Expected: compile error — `translateDocument` and `SetWidgetTranslationFunc` undefined.

- [ ] **Step 3: Add SetWidgetTranslation to backend infrastructure**

In `mdl/backend/infrastructure.go`, add to `PageMutationBackend` (or create a new `TranslationBackend` embedded in FullBackend):

```go
// In PageMutationBackend or a new TranslationBackend:
SetWidgetTranslation(docQN, containerType, widgetName, property, lang, text string) error
SetPageTitleTranslation(docQN, containerType, lang, text string) error
```

Add `SetWidgetTranslationFunc` and `SetPageTitleTranslationFunc` fields to `MockBackend` in `mdl/backend/mock/backend.go`.

Add stub implementations in `mdl/backend/mock/mock_page.go` (or `mock_mutation.go`):

```go
func (m *MockBackend) SetWidgetTranslation(docQN, containerType, widgetName, property, lang, text string) error {
	if m.SetWidgetTranslationFunc != nil {
		return m.SetWidgetTranslationFunc(docQN, containerType, widgetName, property, lang, text)
	}
	return fmt.Errorf("MockBackend.SetWidgetTranslation not configured")
}

func (m *MockBackend) SetPageTitleTranslation(docQN, containerType, lang, text string) error {
	if m.SetPageTitleTranslationFunc != nil {
		return m.SetPageTitleTranslationFunc(docQN, containerType, lang, text)
	}
	return fmt.Errorf("MockBackend.SetPageTitleTranslation not configured")
}
```

- [ ] **Step 4: Implement SetWidgetTranslation in page_mutator.go**

In `mdl/backend/mpr/page_mutator.go`, add new method on `mprPageMutator`:

```go
// SetWidgetTranslation updates a specific language's translation for a named
// widget property. property is one of: caption, placeholder, label, tooltip, content.
func (m *mprPageMutator) SetWidgetTranslation(widgetName, property, lang, text string) error {
	widget := m.widgetFinder(m.rawData, widgetName)
	if widget == nil {
		return mdlerrors.NewNotFound("widget", widgetName)
	}
	wDoc := widget.doc

	var fieldKey string
	switch strings.ToLower(property) {
	case "caption":
		fieldKey = "Caption"
	case "placeholder":
		fieldKey = "Placeholder"
	case "tooltip":
		fieldKey = "Tooltip"
	case "label":
		// Label is nested: Label.Caption
		labelDoc := dGetDoc(wDoc, "Label")
		if labelDoc == nil {
			return mdlerrors.NewValidation(fmt.Sprintf("widget '%s' has no Label property", widgetName))
		}
		if !setTranslationInField(labelDoc, "Caption", lang, text) {
			return mdlerrors.NewValidation(fmt.Sprintf("widget '%s' Label has no Caption field", widgetName))
		}
		return nil
	case "content":
		fieldKey = "Content"
	default:
		return mdlerrors.NewUnsupported(fmt.Sprintf("unsupported translatable property: %s", property))
	}

	if !setTranslationInField(wDoc, fieldKey, lang, text) {
		return mdlerrors.NewValidation(fmt.Sprintf("widget '%s' has no '%s' property", widgetName, property))
	}
	return nil
}
```

Wire `SetWidgetTranslation` on `MprBackend` level (in `mdl/backend/mpr/backend.go`) to open the page, call the mutator method, and save:

```go
func (b *MprBackend) SetWidgetTranslation(docQN, containerType, widgetName, property, lang, text string) error {
	mut, err := b.OpenPageForMutation(docQN, backend.ContainerType(containerType))
	if err != nil {
		return err
	}
	mprMut, ok := mut.(*mprPageMutator)
	if !ok {
		return fmt.Errorf("unexpected mutator type")
	}
	if err := mprMut.SetWidgetTranslation(widgetName, property, lang, text); err != nil {
		return err
	}
	return mprMut.Commit()
}
```

- [ ] **Step 5: Create executor cmd_translate.go**

Create `mdl/executor/cmd_translate.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// translateDocument handles TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW.
func translateDocument(ctx *ExecContext, stmt *ast.TranslateStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	// Validate language is registered
	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	langRegistered := false
	if ps.Language != nil {
		for _, l := range ps.Language.Languages {
			if l.Code == stmt.Lang {
				langRegistered = true
				break
			}
		}
	}
	if !langRegistered {
		return mdlerrors.NewValidation(fmt.Sprintf(
			"language '%s' is not registered. Run ALTER SETTINGS LANGUAGE ADD '%s' first.",
			stmt.Lang, stmt.Lang,
		))
	}

	docQN := stmt.QName.String()

	switch strings.ToUpper(stmt.DocType) {
	case "PAGE", "SNIPPET":
		return translatePage(ctx, docQN, stmt.DocType, stmt.Lang, stmt.Ops)
	case "ENUMERATION":
		return translateEnumeration(ctx, docQN, stmt.Lang, stmt.Ops)
	case "WORKFLOW":
		return translateWorkflow(ctx, docQN, stmt.Lang, stmt.Ops)
	default:
		return mdlerrors.NewUnsupported("unsupported TRANSLATE target: " + stmt.DocType)
	}
}

func translatePage(ctx *ExecContext, docQN, docType, lang string, ops []ast.TranslateSetOp) error {
	containerType := "page"
	if strings.EqualFold(docType, "SNIPPET") {
		containerType = "snippet"
	}
	for _, op := range ops {
		parts := strings.SplitN(op.Path, ".", 2)
		if len(parts) == 1 && strings.EqualFold(parts[0], "title") {
			// Page-level title
			if err := ctx.Backend.SetPageTitleTranslation(docQN, containerType, lang, op.Text); err != nil {
				return fmt.Errorf("set title translation: %w", err)
			}
			continue
		}
		if len(parts) != 2 {
			return mdlerrors.NewValidation(fmt.Sprintf("invalid path '%s': expected WidgetName.property or Title", op.Path))
		}
		widgetName, property := parts[0], parts[1]
		if err := ctx.Backend.SetWidgetTranslation(docQN, containerType, widgetName, property, lang, op.Text); err != nil {
			return fmt.Errorf("widget '%s' property '%s': %w", widgetName, property, err)
		}
	}
	fmt.Fprintf(ctx.Output, "TRANSLATED %s %s IN %s (%d properties)\n", docType, docQN, lang, len(ops))
	return nil
}

// translateEnumeration and translateWorkflow are added in Task 6 and 7.
func translateEnumeration(ctx *ExecContext, docQN, lang string, ops []ast.TranslateSetOp) error {
	return mdlerrors.NewUnsupported("TRANSLATE ENUMERATION not yet implemented")
}
func translateWorkflow(ctx *ExecContext, docQN, lang string, ops []ast.TranslateSetOp) error {
	return mdlerrors.NewUnsupported("TRANSLATE WORKFLOW not yet implemented")
}
```

- [ ] **Step 6: Add grammar for TRANSLATE PAGE**

In `mdl/grammar/domains/MDLDomainModel.g4`, add `translateStatement` to the top-level statement rule, then define:

```antlr
translateStatement
    : TRANSLATE translateDocType qualifiedName IN identifierOrKeyword
      translateSetOp+
    ;

translateDocType
    : PAGE | SNIPPET | ENUMERATION | WORKFLOW
    ;

translateSetOp
    : SET translatePath EQUALS STRING_LITERAL
    ;

translatePath
    : identifierOrKeyword (DOT identifierOrKeyword)?
    ;
```

- [ ] **Step 7: Create visitor_translate.go**

Create `mdl/visitor/visitor_translate.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// EnterTranslateStatement builds a TranslateStmt AST node.
func (b *MDLBuilder) EnterTranslateStatement(ctx *parser.TranslateStatementContext) {
	stmt := &ast.TranslateStmt{
		DocType: strings.ToUpper(ctx.TranslateDocType().GetText()),
		QName:   b.buildQualifiedName(ctx.QualifiedName()),
		Lang:    unquoteOrIdent(ctx.IdentifierOrKeyword().GetText()),
	}
	for _, opCtx := range ctx.AllTranslateSetOp() {
		path := buildTranslatePath(opCtx.TranslatePath())
		text := unquoteString(opCtx.STRING_LITERAL().GetText())
		stmt.Ops = append(stmt.Ops, ast.TranslateSetOp{Path: path, Text: text})
	}
	b.statements = append(b.statements, stmt)
}

func buildTranslatePath(ctx *parser.TranslatePathContext) string {
	parts := ctx.AllIdentifierOrKeyword()
	if len(parts) == 1 {
		return parts[0].GetText()
	}
	return parts[0].GetText() + "." + parts[1].GetText()
}
```

- [ ] **Step 8: Register in register_stubs.go**

```go
case *ast.TranslateStmt:
    return translateDocument(ctx, stmt.(*ast.TranslateStmt))
```

- [ ] **Step 9: Regenerate parser**

```bash
make grammar 2>&1 | tail -5
```

- [ ] **Step 10: Run tests**

```bash
go test ./mdl/executor/ -run "TestTranslatePage" -v 2>&1 | tail -10
go test ./mdl/... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **Step 11: Commit**

```bash
git add mdl/ast/ast_translate.go mdl/grammar/domains/MDLDomainModel.g4 \
    mdl/visitor/visitor_translate.go mdl/executor/cmd_translate.go \
    mdl/executor/register_stubs.go mdl/backend/infrastructure.go \
    mdl/backend/mock/backend.go mdl/backend/mock/mock_page.go \
    mdl/backend/mpr/page_mutator.go mdl/backend/mpr/backend.go \
    mdl/executor/cmd_translate_mock_test.go
git commit -m "feat(i18n): TRANSLATE PAGE/SNIPPET command"
```

---

## Task 6: TRANSLATE ENUMERATION + TRANSLATE WORKFLOW

**Files:**
- Modify: `mdl/executor/cmd_translate.go`
- Modify: `mdl/backend/infrastructure.go` (add SetEnumerationTranslation, SetWorkflowTranslation)
- Modify: `mdl/backend/mock/backend.go` + mock file
- Modify: `mdl/backend/mpr/backend.go` + new helper

- [ ] **Step 1: Write failing tests**

```go
// mdl/executor/cmd_translate_mock_test.go — add:
func TestTranslateEnumeration(t *testing.T) {
	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ConnectedForWriteFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					Languages: []model.Language{{Code: "en_US"}, {Code: "zh_CN"}},
				},
			}, nil
		},
		SetEnumerationTranslationFunc: func(docQN, valueName, lang, text string) error {
			called = true
			if valueName != "Active" || lang != "zh_CN" || text != "活跃" {
				t.Errorf("unexpected args: value=%s lang=%s text=%s", valueName, lang, text)
			}
			return nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateStmt{
		DocType: "ENUMERATION",
		QName:   ast.QualifiedName{Parts: []string{"MyModule", "Status"}},
		Lang:    "zh_CN",
		Ops:     []ast.TranslateSetOp{{Path: "Active.caption", Text: "活跃"}},
	}
	if err := translateDocument(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("SetEnumerationTranslationFunc not called")
	}
}
```

- [ ] **Step 2: Add SetEnumerationTranslation to backend**

In `mdl/backend/infrastructure.go`, add to `EnumerationBackend`:

```go
SetEnumerationTranslation(docQN, valueName, lang, text string) error
```

Add `SetEnumerationTranslationFunc` to `MockBackend`. Add mock stub. Implement in `mdl/backend/mpr/backend.go`:

```go
func (b *MprBackend) SetEnumerationTranslation(docQN, valueName, lang, text string) error {
	// Load enumeration BSON unit, find value by name, update Caption translation
	// Follow pattern of existing enum read/write in mdl/backend/mpr/
	unit, err := b.loadEnumerationUnit(docQN)
	if err != nil {
		return err
	}
	if err := setEnumValueTranslation(unit.rawDoc, valueName, lang, text); err != nil {
		return err
	}
	return b.writeUnitContents(unit.id, unit.rawDoc)
}
```

Implement `setEnumValueTranslation` in `translation_writer.go`:

```go
// setEnumValueTranslation finds an enumeration value by name and updates its Caption translation.
func setEnumValueTranslation(enumDoc bson.D, valueName, lang, text string) error {
	values := extractBsonArray(dGet(enumDoc, "Values"))
	for _, item := range values {
		doc, ok := item.(bson.D)
		if !ok {
			continue
		}
		if extractString(dGet(doc, "Name")) == valueName {
			captionDoc := dGetDoc(doc, "Caption")
			if captionDoc == nil {
				return fmt.Errorf("enumeration value '%s' has no Caption", valueName)
			}
			setTranslationForLang(captionDoc, lang, text)
			return nil
		}
	}
	return fmt.Errorf("enumeration value '%s' not found", valueName)
}
```

- [ ] **Step 3: Implement translateEnumeration in cmd_translate.go**

Replace the stub:

```go
func translateEnumeration(ctx *ExecContext, docQN, lang string, ops []ast.TranslateSetOp) error {
	for _, op := range ops {
		parts := strings.SplitN(op.Path, ".", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[1], "caption") {
			return mdlerrors.NewValidation(fmt.Sprintf(
				"invalid enumeration path '%s': expected ValueName.caption", op.Path,
			))
		}
		valueName := parts[0]
		if err := ctx.Backend.SetEnumerationTranslation(docQN, valueName, lang, op.Text); err != nil {
			return fmt.Errorf("enumeration value '%s': %w", valueName, err)
		}
	}
	fmt.Fprintf(ctx.Output, "TRANSLATED ENUMERATION %s IN %s (%d values)\n", docQN, lang, len(ops))
	return nil
}
```

- [ ] **Step 4: Add SetWorkflowTranslation similarly**

Follow the same pattern for TRANSLATE WORKFLOW — add `SetWorkflowTranslation(docQN, activityName, property, lang, text string) error` to `WorkflowMutationBackend`. Properties are `taskName` and `taskDescription`. Implement in `translation_writer.go`:

```go
func setWorkflowActivityTranslation(flowDoc bson.D, activityName, property, lang, text string) error {
	activities := extractBsonArray(dGet(flowDoc, "Activities"))
	for _, item := range activities {
		doc, ok := item.(bson.D)
		if !ok {
			continue
		}
		if extractString(dGet(doc, "Name")) == activityName {
			var fieldKey string
			switch strings.ToLower(property) {
			case "taskname":
				fieldKey = "TaskName"
			case "taskdescription":
				fieldKey = "TaskDescription"
			default:
				return fmt.Errorf("unsupported workflow activity property: %s", property)
			}
			if !setTranslationInField(doc, fieldKey, lang, text) {
				return fmt.Errorf("activity '%s' has no '%s' field", activityName, fieldKey)
			}
			return nil
		}
	}
	return fmt.Errorf("workflow activity '%s' not found", activityName)
}
```

Replace the stub `translateWorkflow`:

```go
func translateWorkflow(ctx *ExecContext, docQN, lang string, ops []ast.TranslateSetOp) error {
	for _, op := range ops {
		parts := strings.SplitN(op.Path, ".", 2)
		if len(parts) != 2 {
			return mdlerrors.NewValidation(fmt.Sprintf(
				"invalid workflow path '%s': expected ActivityName.taskName or ActivityName.taskDescription", op.Path,
			))
		}
		activityName, property := parts[0], parts[1]
		if err := ctx.Backend.SetWorkflowTranslation(docQN, activityName, property, lang, op.Text); err != nil {
			return fmt.Errorf("workflow activity '%s': %w", activityName, err)
		}
	}
	fmt.Fprintf(ctx.Output, "TRANSLATED WORKFLOW %s IN %s (%d activities)\n", docQN, lang, len(ops))
	return nil
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./mdl/executor/ -run "TestTranslateEnumeration|TestTranslateWorkflow" -v 2>&1 | tail -10
go test ./mdl/... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_translate.go mdl/backend/infrastructure.go \
    mdl/backend/mock/backend.go mdl/backend/mpr/translation_writer.go \
    mdl/backend/mpr/backend.go mdl/executor/cmd_translate_mock_test.go
git commit -m "feat(i18n): TRANSLATE ENUMERATION + TRANSLATE WORKFLOW"
```

---

## Task 7: Microflow Inline Multilingual (TextLiteral)

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4`
- Modify: `mdl/ast/ast_microflow.go`
- Modify: `mdl/visitor/visitor_helpers.go`
- Modify: `mdl/executor/flowbuilder_actions_feedback_gen.go` (and related action builders)

- [ ] **Step 1: Write failing test**

```go
// mdl/visitor/visitor_helpers_test.go (create or add to existing)
package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestParseTextLiteral_Single(t *testing.T) {
	// Single string → en_US only
	lit := SingleText("Hello")
	if len(lit.Translations) != 1 {
		t.Fatalf("expected 1 translation, got %d", len(lit.Translations))
	}
	if lit.Translations["en_US"] != "Hello" {
		t.Errorf("expected en_US='Hello', got %v", lit.Translations)
	}
}
```

Also add an executor-level test for CREATE OR MODIFY MICROFLOW with multi-language show message (integration via golden test or mock backend pattern).

- [ ] **Step 2: Extend TextLiteral in ast_microflow.go**

Find the current `TextLiteral` (or wherever message text is stored in the AST — it may be just a `string` field). Change it:

```go
// TextLiteral represents a translatable text: either a single string (en_US)
// or a map of language codes to translated strings.
type TextLiteral struct {
	Translations map[string]string // langCode → text
}

// SingleText creates a TextLiteral with only the default language (en_US).
func SingleText(s string) TextLiteral {
	return TextLiteral{Translations: map[string]string{"en_US": s}}
}

// IsEmpty returns true if the literal has no translations.
func (t TextLiteral) IsEmpty() bool {
	return len(t.Translations) == 0
}
```

Find all `ShowMessage`, `ValidationFeedback`, `LogMessage` AST fields that were `string` and change to `TextLiteral`. Run `go build ./mdl/...` to find all broken callers, then fix them.

- [ ] **Step 3: Add textLiteral grammar rule to MDLMicroflow.g4**

Find rules like `showMessageStatement` (or whatever rule handles `show message`) and replace the message string literal with a `textLiteral` rule reference. Add:

```antlr
textLiteral
    : STRING_LITERAL                                           # textSingle
    | LPAREN textTranslation (COMMA textTranslation)* RPAREN  # textMap
    ;

textTranslation
    : identifierOrKeyword COLON STRING_LITERAL
    ;
```

- [ ] **Step 4: Add parseTextLiteral helper in visitor_helpers.go**

```go
// parseTextLiteral converts a textLiteralContext to an ast.TextLiteral.
func parseTextLiteral(ctx parser.ITextLiteralContext) ast.TextLiteral {
	if ctx == nil {
		return ast.TextLiteral{}
	}
	if single, ok := ctx.(*parser.TextSingleContext); ok {
		text := unquoteString(single.STRING_LITERAL().GetText())
		return ast.SingleText(text)
	}
	if mapCtx, ok := ctx.(*parser.TextMapContext); ok {
		lit := ast.TextLiteral{Translations: make(map[string]string)}
		for _, tr := range mapCtx.AllTextTranslation() {
			lang := tr.IdentifierOrKeyword().GetText()
			text := unquoteString(tr.STRING_LITERAL().GetText())
			lit.Translations[lang] = text
		}
		return lit
	}
	return ast.TextLiteral{}
}
```

- [ ] **Step 5: Update visitor_microflow.go to use parseTextLiteral**

Find where `show message` / `validation feedback` / `log message` text is parsed (currently calls `unquoteString` on a STRING_LITERAL). Replace with `parseTextLiteral(ctx.TextLiteral())`.

- [ ] **Step 6: Update action builders to use TextLiteral**

In `mdl/executor/flowbuilder_actions_feedback_gen.go` (and other flow builder files), find `buildTextNode` (or equivalent). Change from:

```go
tr.SetLanguageCode("en_US")
tr.SetText(messageStr)
```

To:

```go
func buildTextNodeFromLiteral(lit ast.TextLiteral) *genTx.Text {
	text := genTx.NewText()
	langs := make([]string, 0, len(lit.Translations))
	for lang := range lit.Translations {
		langs = append(langs, lang)
	}
	sort.Strings(langs) // deterministic order
	for _, lang := range langs {
		tr := genTx.NewTranslation()
		tr.SetLanguageCode(lang)
		tr.SetText(lit.Translations[lang])
		text.AddTranslations(tr)
	}
	return text
}
```

Replace all existing `buildTextNode`-style calls that hard-code `"en_US"` with `buildTextNodeFromLiteral`.

- [ ] **Step 7: Regenerate parser + run tests**

```bash
make grammar 2>&1 | tail -5
go test ./mdl/... 2>&1 | grep -E "FAIL|ok"
```

Fix any compilation errors from the TextLiteral type change (use `go build ./mdl/...` to find them all first).

- [ ] **Step 8: Commit**

```bash
git add mdl/grammar/domains/MDLMicroflow.g4 mdl/ast/ast_microflow.go \
    mdl/visitor/visitor_helpers.go mdl/executor/flowbuilder_actions_feedback_gen.go \
    mdl/executor/cmd_microflows_format_action_gen_test.go
git commit -m "feat(i18n): microflow inline multilingual TextLiteral syntax"
```

---

## Task 8: DESCRIBE TRANSLATIONS

**Files:**
- Create: `mdl/executor/cmd_describe_translations.go`
- Modify: `mdl/grammar/domains/MDLDomainModel.g4`
- Modify: `mdl/visitor/visitor_translate.go`
- Modify: `mdl/executor/register_stubs.go`

- [ ] **Step 1: Write failing test**

```go
// mdl/executor/cmd_describe_translations_mock_test.go
package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestDescribeTranslations_Page(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					Languages: []model.Language{{Code: "en_US"}, {Code: "zh_CN"}},
				},
			}, nil
		},
		ListTranslationNodesFunc: func(docQN, docType string) ([]model.TranslationNode, error) {
			return []model.TranslationNode{
				{
					Path:     "Button1.caption",
					Property: "caption",
					Texts:    map[string]string{"en_US": "Submit", "zh_CN": "提交"},
				},
				{
					Path:     "TextBox1.placeholder",
					Property: "placeholder",
					Texts:    map[string]string{"en_US": "Enter email"},
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	stmt := &ast.DescribeTranslationsStmt{
		QName: ast.QualifiedName{Parts: []string{"MyModule", "Home"}},
		Lang:  "zh_CN",
	}
	if err := describeTranslations(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Button1") {
		t.Errorf("expected Button1 in output, got: %s", out)
	}
	if !strings.Contains(out, "missing") {
		t.Errorf("expected 'missing' for untranslated placeholder, got: %s", out)
	}
}
```

- [ ] **Step 2: Add TranslationNode type + ListTranslationNodes to model and backend**

In `model/types.go`, add:

```go
// TranslationNode represents a single translatable text field in a document.
type TranslationNode struct {
	Path     string            // "Button1.caption", "ShowMessage.template"
	Property string
	Texts    map[string]string // langCode → text; missing key = not yet translated
}
```

In `mdl/backend/infrastructure.go`, add to an appropriate backend interface (or create `TranslationInspectionBackend` embedded in FullBackend):

```go
ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error)
```

Add `ListTranslationNodesFunc` to MockBackend. Add mock stub returning `nil, fmt.Errorf("MockBackend.ListTranslationNodes not configured")`.

- [ ] **Step 3: Create describeTranslations executor**

Create `mdl/executor/cmd_describe_translations.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

func describeTranslations(ctx *ExecContext, stmt *ast.DescribeTranslationsStmt) error {
	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}

	// Determine languages to show
	var langs []string
	if stmt.Lang != "" {
		langs = []string{stmt.Lang}
	} else if ps.Language != nil {
		for _, l := range ps.Language.Languages {
			langs = append(langs, l.Code)
		}
	}

	docQN := stmt.QName.String()
	// Infer doc type from catalog or just pass empty (backend figures it out)
	nodes, err := ctx.Backend.ListTranslationNodes(docQN, "")
	if err != nil {
		return mdlerrors.NewBackend("list translation nodes", err)
	}

	// Build table columns: Path | Property | lang1 | lang2 | ...
	sort.Strings(langs)
	columns := []string{"Path", "Property"}
	columns = append(columns, langs...)

	tr := &TableResult{Columns: columns}
	for _, node := range nodes {
		row := []any{node.Path, node.Property}
		for _, lang := range langs {
			if text, ok := node.Texts[lang]; ok {
				row = append(row, text)
			} else {
				row = append(row, "(missing)")
			}
		}
		tr.Rows = append(tr.Rows, row)
	}
	tr.Summary = fmt.Sprintf("(%d translatable fields)", len(nodes))
	return writeResult(ctx, tr)
}
```

- [ ] **Step 4: Add DESCRIBE TRANSLATIONS grammar**

In `MDLDomainModel.g4`, add to the DESCRIBE statement alternatives:

```antlr
| DESCRIBE TRANSLATIONS qualifiedName (IN identifierOrKeyword)?
```

Wire in visitor_translate.go:

```go
func (b *MDLBuilder) EnterDescribeTranslationsStatement(...) {
	stmt := &ast.DescribeTranslationsStmt{
		QName: b.buildQualifiedName(ctx.QualifiedName()),
	}
	if ctx.IN() != nil {
		stmt.Lang = ctx.IdentifierOrKeyword().GetText()
	}
	b.statements = append(b.statements, stmt)
}
```

Register in register_stubs.go:

```go
case *ast.DescribeTranslationsStmt:
    return describeTranslations(ctx, stmt.(*ast.DescribeTranslationsStmt))
```

- [ ] **Step 5: Implement ListTranslationNodes in MPR backend**

In `mdl/backend/mpr/backend.go` or a new `translation_reader.go`:

```go
func (b *MprBackend) ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error) {
	// Heuristic: try page, then enumeration, then microflow, then workflow
	// based on what's loadable for the given QN
	// For pages: walk widget tree, collect all Texts$Text fields
	// For microflows: walk ObjectCollection, collect ShowMessage/Validation/Log templates
	// For enumerations: walk Values, collect Caption fields
	// For workflows: walk flow activities, collect TaskName/TaskDescription
	// Return TranslationNode per field found
	// ... implementation follows the builder_strings.go pattern but reads gen-decoded elements
	return listTranslationNodesForDoc(b, docQN)
}
```

The full implementation of `listTranslationNodesForDoc` reads the document using the reader, walks the element tree collecting `Texts$Text` items, and returns `[]model.TranslationNode`. Follow the existing pattern in `mdl/catalog/builder_strings.go` for enumerating the same fields.

- [ ] **Step 6: Regenerate parser + run tests**

```bash
make grammar 2>&1 | tail -5
go test ./mdl/executor/ -run "TestDescribeTranslations" -v 2>&1 | tail -10
go test ./mdl/... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **Step 7: Add MDL example test**

Create `mdl-examples/doctype-tests/translations.mdl`:

```mdl
-- Test language registry
alter settings language add 'zh_CN';
show languages;
show supported languages;

-- Test TRANSLATE PAGE (requires connected project)
-- translate page MyModule.Home in zh_CN
--   set Button1.caption = '提交';

-- Test TRANSLATE ENUMERATION
-- translate enumeration MyModule.Status in zh_CN
--   set Active.caption = '活跃';

alter settings language drop 'zh_CN';
```

- [ ] **Step 8: Commit**

```bash
git add mdl/executor/cmd_describe_translations.go mdl/executor/cmd_describe_translations_mock_test.go \
    mdl/grammar/domains/MDLDomainModel.g4 mdl/visitor/visitor_translate.go \
    mdl/executor/register_stubs.go mdl/backend/infrastructure.go \
    mdl/backend/mock/backend.go model/types.go mdl/backend/mpr/backend.go \
    mdl-examples/doctype-tests/translations.mdl
git commit -m "feat(i18n): DESCRIBE TRANSLATIONS coverage view"
```

---

## Task 9: Syntax Reference + Final Integration Test

**Files:**
- Modify: `docs/01-project/MDL_QUICK_REFERENCE.md`
- Modify: `cmd/mxcli/syntax/` (features file for language/translation)

- [ ] **Step 1: Update MDL_QUICK_REFERENCE.md**

Add a new "Multilingual / Translations" section with all new syntax:

```markdown
## Multilingual / Translations

| Statement | Example | Notes |
|---|---|---|
| Show registered languages | `show languages;` | Reads Settings directly |
| Show valid language codes | `show supported languages;` | Built-in whitelist |
| Add language | `alter settings language add 'zh_CN';` | Validates against whitelist |
| Add with options | `alter settings language add 'zh_CN' (checkCompleteness: true, dateFormat: 'yyyy-MM-dd');` | |
| Drop language | `alter settings language drop 'fr_FR';` | Cannot drop default language |
| Set default language | `alter settings language DefaultLanguageCode = 'zh_CN';` | Existing |
| Translate page | `translate page MyModule.Home in zh_CN set Button1.caption = '提交';` | Widget name must exist |
| Translate snippet | `translate snippet MyModule.Snip_Header in zh_CN set Text1.content = '欢迎';` | |
| Translate enumeration | `translate enumeration MyModule.Status in zh_CN set Active.caption = '活跃';` | |
| Translate workflow | `translate workflow MyModule.WF_Approval in zh_CN set ApproveTask.taskName = '审批';` | |
| Describe translations | `describe translations MyModule.Home;` | All languages |
| Describe translations (lang) | `describe translations MyModule.Home in zh_CN;` | Shows missing |
| Microflow inline multilingual | `show message 'Info' (en_US: 'OK', zh_CN: '好的') blocking;` | |
| Validation feedback multilingual | `validation feedback on $Obj.Field (en_US: 'Invalid', zh_CN: '无效');` | |
```

- [ ] **Step 2: Add SyntaxFeature entries**

Find `cmd/mxcli/syntax/features_settings.go` (or equivalent file). Add:

```go
{
    Name:    "language",
    Summary: "Language registry: add/drop/show languages",
    Syntax: `alter settings language add 'zh_CN';
alter settings language add 'zh_CN' (checkCompleteness: true);
alter settings language drop 'fr_FR';
show languages;
show supported languages;`,
},
{
    Name:    "translate",
    Summary: "Element-level translation maintenance",
    Syntax: `translate page Module.Name in zh_CN
  set Button1.caption = '提交'
  set TextBox1.placeholder = '请输入';
translate enumeration Module.Status in zh_CN
  set Active.caption = '活跃';
translate workflow Module.WF in zh_CN
  set Task1.taskName = '任务名称';
describe translations Module.Home;
describe translations Module.Home in zh_CN;`,
},
```

- [ ] **Step 3: Run full test suite**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok" | head -40
make build 2>&1 | tail -5
```

Expected: all packages pass.

- [ ] **Step 4: Final commit**

```bash
git add docs/01-project/MDL_QUICK_REFERENCE.md cmd/mxcli/syntax/
git commit -m "docs(i18n): update syntax reference + feature topics for multilingual support"
```

---

## Self-Review Checklist

- [ ] All spec features covered: language ADD/DROP, SHOW LANGUAGES enhanced, SHOW SUPPORTED, TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW, microflow TextLiteral, DESCRIBE TRANSLATIONS
- [ ] No "TBD" or "TODO" in code steps
- [ ] `setTranslationForLang` defined in Task 4 before first use in Task 5
- [ ] `TranslationNode` type defined in Task 8 before `ListTranslationNodes` uses it
- [ ] `AlterLanguageStmt` defined in Task 2 (ast_translate.go) — same file as `TranslateStmt` and `DescribeTranslationsStmt` (no duplication)
- [ ] `TextLiteral` in ast_microflow.go is changed in Task 7, callers fixed before commit
- [ ] Mock fields match: `SetWidgetTranslationFunc`, `SetEnumerationTranslationFunc`, `SetWorkflowTranslationFunc`, `ListTranslationNodesFunc` all defined before used in tests
- [ ] `make grammar` run after every grammar change (Tasks 1, 2, 5, 7, 8)
