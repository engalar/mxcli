# Multilingual MDL Support — Design Spec

**Date:** 2026-05-23
**Revised:** 2026-06-02
**Status:** Approved
**Scope:** Language registry management + multilingual text maintenance via MDL + AI agent translation pipeline

---

## 1. Problem Statement

Mendix projects support multiple languages through `Settings$LanguageSettings` (the language
registry) and `Texts$Text` / `Texts$Translation` BSON nodes embedded in every translatable
element. MDL currently has no way to:

1. Add or remove languages from the project language registry.
2. Read or update translated text for a specific language on a specific element.
3. Know which Mendix language codes are valid.
4. Drive an AI agent translation pipeline: discover missing strings → translate → apply in one
   automated workflow.

The goal is a complete MDL workflow: register a new language, inspect what needs translating,
fill in translations element by element — and optionally hand the entire workflow to an AI agent.

---

## 2. BSON Structure Reference

Every translatable string in Mendix is stored as a `Texts$Text` element with an array of
`Texts$Translation` children, each identified by a `LanguageCode`:

```
Texts$Text {
    $ID:   <ObjectId>           -- unique, stable
    Items: [
        Texts$Translation {
            $ID:          <ObjectId>
            LanguageCode: "en_US"
            Text:         "Submit"
        },
        Texts$Translation {
            $ID:          <ObjectId>
            LanguageCode: "zh_CN"
            Text:         "提交"
        }
    ]
}
```

`Texts$Text` nodes appear as fields in pages, microflows, enumerations, workflows, and
other Mendix document types. Each node has its own `$ID`, making it globally addressable.

### Addressability by element type

| Element type | Child addressing | Viable path |
|---|---|---|
| Page / Snippet | Widget `Name` field, Studio Pro-enforced unique | `WidgetName.property` |
| Enumeration | Value `Name` field | `ValueName.caption` |
| Workflow | Activity `name` field | `ActivityName.taskName` |
| Microflow action | No user name; type + 1-based index within flow | `ShowMessage[1].template` |

Microflow actions have no user-defined identifier — only a GUID and an optional plain-string
`Caption`. The solution is type-index addressing (see Section 6), which avoids the
cross-cutting TextLiteral AST change originally proposed.

---

## 3. Architecture

```
User MDL input
      │
      ├── ALTER SETTINGS LANGUAGE ADD/DROP  ── LanguageRegistry commands
      ├── SHOW LANGUAGES / SHOW SUPPORTED LANGUAGES
      │
      ├── TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW  ── TranslateStmt AST
      ├── TRANSLATE MICROFLOW (type-index)              ── TranslateMicroflowStmt AST
      │
      └── DESCRIBE TRANSLATIONS Mod.Elem [IN lang]      ── DescribeTranslationsStmt AST
                  ├── text output with executable template comment
                  └── JSON output: {missing[], translated[], translate_template}

Grammar  (MDLDomainModel.g4, MDLSettings.g4)
      ↓
AST  (mdl/ast/)
      ↓
Visitor  (mdl/visitor/)
      ↓
Executor  (mdl/executor/)  — no direct BSON access
      ↓
Backend interface  (mdl/backend/)
      │  LanguageBackend      — read/write Settings$LanguageSettings
      │  TranslationBackend   — set translation by path + lang on named elements
      └  TextNodeBackend      — enumerate Texts$Text nodes for DESCRIBE
              ↓
      mdl/backend/mpr/  — BSON implementation
```

### New AST nodes

- `AlterLanguageStmt` — ADD / DROP a language code
- `TranslateStmt` — TRANSLATE PAGE/SNIPPET/ENUMERATION/WORKFLOW with SET ops
- `TranslateMicroflowStmt` — TRANSLATE MICROFLOW with type-index SET ops
- `DescribeTranslationsStmt` — inspect translation coverage

### New backend interface methods

