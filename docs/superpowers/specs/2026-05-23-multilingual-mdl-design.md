# Multilingual MDL Support — Design Spec

**Date:** 2026-05-23  
**Status:** Approved  
**Scope:** Language registry management + multilingual text maintenance via MDL

---

## 1. Problem Statement

Mendix projects support multiple languages through `Settings$LanguageSettings` (the language
registry) and `Texts$Text` / `Texts$Translation` BSON nodes embedded in every translatable
element. MDL currently has no way to:

1. Add or remove languages from the project language registry.
2. Read or update translated text for a specific language on a specific element.
3. Know which Mendix language codes are valid.

The goal is a complete MDL workflow: register a new language, inspect what needs translating,
and fill in translations element by element.

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
| Microflow action | No user-visible name, only GUID | Inline syntax in CREATE OR MODIFY |

Microflow actions cannot be addressed by name because `ActionActivity` has no user-defined
identifier — only a GUID and an optional plain-string `Caption`. The solution is to embed
multilingual text directly in the microflow definition syntax, which is already a full
replacement (CREATE OR MODIFY replaces the entire flow).

---

## 3. Architecture

```
User MDL input
      │
      ├── ALTER SETTINGS LANGUAGE ADD/DROP  ── LanguageRegistry commands
      ├── SHOW LANGUAGES / SHOW SUPPORTED LANGUAGES
      │
      ├── TRANSLATE PAGE/ENUMERATION/WORKFLOW  ── TranslateStmt AST
      │
      ├── DESCRIBE TRANSLATIONS Mod.Elem [IN lang]  ── DescribeTranslationsStmt AST
      │
      └── CREATE OR MODIFY MICROFLOW ... (lang: 'text', ...)
                      └── extends TextLiteral AST node
                             (was: string, now: map[string]string)

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
- `TranslateStmt` — TRANSLATE PAGE/ENUMERATION/WORKFLOW with SET ops
- `DescribeTranslationsStmt` — inspect translation coverage
- Extended `TextLiteral` — `map[string]string` instead of `string`

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
    ListTranslationNodes(docQN string, docType TranslationDocType) ([]TranslationNode, error)
}

type TranslationNode struct {
    Path     string            // "Button_Submit.caption", "ShowMessage.template"
    Property string
    Texts    map[string]string // langCode → text; missing key = not translated
}
```

### Core BSON write primitive

Replaces the current `setTranslatableText` (which blindly overwrites the first translation):

```go
// mdl/backend/mpr/translation_writer.go
func setTranslationForLang(textsBsonD bson.D, langCode, text string) {
    items := extractBsonArray(dGet(textsBsonD, "Items"))
    for _, item := range items {
        doc := extractBsonMap(item)
        if extractString(doc["LanguageCode"]) == langCode {
            dSet(doc, "Text", text)   // found — update in place
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
    appendBsonArrayItem(textsBsonD, "Items", newTr)
}
```

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

-- List registered languages (shows code, default flag, completeness, string count)
SHOW LANGUAGES;

-- List all valid Mendix language codes
SHOW SUPPORTED LANGUAGES;
```

### Built-in language whitelist (representative subset)

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

### Validation rules

| Scenario | Behavior |
|---|---|
| ADD invalid code (e.g. `"chinese"`) | Error: `'chinese' is not a valid Mendix language code. Run SHOW SUPPORTED LANGUAGES.` |
| ADD already-registered language | Idempotent: success with info message |
| DROP non-existent language | Idempotent: success with info message |
| DROP the defaultLanguageCode | Error: `Cannot drop the default language 'en_US'. Change DefaultLanguageCode first.` |
| DROP language with existing translations | Warning + proceed: `Warning: zh_CN has N translated strings. Language registration removed; existing BSON translations are not deleted.` |

### Grammar change (`MDLSettings.g4`)

Extend `alterSettingsClause`:

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

## 5. TRANSLATE Command

Handles named-address elements: Page, Snippet, Enumeration, Workflow.

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
| `ValueName.caption` | Enumeration | Find value where `Name == ValueName` → update `Caption` |
| `ActivityName.taskName` | Workflow UserTask | Find activity where `name == ActivityName` → update `TaskName` |
| `ActivityName.taskDescription` | Workflow UserTask | Same → update `TaskDescription` |

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

## 6. Microflow Inline Multilingual Syntax

Microflow actions have no user-visible name — only a GUID. Since `CREATE OR MODIFY MICROFLOW`
replaces the entire flow, multilingual text is embedded inline in the microflow definition.

### Syntax

Single string literal remains backward-compatible (writes `en_US`):

```mdl
-- Backward compatible (en_US only)
show message 'Information' 'Operation successful' blocking;

-- Inline multilingual map
show message 'Information' (
  en_US: 'Operation successful',
  zh_CN: '操作成功'
) blocking;

validation feedback on $Order.Email (
  en_US: 'Invalid email format',
  zh_CN: '电子邮件格式无效'
);

log 'MyModule' level information message (
  en_US: 'User {1} created',
  zh_CN: '用户 {1} 已创建'
) with $UserName;

