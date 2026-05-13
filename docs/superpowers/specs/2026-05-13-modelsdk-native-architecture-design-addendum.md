# Modelsdk-Native Architecture — Stage 1 PoC Addendum

**Date:** 2026-05-13
**Status:** PoC complete — go decision with spec modifications
**References:** Original spec `2026-05-13-modelsdk-native-architecture-design.md`, plan `docs/superpowers/plans/2026-05-13-modelsdk-native-stage1-poc.md`

---

## Summary

Five tests across four blockers ran against the canonical fixture
`testdata/expr-checker/minimal.mpr` (v2 MPR with 16 `Microflows$Microflow`
units, largest 25,305 bytes). All five tests pass. Three benchmarks
quantify the encoder's incremental-rebuild behaviour. Three blockers pass
unconditionally; Blocker 3 is **MARGINAL** at the chosen 5x threshold and
forces one spec change (PageMutator interface for large unit ALTER).

The full PoC suite is `modelsdk/codec/poc/` and is included in the
standard `go test ./modelsdk/...` regression run with zero failures.

---

## Blocker Outcomes

### Blocker 1: Encode freshly-constructed gen object

**Result:** PASS

**Test:** `modelsdk/codec/poc/blocker1_fresh_encode_test.go`
- `TestBlocker1_EncodeFreshMicroflow`: PASS — a `*genMf.Microflow` built
  via `NewMicroflow()` + `SetID()` + `SetName()` encodes to BSON whose
  `$Type` field is `"Microflows$Microflow"` and whose `Name` field
  matches the value set on the gen object.
- `TestBlocker1_DecodeRoundTrip`: PASS — encode then decode produces a
  `*genMf.Microflow` with the same `ID()` and `Name()` as the original.

**Findings:**

- The encoder is a zero-value struct. The exposed method is `Encode(elem element.Element)` returning a byte slice and an error. No constructor is required.
- The decoder is constructed once: `codec.NewDecoder(codec.DefaultRegistry).Decode(raw bson.Raw) (element.Element, error)`.
- `codec.DefaultRegistry` is populated at package init time by every `modelsdk/gen/*` package's `init()` function, so a process that imports `genMf` (or any gen subpackage that's transitively imported) automatically registers `Microflows$Microflow` and the rest of that domain's types.
- For a fresh element (raw bytes are nil), `Encoder.Encode` falls through to `buildDoc`, which seeds `$ID` (binary subtype 0) and `$Type` from the element's identity, then overlays dirty properties. No special "new element" code path is needed at the call site.
- The decoded element is the concrete generated type (`*genMf.Microflow`) — the assertion `elem.(*genMf.Microflow)` succeeds. Generic `*element.Base` is only returned for unknown `$Type` values.

**Implication for Stage 2-4:** The `repo.Create(...)` flow of "build gen
object in memory → `codec.Encoder.Encode` → `mmpr.Writer.InsertUnit`" is
viable end-to-end. No design change is needed for the basic write path.

---

### Blocker 2: gen type ID/Container setters

**Result:** PASS, with one design clarification that updates spec section 5.

**Test:** `modelsdk/codec/poc/blocker2_id_setters_test.go`
- `TestBlocker2_GenTypeIDSetters`: PASS

**Findings (actual API surface on `*genMf.Microflow`):**

| Method | Source | Signature |
|---|---|---|
| `SetID` / `ID` | `element.Base` | `SetID(element.ID)` / `ID() element.ID` |
| `SetTypeName` / `TypeName` | `element.Base` | `SetTypeName(string)` / `TypeName() string` |
| `SetContainer` / `Container` | `element.Base` | `SetContainer(Element)` / `Container() Element` |
| `SetName` / `Name` | codegen (per-type property) | `SetName(string)` / `Name() string` |
| `SetUnit` / `Unit` | `element.Base` | `SetUnit(Unit)` / `Unit() Unit` |

`element.ID` is `type ID string`. Minting an ID is therefore
`element.ID(mmpr.GenerateID())`.

**Crucial absence:** there is **no** `SetContainerID(ID)` and **no**
`SetContainmentName(string)` on either `element.Base` or codegen output.
The container UUID and the SQL `ContainmentName` are passed positionally
to `mmpr.Writer.InsertUnit(unitID, containerID, containmentName, unitType, contents)`
at write time. They are not stored on the gen object.

`SetContainer(Element)` exists but is purely an in-memory parent
pointer used for dirty propagation through the Element tree. It does
not influence BSON serialisation of the child.

`NewMicroflow()` pre-populates `TypeName` to `"Microflows$Microflow"`
(verified: `mf.TypeName() != ""` immediately after construction).

**Implication for Stage 2-4:** The repository `Create` interface signature
must accept the parent UUID and the containment name as explicit
arguments rather than reading them off the element:

