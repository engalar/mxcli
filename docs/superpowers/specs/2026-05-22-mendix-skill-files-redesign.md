# Mendix Skill Files Redesign — Design Spec

**Date:** 2026-05-22  
**Status:** Approved  
**Scope:** 8 files (2 new + 6 updated) in `cmd/mxcli/skills/`

---

## Problem Statement

The 50+ Mendix skill files in `cmd/mxcli/skills/` are plain Markdown reference documents. They lack:

1. **YAML frontmatter** — no `name:` / `description:` fields, so the Claude Code `Skill` tool cannot reliably trigger them or surface them to the AI
2. **Trigger conditions** — no "When to Use" section, so the AI cannot decide when to consult the skill
3. **Executable checklists** — current content is descriptive prose, not actionable step sequences
4. **Known Limitations** — critical implementation gaps (e.g., DataGrid2 filter bar) are not documented, causing the AI to generate non-working code
5. **New content gaps** — filter widgets, NPE datasource patterns, data container design are missing or incomplete

---

## Solution

Rewrite 6 existing files and create 2 new files using the **Superpowers skill format**:

```
---
name: skill-name
description: Use when [trigger conditions] — keyword1 keyword2 keyword3
---

## When to Use This Skill
## Checklist
## Quick Syntax Reference
## Core Patterns
## Known Limitations
## Validation
```

All files live in `cmd/mxcli/skills/` and are synced to `.claude/skills/mendix/` via `make sync-skills`, then distributed to user Mendix projects via `mxcli init`.

---

## File Inventory

| # | File | Operation | Skill name | Primary new content |
|---|------|-----------|------------|---------------------|
| 1 | `datagrid2-filters.md` | **NEW** | `mendix:datagrid2-filters` | All 4 filter types, column-level vs grid-level status, filtertype variants, PLUGGABLEWIDGET escape hatch, implementation gap docs |
| 2 | `data-containers.md` | **NEW** | `mendix:page-data-design` | Data container type selection, datasource strategy, nesting patterns, NPE constraints, client vs server tradeoffs |
| 3 | `create-page.md` | **REWRITE** | `mendix:create-page` | DataGrid2 complete section (filters, column props, selection, paging, designprops); NPE datasource; page variables |
| 4 | `generate-domain-model.md` | **UPDATE** | `mendix:generate-domain-model` | Non-persistent entity complete section: declaration, page datasource constraints, microflow/nanoflow as bridge |
| 5 | `write-microflows.md` | **UPDATE** | `mendix:write-microflows` | DSO_ naming pattern (page datasource microflows), returning `list of NPE`, nanoflow vs microflow datasource guidance |
| 6 | `overview-pages.md` | **REWRITE** | `mendix:overview-pages` | Superpowers format; DataGrid2 CRUD pattern; controlbar with action buttons |
| 7 | `master-detail-pages.md` | **REWRITE** | `mendix:master-detail-pages` | Gallery filter bar (`attributes:[...]` multi-attribute search); DataGrid column filter + selection; Listen to widget deep dive |
| 8 | `alter-page.md` | **UPDATE** | `mendix:alter-page` | DataGrid2 column SET/DROP/INSERT; filter widget ALTER considerations |

**Execution order:** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 (each file cross-references earlier ones)

---

## Standard Template Structure

Every file follows this exact skeleton. Sections scale in depth to their complexity.

```markdown
---
name: skill-name
description: Use when [one-line trigger] — keyword1 keyword2 keyword3
---

## When to Use This Skill

Bullet list of concrete trigger scenarios. These are what the Skill tool
uses to decide whether to invoke this skill.

## Checklist

Ordered, executable steps. Each item is independently actionable:
- [ ] Step 1 (choose something)
- [ ] Step 2 (declare something)
- [ ] Step 3 (validate with mxcli check)

## Quick Syntax Reference

| Element | Syntax | Notes |
|---------|--------|-------|
| … | … | … |

## Core Patterns

### Pattern N: Pattern Name

One-sentence description. MDL code block follows.

```sql
-- example
```

## Known Limitations

- ✅ Feature X — works normally
- ⚠️ Feature Y — parsed but not forwarded to BSON (fix: use PLUGGABLEWIDGET)
- ❌ Feature Z — not implemented; use workaround W

## Validation

```bash
./bin/mxcli check script.mdl
./bin/mxcli check script.mdl -p app.mpr --references
```
```

---

## File 1: `datagrid2-filters.md` (NEW)

