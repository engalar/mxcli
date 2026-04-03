# MDL Internationalization (i18n) Support

**Date:** 2026-04-03
**Status:** Proposal
**Author:** @anthropics/claude-code

## Problem

MDL currently handles all translatable text fields (page titles, widget captions, enumeration captions, microflow message templates) as single-language strings. When creating or describing model elements, only the default language is read or written. All other translations are silently dropped.

This means:
- `DESCRIBE PAGE` output loses translations — roundtripping a page strips non-default languages
- `CREATE PAGE` can only set one language — multi-language projects require Studio Pro for translation
- No way to audit translation coverage from the CLI

Mendix stores translations as `Texts$Text` objects containing an array of `Texts$Translation` entries (one per language). The mxcli internal model (`model.Text`) already represents translations as `map[string]string`, and the BSON reader/writer already handles multi-language serialization. The gap is purely at the MDL syntax and command layer.

## Scope

**In scope (syntax-layer extension):**
- Inline multi-language text literal syntax for CREATE/ALTER
- DESCRIBE WITH TRANSLATIONS output mode
- SHOW TRANSLATIONS query command
- Writer changes to serialize multi-language BSON correctly

**Out of scope:**
- Batch export/import (CSV, XLIFF) — future proposal
- ALTER TRANSLATION standalone command — future proposal
- Translation memory or machine translation integration

## Design

### 1. Translated Text Literal Syntax

Any MDL property that accepts a string literal `'text'` can alternatively accept a translation map:

```sql
-- Single language (backward compatible, unchanged)
Title: 'Hello World'

-- Multi-language
Title: {
  en_US: 'Hello World',
  zh_CN: '你好世界',
  nl_NL: 'Hallo Wereld'
}
```

**Grammar (ANTLR4):**

```antlr
translatedText
    : STRING_LITERAL
    | '{' translationEntry (',' translationEntry)* ','? '}'
    ;

translationEntry
    : IDENTIFIER ':' STRING_LITERAL
    ;
```

**AST node:**

```go
type TranslatedText struct {
    Translations map[string]string // languageCode → text
    IsMultiLang  bool              // false = single bare string
}
```

**Semantics:**
- Bare string `'text'` writes to the project's `DefaultLanguageCode`. Existing translations in other languages are preserved.
- Map `{ lang: 'text', ... }` writes the specified languages. Languages not mentioned in the map are preserved (merge, not replace).
- No syntax for deleting a translation (use Studio Pro).

### 2. DESCRIBE WITH TRANSLATIONS

```sql
-- Default: single language output (backward compatible)
DESCRIBE PAGE Module.MyPage;
-- Output: Title: 'Hello World'

-- New: all translations
DESCRIBE PAGE Module.MyPage WITH TRANSLATIONS;
-- Output:
-- Title: {
--   en_US: 'Hello World',
--   zh_CN: '你好世界'
-- }
```

**Rules:**
- Without `WITH TRANSLATIONS`: outputs only the default language as a bare string (current behavior).
- With `WITH TRANSLATIONS`: if only one language exists, still uses bare string; if ≥2 languages, uses map syntax.
- Output must be re-parseable by the MDL parser (roundtrip guarantee).

**Grammar:**

```antlr
describeStatement
    : DESCRIBE objectType qualifiedName withTranslationsClause?
    ;

withTranslationsClause
    : WITH TRANSLATIONS
    ;
```

**Affected commands:**
- DESCRIBE PAGE / SNIPPET — Title, widget Caption, Placeholder
- DESCRIBE ENTITY — validation rule messages
- DESCRIBE MICROFLOW / NANOFLOW — LogMessage, ShowMessage, ValidationFeedback templates
- DESCRIBE ENUMERATION — value captions
- DESCRIBE WORKFLOW — task names, descriptions, outcome captions

### 3. SHOW TRANSLATIONS

```sql
-- All translations in a module
SHOW TRANSLATIONS IN Module;

-- Only missing translations
SHOW TRANSLATIONS IN Module MISSING;

-- All translations project-wide
SHOW TRANSLATIONS MISSING;
```

**Output (tabular):**