-- Microflow-level concurrency error message
create or modify microflow MyModule.ACT_Process () returns Nothing
  concurrencyErrorMessage: (
    en_US: 'Process is already running',
    zh_CN: '流程正在运行中'
  )
begin
  ...
end;
```

### Grammar change

New `textLiteral` rule replaces all `STRING_LITERAL` in message positions:

```antlr
textLiteral
    : STRING_LITERAL                                          # textSingle
    | LPAREN textTranslation (COMMA textTranslation)* RPAREN # textMap
    ;

textTranslation
    : identifierOrKeyword COLON STRING_LITERAL
    ;
```

### AST change

```go
// mdl/ast/ast_microflow.go
type TextLiteral struct {
    Translations map[string]string  // langCode → text
}

// Backward-compat constructor: single string → en_US
func SingleText(s string) TextLiteral {
    return TextLiteral{Translations: map[string]string{"en_US": s}}
}
```

### Write behavior

```go
func buildTextNode(lit ast.TextLiteral) *genTx.Text {
    text := genTx.NewText()
    for lang, txt := range lit.Translations {
        tr := genTx.NewTranslation()
        tr.SetLanguageCode(lang)
        tr.SetText(txt)
        text.AddTranslations(tr)
    }
    return text
}
```

All action builders (`buildShowMessageAction`, `buildValidationFeedbackAction`,
`buildLogMessageAction`) switch from `string` to `TextLiteral`.

---

## 7. DESCRIBE TRANSLATIONS

Provides translation coverage inspection for any document type.

### Syntax

```mdl
-- All languages for an element
DESCRIBE TRANSLATIONS MyModule.Home;

-- Specific language (shows missing entries)
DESCRIBE TRANSLATIONS MyModule.Home IN zh_CN;

-- Microflow (shows action message texts)
DESCRIBE TRANSLATIONS MyModule.ACT_Validate;
DESCRIBE TRANSLATIONS MyModule.ACT_Validate IN zh_CN;
```

### Output format

```
Path                    Property     en_US                           zh_CN       nl_NL
Button_Submit           caption      Submit                          提交         (missing)
TextBox_Email           placeholder  Enter email                     (missing)   (missing)
Title                   title        Login                           登录         Inloggen
```

For microflows:

```
Type               Property          en_US                            zh_CN
ShowMessage        template          Please fill in required fields   (missing)
ValidationFeedback feedbackTemplate  Invalid email format             请输入正确邮箱
```

### AST node

```go
type DescribeTranslationsStmt struct {
    QName QualifiedName
    Lang  string   // empty = show all project languages
}
```

---

## 8. Full-Stack Implementation Checklist

Each new feature must be wired through the full pipeline per the PR checklist:

| Layer | Files to change |
|---|---|
| **Grammar** | `MDLSettings.g4` (ADD/DROP language, languageOptions), `MDLDomainModel.g4` (translateStatement, textLiteral, DESCRIBE TRANSLATIONS) |
| **Regenerate parser** | `make grammar` — do NOT commit generated files |
| **AST** | `mdl/ast/ast_translate.go` (new), `mdl/ast/ast_microflow.go` (TextLiteral), `mdl/ast/ast_query.go` (DescribeTranslations variant) |
| **Visitor** | `mdl/visitor/visitor_settings.go` (ADD/DROP), `mdl/visitor/visitor_entity.go` or new file (TRANSLATE, DESCRIBE TRANSLATIONS), `mdl/visitor/visitor_helpers.go` (textLiteral parsing) |
| **Executor** | `mdl/executor/cmd_languages.go` (ADD/DROP/SHOW SUPPORTED), `mdl/executor/cmd_translate.go` (new), `mdl/executor/cmd_describe_translations.go` (new); update all action builders for TextLiteral |
| **Backend interface** | `mdl/backend/language.go` (new), `mdl/backend/translation.go` (new) |
| **Backend MPR impl** | `mdl/backend/mpr/language_compat.go` (new, ADD/DROP writes Settings BSON), `mdl/backend/mpr/translation_writer.go` (new, `setTranslationForLang`), `mdl/backend/mpr/page_mutator.go` (replace `setTranslatableText` calls), `mdl/backend/mpr/enum_constant_gen.go` (translation-aware writes) |
| **Backend mock** | `mdl/backend/mock/` — stub all new interface methods |
| **Tests** | `mdl/executor/cmd_languages_mock_test.go` (extend), new `cmd_translate_mock_test.go`, `cmd_describe_translations_mock_test.go`; MDL examples in `mdl-examples/doctype-tests/` |
| **Syntax reference** | `docs/01-project/MDL_QUICK_REFERENCE.md` |
| **Syntax topics** | `cmd/mxcli/syntax/features_*.go` |

---

## 9. Out of Scope

- **Bulk export / import of translations** (CSV, XLIFF): not in this spec. The DESCRIBE
  TRANSLATIONS output + TRANSLATE command covers the targeted workflow.
- **Translation of system texts** (`SystemTextCollection`): these are managed by Mendix
  runtime and are not user-editable via MDL.
- **Translation memory or suggestions**: no AI-assisted translation in this spec.
- **TRANSLATE MICROFLOW with action-level targeting**: deferred; inline syntax handles
  all microflow translation needs within the current CREATE OR MODIFY workflow.