**Skill name:** `mendix:datagrid2-filters`  
**Description trigger keywords:** `textfilter numberfilter dropdownfilter datefilter column filter filtertype attributes datasource filter bar datagrid2`

### When to Use
- Adding filter widgets to DataGrid2 columns
- Configuring a filter bar above the grid
- Setting `filtertype:` default comparison
- Configuring advanced filter properties (placeholder, adjustable, multiselect)

### Implementation Status Table (in Known Limitations)

| Pattern | Status | Root cause / workaround |
|---------|--------|------------------------|
| Column-level filter (`column {} { textfilter }`) | ✅ Works | Auto-wired to column attribute |
| Gallery filter bar (`filter {} { textfilter }`) | ✅ Works | Routes through pluggable widget engine |
| DataGrid2 filter bar (`controlbar {} { textfilter }`) | ⚠️ Gap | `buildWidgetBSON()` produces DivContainer for unknown widget types; fix: add filter-widget case to `buildWidgetBSON()` and extend `FilterWidgetSpec` with `Attributes []string` + `FilterType string` |
| `filtertype:` property | ⚠️ Gap | Parsed by visitor, stored in AST, not forwarded to `BuildFilterWidgetGen()`; workaround: `PLUGGABLEWIDGET` with `defaultFilter:` |
| PLUGGABLEWIDGET advanced props | ✅ Works | Full widget property access |

### Auto-type Selection (column-level)

| Attribute type | Filter widget |
|---|---|
| String | textfilter |
| Integer / Long / Decimal / AutoNumber | numberfilter |
| DateTime | datefilter |
| Enumeration / Boolean | dropdownfilter |

### Core Patterns (5 patterns)

1. Column-level filter — 4 types (textfilter, numberfilter, datefilter, dropdownfilter)
2. filtertype variants — startsWith, equal, between, greaterEqual, etc.
3. Gallery filter bar — multi-attribute `attributes:[A, B]` search (OR within widget)
4. PLUGGABLEWIDGET — placeholder, adjustable, multiselect, clearable, applyAfterMs
5. Column filter + CRUD combination (controlbar for actions only)

---

## File 2: `data-containers.md` (NEW)

**Skill name:** `mendix:page-data-design`  
**Description trigger keywords:** `dataview datasource microflow nanoflow database selection association NPE non-persistent data container design page data flow`

### When to Use
- Deciding which widget to use for displaying data (DataView / DataGrid2 / ListView / Gallery)
- Choosing a datasource type (database vs microflow vs nanoflow vs page parameter vs selection)
- Designing pages that use non-persistent entities
- Nesting data containers (DataView + DataGrid, master-detail)
- Avoiding N+1 and unnecessary server round-trips

### Datasource Decision Tree

```
Is the object already available as a page parameter?
  YES → datasource: $paramName
  NO → Does the page need a list or a single object?
    SINGLE → Use microflow / nanoflow datasource
    LIST → Is it a persistent entity?
      YES → datasource: database [with WHERE/SORT] (preferred, avoids extra microflow)
           OR microflow if complex logic needed
      NO (NPE) → MUST use microflow or nanoflow datasource
               (database source not available for NPEs)
```

### Container Type Selection

| Widget | Holds | Best for |
|--------|-------|----------|
| DataView | 1 object | Edit forms, detail panels, any single-object display |
| DataGrid2 | List | Tabular overview with sorting, paging, filtering |
| ListView | List | Simple vertical list, custom row templates |
| Gallery | List | Card grid layout with selection |

### NPE Constraints (critical section)
- Cannot use `datasource: database` — NPEs have no DB table
- Cannot use `datasource: association` in isolation — only works if object is in memory via parent context
- `Keep Selection` in DataGrid2 does NOT work with NPEs (IDs change on refresh)
- Pages with NPE parameters cannot have a `url:` (no deeplink support)

### Nesting Patterns (3 patterns)
1. DataView (page param) → nested DataGrid (association datasource)
2. Gallery (selection: single) → DataView (datasource: selection galleryName)
3. DataView (microflow single) + DataGrid (database, related entity)

---

## File 3: `create-page.md` (REWRITE)

**Additions to existing content:**

### DataGrid2 section (new, comprehensive)
Subsections:
- Datasource types (database / microflow / nanoflow / association / $param / selection)
- Column properties reference table (all 13 properties)
- Column-level filter widgets (4 types, auto-selection table)
- Page variables for column visibility (`variables: { $showCol: boolean = 'true' }`)
- Selection modes (none / single / multiple)
- Paging modes (buttons / virtualScrolling / loadMore) + position + showPagingButtons
- Design properties (Compact / Striped / Hover / Borders)
- DynamicCellClass + DynamicRowClass expressions
- CRUD pattern (controlbar New + column Edit/Delete)