```
Element                    Context        en_US          zh_CN          nl_NL
─────────────────────────────────────────────────────────────────────────────
Module.MyPage              page_title     Hello World    你好世界        ✗
Module.MyPage.SaveButton   caption        Save           保存            ✗
Module.Status.Active       enum_caption   Active         活跃            ✗
```

`✗` indicates a missing translation. The `MISSING` filter shows only rows with at least one gap.

**Implementation:** Reuses the existing catalog `strings` FTS5 table. Pivots rows by language code into a wide-format table. Requires `REFRESH CATALOG FULL` to index strings first.

### 4. Writer Layer Changes

When executing CREATE/ALTER with multi-language text, the writer serializes all provided translations into the standard Mendix BSON format:

```go
titleItems := bson.A{int32(2)} // marker for non-empty
for langCode, text := range translatedText.Translations {
    titleItems = append(titleItems, bson.D{
        {Key: "$ID", Value: generateUUID()},
        {Key: "$Type", Value: "Texts$Translation"},
        {Key: "LanguageCode", Value: langCode},
        {Key: "Text", Value: text},
    })
}
```

**Merge semantics for bare strings:**
When a bare string `'text'` is used, the writer must:
1. Read the existing `Texts$Text` from the MPR
2. Update only the `DefaultLanguageCode` entry
3. Preserve all other language entries unchanged

**Affected writer functions:**
- `writer_pages.go` — Page Title, widget Caption/Placeholder
- `writer_enumeration.go` — EnumerationValue Caption
- `writer_microflow.go` — StringTemplate (log/show/validation messages)
- `writer_widgets.go` — all widget Caption/Placeholder properties

## Translatable Fields Inventory

The following fields use `Texts$Text` and are affected by this proposal:

| Category | StringContext | Count | Examples |
|----------|-------------|-------|---------|
| Page metadata | `page_title` | 1 | Page.Title |
| Enumeration values | `enum_caption` | per value | EnumerationValue.Caption |
| Microflow actions | `log_message`, `show_message`, `validation_message` | 3 | LogMessageAction, ShowMessageAction |
| Workflow objects | `task_name`, `task_description`, `outcome_caption`, `activity_caption` | 4 | UserTask.Name, UserTask.Description |
| Widget properties | `caption`, `placeholder` | 7+ | ActionButton.Caption, TextInput.Placeholder |

**Note:** Widget-level translations (caption, placeholder) are not currently indexed in the catalog `strings` table. A follow-up task should extend `catalog/builder_strings.go` to extract these.

## Implementation Phases

| Phase | Scope | Dependency |
|-------|-------|------------|
| **P1** | Grammar + AST: `translatedText` rule, `TranslatedText` node | None |
| **P2** | Visitor: parse `{ lang: 'text' }` into AST | P1 |
| **P3** | DESCRIBE WITH TRANSLATIONS: all describe commands output multi-language | P1 (reuses AST) |
| **P4** | Writer: CREATE/ALTER write multi-language BSON | P1 + P2 |
| **P5** | SHOW TRANSLATIONS: catalog query command | None (independent) |
| **P6** | Widget translation indexing: extend catalog builder for widget-level translations | P5 |

Each phase is independently deliverable and testable.

## Compatibility

- **Backward compatible**: existing MDL scripts with bare strings continue to work identically.
- **Forward compatible**: MDL scripts using `{ lang: 'text' }` syntax will fail gracefully on older mxcli versions with a parse error pointing to the `{` token.
- **DESCRIBE roundtrip**: `DESCRIBE ... WITH TRANSLATIONS` output can be fed back to `CREATE OR REPLACE` to reproduce the same translations.

## Risks

| Risk | Mitigation |
|------|-----------|
| `{` ambiguity with widget body blocks | Grammar context: `translatedText` only appears in property value position, not statement position. Widget bodies follow `)` not `:`. |
| Translation ordering in BSON | Mendix does not depend on translation order within `Items` array. Sort by language code for deterministic output. |
| Large translation maps cluttering DESCRIBE output | `WITH TRANSLATIONS` is opt-in; default remains single-language. |
