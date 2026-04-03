# MDL Internationalization (i18n) Support

**Date:** 2026-04-03 (updated 2026-04-03)
**Status:** Proposal (revised per review feedback)
**Author:** @engalar

## Problem

MDL currently handles all translatable text fields (page titles, widget captions, enumeration captions, microflow message templates) as single-language strings. When creating or describing model elements, only the default language is read or written. All other translations are silently dropped.

This means:
- `DESCRIBE PAGE` output loses translations — roundtripping a page strips non-default languages
- `CREATE PAGE` can only set one language — multi-language projects require Studio Pro for translation
- No way to see all translations in context (note: `SHOW LANGUAGES` and `QUAL005 MissingTranslations` linter rule already provide language inventory and gap detection via the catalog `strings` table)

Mendix stores translations as `Texts$Text` objects containing an array of `Texts$Translation` entries (one per language). The mxcli internal model (`model.Text`) already represents translations as `map[string]string`, and the BSON reader/writer already handles multi-language serialization. The gap is purely at the MDL syntax and command layer.

## Scope

**In scope (syntax-layer extension):**
- Inline multi-language text literal syntax for CREATE/ALTER/ALTER PAGE SET
- DESCRIBE WITH TRANSLATIONS output mode
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

New rule:

```antlr
translationMap
    : LBRACE translationEntry (COMMA translationEntry)* COMMA? RBRACE
    ;

translationEntry
    : IDENTIFIER COLON STRING_LITERAL
    ;
```

Integration into `propertyValueV3` (MDLParser.g4 line ~1961):

```antlr
propertyValueV3
    : STRING_LITERAL
    | translationMap                                        // NEW: { en_US: 'Hello', zh_CN: '你好' }
    | NUMBER_LITERAL
    | booleanLiteral
    | qualifiedName
    | IDENTIFIER
    | H1 | H2 | H3 | H4 | H5 | H6
    | LBRACKET (expression (COMMA expression)*)? RBRACKET
    ;
```

**Disambiguation from widget body `{`**: `translationMap` only appears inside `propertyValueV3`, which follows `COLON` or `EQUALS` in property definitions. Widget bodies (`widgetBodyV3`) follow `)` at statement level, never after `:`. The parser sees `Caption: {` and enters `propertyValueV3 → translationMap` — there is no ambiguity because `widgetBodyV3` is a separate production in `widgetStatementV3` that requires `(...)` before `{`.

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

### 3. ALTER PAGE SET with Translation Maps

Translation maps work in ALTER PAGE SET, enabling in-place translation updates:

```sql
ALTER PAGE Module.MyPage
  SET WIDGET saveButton Caption: { en_US: 'Save', zh_CN: '保存' };
```

This reuses the `translationMap` rule inside `propertyValueV3` — no additional grammar changes needed since ALTER PAGE SET already uses `propertyValueV3` for values.

### 4. Relationship to Existing Translation Features

`SHOW LANGUAGES` (commit a060152) already lists project languages with string counts. `QUAL005 MissingTranslations` linter rule already detects missing translations. The catalog `strings` FTS5 table already stores per-language text with `SELECT * FROM CATALOG.strings WHERE Language = 'nl_NL'`.

This proposal does **not** duplicate those features. It addresses the gap they cannot fill: **writing and round-tripping multi-language text in MDL syntax**.

### 5. Writer Layer Changes

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

**Merge semantics for bare strings (architectural change):**

Currently, all writer functions construct `Texts$Text` from scratch — e.g. `writer_pages.go:219-247` builds a new `Items` array every time. Bare-string merge semantics require a **read-modify-write cycle**:

1. Read the existing `Texts$Text` BSON from the MPR via `GetRawUnit`
2. Parse existing `Items` array to find the entry for `DefaultLanguageCode`
3. Update that entry's `Text` field (or insert if missing)
4. Preserve all other `Texts$Translation` entries unchanged
5. Write back the modified `Items` array

This is a significant change to writer architecture. A shared helper should be introduced:

```go
// mergeTranslation reads existing Texts$Text, merges new translations, returns updated BSON.
// For bare strings: translations = {defaultLang: text}
// For maps: translations = the full map
func mergeTranslation(existingBSON bson.D, translations map[string]string) bson.D
```

**Affected writer functions (11+ call sites):**
- `writer_pages.go` — Page Title, widget Caption/Placeholder
- `writer_enumeration.go` — EnumerationValue Caption
- `writer_microflow.go` — StringTemplate (log/show/validation messages)
- `writer_widgets.go` — all widget Caption/Placeholder properties
- `writer_widgets_action.go`, `writer_widgets_display.go`, `writer_widgets_input.go`

**Serialization ordering:** Translations within `Items` array must be sorted by language code for deterministic BSON output and diff-friendly DESCRIBE.

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

| Phase | Scope | Dependency | Risk |
|-------|-------|------------|------|
| **P1** | DESCRIBE WITH TRANSLATIONS: all describe commands output multi-language | None — read-only, no grammar change | Low |
| **P2** | Grammar + AST: `translationMap` rule, `TranslatedText` node | None | Low |
| **P3** | Visitor: parse `{ lang: 'text' }` into AST | P2 | Low |
| **P4** | Writer `mergeTranslation` helper + multi-lang BSON write | P3 | **High** — architectural change to writer, must test against Studio Pro |
| **P5** | Widget translation indexing: extend catalog builder for widget-level translations | None (independent) | Low |

P1 first — highest user value, zero risk. P4 is the riskiest phase.

**Dropped**: SHOW TRANSLATIONS command — `SHOW LANGUAGES` + `QUAL005` + `SELECT ... FROM CATALOG.strings` already cover translation auditing.

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
