# Proposal: Version-Aware Agent Support

**Status:** Draft
**Date:** 2026-04-02

## Problem Statement

Three use cases require mxcli to be version-aware at the MDL level:

1. **Generate**: An AI coding agent creates MDL for a Mendix project but doesn't know which features are available in the project's version. It writes `CREATE VIEW ENTITY` for a 9.x project and gets a cryptic BSON error.

2. **Validate**: A user runs an MDL script that uses 11.0+ syntax on a 10.24 project. The script executes, writes corrupt BSON, and the error only surfaces when Studio Pro tries to open the project.

3. **Upgrade**: A customer wants to migrate from 10.24 to 11.6 and leverage new capabilities. They need to know what patterns can be modernized and what new features become available.

Today, version knowledge is scattered across:
- `sdk/mpr/version/version.go` (6 hardcoded features)
- Comments in MDL example scripts (`-- @version: 11.0+`)
- CLAUDE.md documentation (prose, not machine-readable)
- Implicit knowledge in the BSON writers (version-conditional serialization)

There is no way for an agent or user to **query** what's available, no **pre-flight check** before writing, and no **upgrade advisor** for migration.

## Design Principles

1. **Single source of truth** -- One structured data format defines version capabilities. Everything else reads from it.
2. **Machine-readable first** -- An AI agent should be able to query capabilities, not read prose.
3. **Queryable at runtime** -- `SHOW FEATURES` returns capabilities for the connected project's version.
4. **Fail-fast with actionable messages** -- Error before writing BSON, not after Studio Pro crashes.
5. **Incremental updates** -- Adding a new version's capabilities should be a data change, not a code change.
6. **Reuse existing infrastructure** -- Linter rules, skills, executor commands follow established patterns.

## Architecture

```
                    Version Feature Registry
                    (sdk/versions/*.yaml)
                           |
            +--------------+--------------+
            |              |              |
       SHOW FEATURES   Executor       Linter
       (MDL command)   Pre-checks    Rules (VER0xx)
            |              |              |
            v              v              v
       Agent queries   Error before   Upgrade
       capabilities    BSON write     recommendations
```

### Layer 1: Version Feature Registry

A structured YAML file per major version, embedded via `go:embed`:

```
sdk/versions/
  mendix-9.yaml
  mendix-10.yaml
  mendix-11.yaml
```

Each file defines features, their introduction version, syntax, deprecations, and upgrade hints:

```yaml
# sdk/versions/mendix-10.yaml
major: 10
supported_range: "10.0..10.24"
lts_versions: ["10.24"]
mts_versions: ["10.6", "10.12", "10.18"]

features:
  domain_model:
    entities:
      introduced: "10.0"
      mdl: "CREATE PERSISTENT ENTITY Module.Name (...)"
    view_entities:
      introduced: "10.18"
      mdl: "CREATE VIEW ENTITY Module.Name (...) AS SELECT ..."
      notes: "OQL stored inline on OqlViewEntitySource (Oql field)"
    calculated_attributes:
      introduced: "10.0"
      mdl: "CALCULATED BY Module.Microflow"
    entity_generalization:
      introduced: "10.0"
      mdl: "EXTENDS Module.ParentEntity"

  microflows:
    basic:
      introduced: "10.0"
    show_page_with_params:
      introduced: null
      available_in: "11.0+"
      workaround: "Pass data via a non-persistent entity or microflow parameter"
    send_rest_request:
      introduced: "10.1"
      notes: "Query parameters require 11.0+"

  pages:
    page_parameters:
      introduced: null
      available_in: "11.0+"
    pluggable_widgets:
      introduced: "10.0"
      widgets:
        - ComboBox
        - DataGrid2
        - Gallery
        - Image
      notes: "Widget templates are version-specific; MPK augmentation handles drift"
    design_properties_v3:
      introduced: null
      available_in: "11.0+"
      notes: "Atlas v3 design properties (Card style, Disable row wrap)"

  security:
    module_roles:
      introduced: "10.0"
    demo_users:
      introduced: "10.0"

  integration:
    rest_client:
      introduced: "10.1"
      notes: "Full BSON format requires 11.0+"
    database_connector:
      introduced: "10.6"
      notes: "EXECUTE DATABASE QUERY BSON format requires 11.0+"
    business_events:
      introduced: "10.0"

  workflows:
    basic:
      introduced: null
      available_in: "9.0+"

deprecated:
  - id: "DEP001"
    pattern: "Persistable: false on view entities"
    replaced_by: "Persistable: true (auto-set)"
    since: "10.18"
    severity: "info"

upgrade_opportunities:
  from_10_to_11:
    - feature: "page_parameters"
      description: "Replace non-persistent entity parameter passing with direct page parameters"
      effort: "low"
    - feature: "design_properties_v3"
      description: "Atlas v3 design properties available for richer styling"
      effort: "low"
    - feature: "association_storage"
      description: "New association storage format (automatic on project upgrade)"
      effort: "none"
```

