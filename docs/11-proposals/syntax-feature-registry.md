# Proposal: Self-Describing Syntax Feature Registry

**Date**: 2026-04-10
**Status**: Draft
**Branch**: `research/recursive-help-discovery`

## Problem

mxcli's help system is flat — `mxcli syntax` has 26 topics, each a monolithic text file. There is no way to:

1. **Drill down** from a broad topic to a specific feature
2. **Get structured output** for LLM consumption
3. **Discover by business concept** (e.g., "who can handle a workflow task" → `TARGETING`)

The primary user is an LLM (Claude Code). LLMs can't interactively browse — they need structured data in one shot to match business questions to syntax.

Adding a new topic currently requires: creating a `.txt` file, adding a `case` in a switch statement, and updating the command's `Long` description. Three files for one concern.

## Design

### Data Model

Each discoverable syntax feature is a `SyntaxFeature`:

```go
type SyntaxFeature struct {
    Path       string   // Hierarchical key: "workflow.user-task.targeting"
    Summary    string   // One-line business description
    Keywords   []string // Synonyms for LLM matching
    Syntax     string   // MDL syntax pattern
    Example    string   // Working example
    MinVersion string   // Minimum Mendix version (optional)
    SeeAlso    []string // Related feature paths
}
```

**Path convention**: `domain.concept[.detail]`
- `workflow` → overview
- `workflow.user-task` → USER TASK activity type
- `workflow.user-task.targeting` → TARGETING clause

### Query Interface

**CLI (for LLMs and humans):**

```bash
# Full index — LLM gets everything in one call
mxcli syntax --json
# → [{"path":"workflow.user-task.targeting", "summary":"...", "keywords":[...]}, ...]

# Filter by topic prefix
mxcli syntax workflow --json

# Human-readable (default) — auto-generated from registry, grouped by hierarchy
mxcli syntax workflow

# Drill down to specific feature
mxcli syntax workflow user-task
```

**MDL REPL:**

```sql
HELP;                        -- Overview (unchanged)
HELP WORKFLOW;               -- Same as: mxcli syntax workflow
HELP WORKFLOW USER TASK;     -- Drill down
```

`HELP` reads from the same registry — no more hardcoded strings.

**Typical LLM flow:**

1. `mxcli syntax --json` → get full index (cache in context)
2. Match keywords/summary → find `workflow.user-task.targeting`
3. Write MDL directly, or `mxcli syntax workflow user-task --json` for details

One call solves 90% of discovery.

### File Organization

Separate package, registrations grouped by domain:

```
cmd/mxcli/
  syntax/
    registry.go                # Registry type + Register() + query methods
    format.go                  # Text/JSON output formatting
    features_workflow.go       # Workflow feature registrations
    features_security.go       # Security feature registrations
    features_domain_model.go   # Entity/association/enumeration
    features_microflow.go
    features_page.go
    features_navigation.go
    ...
```

Each `features_*.go` contains only `init()` + `Register()` calls (~50-150 lines).

### Adding a New Feature (single file change)

```go
// features_workflow.go
syntax.Register(SyntaxFeature{
    Path:       "workflow.boundary-event.timer",
    Summary:    "Attach a timer to a user task that triggers on timeout",
    Keywords:   []string{"timeout", "deadline", "SLA", "escalation"},
    Syntax:     "BOUNDARY TIMER ON <task-name> AFTER '<duration>' { <activities> }",
    Example:    "BOUNDARY TIMER ON ReviewTask AFTER 'P3D' {\n  CALL MICROFLOW Module.Escalate;\n}",
    MinVersion: "10.6.0",
    SeeAlso:    []string{"workflow.user-task"},
})
```

Automatically appears in: `--json` index, text output, `HELP` command, drill-down navigation.

No switch case, no .txt file, no manual topic list update.

### Migration Path

| Phase | Scope | Approach |
|-------|-------|----------|
| 1 | Registry core + `--json` | Build registry, register workflow + security topics, validate |
| 2 | Migrate remaining topics | Convert each `.txt` → `Register()` calls, delete `.txt` + switch case |
| 3 | MDL `HELP` with parameters | Parse `HELP <topic>` in grammar, read from registry |
| 4 | Cleanup | Remove `help_topics/` directory and old switch statement |

**Backward compatibility**: During migration, unregistered topics fall back to `.txt` files. `syntaxCmd` checks registry first, then `.txt`.

### Relationship to Skills

Skills (`.claude/skills/`) provide **guidance** — when to use, best practices, gotchas.
Registry provides **discovery** — what exists, how to write it.

Complementary, no overlap.

## PR Checklist Addition

```markdown
- [ ] New MDL syntax registered as `SyntaxFeature` (path, summary, keywords, example)
```
