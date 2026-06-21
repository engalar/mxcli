# Proposal: Version-Aware Agent Support

**Status:** Draft
**Date:** 2026-04-02

## Problem Statement

Three use cases require mxcli to be version-aware at the MDL level:

1. **Generate**: An AI coding agent creates MDL for a Mendix project but doesn't know which features are available in the project's version. It writes `create view entity` for a 9.x project and gets a cryptic BSON error.

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
3. **Queryable at runtime** -- `show features` returns capabilities for the connected project's version.
4. **Fail-fast with actionable messages** -- Error before writing BSON, not after Studio Pro crashes.
5. **Incremental updates** -- Adding a new version's capabilities should be a data change, not a code change.
6. **Reuse existing infrastructure** -- Linter rules, skills, executor commands follow established patterns.

## Prior Art: Mendix Content API

The Mendix Content API (Marketplace) already solves a similar version-availability problem for add-on modules. Each Marketplace component release carries a minimum (and sometimes maximum) Studio Pro version, and Studio Pro resolves the highest compatible release at download time. The conceptual model is the same — mapping capabilities to version ranges — just applied to external packages rather than internal platform features.

This proposal adopts the same **`min_version` / `max_version`** bound format used by the Content API. A single version-comparison implementation can then handle both "is this platform feature available?" and "is this Marketplace module compatible?" — which matters for agents that reason about both in the same session (e.g., creating entities *and* importing a module).

The proposal diverges from the Content API in scope: we use a per-feature YAML registry rather than the Marketplace's full component → releases → packages data model. The alignment is specifically about version bound format and comparison semantics.

## Architecture

```
                    version Feature Registry
                    (modelsdk/version/*.yaml)
                           |
            +--------------+--------------+
            |              |              |
       show features   Executor       Linter
       (MDL command)   Pre-checks    rules (VER0xx)
            |              |              |
            v              v              v
       agent queries   error before   Upgrade
       capabilities    BSON write     recommendations
```

### Layer 1: Version Feature Registry

A structured YAML file per major version, embedded via `go:embed`:

```
modelsdk/version/
  mendix-9.yaml
  mendix-10.yaml
  mendix-11.yaml
```

Each file defines features with consistent `min_version` / `max_version` bounds (aligned with Mendix Content API conventions), syntax examples, deprecations, and upgrade hints:

```yaml
# modelsdk/version/mendix-10.yaml
major: 10
supported_range: "10.0.0..10.24.99"
lts_versions: ["10.24"]
mts_versions: ["10.6", "10.12", "10.18"]

features:
  # --- Domain Model ---
  domain_model:
    entities:
      min_version: "10.0.0"
      mdl: "create persistent entity Module.Name (...)"
    non_persistent_entities:
      min_version: "10.0.0"
      mdl: "create non-persistent entity Module.Name (...)"
    view_entities:
      min_version: "10.18.0"
      mdl: "create view entity Module.Name (...) as select ..."
      notes: "OQL stored inline on OqlViewEntitySource in 10.x"
    calculated_attributes:
      min_version: "10.0.0"
      mdl: "calculated by Module.Microflow"
    entity_generalization:
      min_version: "10.0.0"
      mdl: "extends Module.ParentEntity"
    alter_entity:
      min_version: "10.0.0"
      mdl: "alter entity Module.Name add attribute ..."

  # --- OQL Functions (used in VIEW ENTITY queries) ---
  oql_functions:
    basic_select:
      min_version: "10.18.0"
      mdl: "select, from, where, as, and, or, not"
    aggregate_functions:
      min_version: "10.18.0"
      mdl: "count, sum, avg, min, max, GROUP by"
    subqueries:
      min_version: "10.18.0"
      mdl: "Inline subqueries in select and where"
    join_types:
      min_version: "10.18.0"
      mdl: "inner join, left join, right join, full join"
    # New OQL functions added per minor release:
    # string_length: { min_version: "10.20.0" }
    # date_diff: { min_version: "10.21.0" }
    # (to be populated from release notes)

  # --- Microflows ---
  microflows:
    basic:
      min_version: "10.0.0"
      mdl: "create microflow Module.Name (...) begin ... end"
    show_page_with_params:
      min_version: "11.0.0"
      workaround:
        description: "Pass data via a non-persistent entity or microflow parameter"
        max_version: "10.99.99"
    send_rest_request:
      min_version: "10.1.0"
      mdl: "send rest request ..."
    send_rest_query_params:
      min_version: "11.0.0"
      notes: "query parameters in rest requests"
    execute_database_query:
      min_version: "10.6.0"
      mdl: "execute database query ..."
    execute_database_query_runtime_connection:
      min_version: "11.0.0"
      notes: "runtime connection override for database queries"
    loop_in_branch:
      min_version: "10.0.0"
      notes: "loop inside if/else branches"

  # --- Pages ---
  pages:
    basic:
      min_version: "10.0.0"
      mdl: "create page Module.Name (...) { ... }"
    page_parameters:
      min_version: "11.0.0"
      workaround:
        description: "use non-persistent entity or microflow parameter"
        max_version: "10.99.99"
    page_variables:
      min_version: "11.0.0"
    pluggable_widgets:
      min_version: "10.0.0"
      widgets: [combobox, DataGrid2, gallery, image, textfilter, numberfilter, datefilter, dropdownfilter]
      notes: "widget templates are version-specific; MPK augmentation handles drift"
    design_properties_v3:
      min_version: "11.0.0"
      notes: "Atlas v3 design properties (Card style, Disable row wrap)"
    alter_page:
      min_version: "10.0.0"
      mdl: "alter page Module.Name set/insert/drop/replace ..."

  # --- Security ---
  security:
    module_roles:
      min_version: "10.0.0"
    user_roles:
      min_version: "10.0.0"
    entity_access:
      min_version: "10.0.0"
    demo_users:
      min_version: "10.0.0"

  # --- Integration ---
  integration:
    rest_client_basic:
      min_version: "10.1.0"
      mdl: "create rest client Module.Name ..."
    rest_client_query_params:
      min_version: "11.0.0"
    rest_client_headers:
      min_version: "10.4.0"
    database_connector_basic:
      min_version: "10.6.0"
      mdl: "create database connection Module.Name ..."
    database_connector_execute:
      min_version: "10.6.0"
      mdl: "execute database query in microflows"
      notes: "full BSON format requires 11.0+"
    business_events:
      min_version: "10.0.0"
      mdl: "create business event service Module.Name ..."
    odata_client:
      min_version: "10.0.0"

  # --- Workflows ---
  workflows:
    basic:
      min_version: "9.0.0"
      mdl: "create workflow Module.Name ..."
    parallel_split:
      min_version: "9.0.0"
    user_task:
      min_version: "9.0.0"

  # --- Navigation ---
  navigation:
    profiles:
      min_version: "10.0.0"
    menu_items:
      min_version: "10.0.0"
    home_pages:
      min_version: "10.0.0"

deprecated:
  - id: "DEP001"
    pattern: "Persistable: false on view entities"
    replaced_by: "Persistable: true (auto-set)"
    since: "10.18.0"
    severity: "info"

upgrade_opportunities:
  from_10_to_11:
    - feature: "page_parameters"
      description: "replace non-persistent entity parameter passing with direct page parameters"
      effort: "low"
    - feature: "design_properties_v3"
      description: "Atlas v3 design properties available for richer styling"
      effort: "low"
    - feature: "rest_client_query_params"
      description: "rest clients can now define query parameters directly"
      effort: "low"
    - feature: "database_connector_runtime_connection"
      description: "database queries can override connection at runtime"
      effort: "low"
    - feature: "association_storage"
      description: "New association storage format (automatic on project upgrade)"
      effort: "none"
```

### Layer 2: MDL Commands

#### `show features`

Lists all features available for the connected project's Mendix version, or for a specified version without a project connection:

```sql
-- When connected to a project (uses project's Mendix version)
show features;

-- Without a project connection (query any version)
show features for version 10.24;

-- Filter by area
show features in integration;

-- Show only features available in the project's version
show features where available = true;
```

Output:
```
| Feature                | Available | since  | Notes                                    |
|------------------------|-----------|--------|------------------------------------------|
| persistent entities    | Yes       | 10.0   |                                          |
| view entities          | Yes       | 10.18  | OQL stored inline on source object       |
| page parameters        | No        | 11.0+  | use non-persistent entity workaround     |
| Pluggable widgets      | Yes       | 10.0   | combobox, DataGrid2, gallery, image      |
| design properties v3   | No        | 11.0+  | Atlas v3 required                        |
| rest client            | Partial   | 10.1   | query parameters require 11.0+           |
| database connector     | Partial   | 10.6   | execute database query requires 11.0+    |
| business events        | Yes       | 10.0   |                                          |
| workflows              | Yes       | 9.0    |                                          |
```

#### `show features added since <version>`

Shows what becomes available when upgrading:

```sql
show features added since 10.24;
```

Output:
```
| Feature              | Available in | description                              | Effort |
|----------------------|-------------|------------------------------------------|--------|
| page parameters      | 11.0        | Direct page parameter passing            | Low    |
| design properties v3 | 11.0        | Atlas v3 Card style, Disable row wrap    | Low    |
| rest query params    | 11.0        | query parameter support in rest clients  | Low    |
| Portable app format  | 11.6        | New deployment format                    | none   |
```

#### `show UPGRADE OPPORTUNITIES`

When connected, analyzes the current project for patterns that could benefit from the project's own version or a target version:

```sql
-- What can I improve using my current version's capabilities?
show UPGRADE OPPORTUNITIES;

-- What would upgrading to 11.6 enable?
show UPGRADE OPPORTUNITIES to 11.6;
```