```go
// mdl/backend/language.go
type LanguageBackend interface {
    AddLanguage(code string, opts LanguageOptions) error
    DropLanguage(code string) error
    ListProjectLanguages() ([]model.Language, error)
}

// mdl/backend/translation.go
type TranslationBackend interface {
    SetTranslation(docQN string, docType TranslationDocType, path, lang, text string) error
    SetMicroflowActionTranslation(docQN, actionType string, index int, property, lang, text string) error
    ListTranslationNodes(docQN string, docType TranslationDocType) ([]TranslationNode, error)
}

type TranslationNode struct {
    Path     string            // "Button_Submit.caption", "ShowMessage[1].template"
    Property string
    Texts    map[string]string // langCode → text; missing key = not translated
}
```

### Core BSON write primitive

Replaces the current pattern of blindly overwriting the first translation:

```go
// mdl/backend/mpr/translation_writer.go
func setTranslationForLang(textsBsonD bson.D, langCode, text string) {
    items := extractBsonArray(dGet(textsBsonD, "Items"))
    for i, item := range items {
        doc, ok := item.(bson.D)
        if !ok {
            continue
        }
        if extractString(dGet(doc, "LanguageCode")) == langCode {
            dSet(&items[i], "Text", text)   // found — update in place
            dSet(&textsBsonD, "Items", bson.A(items))
            return
        }
    }
    // not found — append new Translation entry
    newTr := bson.D{
        {Key: "$ID",          Value: newMendixID()},
        {Key: "$Type",        Value: "Texts$Translation"},
        {Key: "LanguageCode", Value: langCode},
        {Key: "Text",         Value: text},
    }
    newItems := append(items, newTr)
    dSet(&textsBsonD, "Items", bson.A(newItems))
}
```

**Pre-implementation check:** Before writing `translation_writer.go`, verify that `dGet`,
`dSet`, `extractBsonArray`, `extractString`, and `newMendixID` all exist in
`mdl/backend/mpr/` with compatible signatures. Run `grep -n "^func dSet\|^func dGet\|^func extractBsonArray\|^func newMendixID" mdl/backend/mpr/*.go`.

---

## 4. Language Registry Commands

### Syntax

```mdl
-- Add a language
ALTER SETTINGS LANGUAGE ADD 'zh_CN';
ALTER SETTINGS LANGUAGE ADD 'zh_CN' (
  checkCompleteness: true,
  dateFormat:        'yyyy-MM-dd',
  dateTimeFormat:    'yyyy-MM-dd HH:mm',
  timeFormat:        'HH:mm'
);

-- Remove a language
ALTER SETTINGS LANGUAGE DROP 'fr_FR';

-- Change default language (existing)
ALTER SETTINGS LANGUAGE DefaultLanguageCode = 'zh_CN';

-- List registered languages (reads Settings$LanguageSettings directly, no catalog required)
-- Output: Code | Default | CheckCompleteness | DateFormat | DateTimeFormat
SHOW LANGUAGES;

-- List all valid Mendix language codes
SHOW SUPPORTED LANGUAGES;
```

### Built-in language whitelist

Stored as a Go map constant in `mdl/executor/cmd_languages.go`.

| Code | Language | Code | Language |
|---|---|---|---|
| `en_US` | English (US) | `zh_CN` | Chinese (Simplified) |
| `en_GB` | English (UK) | `zh_TW` | Chinese (Traditional) |
| `nl_NL` | Dutch | `ja_JP` | Japanese |
| `de_DE` | German | `ko_KR` | Korean |
| `fr_FR` | French | `ar_SA` | Arabic |
| `es_ES` | Spanish | `pt_BR` | Portuguese (Brazil) |
| `it_IT` | Italian | `ru_RU` | Russian |
| `pl_PL` | Polish | `tr_TR` | Turkish |
| `sv_SE` | Swedish | `da_DK` | Danish |
| `fi_FI` | Finnish | `nb_NO` | Norwegian |
| `hu_HU` | Hungarian | `cs_CZ` | Czech |
| `ro_RO` | Romanian | `el_GR` | Greek |
| `th_TH` | Thai | `id_ID` | Indonesian |
| `vi_VN` | Vietnamese | `uk_UA` | Ukrainian |

### Validation rules