### Layer 2: MDL Commands

#### `SHOW FEATURES`

Lists all features available for the connected project's Mendix version:

```sql
SHOW FEATURES;
```

Output:
```
| Feature                | Available | Since  | Notes                                    |
|------------------------|-----------|--------|------------------------------------------|
| Persistent entities    | Yes       | 10.0   |                                          |
| View entities          | Yes       | 10.18  | OQL stored inline on source object       |
| Page parameters        | No        | 11.0+  | Use non-persistent entity workaround     |
| Pluggable widgets      | Yes       | 10.0   | ComboBox, DataGrid2, Gallery, Image      |
| Design properties v3   | No        | 11.0+  | Atlas v3 required                        |
| REST client            | Partial   | 10.1   | Query parameters require 11.0+           |
| Database connector     | Partial   | 10.6   | EXECUTE DATABASE QUERY requires 11.0+    |
| Business events        | Yes       | 10.0   |                                          |
| Workflows              | Yes       | 9.0    |                                          |
```

#### `SHOW FEATURES ADDED SINCE <version>`

Shows what becomes available when upgrading:

```sql
SHOW FEATURES ADDED SINCE 10.24;
```

Output:
```
| Feature              | Available In | Description                              | Effort |
|----------------------|-------------|------------------------------------------|--------|
| Page parameters      | 11.0        | Direct page parameter passing            | Low    |
| Design properties v3 | 11.0        | Atlas v3 Card style, Disable row wrap    | Low    |
| REST query params    | 11.0        | Query parameter support in REST clients  | Low    |
| Portable app format  | 11.6        | New deployment format                    | None   |
```

#### `SHOW DEPRECATED`

Lists deprecated patterns in the current project:

```sql
SHOW DEPRECATED;
```

### Layer 3: Executor Pre-Checks

Before writing BSON, the executor checks version compatibility and produces actionable errors:

```go
// In cmd_entities.go, before creating a view entity:
if s.IsViewEntity {
    pv := e.reader.ProjectVersion()
    if !pv.IsAtLeast(10, 18) {
        return fmt.Errorf(
            "CREATE VIEW ENTITY requires Mendix 10.18+ (project is %s)\n"+
            "  hint: upgrade your project or use a regular entity with a microflow data source",
            pv.ProductVersion,
        )
    }
}
```

This pattern already exists informally in the codebase (version-conditional BSON writing). The proposal formalizes it with:

1. A `CheckFeature(feature, version)` function that returns a user-friendly error
2. Pre-checks at the start of each executor command
3. Consistent error format with hints

### Layer 4: Linter Rules (VER prefix)

New linter rule category `VER` for version-related checks:

| Rule | Name | Description |
|------|------|-------------|
| VER001 | UnsupportedFeature | Feature used that's not available in project version |
| VER002 | DeprecatedPattern | Deprecated pattern that has a modern replacement |
| VER003 | UpgradeOpportunity | Pattern that can be simplified on a newer version |

**VER001** runs during `mxcli check` and `mxcli lint`:
```
[VER001] CREATE VIEW ENTITY requires Mendix 10.18+ (project is 10.12.0)
  at line 42 in script.mdl
  hint: upgrade to 10.18+ or use a microflow data source
```

**VER003** runs during `mxcli lint --upgrade-hints`:
```
[VER003] Page MyModule.EditCustomer uses non-persistent entity for parameter passing
  This pattern can be replaced with page parameters in Mendix 11.0+
  effort: low
```

### Layer 5: Skills (AI Agent Guidance)

One skill file: `.claude/skills/version-awareness.md`

```markdown
# Version Awareness

## Before Generating MDL

Always check the project's Mendix version before writing MDL:

    SHOW STATUS;           -- shows connected project version
    SHOW FEATURES;         -- shows available features

## Version-Conditional Patterns

If a feature is not available, use the documented workaround:

    SHOW FEATURES WHERE name = 'page_parameters';
    -- If not available, use non-persistent entity pattern instead

## Upgrade Workflow

When migrating to a newer version:

    SHOW FEATURES ADDED SINCE 10.24;    -- what's new
    SHOW DEPRECATED;                     -- what to update
    mxcli lint --upgrade-hints -p app.mpr  -- automated suggestions
```

This skill is small and stable -- it teaches the agent to **query** mxcli rather than embedding version knowledge in the skill itself. The version data lives in the registry.

### Layer 6: Keeping Data Current

