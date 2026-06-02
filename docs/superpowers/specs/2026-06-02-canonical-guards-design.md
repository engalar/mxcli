# Canonical Model Layer — Architectural Guards Design

**Date:** 2026-06-02
**Goal:** Prevent regression in the `mdl/canonical/` refactoring via failing-first
architectural tests that double as AI-actionable fix guides.
**Approach:** Five guard files using a SOLID `internal/archtest` rule engine.
All guards start red; they green as implementation converges.

---

## Problem

The canonical model layer (`mdl/model/`, to be renamed `mdl/canonical/`) was
implemented incrementally and has accumulated SOLID violations:

- **DIP**: root package imports `mdl/backend`, `modelsdk/gen/`, `mdl/ast`
- **LSP**: `Document.ToMDL()` and `ToMDLStatement(bool)` co-exist; interface
  contract is bypassed in the describe path
- **ISP**: `PersistContext` carries entity-specific fields that are meaningless
  for associations or microflows
- **OCP**: executor.go must be manually updated every time a new domain is added
- **Executor bypass**: `cmd_entities_gen.go` calls `entity.HydrateWithModule()`
  directly instead of routing through the codec registry

AI implementation is fast but drifts. Guards make the target architecture
self-enforcing: a failing test tells the implementer exactly what to fix.

---

## Constraints

- Guards run under `make test` (no new tooling, no separate CI step)
- Use `go/parser` + standard library only (no third-party arch tools)
- Follow the existing `mdl/executor/import_guard_test.go` pattern
- Target package path: `mdl/canonical/` (renamed from `mdl/model/`)
- Guards start **all red**; implementation drives them to green (TDD for architecture)

---

## `internal/archtest` — Rule Engine

A SOLID rule engine. Each concern is a separate type; the `Check()` entry point
depends only on the `Rule` interface.

### File layout

```
internal/archtest/
    archtest.go        # Rule, Violation, Package, File, Check()
    no_import.go       # NoImport rule
    codec_complete.go  # CodecComplete rule
    package_name.go    # PackageName rule
```

### Core types

```go
// Rule is the single abstraction. Each implementation checks one property.
type Rule interface {
    Name()  string
    Check(pkg Package) []Violation
}

// Violation is one rule failure. Hint is an AI-actionable fix guide;
// it is written by the guard-file author, not the rule implementation.
type Violation struct {
    File    string
    Line    int
    Message string
    Hint    string
}

// Package is the input to file-scanning rules.
type Package struct{ Dir string }

func (p Package) Files() []File  // root-level *.go files, excluding _test.go
func (p Package) Walk()  []File  // recursive, including subdirectories

type File struct {
    Name    string
    Path    string
    Imports []string
    Line    func(importPath string) int
}

// Check runs rules against dir and reports all violations via t.Errorf.
// Hint is appended to the error message when non-empty.
func Check(t testing.TB, dir string, rules ...Rule)
```

### Concrete rules

**`NoImport`** — scans `pkg.Files()`, skips Allowlist entries, fails on any
file whose imports contain a forbidden path prefix:

```go
type NoImport struct {
    Forbidden []string
    Allowlist map[string]bool  // file basenames exempt from the rule
    Hint      string           // propagated to every Violation
}
```

**`PackageName`** — uses `pkg.Walk()`, checks that every subdirectory's
`package` declaration matches the directory name and nesting depth ≤ MaxDepth:

```go
type PackageName struct {
    MaxDepth       int
    NameMatchesDir bool
    Hint           string
}
```

**`CodecComplete`** — does **not** scan files; inspects a live registry.
The `Package` argument is ignored. This is the only rule that operates on
runtime state rather than source:

```go
type CodecComplete struct {
    BuildRegistry func() *canonical.DefaultRegistry
    Required      []string  // gen TypeNames that must be registered
    Hint          string
}
```

`DefaultRegistry` must expose a `Lookup(typeName string) (Codec, bool)` method
to support this rule.

### SOLID properties

| Principle | How it is satisfied |
|-----------|---------------------|
| S | Rule / Violation / Package / File / Check each have one job |
| O | New rules implement `Rule`; `check.go` is never modified |
| L | All rules are substitutable through `Rule`; `Check()` is unaware of specifics |
| I | `Rule` interface has exactly two methods |
| D | Guard files depend on `Rule` abstraction; `go/parser` details are hidden |

### Output format

```
--- FAIL: TestImportDirection (0.01s)
    context.go:12: imports "mendixlabs/mxcli/mdl/backend" [NoImport]
    HINT: mdl/canonical/ root package is the stable center; it must not depend on any volatile layer ...
```

---

## Five Guards

### Guard 1 — Import Direction

**File:** `mdl/canonical/import_guard_test.go`
**Rule:** `NoImport`
**Scans:** root-level files only (not subdirectories)

```go
func TestImportDirection(t *testing.T) {
    archtest.Check(t, ".",
        archtest.NoImport{
            Forbidden: []string{
                "mendixlabs/mxcli/mdl/ast",
                "mendixlabs/mxcli/modelsdk/gen/",
                "mendixlabs/mxcli/mdl/backend",
            },
            Hint: `mdl/canonical/ root package is the stable center; it must not
depend on any volatile layer.
Fix:
  1. Replace PersistContext.Backend with a Writer interface defined in this package:
       type Writer interface { CreateDoc(id ID, doc Persistable) error }
  2. backend/mpr implements Writer; executor passes the implementation in.
  3. context.go, doc.go, registry.go may only import stdlib and model/ (for ID).`,
        },
    )
}
```