| Scenario | Behavior |
|---|---|
| ADD invalid code (e.g. `"chinese"`) | Error: `'chinese' is not a valid Mendix language code. Run SHOW SUPPORTED LANGUAGES.` |
| ADD already-registered language | Idempotent: success with info message |
| DROP non-existent language | Idempotent: success with info message |
| DROP the defaultLanguageCode | Error: `Cannot drop the default language 'en_US'. Change DefaultLanguageCode first.` |
| DROP language with existing translations | Warning + proceed: translations BSON is not deleted; only registration removed |

### Grammar change (`MDLLexer.g4` + `MDLSettings.g4`)

New tokens required in `MDLLexer.g4`:

```antlr
TRANSLATE   : T R A N S L A T E ;
TRANSLATIONS: T R A N S L A T I O N S ;
SUPPORTED   : S U P P O R T E D ;
```

Also add `TRANSLATE`, `TRANSLATIONS`, `SUPPORTED` to the `keyword` rule in `MDLSettings.g4`
so they can appear as identifiers where needed.

Extend `alterSettingsClause` in `MDLSettings.g4`:

```antlr
alterSettingsClause
    : settingsSection settingsAssignment (COMMA settingsAssignment)*
    | CONSTANT STRING_LITERAL (VALUE settingsValue | DROP) (IN CONFIGURATION STRING_LITERAL)?
    | DROP CONSTANT STRING_LITERAL (IN CONFIGURATION STRING_LITERAL)?
    | CONFIGURATION STRING_LITERAL settingsAssignment (COMMA settingsAssignment)*
    | LANGUAGE ADD STRING_LITERAL (LPAREN languageOptions RPAREN)?   // NEW
    | LANGUAGE DROP STRING_LITERAL                                    // NEW
    ;

languageOptions
    : languageOption (COMMA languageOption)*
    ;

languageOption
    : identifierOrKeyword COLON settingsValue
    ;
```

---

## 5. TRANSLATE Command (Named-Address Elements)

Handles elements with stable user-visible names: Page, Snippet, Enumeration, Workflow.

### Syntax

```mdl
-- Page / Snippet
TRANSLATE PAGE MyModule.Home IN zh_CN
  SET Button_Submit.caption     = '提交'
  SET Button_Cancel.caption     = '取消'
  SET TextBox_Email.placeholder = '请输入邮箱'
  SET TextBox_Email.label       = '邮箱'
  SET Title                     = '登录';

TRANSLATE SNIPPET MyModule.Snip_Header IN zh_CN
  SET Text_Welcome.content = '欢迎使用';

-- Enumeration
TRANSLATE ENUMERATION MyModule.OrderStatus IN zh_CN
  SET Active.caption   = '活跃'
  SET Inactive.caption = '非活跃'
  SET Pending.caption  = '待处理';

-- Workflow
TRANSLATE WORKFLOW MyModule.WF_Approval IN zh_CN
  SET ApproveTask.taskName        = '审批任务'
  SET ApproveTask.taskDescription = '请审核并做出决定'
  SET RejectTask.taskName         = '驳回任务';
```

### Path resolution rules

| Path format | Scope | Resolution |
|---|---|---|
| `Title` | Page level | Update page `Title` Texts$Text directly |
| `WidgetName.caption` | Any widget | `findBsonWidget(name)` → update `Caption` |
| `WidgetName.placeholder` | Input widgets | `findBsonWidget(name)` → update `Placeholder` |
| `WidgetName.label` | Input widgets | `findBsonWidget(name)` → update `Label.Caption` |
| `WidgetName.tooltip` | Any widget | `findBsonWidget(name)` → update `Tooltip` |
| `WidgetName.content` | Text widgets | `findBsonWidget(name)` → update `Content` |
| `ValueName.caption` | Enumeration | Find value where `Name == ValueName` → update `Caption` |
| `ActivityName.taskName` | Workflow UserTask | Find activity where `name == ActivityName` → update `TaskName` |
| `ActivityName.taskDescription` | Workflow UserTask | Same → update `TaskDescription` |

### TRANSLATE is idempotent

Re-running TRANSLATE with the same text is a no-op (not an error). Re-running with
different text overwrites the previous translation. This is required for AI agent
retry-safety.

### AST node

