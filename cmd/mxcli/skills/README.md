# Mendix MDL Skills

Skills for writing Mendix Definition Language (MDL) code correctly.

## Quick Reference (Cheat Sheets)

Start here for quick syntax lookups:

| Skill | Purpose | Use When |
|-------|---------|----------|
| [cheatsheet-variables.md](cheatsheet-variables.md) | Variable declaration syntax | Declaring variables, fixing declaration errors |
| [cheatsheet-errors.md](cheatsheet-errors.md) | Common errors and fixes | Debugging MDL syntax errors |

## Syntax Reference (By Document Type)

Detailed syntax for each MDL document type:

| Skill | Purpose | Use When |
|-------|---------|----------|
| [mdl-entities.md](mdl-entities.md) | Entity, attribute, association syntax | Creating domain models |
| [write-microflows.md](write-microflows.md) | Microflow syntax reference | Writing microflow logic |
| [write-oql-queries.md](write-oql-queries.md) | OQL query syntax | Creating VIEW entities |
| [write-workflows.md](write-workflows.md) | Workflow syntax reference | Creating and exploring workflows |
| [create-page.md](create-page.md) | Page and widget syntax | Creating pages |
| [fragments.md](fragments.md) | Fragment (reusable widget group) syntax | Reusing widget patterns across pages |

## Patterns (By Use Case)

Common implementation patterns:

| Skill | Purpose | Use When |
|-------|---------|----------|
| [patterns-crud.md](patterns-crud.md) | Create/Read/Update/Delete patterns | Building CRUD functionality |
| [patterns-data-processing.md](patterns-data-processing.md) | Loops, aggregates, batch processing | Processing lists of data |
| [validation-microflows.md](validation-microflows.md) | Validation feedback patterns | Building form validation |

## Integration Skills

External system integration:

| Skill | Purpose | Use When |
|-------|---------|----------|
| [database-connections.md](database-connections.md) | Mendix Database Connector | Connecting to Oracle, PostgreSQL, etc. via JDBC |
| [demo-data.md](demo-data.md) | Demo data & IMPORT | Seeding data, `IMPORT FROM` bulk import from external DB |
| [rest-client.md](rest-client.md) | REST API consumption | Calling external REST APIs |
| [java-actions.md](java-actions.md) | Custom Java actions | Extending with Java code |

## Page Patterns

Page-specific patterns:

| Skill | Purpose | Use When |
|-------|---------|----------|
| [overview-pages.md](overview-pages.md) | List/grid pages | Building overview screens |
| [master-detail-pages.md](master-detail-pages.md) | Master-detail layouts | Building selection-based UIs |
| [bulk-widget-updates.md](bulk-widget-updates.md) | Bulk widget property updates | Changing widget settings across pages |

## Specialized Skills

| Skill | Purpose | Use When |
|-------|---------|----------|
| [generate-domain-model.md](generate-domain-model.md) | Complete domain model generation | Generating full domain models |
| [debug-bson.md](debug-bson.md) | BSON debugging | Troubleshooting SDK issues |

---

## Skill Loading Guide

### For LLMs (Claude Code)

Load skills based on the task:

| User Request | Load These Skills |
|--------------|-------------------|
| "Create entity/domain model" | `mdl-entities.md` |
| "Write microflow" | `write-microflows.md`, `cheatsheet-variables.md` |
| "Create validation" | `validation-microflows.md`, `patterns-crud.md` |
| "Add CRUD operations" | `patterns-crud.md` |
| "Process list of items" | `patterns-data-processing.md` |
| "Merge/sync/reconcile data" | `patterns-data-processing.md` |
| "Delta update/import transform" | `patterns-data-processing.md`, `demo-data.md` |
| "Match/find/lookup in list" | `patterns-data-processing.md` |
| "Fix MDL error" | `cheatsheet-errors.md` |
| "Import data from database" | `demo-data.md` |
| "Seed/populate test data" | `demo-data.md` |
| "Update widget properties" | `bulk-widget-updates.md` |
| "Change widgets in bulk" | `bulk-widget-updates.md` |
| "Reuse widgets across pages" | `fragments.md` |
| "Define a fragment" | `fragments.md` |
| "Create workflow" | `write-workflows.md` |
| "Show/describe workflows" | `write-workflows.md` |
| "Approval process" | `write-workflows.md` |