Output:
```
| Opportunity                | Target  | description                                          | Effort |
|----------------------------|---------|------------------------------------------------------|--------|
| page parameters            | 11.0    | 12 pages use NP entity workaround for parameter passing | Low  |
| design properties v3       | 11.0    | 8 containers could use Card style / row wrap         | Low    |
| rest query parameters      | 11.0    | 3 rest clients build query strings manually          | Low    |
| database runtime connection| 11.0    | 2 microflows hardcode DB connection                  | Low    |
```

#### `show deprecated`

Lists deprecated patterns in the current project:

```sql
show deprecated;
```

### Layer 3: Executor Pre-Checks

Before writing BSON, the executor checks version compatibility and produces actionable errors:

```go
// in cmd_entities.go, before creating a view entity:
if s.IsViewEntity {
    pv := e.reader.ProjectVersion()
    if !pv.IsAtLeast(10, 18) {
        return fmt.Errorf(
            "create view entity requires Mendix 10.18+ (project is %s)\n"+
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
[VER001] create view entity requires Mendix 10.18+ (project is 10.12.0)
  at line 42 in script.mdl
  hint: upgrade to 10.18+ or use a microflow data source
```

**VER003** runs during `mxcli lint --upgrade-hints`:
```
[VER003] page MyModule.EditCustomer uses non-persistent entity for parameter passing
  This pattern can be replaced with page parameters in Mendix 11.0+
  effort: low
```

### Layer 5: Skills (AI Agent Guidance)

One skill file: `.claude/skills/version-awareness.md`

```markdown
# version Awareness

## before Generating MDL

Always check the project's Mendix version before writing MDL:

    show status;           -- shows connected project version
    show features;         -- shows available features

## version-Conditional Patterns

if a feature is not available, use the documented workaround:

    show features where name = 'page_parameters';
    -- If not available, use non-persistent entity pattern instead

## Upgrade workflow

when migrating to a newer version:

    show features added since 10.24;    -- what's new
    show deprecated;                     -- what to update
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

1. Create `modelsdk/version/` package with YAML loader and `go:embed`
2. Create YAML files for Mendix 9, 10, 11 (initial feature set from existing knowledge)
3. Implement `show features` command in executor
4. Implement `show features added since <version>` variant
5. Wire into AST/grammar: add `features` keyword to MDLParser.g4

**Deliverable**: Agent can query `show features` and get machine-readable output.

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
2. Implement `show deprecated` command
3. Implement `show features added since` with effort estimates
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
user/agent Layer     This Proposal        "What can I do?"
                          |
MDL Layer            Executor pre-checks  "Will this work?"
                          |
BSON Layer           schema Registry      "How do I serialize this?"
```

## Design Decisions

1. **YAML as source of truth.** Human-readable, easy to edit, reviewed in PRs. Compiled to embedded Go structs at build time via `go:embed` + YAML parser. Version bounds use `min_version` / `max_version` semver notation consistent with the Mendix Content API (see Prior Art section).

2. **Consistent version bound format.** The registry uses `min_version` / `max_version` fields with semver-style notation (e.g., `"10.18.0"`), aligned with the Mendix Content API's versioning conventions. This replaces an earlier draft that mixed `introduced`, `available_in`, and `null` — which was ad-hoc and required special-case handling. With consistent bounds, a single `IsInRange(projectVersion, min, max)` comparison handles all checks. The `max_version` field is optional and defaults to unbounded (feature available in all later versions). This format also enables future unification: the same version-comparison logic can evaluate both platform feature availability and Marketplace module compatibility.

3. **Fine-grained features with feature-level bounds and optional property overrides.** Mendix adds capabilities per minor release across all areas. The registry tracks at the **per-capability** level, not just per-concept. Each capability has its own `min_version`. Where a capability gains sub-features in later releases, property-level overrides can be added without changing the parent entry:

    ```yaml
    oql_functions:
      string_length: { min_version: "10.20.0" }
      date_diff: { min_version: "10.21.0" }
      coalesce: { min_version: "10.14.0" }
    database_connector:
      basic_query: { min_version: "10.6.0" }
      parameterized_query: { min_version: "10.12.0" }
      runtime_connection: { min_version: "11.0.0" }
    ```

    This mirrors how the Marketplace handles it — a module has one compatibility range, but release notes call out property-level changes.

4. **SHOW FEATURES works without a project.** `show features for version 10.24` queries the registry directly without an MPR connection. When connected, it defaults to the project's version. Useful for upgrade planning: `show features added since 10.24` shows what upgrading to the latest would gain; `show features for version 11.6` shows the full capability set of a target version. When connected, also supports: `show UPGRADE OPPORTUNITIES` to identify patterns in the current project that could benefit from the project's own version capabilities (e.g., detecting workarounds that are no longer needed).

5. **Minor version granularity by default.** Metamodel changes happen at the minor release level. Patch-level overrides supported where needed but expected to be rare. All `min_version` values use three-part semver (`major.minor.patch`) for consistency, with `.0` patch for minor-release features.

6. **Non-interactive upgrade advisor for now.** `mxcli lint --upgrade-hints --target-version 11.6` outputs a report. Interactive `mxcli upgrade` can be added later as the tooling matures.