**Red state:** `context.go` imports `mdl/backend`.
**Green when:** `PersistContext.Backend` is replaced by a local `Writer` interface.

---

### Guard 2 — Interface Compliance

**File:** `mdl/canonical/entity/comply_test.go` (one per domain)
**Mechanism:** compile-time `var _` assertions; no runtime logic.

```go
package entity_test

import (
    "github.com/mendixlabs/mxcli/mdl/canonical"
    "github.com/mendixlabs/mxcli/mdl/canonical/entity"
)

var _ canonical.Document    = (*entity.EntityModel)(nil)
var _ canonical.Persistable = (*entity.EntityModel)(nil)
```

The compiler error names the missing method exactly. No Hint needed.

**Scaling rule:** when adding a new domain, create `mdl/canonical/{domain}/comply_test.go`
with two lines changed. No other file is modified.

**Red state:** package path is still `mdl/model/entity`; the import path fails.
**Green when:** directory renamed to `mdl/canonical/entity/`.

---

### Guard 3 — Codec Completeness

**File:** `mdl/canonical/codec_guard_test.go`
**Rule:** `CodecComplete`

```go
func TestCodecComplete(t *testing.T) {
    archtest.Check(t, ".",
        archtest.CodecComplete{
            BuildRegistry: func() *canonical.DefaultRegistry {
                r := canonical.NewDefaultRegistry()
                entity.RegisterCodec(r)
                // Phase 2: association.RegisterCodec(r)
                // Phase 3: microflow.RegisterCodec(r)
                return r
            },
            Required: []string{
                "DomainModels$Entity",
                "DomainModels$EntityImpl",
                // Phase 2: "DomainModels$Association"
            },
            Hint: `Every Required gen TypeName must be registered with non-nil LiftFn and HydrateFn.
Fix:
  1. Confirm domain/codec.go calls r.Register(...) for all listed TypeNames.
  2. LiftFn  — func(stmt any) (Persistable, error): converts AST stmt to Model.
  3. HydrateFn — func(el any, ctx HydrateCtx) (Document, []Warning, error):
       ctx.ModuleName carries the owning module; never hardcode "".
  4. When adding a new domain: add RegisterCodec call to BuildRegistry AND
     add its TypeName(s) to Required. Both must be updated together.`,
        },
    )
}
```

**Red state:** `HydrateFn` passes empty module name (violates HydrateCtx contract).
**Green when:** all Required types registered; `HydrateFn` uses `HydrateCtx`.

---

### Guard 4 — Executor Boundary

**File:** `mdl/executor/boundary_guard_test.go`
**Rule:** `NoImport`

```go
func TestExecutorBoundary(t *testing.T) {
    archtest.Check(t, ".",
        archtest.NoImport{
            Forbidden: []string{"mendixlabs/mxcli/mdl/canonical/"},
            Allowlist: map[string]bool{
                "executor.go": true, // sole file allowed to import domain subpackages
            },
            Hint: `cmd_*.go files must not import mdl/canonical/{domain}/ subpackages directly.
Fix:
  Describe path:
    Replace: entitymodel.HydrateWithModule(modName, e)
    With:    doc, warns, err := ctx.ModelCodecs.HydrateFrom(e,
                 canonical.HydrateCtx{ModuleName: modName})
  Create path:
    Replace: entity.Lift(s)
    With:    ctx.ModelCodecs.LiftFrom(s)
  RegisterCodec calls remain in executor.go only; all other files dispatch
  through ctx.ModelCodecs.`,
        },
    )
}
```

**Red state:** `cmd_entities_gen.go` imports `mdl/model/entity` directly.
**Green when:** describe/create paths route through `ctx.ModelCodecs`.

---

### Guard 5 — Package Structure

**File:** `mdl/canonical/naming_guard_test.go`
**Rule:** `PackageName`

```go
func TestPackageStructure(t *testing.T) {
    archtest.Check(t, ".",
        archtest.PackageName{
            MaxDepth:       1,
            NameMatchesDir: true,
            Hint: `mdl/canonical/ allows exactly one level of subpackages (entity/, association/).
Do not create nested subdirectories inside domain packages (entity/conversion/, etc.).
To split a large file: add files in the same directory with the same package name.`,
        },
    )
}
```

**Red state:** current path prefix is `mdl/model/`, not `mdl/canonical/`.
**Green when:** directories renamed to `mdl/canonical/{domain}/`.

---

## Red → Green Summary

| Guard | Current red point | Minimum action to green |
|-------|-------------------|-------------------------|
| 1 Import direction | `context.go` imports `mdl/backend` | Define `Writer` interface in `mdl/canonical/`; remove backend import |
| 2 Interface compliance | Path is `mdl/model/entity` | Rename directory to `mdl/canonical/entity/` |
| 3 Codec completeness | `HydrateFn` hardcodes `""` module name | Change `HydrateFn` to accept `HydrateCtx`; update `HydrateFrom` signature |
| 4 Executor boundary | `cmd_entities_gen.go` imports entity directly | Route describe/create through `ctx.ModelCodecs` |
| 5 Package structure | Prefix is `mdl/model/` | Rename directories |

Guards 2 and 5 share the same trigger (directory rename) and green together.
Guards 3 and 4 require independent changes and green independently.
Guard 1 is the deepest structural change and should green last.

---

## What This Does NOT Cover

- Correctness of `ToMDL()` output (covered by existing roundtrip tests)
- BSON field validity (covered by existing `TestNoRawBSONTypeStringsInExecutor`)
- Event handler completeness in EntityModel (tracked separately as a known gap)
- Page canonical model (page architecture is fundamentally different; separate spec needed)