### For Error Recovery

When encountering errors:

| Error Type | Load This Skill |
|------------|-----------------|
| Variable not declared | `cheatsheet-variables.md` |
| Entity type syntax | `cheatsheet-errors.md` |
| Association path error | `cheatsheet-errors.md` |
| Microflow structure error | `write-microflows.md` |
| OQL syntax error | `write-oql-queries.md` |

---

## Common Mistakes Summary

| Mistake | Frequency | Quick Fix |
|---------|-----------|-----------|
| SET without DECLARE | High | Add `DECLARE $var Type = value;` before SET |
| Missing AS for entity | High | Use `DECLARE $var AS Module.Entity;` |
| Unqualified association | Medium | Use `$var/Module.Assoc/Attr` |
| String enum comparison | Medium | Use `Module.Enum.Value` not `'string'` |
| Missing RETURN | Low | Add `RETURN $value;` at end |

---

## Integration Points

### REPL Integration
```bash
mdl> help variables    # Load cheatsheet-variables.md
mdl> help errors       # Load cheatsheet-errors.md
mdl> help crud         # Load patterns-crud.md
```

### Check Command
```bash
mxcli check script.mdl -p app.mpr --references
mxcli check script.mdl --format json
mxcli check script.mdl --format sarif
```

### Linter
```bash
mxcli lint -p app.mpr
mxcli lint -p app.mpr --format json
```

---

## Rule ID Naming Convention

Rule ID prefixes reflect the **input** needed to run the rule:

| Prefix | Meaning | Input | Tool |
|--------|---------|-------|------|
| `MDL` | MDL source file checks | `.mdl` file (no project) | `mxcli check` |
| `MPR` | Project model checks | `.mpr` project | `mxcli lint` (built-in Go) |
| `SEC` | Security | `.mpr` project | `mxcli lint` (built-in + Starlark) |
| `CONV` | Mendix conventions | `.mpr` project | `mxcli lint` (Starlark) |
| `QUAL` | Code quality | `.mpr` project | `mxcli lint` (Starlark) |
| `ARCH` | Architecture | `.mpr` project | `mxcli lint` (Starlark) |
| `DESIGN` | Design patterns | `.mpr` project | `mxcli lint` (Starlark) |

### MDL Rules (mxcli check)

| Rule | Check | Severity |
|------|-------|----------|
| MDL001 | Nested LOOP (O(N^2) anti-pattern) | Warning |
| MDL002 | Empty list variable used as loop source | Warning |
| MDL003 | Missing RETURN on non-void path | Error |
| MDL004 | RETURN value/type mismatch | Error |
| MDL005 | Variable declared in branch, used outside | Warning |
| MDL006 | Error handling type invalid inside loop | Warning |
| MDL007 | Empty VALIDATION FEEDBACK message | Warning |
| MDL010 | Enumeration value is reserved word | Error |
| MDL020 | Entity attribute conflicts with system name | Error |
| MDL030 | OQL syntax issues (paths, aliases, ORDER BY) | Error |
| MDL031 | OQL type mismatch vs declared attributes | Error |

### MPR Rules (mxcli lint)

| Rule | Check | Severity |
|------|-------|----------|
| MPR001 | PascalCase naming conventions | Warning |
| MPR002 | Empty microflows (no activities) | Warning |
| MPR003 | Domain model size (>15 persistent entities) | Warning |
| MPR004 | Empty validation feedback message (CE0091) | Warning |
| MPR005 | Unconfigured image widget source | Warning |
| MPR006 | Empty containers (runtime crash) | Warning |
| MPR007 | Navigation page without allowed role (CE0557) | Warning |

---

## Maintenance

When updating skills:
1. Keep each skill focused (100-200 lines target)
2. Include working code examples
3. Document common mistakes with fixes
4. Update this README when adding new skills
5. Test examples with actual MDL execution
