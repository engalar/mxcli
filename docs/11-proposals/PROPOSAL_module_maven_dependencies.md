# Proposal: Module Maven Dependencies (JAR Dependencies)

**Status:** Draft  
**Date:** 2026-05-10  
**Supersedes:** `show-describe-module-settings.md` (partial — that proposal covers the broader
`ModuleSettings` document; this proposal focuses specifically on the Maven/JAR dependency subset
and adds the full read/write lifecycle)

---

## Problem Statement

Mendix modules can declare Maven dependencies that are automatically resolved at build time
(`Project > App Settings > Manage App Store modules > Module Settings`). Today, `mxcli`
creates new modules with a hardcoded empty `JarDependencies` array in `serializeModuleSettings`
and provides no way to read, add, update, or remove these dependencies from MDL.

Affected users:
- Teams that maintain in-house Mendix modules with Java actions backed by third-party JARs
- Platform engineers who want to audit or standardise JAR versions across modules
- CI pipelines that need to reproduce the exact dependency set from source control

---

## BSON Structure (verified with `mx dump-mpr` on test5.mpr, Mendix 11.9)

`Projects$ModuleSettings` is a **model unit** (one `.mxunit` file per module). It is a child
of `Projects$ModuleImpl` via `ContainerProperty = "moduleSettings"`. The writer already creates
this unit for every new module; it just writes an empty `JarDependencies` array.

```
Projects$ModuleSettings
  $ID                  UUID
  $Type                "Projects$ModuleSettings"   ← no alias, storage name = qualified name
  $ContainerID         <ModuleImpl.$ID>
  $ContainerProperty   "moduleSettings"
  exportLevel          "Source" | "Protected"
  protectedModuleType  "AddOn" | "Solution"
  version              "1.0.0"
  basedOnVersion       ""
  extensionName        ""
  solutionIdentifier   ""
  jarDependencies      []   ← listType=2 (PART, embedded docs)
    Projects$JarDependency
      $ID          UUID (generated)
      $Type        "Projects$JarDependency"
      groupId      "org.duckdb"
      artifactId   "duckdb_jdbc"
      version      "1.5.2.1"
      isIncluded   true
      exclusions   []   ← listType=2, added in Mendix 10.20
        Projects$JarDependencyExclusion
          $ID        UUID
          $Type      "Projects$JarDependencyExclusion"
          groupId    "com.example"
          artifactId "some-artifact"
```

**Version notes:**
- `JarDependency` (without `exclusions`) present from **Mendix 10.0**.
- `exclusions` field + `JarDependencyExclusion` type added in **Mendix 10.20**.
- No version gate needed for basic add/remove; `exclusions` clause requires `checkFeature`.

---

## Proposed MDL Syntax

The Maven coordinate `groupId:artifactId:version` is the natural identifier — identical to how
Maven, Gradle, and every dependency tool express a dependency. MDL uses the same notation.

### LIST JAR DEPENDENCIES

```sql
list jar dependencies in MyModule;
list jar dependencies;               -- all modules
```

Tabular output:

```
Module        Group                      Artifact             Version      Included
------------  -------------------------  -------------------  -----------  --------
JaTest        org.apache.camel           camel-core           4.20.0       yes
JaTest        org.duckdb                 duckdb_jdbc          1.5.2.1      yes
SqlConnector  com.zaxxer                 HikariCP             5.0.1        yes
SqlConnector  org.postgresql             postgresql           42.4.4       yes
(4 jar dependencies)
```

### DESCRIBE JAR DEPENDENCY

```sql
describe jar dependency JaTest org.duckdb:duckdb_jdbc;
```

Output (roundtrippable MDL):

```sql
alter module JaTest add jar dependency (
  group    = 'org.duckdb',
  artifact = 'duckdb_jdbc',
  version  = '1.5.2.1',
  included = true
);
/
```

### ALTER MODULE — add jar dependency

```sql
alter module JaTest add jar dependency (
  group    = 'org.apache.camel',
  artifact = 'camel-core',
  version  = '4.20.0'
);
```

`included` defaults to `true` when omitted. The dependency is appended to the existing list;
if the same `group:artifact` pair already exists, the command fails with a clear error
(use `set` to update an existing version — see below).

### ALTER MODULE — set jar dependency version (update existing)

```sql
alter module JaTest set jar dependency org.apache.camel:camel-core version '4.21.0';
```

Updates the `version` field of an existing dependency. Error if the coordinate does not exist.

### ALTER MODULE — drop jar dependency

```sql
alter module JaTest drop jar dependency org.apache.camel:camel-core;
```

Removes the dependency with that group:artifact. Error if not found.

### ALTER MODULE — set included flag

```sql
alter module JaTest set jar dependency org.duckdb:duckdb_jdbc included false;
```

Allows toggling the `isIncluded` flag without removing the entry (matches Studio Pro behaviour
where unchecking a dependency keeps it in the list).

### ALTER MODULE — add exclusion (Mendix 10.20+)

```sql
alter module JaTest set jar dependency org.apache.camel:camel-core
  add exclusion commons-logging:commons-logging;
```

```sql
alter module JaTest set jar dependency org.apache.camel:camel-core
  drop exclusion commons-logging:commons-logging;
```

### Batch — add multiple dependencies at once

The `add jar dependency` clause can be repeated in a single `alter`:

```sql
alter module JaTest
  add jar dependency (group = 'org.slf4j',     artifact = 'slf4j-api',  version = '2.0.7')
  add jar dependency (group = 'com.google.code.gson', artifact = 'gson', version = '2.13.1');
```

