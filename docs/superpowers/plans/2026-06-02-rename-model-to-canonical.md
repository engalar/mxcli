# Rename `mdl/model` → `mdl/canonical` Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the `mdl/model/` package to `mdl/canonical/` so the import path and package name are self-describing and avoid aliasing conflicts with the top-level `model/` (ID types) package.

**Architecture:** Pure mechanical rename — no logic changes. After this plan, `package model` becomes `package canonical`, `canonicalmodel.X` becomes `canonical.X`, and all import paths change from `mdl/model` to `mdl/canonical`. Subpackages (`entity/`, `layout/`) keep their own package names unchanged; only their import paths change.

**Tech Stack:** Go 1.26, `git mv`, `sed`.

**Must run before:** `2026-06-02-entity-canonical-completion.md`, `2026-06-02-association-canonical-model.md`, `2026-06-02-page-migration.md`.

---

## Affected files

**Root package declaration change** (`package model` → `package canonical`):

| File |
|------|
| `mdl/model/doc.go` |
| `mdl/model/context.go` |
| `mdl/model/datatype.go` |
| `mdl/model/registry.go` |

**Test package declaration change** (`package model_test` → `package canonical_test`):

| File |
|------|
| `mdl/model/import_guard_test.go` |
| `mdl/model/codec_guard_test.go` |
| `mdl/model/naming_guard_test.go` |

**Import path updates** (mdl/model → mdl/canonical, mdl/model/entity → mdl/canonical/entity, mdl/model/layout → mdl/canonical/layout):

| File | Old import | New import |
|------|-----------|-----------|
| `mdl/model/entity/model.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/entity/lift.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/entity/hydrate.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/entity/serialize.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/entity/persist.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/entity/codec.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/model/codec_guard_test.go` | `"github.com/mendixlabs/mxcli/mdl/model/entity"` | `"github.com/mendixlabs/mxcli/mdl/canonical/entity"` |
| `mdl/executor/executor.go` | `canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/executor/executor.go` | `entitymodel "github.com/mendixlabs/mxcli/mdl/model/entity"` | `entitymodel "github.com/mendixlabs/mxcli/mdl/canonical/entity"` |
| `mdl/executor/exec_context.go` | `canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/executor/cmd_create_entity_gen.go` | `canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |
| `mdl/executor/cmd_entities_gen.go` | `entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"` | `entityModel "github.com/mendixlabs/mxcli/mdl/canonical/entity"` |
| `mdl/executor/cmd_diff_mdl.go` | `entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"` | `entityModel "github.com/mendixlabs/mxcli/mdl/canonical/entity"` |
| `mdl/executor/boundary_guard_test.go` | `"github.com/mendixlabs/mxcli/mdl/model/entity"` | `"github.com/mendixlabs/mxcli/mdl/canonical/entity"` |
| `mdl/backend/mpr/domainmodel_layout.go` | `"github.com/mendixlabs/mxcli/mdl/model/layout"` | `"github.com/mendixlabs/mxcli/mdl/canonical/layout"` |
| `internal/archtest/codec_complete.go` | `"github.com/mendixlabs/mxcli/mdl/model"` | `"github.com/mendixlabs/mxcli/mdl/canonical"` |

**Usage references change** (after alias removal, `canonicalmodel.X` → `canonical.X`):

Affects: `exec_context.go`, `executor.go`, `cmd_create_entity_gen.go`, and any file that used the `canonicalmodel` alias.

---

## Task 1: Move directory and update all import paths

- [ ] **Step 1: Move the directory**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git mv mdl/model mdl/canonical
```

- [ ] **Step 2: Update root package declarations**

```bash
sed -i 's/^package model$/package canonical/' \
    mdl/canonical/doc.go \
    mdl/canonical/context.go \
    mdl/canonical/datatype.go \
    mdl/canonical/registry.go
```

- [ ] **Step 3: Update test package declarations**

```bash
sed -i 's/^package model_test$/package canonical_test/' \
    mdl/canonical/import_guard_test.go \
    mdl/canonical/codec_guard_test.go \
    mdl/canonical/naming_guard_test.go
