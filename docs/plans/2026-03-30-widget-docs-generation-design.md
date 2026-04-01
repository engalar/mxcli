# Widget Documentation Generation Design

## Problem

Pluggable widgets have BSON templates for creation, but Claude (and users) have no reference for what properties each widget supports, which are required, and how to use them in MDL. The `create-page.md` skill only documents built-in widgets.

## Solution

Auto-generate per-widget markdown documentation from MPK XML definitions + embedded templates, placed in `.claude/skills/widgets/` for Claude to reference during page creation.

## Architecture

### Output Structure

```
project/.claude/skills/widgets/
├── badge.md              # Auto-generated + manual supplements
├── accordion.md
├── switch.md
├── ...
└── _index.md             # Summary index of all available widgets
```

### Generation Pipeline

1. **Source 1: MPK XML** (`widgets/*.mpk`) — property names, types, required flags, categories, descriptions, enumeration values
2. **Source 2: Embedded templates** (`sdk/widgets/templates/`) — default values, widget ID mapping
3. **Merge** — combine into structured markdown per widget
4. **Preserve manual content** — `<!-- AUTO-GENERATED -->` / `<!-- END AUTO-GENERATED -->` markers; content outside markers is preserved on regeneration

### Generated Content Per Widget

- Widget ID and display name
- Property table: name, type, required, default, description
- Basic MDL example using CUSTOMWIDGET syntax
- Entity context requirement flag

### Integration Points

- **`mxcli init`** — generates widget docs alongside skill sync
- **`create-page.md` skill** — references `.claude/skills/widgets/` for pluggable widget docs
- **New command**: `mxcli generate widget-docs` — standalone regeneration

### Prerequisites

- Generic CUSTOMWIDGET executor support (WidgetType property + template loading)
- Grammar: `WIDGETTYPE COLON STRING_LITERAL` in `widgetPropertyV3`

## Files to Modify

| File | Change |
|------|--------|
| `mdl/grammar/MDLParser.g4` | Add `WIDGETTYPE COLON STRING_LITERAL` to `widgetPropertyV3` |
| `mdl/visitor/visitor_page_v3.go` | Store WidgetType in widget properties |
| `mdl/executor/cmd_pages_builder_v3.go` | Add `case "CUSTOMWIDGET":` using `GetTemplateFullBSON` |
| `cmd/mxcli/cmd_generate.go` (new) | Widget docs generation command |
| `cmd/mxcli/init.go` | Call widget docs generation during init |
| `.claude/skills/mendix/create-page.md` | Add QUOTED_IDENTIFIER docs + reference to widget docs |
| `reference/mendix-repl/templates/.claude/skills/` | Include widget docs template |