```go
// mdl/ast/ast_translate.go
type TranslateStmt struct {
    DocType string          // "PAGE", "SNIPPET", "ENUMERATION", "WORKFLOW"
    QName   QualifiedName
    Lang    string          // "zh_CN"
    Ops     []TranslateSetOp
}

type TranslateSetOp struct {
    Path string             // "Button_Submit.caption" or "Title"
    Text string
}
```

### Error handling

| Scenario | Error message |
|---|---|
| `lang` not registered | `Language 'zh_CN' is not registered. Run ALTER SETTINGS LANGUAGE ADD 'zh_CN' first.` |
| Widget not found | `Widget 'Button_Submit' not found in MyModule.Home` |
| Widget has no such property | `Widget 'Button_Submit' has no 'placeholder' property` |
| Enumeration value not found | `Enumeration value 'Active' not found in MyModule.OrderStatus` |
| Workflow activity not found | `Activity 'ApproveTask' not found in MyModule.WF_Approval` |

### Grammar change (`MDLDomainModel.g4`)

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

---

## 6. TRANSLATE MICROFLOW (Type-Index Addressing)

Microflow actions have no user-defined names — only GUIDs. The original design proposed
extending the `TextLiteral` AST type from `string` to `map[string]string`, but that change
is cross-cutting (affects 30–50 files across AST, visitor, and all action builders) and is
deferred to Phase C. Instead, microflow actions are addressed by action type + 1-based
appearance index within the flow.

### Syntax

```mdl
-- Translate the 1st ShowMessage action in the flow (top-to-bottom order)
TRANSLATE MICROFLOW MyModule.ACT_Process IN zh_CN
  SET ShowMessage[1].template         = '操作成功'
  SET ValidationFeedback[1].message   = '电子邮件格式无效'
  SET ValidationFeedback[2].message   = '用户名不能为空'
  SET Log[1].template                 = '用户 {1} 已创建';
```

### Type-index rules

| Path format | Action type | Property |
|---|---|---|
| `ShowMessage[N].template` | ShowMessage | `MessageTemplate` (Texts$Text) |
| `ValidationFeedback[N].message` | ValidationFeedback | `FeedbackTemplate` (Texts$Text) |
| `Log[N].template` | Log | `MessageTemplate` (Texts$Text) |

- Indices are **1-based**
- Order is top-to-bottom within the microflow's ObjectCollection
- For sub-flows (LoopedActivity), the inner activities are traversed depth-first
- If index N exceeds the count of that action type, return a clear error:
  `ShowMessage[3] not found in MyModule.ACT_Process (only 2 ShowMessage actions exist)`

### DESCRIBE TRANSLATIONS for microflows

`DESCRIBE TRANSLATIONS MyModule.ACT_Process` walks the ObjectCollection and outputs
type-indexed paths:

```
Path                       Property   en_US                     zh_CN
ShowMessage[1]             template   Please fill all fields    (missing)
ValidationFeedback[1]      message    Invalid email format      电子邮件格式无效
Log[1]                     template   User {1} created          (missing)
```

### AST node

```go
// mdl/ast/ast_translate.go
type TranslateMicroflowStmt struct {
    QName QualifiedName
    Lang  string
    Ops   []TranslateMicroflowSetOp
}

type TranslateMicroflowSetOp struct {
    ActionType string // "ShowMessage", "ValidationFeedback", "Log"
    Index      int    // 1-based
    Property   string // "template", "message"
    Text       string
}
```

### Grammar change (`MDLMicroflow.g4` or `MDLDomainModel.g4`)

```antlr
translateMicroflowStatement
    : TRANSLATE MICROFLOW qualifiedName IN identifierOrKeyword
      translateMicroflowSetOp+
    ;

translateMicroflowSetOp
    : SET identifierOrKeyword LBRACKET INTEGER_LITERAL RBRACKET DOT identifierOrKeyword EQUALS STRING_LITERAL
    ;
```

### Phase C (Deferred): Inline TextLiteral

The original proposal to change `TextLiteral` from `string` to `map[string]string` in the
microflow AST is deferred. It would allow:

```mdl
show message 'Info' (en_US: 'OK', zh_CN: '好的') blocking;
```

This is attractive for new microflow creation but imposes a large cross-cutting change.
Defer until Phase C once the TRANSLATE MICROFLOW approach has been validated in production.