```go
type MicroflowRepo interface {
    Create(parentUUID string, containmentName string, mf *genMf.Microflow) error
    ...
}
```

This actually simplifies the spec — no need for an `IDGenerator` to mutate
container fields, and the repo never has to walk a chain of `Container()`
pointers to find a parent UUID. The caller (executor or `api/` builder)
already knows where the new unit lives.

---

### Blocker 3: Incremental encoding (dirty tracking)

**Result:** MARGINAL — Full vs Incremental ratio is 1.06x, not the 5x the
plan required. CleanPassthrough is 74,651x faster than Incremental, which
isolates *where* the cost lives but does not satisfy the "large ALTER is
cheap" assumption the original spec relied on.

**Benchmark output (verbatim from the second run captured in `/tmp/blocker3_bench.txt`):**

```
goos: linux
goarch: amd64
pkg: github.com/mendixlabs/mxcli/modelsdk/codec/poc
cpu: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
BenchmarkBlocker3_FullEncode-12           	    3277	    419059 ns/op	  168542 B/op	    3591 allocs/op
BenchmarkBlocker3_IncrementalEncode-12    	    3244	    393915 ns/op	  168169 B/op	    3591 allocs/op
BenchmarkBlocker3_CleanPassthrough-12     	225474939	         5.277 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/mendixlabs/mxcli/modelsdk/codec/poc	5.292s
```

Fixture: the largest microflow in the canonical PoC fixture
(`testdata/expr-checker/minimal.mpr`), 25,305 bytes raw, selected from
16 candidates. Hardware: Intel i7-8700 @ 3.20GHz, 12 logical CPUs.

**Findings:**

- `CleanPassthrough` (no dirty bits, raw passthrough fast path) costs 5.3 ns/op and zero allocations — the encoder simply returns `raw` unchanged when `raw != nil && !elem.IsDirty()`.
- `IncrementalEncode` (one root-level scalar dirty: `mf.SetName("ChangedNameOnly")`) costs 393,915 ns/op and 168 KB / 3,591 allocs.
- `FullEncode` (force everything: `mf.SetName(mf.Name())` to set the same dirty bit, then encode) costs 419,059 ns/op and 168 KB / 3,591 allocs.
- Full and Incremental are statistically indistinguishable (Incremental is 6% faster, well within run-to-run noise). They exercise the same `buildDoc` path because both set the same root-level dirty bit on `Name`.
- The encoder's selective-rebuild logic only short-circuits **child elements** that themselves have `raw != nil && !IsDirty()`. There is no overlay mechanism for "rebuild this scalar in place on a clean root element"; any root-level dirty bit forces the full document to be unmarshalled into `bson.D`, mutated, and re-marshalled.

**Implication for Stage 2-4:**

- For microflow ALTER on KB-class units (the entire 25 KB fixture rebuilds in 0.4 ms): **acceptable**. Even at 1000 ALTER operations per script, this is sub-second cost and 168 MB of transient allocations — well within the working set the GC handles.
- For Page or Workflow ALTER on widget trees that routinely exceed 100 KB and contain hundreds of widgets: **insufficient**. A single property change at the page root rebuilds the entire serialised tree, multiplied by the ALTER cadence the executor produces.
- **Spec change required:** Stage 2 must add a `PageMutator` (and `WorkflowMutator`) interface to `mdl/repos`. Mutators take a unit's raw BSON, edit specific child subtrees in place using `bson.RawValue` lookups, and write back the result without round-tripping the full element tree through `buildDoc`. This is the same shape as `mdl/backend/mpr.OpenPageForMutation` already in the legacy backend — it can be ported rather than redesigned.
- **Out of scope (deferred):** redesigning `Encoder.buildDoc` to overlay only the dirty fields onto `raw` rather than re-emitting from `bson.D`. This would close the gap between Clean (5 ns) and Incremental (400 µs) but requires non-trivial encoder rewrite. Park this as a Stage 5+ optimisation if PageMutator turns out to be insufficient ergonomically.

---

### Blocker 4: Write-then-read cache consistency

**Result:** PASS — cache invalidation is automatic on every write path tested.

**Test:** `modelsdk/codec/poc/blocker4_cache_consistency_test.go`
- `TestBlocker4_WriteThenRead`: PASS (0.23 s) — after `InsertUnit`, the same `*mmpr.Reader` instance returns count + 1 from `ListUnitsByType` and the new unit's bytes from `GetRawUnitBytes` (verified `bytes.Equal` against the originally encoded contents).
- `TestBlocker4_TransactionWriteThenRead`: PASS (0.20 s) — after `InsertUnit` to seed a unit, then `BeginWriteTransaction` → `WriteUnit` → `Commit`, the same `*mmpr.Reader` returns the updated bytes (verified `bytes.Equal` against the second encoding).