### NPE datasource section (new)
- `datasource: microflow Module.DSO_GetNPEList` pattern
- `datasource: nanoflow Module.NANO_GetNPEList` pattern  
- `datasource: $pageParam` for NPE passed in
- Cross-reference to `mendix:page-data-design`

---

## File 4: `generate-domain-model.md` (UPDATE)

### NPE section to add
```sql
-- Declaration syntax
create non-persistent entity Module.ResultDto (
  Field1: String(200),
  Field2: Boolean default false,
  Score:  Decimal
);
```

Key rules:
- No database table — never use `datasource: database` on pages
- No `create list of NPE` in microflows (CE0053 — blocked)
- Must be returned from a microflow/nanoflow to be shown in a list widget
- Use for: validation results, search results, aggregation views, DTO transfer objects

---

## File 5: `write-microflows.md` (UPDATE)

### DSO_ naming convention (new section)
DataSource Object microflows follow `DSO_` prefix and return a list or single object for page datasource use:

```sql
create microflow Module.DSO_GetActiveItems ()
  returns list of Module.Item
begin
  retrieve $Items from Module.Item where [IsActive = true()] sort by Name asc;
  return $Items;
end;
```

Rules:
- Never accept page-context variables as parameters (use retrieve instead)
- Must return the correct type (list vs single object matches what the widget expects)
- For NPE: build and return NPE objects directly, no commit needed

### Nanoflow vs microflow datasource guidance (new)
- Use nanoflow when: pure client-side calculation, no DB access needed, reduce server load
- Use microflow when: DB retrieve, complex logic, Java action calls, NPE built from server data

---

## File 6: `overview-pages.md` (REWRITE)

Superpowers format with these Checklist items:
- [ ] Create entity + enumeration (if not exists)
- [ ] Choose layout (Atlas_Default for overview, PopupLayout for edit form)
- [ ] Create edit page with page params (Mendix 11.0+)
- [ ] Create overview page: DataGrid2 + controlbar New button
- [ ] Add columns (attribute, caption, optional column filter)
- [ ] Add Actions column (Edit + Delete with `$currentObject`)
- [ ] Run `mxcli check` to validate
- [ ] Run `mx check` to validate BSON

Core patterns: Basic CRUD DataGrid, DataGrid + column filters, DataGrid + microflow datasource

---

## File 7: `master-detail-pages.md` (REWRITE)

Superpowers format. Core patterns:
1. Gallery (single select) + DataView (selection datasource) — side-by-side
2. Gallery (single select) + DataView (selection datasource) — same page, stacked
3. DataGrid + DataView (selection datasource)
4. Gallery filter bar (`attributes:[...]` multi-attribute search)
5. Nested: DataView (page param) → DataGrid (association datasource)

Known limitations:
- `datasource: selection widgetName` requires selection mode enabled on the source widget
- NPE lists cannot use `Keep Selection` — selection resets on filter/refresh

---

## File 8: `alter-page.md` (UPDATE)

Add section: **DataGrid2 column operations**
- `SET attribute = NewAttr ON dgName.colName`
- `DROP WIDGET dgName.colName`
- `INSERT AFTER dgName.colName { column newCol (...) }`

Add note: Filter widgets inside columns cannot currently be addressed by dotted notation for ALTER — replace the column instead.

---

## Validation Criteria

Each file is considered complete when:
1. YAML frontmatter has `name:` and `description:` with sufficient trigger keywords
2. Every code block passes `./bin/mxcli check` (no syntax errors)
3. `Known Limitations` accurately reflects current Go implementation state
4. `Checklist` items are independently executable (no vague verbs like "configure" without specifics)
5. Cross-references between files use the `mendix:skill-name` format

---

## Out of Scope

- Go code fixes (`buildWidgetBSON` filter bar gap, `filtertype:` forwarding) — tracked as separate issues
- Non-page skill files (`rest-client.md`, `java-actions.md`, etc.)
- VS Code extension changes
- Grammar changes for new filter properties

---

## Sync After Completion

```bash
make sync-skills        # copies cmd/mxcli/skills/ → .claude/skills/mendix/
make build              # runs sync as part of build
```

Test by invoking `Skill` tool with `mendix:datagrid2-filters` and `mendix:page-data-design` in the Mx2026AIDay project.