### CREATE MODULE with dependencies (inline, optional)

For completeness, `create module` should accept an optional `jar dependencies` block so that
a module's dependencies can be declared in a single statement:

```sql
create module JaTest (
  jar dependency (group = 'org.duckdb',        artifact = 'duckdb_jdbc', version = '1.5.2.1'),
  jar dependency (group = 'org.apache.camel',  artifact = 'camel-core',  version = '4.20.0')
);
```

This is a nice-to-have; the `alter module` path covers all real-world workflows.

---

## DESCRIBE / SHOW MODULE SETTINGS (broader context)

The existing `show-describe-module-settings.md` proposal covers the other fields on
`Projects$ModuleSettings` (`exportLevel`, `protectedModuleType`, `version`, `basedOnVersion`)
as well as the `ModuleImpl` fields (`fromAppStore`, `appStoreGuid`, `appStoreVersion`,
`isThemeModule`). Those are read-only metadata useful for auditing App Store modules.

Recommendation: implement the `show module settings` / `describe module settings` commands
from that proposal in the same PR as this one, since they share the same backend read path.
The write path (`alter module ... set version ...` etc.) is lower priority and can be a
follow-up.

---

## Implementation Plan

### Files to create / modify

| File | Change |
|------|--------|
| `mdl/grammar/MDLLexer.g4` | Add tokens: `JAR`, `DEPENDENCY`, `DEPENDENCIES`, `GROUP`, `ARTIFACT`, `EXCLUSION`, `INCLUDED` |
| `mdl/grammar/MDLParser.g4` | Add `listJarDependencies`, `describeJarDependency`, `alterModuleJarDep` rules |
| `mdl/ast/ast.go` | `AlterModuleJarDepStmt`, `JarDependencySpec`, `ExclusionSpec` nodes |
| `mdl/visitor/visitor.go` | Bridge parse tree → AST for the new rules |
| `mdl/executor/cmd_modules.go` | `execListJarDependencies`, `describeJarDependency`, `execAlterModuleJarDep` |
| `mdl/backend/interface.go` | `GetModuleSettings`, `UpdateModuleSettings` (or `UpdateJarDependencies`) |
| `mdl/backend/mpr/module_settings.go` | New file: read + write `Projects$ModuleSettings` with JarDependencies |
| `mdl/backend/mock/mock.go` | Stub for new backend methods |
| `sdk/mpr/reader_documents.go` | `ReadModuleSettings` (parses `Projects$ModuleSettings` unit) |
| `sdk/mpr/writer_modules.go` | `serializeModuleSettings` — replace hardcoded empty array with actual deps |
| `mdl/types/module_settings.go` | New: `ModuleSettings`, `JarDependency`, `JarDependencyExclusion` types |
| `mdl/executor/convert.go` | Conversion functions for the new types |
| `cmd/mxcli/syntax/features_modules.go` | Add syntax feature entries for new commands |
| `docs/01-project/MDL_QUICK_REFERENCE.md` | Add `ALTER MODULE jar dependency` syntax |
| `.claude/skills/mendix/java-actions.md` | Add note about module-level Maven deps |

### Order of operations

1. `mdl/types/module_settings.go` — define types
2. `sdk/mpr/reader_documents.go` + `sdk/mpr/writer_modules.go` — read/write BSON
3. Backend interface + MPR implementation + mock stub
4. Grammar + regenerate parser
5. AST + visitor
6. Executor commands
7. Syntax feature registry + quick reference
8. MDL test examples

---

## Version Compatibility

| Feature | Min Mendix version | Notes |
|---------|-------------------|-------|
| `JarDependency` (basic) | 10.0 | No gate needed; present in all supported versions |
| `exclusions` clause | 10.20 | Gate with `checkFeature("jar-dependency-exclusions")` |

The `exclusions` field should be written only when the project version is ≥ 10.20. For older
projects, silently omit the `exclusions` field from the serialised BSON (Studio Pro will
add it on next save if needed).

---

## Test Plan

- `mdl-examples/doctype-tests/10-module-jar-dependencies.mdl`:
  - `alter module JaTest add jar dependency` for two deps
  - `list jar dependencies in JaTest` → assert output
  - `describe jar dependency` → assert roundtrip MDL
  - `alter module JaTest set jar dependency ... version ...`
  - `alter module JaTest drop jar dependency ...`
  - `list jar dependencies in JaTest` → assert empty after drops

- `mdl-examples/bug-tests/` — not applicable (no known bug)

- Round-trip test: apply via MDL, read back via reader, verify fields match

- Studio Pro validation: load the project after `add jar dependency` and confirm
  Maven settings panel shows the expected coordinates

---

## Open Questions

1. **`or modify` semantics for `add`**: Should `add jar dependency` be idempotent (i.e.
   update the version if the group:artifact already exists) or strict (error on duplicate)?
   Strict is safer; `set jar dependency ... version` is the explicit update path.

2. **`included = false` in `add`**: Should a newly added dependency with `included = false`
   be allowed? Studio Pro allows it. Recommend yes for completeness.

3. **`show module settings` scope**: Include the broader `ModuleSettings` fields
   (`exportLevel`, `version`, etc.) in the same PR, or keep this PR strictly to
   `JarDependencies`? Recommendation: same PR — the read path is shared and the scope
   is manageable.

4. **Exclusion support priority**: `exclusions` are rarely used in practice. Could be
   deferred to a follow-up PR gated on `checkFeature`.