**Findings:**

- `(*mmpr.Writer).InsertUnit` calls `w.reader.InvalidateCache()` and `w.updateTransactionID()` after committing the file write and the SQL row.
- `(*mmpr.WriteTransaction).Commit` calls `wt.writer.reader.InvalidateCache()` after the database commit succeeds and the temp-file rename completes (file system order: temp write → DB commit → rename → invalidate).
- The cache lives on the `Reader` (`r.unitCache`, `r.unitCacheValid`); invalidation just sets `unitCacheValid = false`, and the next `ListUnitsByType` call rebuilds it from the SQLite `Unit` table plus the `mprcontents/` files.
- Both write paths share the same `Reader` instance via `Writer.reader`, so the same `Reader` returned by `Writer.ConcreteReader()` to the caller sees the invalidation immediately.

**Implication for Stage 2-4:**

- Repository `Create` and `Update` methods do **not** need to call cache invalidation explicitly. The `mmpr.Writer` already handles it.
- `ReaderCache.Invalidate()` can still appear in the spec as an explicit-control hook (useful for cross-process scenarios where another tool writes to the .mpr/mprcontents directly, or for tests that bypass the writer), but it is not a hard requirement of the per-repo write path.
- The two-phase ordering inside `WriteTransaction.Commit` (DB commit before file rename) leaves a narrow partial-failure window if `Rename` fails after `tx.Commit`. This is an existing characteristic of the writer, not a new finding from this PoC; spec section "Failure modes" should reference it but no PoC change is needed.

---

## Open Decisions Resolved

These were deferred from the original spec; the PoC results now resolve them.

| Decision | Resolution | Source |
|---|---|---|
| PageMutator (and WorkflowMutator) interface presence | **Yes — required for Stage 2.** Add to `mdl/repos`; port the existing `OpenPageForMutation` shape from the legacy MPR backend. | Blocker 3 outcome |
| Cache invalidation strategy | **Automatic in `mmpr.Writer`.** Repos rely on writer; expose `ReaderCache.Invalidate()` as an optional explicit hook for cross-process / test scenarios only. | Blocker 4 outcome |
| gen type constructor location | **Already complete in `modelsdk/gen/*`.** `NewMicroflow()` (and equivalents) are codegen-emitted, no codegen change required for ID/Container setters. | Blocker 2 outcome |
| `repo.Create` signature shape | **`Create(parentUUID string, containmentName string, elem Element) error`** — pass parent identity positionally, do not read it from the element. | Blocker 2 outcome |
| TransactionFactory implementation | **Direct wrap of `mmpr.Writer.BeginWriteTransaction()`.** No custom transaction object needed; the existing `*WriteTransaction.WriteUnit` + `Commit` API is sufficient and already drives cache invalidation correctly. | Blocker 4 transactional test |
| Encoder selective-rebuild ergonomics | **Adequate for microflow-class units (25 KB, 0.4 ms / encode); insufficient for page-class units (100 KB+).** Cover the gap with PageMutator rather than rewriting the encoder for Stage 2. | Blocker 3 outcome |

---

## Go/No-Go Decision

**Recommendation:** Proceed to Stage 2 with two spec modifications.

**Rationale:**

1. The basic write path (gen-object construction → `codec.Encoder.Encode` → `mmpr.Writer.InsertUnit` → cache-coherent read) works end-to-end on real fixture data with zero modifications to existing code. Three of four blockers pass unconditionally and validate the core architecture.
2. Blocker 3 forces one new interface in Stage 2: `PageMutator` for large-unit ALTER paths. This is a known shape (already present in the legacy backend as `OpenPageForMutation`) and carries no architectural risk.
3. Blocker 2 simplifies the spec: `repo.Create` takes parent identity as arguments rather than reading it from a "container-aware" element. This removes a class of complexity from the planned `IDGenerator` design and clarifies the executor → repo boundary.
4. Blocker 4 confirms the existing writer's cache invalidation is sufficient — no per-repo invalidation discipline needs to be enforced in code review.

**Stage 2 spec inputs (changes to apply before writing the Stage 2 plan):**

- Section 5 (Repository interfaces): adopt the `Create(parentUUID, containmentName, elem)` signature.
- Section 5 (Repository interfaces): add `PageMutator` and `WorkflowMutator` interfaces alongside `PageRepo` / `WorkflowRepo`. Mutator methods edit raw BSON subtrees; non-mutator methods use the standard encode-and-write path.
- Section "Cache management": downgrade `ReaderCache.Invalidate()` from a required repo discipline to an optional explicit hook.
- Section "Risks / open questions": close the four open items above with the table entries.