```

- [ ] **Step 4: Update all import paths in Go files**

```bash
find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.go" -not -path "*/vendor/*" | xargs sed -i \
    's|github\.com/mendixlabs/mxcli/mdl/model/entity|github.com/mendixlabs/mxcli/mdl/canonical/entity|g'

find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.go" -not -path "*/vendor/*" | xargs sed -i \
    's|github\.com/mendixlabs/mxcli/mdl/model/layout|github.com/mendixlabs/mxcli/mdl/canonical/layout|g'

find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.go" -not -path "*/vendor/*" | xargs sed -i \
    's|github\.com/mendixlabs/mxcli/mdl/model"|github.com/mendixlabs/mxcli/mdl/canonical"|g'
```

**Important:** the last `sed` uses `mdl/model"` (with closing quote) to avoid replacing `mdl/model/entity` or `mdl/model/layout` twice.

- [ ] **Step 5: Remove the `canonicalmodel` alias — package name is now `canonical`, no alias needed**

```bash
# Remove alias from import declarations
find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.go" -not -path "*/vendor/*" | xargs sed -i \
    's|canonicalmodel "github\.com/mendixlabs/mxcli/mdl/canonical"|"github.com/mendixlabs/mxcli/mdl/canonical"|g'

# Update all usages: canonicalmodel.X → canonical.X
find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.go" -not -path "*/vendor/*" | xargs sed -i \
    's/canonicalmodel\./canonical\./g'
```

- [ ] **Step 6: Verify build**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./...
```

Expected: no errors. If there are errors, they will be "undefined: canonical.X" or "imported and not used" — fix manually. Common issues:
- A file that still references `canonicalmodel` but the alias was removed
- A `sed` miss on a line that had a different spacing pattern

Fix any errors, then re-run `go build ./...` until clean.

- [ ] **Step 7: Update plan string literals in guard files**

The boundary guard has string literals for forbidden paths that reference `mdl/model`:

```bash
# Update the Forbidden list in boundary_guard_test.go
sed -i 's|mdl/model/entity|mdl/canonical/entity|g' \
    /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/boundary_guard_test.go
```

- [ ] **Step 8: Run all guards to verify they still hold**

```bash
go test ./mdl/canonical/ -run "TestImportDirection|TestCodecComplete|TestPackageStructure" -v
go test ./mdl/canonical/entity/ -count=1 2>&1 | head -5
go test ./mdl/executor/ -run TestExecutorBoundary -v
go test ./internal/archtest/... -count=1 2>&1 | tail -1
```

Expected:
- `TestImportDirection`: FAIL (RED — context.go still imports backend; this will be fixed in the entity completion plan)
- `TestCodecComplete`: PASS
- `TestPackageStructure`: PASS
- `TestExecutorBoundary`: FAIL (RED — cmd_entities_gen.go still imports canonical/entity; also fixed in entity completion plan)
- archtest suite: PASS

The two RED guards are expected — they were already red before the rename.

- [ ] **Step 9: Run full test suite**

```bash
go test ./... -count=1 2>&1 | grep -E "^FAIL|^ok" | grep -v "^ok" | head -20
```

Any NEW failures (beyond the two expected red guards) must be fixed before committing.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "refactor(model): rename mdl/model → mdl/canonical — package name matches directory convention

Import path: github.com/mendixlabs/mxcli/mdl/model → mdl/canonical
Package name: model → canonical (eliminates canonicalmodel alias)
Subpackages entity/ and layout/ keep their names; only import paths change.
Guards 1 and 4 remain RED (pre-existing violations, fixed in entity completion plan)."
```

---

## Self-Review

| Requirement | Step |
|-------------|------|
| Directory moved | 1 |
| Root package declarations updated | 2 |
| Test package declarations updated | 3 |
| Import paths updated (subpackages) | 4 |
| Import paths updated (root) | 4 |
| `canonicalmodel` alias removed | 5 |
| `canonicalmodel.X` → `canonical.X` usage updated | 5 |
| Build clean | 6 |
| Guard boundary string literals updated | 7 |
| Guards unchanged from pre-rename state | 8 |
| No new test failures | 9 |