The version registry needs updates when Mendix releases new versions. Proposed pipeline:

1. **Automated**: `mxcli diff-schemas 11.5 11.6` compares reflection data between versions, outputs added/removed types and properties as a diff report.

2. **Semi-automated**: An agent reads the diff report + Mendix release notes and proposes updates to the YAML registry. Human reviews and merges.

3. **On-demand**: `mxcli update-features` downloads the latest registry from a central source (GitHub release asset), similar to how `mxcli setup mxbuild` downloads tooling.

4. **Community**: The `-- @version:` directives in MDL test scripts serve as executable documentation. If a test fails on a version, the directive gets updated — and that update feeds back into the registry.

## Implementation Plan

### Phase 1: Version Feature Registry + SHOW FEATURES (foundation)

1. Create `sdk/versions/` package with YAML loader and `go:embed`
2. Create YAML files for Mendix 9, 10, 11 (initial feature set from existing knowledge)
3. Implement `SHOW FEATURES` command in executor
4. Implement `SHOW FEATURES ADDED SINCE <version>` variant
5. Wire into AST/grammar: add `FEATURES` keyword to MDLParser.g4

**Deliverable**: Agent can query `SHOW FEATURES` and get machine-readable output.

### Phase 2: Executor Pre-Checks (fail-fast)

1. Add `CheckFeatureAvailable(feature string)` method to Executor
2. Add version checks to CREATE VIEW ENTITY, CREATE REST CLIENT, CREATE PAGE (with Params), EXECUTE DATABASE QUERY
3. Produce error messages with version requirement, current version, and workaround hint
4. Test: run MDL scripts with version-gated features on older projects

**Deliverable**: Unsupported features fail immediately with actionable error instead of corrupting BSON.

### Phase 3: Linter Rules (VER category)

1. Implement VER001 (UnsupportedFeature) -- reads from version registry
2. Implement VER002 (DeprecatedPattern) -- reads deprecated list from registry
3. Wire into `mxcli lint` and `mxcli check`
4. Add SARIF output support for CI integration

**Deliverable**: `mxcli lint -p app.mpr` reports version issues.

### Phase 4: Upgrade Advisor

1. Implement VER003 (UpgradeOpportunity) linter rule
2. Implement `SHOW DEPRECATED` command
3. Implement `SHOW FEATURES ADDED SINCE` with effort estimates
4. Implement `mxcli lint --upgrade-hints --target-version 11.6`

**Deliverable**: Migration planning from any version to any newer version.

### Phase 5: Skills + Agent Integration

1. Create `.claude/skills/version-awareness.md`
2. Update `.claude/skills/check-syntax.md` to include version pre-check
3. Update `mxcli init` to include version-awareness skill in project setup
4. Test: AI agent generates valid MDL for both 10.24 and 11.6 projects

**Deliverable**: AI agents automatically adapt to project version.

### Phase 6: Automated Registry Updates

1. Implement `mxcli diff-schemas <from> <to>` using reflection data
2. Create agent workflow: diff-schemas output + release notes -> YAML update PR
3. Implement `mxcli update-features` for on-demand downloads
4. Add to nightly CI: verify registry matches reflection data

**Deliverable**: Registry stays current with minimal manual effort.

## Relationship to BSON Schema Registry Proposal

This proposal complements `BSON_SCHEMA_REGISTRY_PROPOSAL.md`:

- **Schema Registry** handles **structural** version differences (field names, defaults, encoding) at the BSON level
- **This proposal** handles **feature-level** version differences (what MDL commands are available) at the user/agent level

The version feature registry (YAML) is simpler and more immediately useful than the full schema registry. It can be built first and later integrated with the schema registry as that matures.

```
User/Agent Layer     This Proposal        "What can I do?"
                          |
MDL Layer            Executor pre-checks  "Will this work?"
                          |
BSON Layer           Schema Registry      "How do I serialize this?"
```

## Open Questions

1. **YAML vs JSON for registry?** YAML is more readable for humans editing it; JSON is easier to parse. Could use YAML as source, compile to embedded JSON at build time.

2. **Granularity of features?** Per-statement (`CREATE VIEW ENTITY`), per-property (`Params:` on pages), or per-concept (`view_entities`)? Probably per-concept with per-property notes.

3. **Should SHOW FEATURES work without a connected project?** Could accept `SHOW FEATURES FOR VERSION 10.24` without needing an MPR file. Useful for planning.

4. **How to handle patch-level differences?** Most changes are at the minor level, but some patch releases introduce fixes. Use minor as the default, with patch-level overrides where needed.

5. **Should the upgrade advisor be interactive?** E.g., `mxcli upgrade --from 10.24 --to 11.6 --dry-run` that shows a migration plan and optionally applies changes.