---

## 7. DESCRIBE TRANSLATIONS

Provides translation coverage inspection for any document type, with output designed for
both human review and AI agent consumption.

### Syntax

```mdl
-- All languages for an element
DESCRIBE TRANSLATIONS MyModule.Home;

-- Specific language (shows missing entries)
DESCRIBE TRANSLATIONS MyModule.Home IN zh_CN;

-- Microflow (uses type-index paths)
DESCRIBE TRANSLATIONS MyModule.ACT_Validate;
DESCRIBE TRANSLATIONS MyModule.ACT_Validate IN zh_CN;
```

### Text output

```
-- describe translations MyModule.Home in zh_CN
Path                    Property     en_US                     zh_CN
Button_Submit           caption      Submit                    (missing)
TextBox_Email           placeholder  Enter email               (missing)
Title                   title        Login                     登录

(3 translatable fields, 2 missing in zh_CN)

-- Ready-to-execute template (replace '?' with translations):
-- translate page MyModule.Home in zh_CN
--   set Button_Submit.caption = '?'
--   set TextBox_Email.placeholder = '?';
```

The trailing commented TRANSLATE block is the "describe → translate" roundtrip: the user
(or AI agent) can copy it, fill in the `'?'` values, and run it directly.

### JSON output (AI agent consumption)

When `ctx.Format == FormatJSON`, output:

```json
{
  "document": "MyModule.Home",
  "document_type": "PAGE",
  "target_language": "zh_CN",
  "project_languages": ["en_US", "nl_NL", "zh_CN"],
  "missing": [
    {"path": "Button_Submit.caption", "property": "caption",
     "source_lang": "en_US", "source": "Submit"},
    {"path": "TextBox_Email.placeholder", "property": "placeholder",
     "source_lang": "en_US", "source": "Enter email"}
  ],
  "translated": [
    {"path": "Title", "property": "title",
     "source_lang": "en_US", "source": "Login", "text": "登录"}
  ],
  "translate_template": "translate page MyModule.Home in zh_CN\n  set Button_Submit.caption = '?'\n  set TextBox_Email.placeholder = '?';"
}
```

Fields:
- `missing[]`: fields not yet translated in the target language; includes `source` text to enable AI translation
- `translated[]`: fields that already have a translation
- `translate_template`: executable MDL with `'?'` placeholders for all missing fields

### AST node

```go
type DescribeTranslationsStmt struct {
    QName QualifiedName
    Lang  string   // empty = show all project languages
}
```

---

## 8. AI Agent Translation Workflow

The combination of `DESCRIBE TRANSLATIONS` (JSON) and `TRANSLATE` commands is designed for
AI agent-driven translation pipelines. This is the primary motivation for the JSON output
format and the `translate_template` field.

### Workflow

```
1. Discover missing strings
   describe translations MyModule.Home in zh_CN;
   → JSON: {missing: [{path: "Button_Submit.caption", source: "Submit"}, ...]}

2. AI agent translates source strings to target language
   (calls LLM or translation service with each `source` value)

3. AI agent executes TRANSLATE command with filled translations
   translate page MyModule.Home in zh_CN
     set Button_Submit.caption = '提交'
     set TextBox_Email.placeholder = '请输入邮箱';

4. Verify coverage
   describe translations MyModule.Home in zh_CN;
   → JSON: {missing: [], translated: [...]}
```

### Design requirements

- `missing[].source` MUST contain the source language text so the agent can translate without
  a second lookup
- `translate_template` MUST be valid MDL when `'?'` placeholders are replaced with strings
- TRANSLATE MUST be idempotent (re-running with same text = no error) for retry-safety
- JSON output MUST be the default when `--format json` is set; text output is the default
  otherwise

### Example: full-project translation agent script

```mdl
-- 1. Register target language
alter settings language add 'zh_CN';

-- 2. For each module/page, describe and translate (AI fills in translations)
describe translations MyModule.Home in zh_CN;
-- AI executes generated translate_template with filled values

describe translations MyModule.OrderStatus in zh_CN;
-- AI executes generated translate_template for enumeration

-- 3. Verify overall coverage
show languages;
```

---

## 9. Full-Stack Implementation Checklist

Each new feature must be wired through the full pipeline per the PR checklist:

| Layer | Files to change |
|---|---|
| **Grammar** | `MDLLexer.g4` (TRANSLATE, TRANSLATIONS, SUPPORTED), `MDLSettings.g4` (ADD/DROP language, languageOptions, keyword additions), `MDLDomainModel.g4` (translateStatement, translateMicroflowStatement, DESCRIBE TRANSLATIONS) |
| **Regenerate parser** | `make grammar` — do NOT commit generated files |
| **AST** | `mdl/ast/ast_translate.go` (new: AlterLanguageStmt, TranslateStmt, TranslateMicroflowStmt, DescribeTranslationsStmt) |
| **Visitor** | `mdl/visitor/visitor_settings.go` (ADD/DROP), `mdl/visitor/visitor_translate.go` (new: TRANSLATE, TRANSLATE MICROFLOW, DESCRIBE TRANSLATIONS) |
| **Executor** | `mdl/executor/cmd_languages.go` (ADD/DROP/SHOW SUPPORTED/SHOW enhanced), `mdl/executor/cmd_translate.go` (new), `mdl/executor/cmd_describe_translations.go` (new) |
| **Backend interface** | `mdl/backend/language.go` (new), `mdl/backend/translation.go` (new) |
| **Backend MPR impl** | `mdl/backend/mpr/translation_writer.go` (new: setTranslationForLang + per-type helpers), `mdl/backend/mpr/page_mutator.go` (SetWidgetTranslation), `mdl/backend/mpr/backend.go` (SetMicroflowActionTranslation, ListTranslationNodes) |
| **Backend mock** | `mdl/backend/mock/` — stub all new interface methods |
| **Tests** | `mdl/executor/cmd_languages_mock_test.go` (extend), `cmd_translate_mock_test.go` (new), `cmd_describe_translations_mock_test.go` (new); MDL examples in `mdl-examples/doctype-tests/` |
| **Syntax reference** | `docs/01-project/MDL_QUICK_REFERENCE.md` |
| **Syntax topics** | `cmd/mxcli/syntax/features_*.go` |

---

## 10. Implementation Phases

### Phase A — Foundation (low risk, high value)

Implement first; these are safe and foundational:

1. `SHOW SUPPORTED LANGUAGES` + language whitelist
2. Enhanced `SHOW LANGUAGES` (reads Settings directly, no catalog required)
3. `setTranslationForLang` BSON primitive + helper verification
4. `DESCRIBE TRANSLATIONS` with JSON output + `translate_template`
5. `ALTER SETTINGS LANGUAGE ADD/DROP`

### Phase B — Write Operations

Build on Phase A; require the BSON primitive from Phase A Task 3:

6. `TRANSLATE PAGE / SNIPPET`
7. `TRANSLATE ENUMERATION`
8. `TRANSLATE WORKFLOW`
9. `TRANSLATE MICROFLOW` (type-index addressing)

### Phase C — Deferred

Re-evaluate after Phase B is stable in production:

- Inline TextLiteral `(en_US: '...', zh_CN: '...')` in microflow CREATE OR MODIFY
- Bulk CSV/XLIFF export/import
- Translation memory / AI suggestion integration

---

## 11. Out of Scope

- **Inline `TextLiteral` syntax** in microflow CREATE OR MODIFY: deferred to Phase C.
  The TRANSLATE MICROFLOW type-index approach covers the same use cases without the
  cross-cutting AST change.
- **Bulk XLIFF/CSV export/import**: not in this spec. DESCRIBE TRANSLATIONS JSON +
  TRANSLATE covers targeted workflows; bulk is a separate feature.
- **Translation of system texts** (`SystemTextCollection`): managed by Mendix runtime,
  not user-editable via MDL.
- **Translation memory or AI suggestions**: no built-in AI translation in MDL itself.
  The workflow in Section 8 uses external AI (the agent calling MDL), not embedded AI.
- **TRANSLATE MICROFLOW with action-level name addressing**: not possible because
  ActionActivity has no user-defined name. Type-index is the supported alternative.
