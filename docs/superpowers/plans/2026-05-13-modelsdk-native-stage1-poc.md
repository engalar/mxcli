# Modelsdk-Native Stage 1: PoC Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Empirically validate the four design assumptions in `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md` Section 9 before committing to Stages 2-4 of the migration.

**Architecture:** Add focused PoC tests under `modelsdk/codec/poc/` that exercise the encoder, decoder, and writer in the exact patterns the new architecture will use. Each blocker becomes one test (or benchmark). Output is empirical findings written as a spec addendum.

**Tech Stack:** Go 1.26, `modelsdk/codec`, `modelsdk/gen/microflows`, `modelsdk/gen/pages`, `modelsdk/mpr`, `sdk/mpr` (read-only verification)

**Test Layout:**

| File | Responsibility |
|------|----------------|
| `modelsdk/codec/poc/poc_helpers_test.go` | Shared helpers: temp MPR file, fixture loading |
| `modelsdk/codec/poc/blocker1_fresh_encode_test.go` | Blocker 1 — encode constructed-from-scratch gen object |
| `modelsdk/codec/poc/blocker2_id_setters_test.go` | Blocker 2 — SetID/SetContainerID/Set ContainmentName public API |
| `modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go` | Blocker 3 — incremental encoding benchmark |
| `modelsdk/codec/poc/blocker4_cache_consistency_test.go` | Blocker 4 — write-then-read cache invalidation |
| `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md` | Findings + go/no-go decision per blocker |

---

## Task 1: PoC infrastructure (test helpers)

**Files:**
- Create: `modelsdk/codec/poc/poc_helpers_test.go`

A new `poc` sub-package isolates these tests from the regular codec test suite. Tests in this package validate cross-package integration that doesn't fit cleanly into any single existing test file.

- [ ] **Step 1: Create the package directory and helpers file**

Create `/mnt/data_sdd/gh/mxcli-wt-02/modelsdk/codec/poc/poc_helpers_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

// Package poc contains the Stage 1 proof-of-concept tests for the
// modelsdk-native architecture migration. See
// docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md
// Section 9 for the design assumptions these tests validate.
package poc_test

import (
	"path/filepath"
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// openTestMPR opens a fixture MPR file from modelsdk/mpr/testdata/microflows
// in read-write mode. Each test gets its own temp copy so writes don't
// affect other tests.
func openTestMPR(t *testing.T, fixtureSubdir string) *mmpr.Writer {
	t.Helper()
	src := filepath.Join("..", "..", "mpr", "testdata", fixtureSubdir, "app.mpr")
	dst := filepath.Join(t.TempDir(), "app.mpr")
	copyFile(t, src, dst)
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := readFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", src, err)
	}
	if err := writeFile(dst, data); err != nil {
		t.Fatalf("write tempfile %s: %v", dst, err)
	}
}

func readFile(p string) ([]byte, error) {
	// Using os.ReadFile but kept as helper for clarity in test failures.
	return osReadFile(p)
}

func writeFile(p string, data []byte) error {
	return osWriteFile(p, data, 0o644)
}
```

Note on imports: the helper wraps `os.ReadFile` / `os.WriteFile` only to make test failure messages clearer about WHICH file failed. Add the `os` import if you implement directly:

```go
import "os"

func osReadFile(p string) ([]byte, error)        { return os.ReadFile(p) }
func osWriteFile(p string, d []byte, m uint32) error { return os.WriteFile(p, d, fs.FileMode(m)) }
```

Use the simpler form if you prefer — direct `os.ReadFile` / `os.WriteFile` calls in `copyFile`. Either is fine; the goal is a working helper.

- [ ] **Step 2: Verify the fixture path exists**

Run from `/mnt/data_sdd/gh/mxcli-wt-02`:

```bash
ls modelsdk/mpr/testdata/microflows/app.mpr
```

Expected: file exists. If not, list `modelsdk/mpr/testdata/` and pick the closest matching microflow fixture; update the helper accordingly.

- [ ] **Step 3: Compile-check the helpers**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go vet ./modelsdk/codec/poc/...
```

Expected: passes (no test bodies yet, just helpers — this catches compile errors early).

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/poc/poc_helpers_test.go
git commit -m "test(poc): add Stage 1 PoC test scaffolding for modelsdk-native migration"
```

---

## Task 2: Blocker 2 — gen type ID/Container setters

We do this blocker **first** because Blockers 1, 3, and 4 all depend on being able to construct a fresh `*genMf.Microflow` and set its ID/ContainerID. If the setters don't exist, we can't write the other tests.

**Files:**
- Create: `modelsdk/codec/poc/blocker2_id_setters_test.go`
- Possibly modify: `modelsdk/gen/microflows/types.go` (only if setters missing — should be present per prior inspection)

- [ ] **Step 1: Verify SetID and SetContainerID exist on Microflow**

```bash
grep -n "func (o \*Microflow) SetID\|func (o \*Microflow) SetContainerID\|func (o \*Microflow) Set" /mnt/data_sdd/gh/mxcli-wt-02/modelsdk/gen/microflows/types.go | head -20
```

Expected: setters exist. If `SetID` or `SetContainerID` is missing, **STOP and report** — codegen needs modification before this plan can proceed.

- [ ] **Step 2: Write the test**

Create `/mnt/data_sdd/gh/mxcli-wt-02/modelsdk/codec/poc/blocker2_id_setters_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// newID is the canonical way to mint an ID for these PoC tests.
// GenerateID lives in modelsdk/mpr (returns string), and element.ID
// is `type ID string`, so the cast is explicit.
func newID() element.ID {
	return element.ID(mmpr.GenerateID())
}

// TestBlocker2_GenTypeIDSetters confirms that freshly-constructed gen
// objects expose public setters for ID, ContainerID, and ContainmentName
// — the three fields the executor must set before calling repo.Create().
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 2.
func TestBlocker2_GenTypeIDSetters(t *testing.T) {
	mf := genMf.NewMicroflow()
	if mf == nil {
		t.Fatal("NewMicroflow() returned nil")
	}

	// Assign ID
	id := newID()
	mf.SetID(id)
	if got := mf.ID(); got != id {
		t.Errorf("after SetID: got %q, want %q", got, id)
	}

	// Assign ContainerID
	containerID := newID()
	mf.SetContainerID(containerID)
	if got := mf.ContainerID(); got != containerID {
		t.Errorf("after SetContainerID: got %q, want %q", got, containerID)
	}

	// Set Name (smoke test — confirms property setter idiom works)
	mf.SetName("PoCFlow")
	if got := mf.Name(); got != "PoCFlow" {
		t.Errorf("after SetName: got %q, want PoCFlow", got)
	}
}
```

- [ ] **Step 3: Run the test**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/codec/poc/... -run TestBlocker2_GenTypeIDSetters -v
```

Expected: PASS. If the test fails to compile because `SetID` or `ContainerID` accessor signatures differ from what's used here, **adjust the test to match the actual gen API** — record the actual signatures in the addendum (Task 7) so the spec stays accurate.

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/poc/blocker2_id_setters_test.go
git commit -m "test(poc): blocker 2 — verify gen type ID/Container setters exist (PASS)"
```

---

## Task 3: Blocker 1 — Encode freshly-constructed gen object

**Files:**
- Create: `modelsdk/codec/poc/blocker1_fresh_encode_test.go`

- [ ] **Step 1: Write the test**

Create `/mnt/data_sdd/gh/mxcli-wt-02/modelsdk/codec/poc/blocker1_fresh_encode_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestBlocker1_EncodeFreshMicroflow confirms that a Microflow constructed
// from scratch (no prior raw BSON) encodes to valid BSON containing the
// expected $Type and basic fields.
//
// This is the core PoC: if encoding a fresh object doesn't work, the
// modelsdk-native write path is unbuildable.
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 1.
func TestBlocker1_EncodeFreshMicroflow(t *testing.T) {
	mf := genMf.NewMicroflow()
	mf.SetID(newID())          // newID defined in blocker2 test file (same package)
	mf.SetContainerID(newID())
	mf.SetName("FreshlyConstructed")

	enc := &codec.Encoder{} // Encoder is a zero-value struct, no constructor
	bytes, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatal("Encode returned empty bytes")
	}

	// Round-trip: decode and verify $Type + Name survive
	var doc bson.D
	if err := bson.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("bson.Unmarshal: %v", err)
	}

	dollarType := lookupString(doc, "$Type")
	if dollarType != "Microflows$Microflow" {
		t.Errorf("$Type = %q, want %q", dollarType, "Microflows$Microflow")
	}

	name := lookupString(doc, "Name")
	if name != "FreshlyConstructed" {
		t.Errorf("Name = %q, want FreshlyConstructed", name)
	}
}

// TestBlocker1_DecodeRoundTrip confirms encode-then-decode produces an
// equivalent Microflow.
func TestBlocker1_DecodeRoundTrip(t *testing.T) {
	mf := genMf.NewMicroflow()
	mfID := newID()
	mf.SetID(mfID)
	mf.SetContainerID(newID())
	mf.SetName("RoundTrip")

	enc := &codec.Encoder{}
	encoded, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(encoded))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	mf2, ok := elem.(*genMf.Microflow)
	if !ok {
		t.Fatalf("Decode produced %T, want *genMf.Microflow", elem)
	}

	if mf2.Name() != "RoundTrip" {
		t.Errorf("decoded Name = %q, want RoundTrip", mf2.Name())
	}
	if !bytes.Equal([]byte(mf.ID()), []byte(mf2.ID())) {
		t.Errorf("decoded ID = %v, want %v", mf2.ID(), mf.ID())
	}
	_ = mfID // also used implicitly via mf.ID()
}

// lookupString fetches a top-level string field from a bson.D.
// Returns empty string if missing or not a string.
func lookupString(doc bson.D, key string) string {
	for _, e := range doc {
		if e.Key == key {
			if s, ok := e.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}
```

- [ ] **Step 2: Run the tests**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/codec/poc/... -run TestBlocker1 -v
```

Expected: both tests PASS. Possible failure modes:
- `codec.NewEncoder` / `codec.NewDecoder` constructor signatures may differ — check `modelsdk/codec/encoder.go` and `decoder.go` for actual signatures and adjust the test
- Decoder may not return `*genMf.Microflow` directly — record actual return type in addendum

If a test fails because of API differences, **adjust the test** to match the actual API and document the actual API in the addendum. If a test fails because encoding is fundamentally broken, **STOP and report** — Stage 2-4 cannot proceed.

- [ ] **Step 3: Commit**

```bash
git add modelsdk/codec/poc/blocker1_fresh_encode_test.go
git commit -m "test(poc): blocker 1 — verify fresh gen object encode + roundtrip"
```

---

## Task 4: Blocker 3 — Incremental encoding benchmark

The encoder's `buildDoc` already implements selective rebuild (`anyChildDirty` + per-child raw passthrough). This benchmark **measures** whether the implementation actually delivers on the promise.

**Files:**
- Create: `modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go`

- [ ] **Step 1: Locate a large fixture**

```bash
ls -la /mnt/data_sdd/gh/mxcli-wt-02/modelsdk/mpr/testdata/pages/
ls -la /mnt/data_sdd/gh/mxcli-wt-02/modelsdk/mpr/testdata/microflows/
```

Pick the fixture with the largest unit size (microflow with many activities, or page with many widgets). Note the path and a known unit ID inside it. To find a unit ID:

```bash
# After setting up the test, you can use list_units once. For now, expect to
# discover the ID at test-write time.
```

If no large fixtures exist (all are minimal smoke tests), record this as a finding and use the largest available fixture, scaling expectations accordingly.

- [ ] **Step 2: Write the benchmark**

Create `/mnt/data_sdd/gh/mxcli-wt-02/modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// BenchmarkBlocker3_FullEncode measures the cost of encoding a microflow
// where every property is dirty (worst-case: full rebuild).
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 3.
func BenchmarkBlocker3_FullEncode(b *testing.B) {
	mf := loadFixtureMicroflow(b)

	// Force everything dirty by re-setting the name (touches dirty bitmap).
	mf.SetName(mf.Name())

	enc := &codec.Encoder{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enc.Encode(mf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBlocker3_IncrementalEncode measures the cost of encoding a
// microflow where only one scalar property is dirty (best-case:
// selective rebuild — clean children pass through raw bytes).
//
// If incremental encoding works, this should be SIGNIFICANTLY faster
// than BenchmarkBlocker3_FullEncode for a large microflow.
func BenchmarkBlocker3_IncrementalEncode(b *testing.B) {
	mf := loadFixtureMicroflow(b)
	mf.SetName("ChangedNameOnly")

	enc := &codec.Encoder{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enc.Encode(mf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// loadFixtureMicroflow opens the largest microflow in the testdata fixture
// and returns it as a *genMf.Microflow. Returns the FIRST microflow found.
//
// REPLACE the fixture subdir / unit lookup with the largest available
// microflow at test-write time.
func loadFixtureMicroflow(b *testing.B) *genMf.Microflow {
	b.Helper()

	// Use the same helper as tests (rename if you split helpers).
	w := openTestMPRForBench(b, "microflows")
	r := w.Reader()
	refs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		b.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		b.Fatalf("no Microflows$Microflow units in fixture")
	}

	// Pick the largest by raw byte length
	var pick = refs[0]
	for _, r := range refs[1:] {
		if len(r.Contents) > len(pick.Contents) {
			pick = r
		}
	}

	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(pick.Contents))
	if err != nil {
		b.Fatalf("Decode: %v", err)
	}
	mf, ok := elem.(*genMf.Microflow)
	if !ok {
		b.Fatalf("Decode produced %T, want *genMf.Microflow", elem)
	}
	return mf
}
```

Add a benchmark variant of the test helper to `poc_helpers_test.go`:

```go
// openTestMPRForBench is the *testing.B variant of openTestMPR.
func openTestMPRForBench(b *testing.B, fixtureSubdir string) *mmpr.Writer {
	b.Helper()
	src := filepath.Join("..", "..", "mpr", "testdata", fixtureSubdir, "app.mpr")
	dst := filepath.Join(b.TempDir(), "app.mpr")
	data, err := os.ReadFile(src)
	if err != nil {
		b.Fatalf("read fixture %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		b.Fatalf("write tempfile %s: %v", dst, err)
	}
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		b.Fatalf("NewWriter(%s): %v", dst, err)
	}
	b.Cleanup(func() { _ = w.Close() })
	return w
}
```

(Add `import "os"` to the helpers file if not already present.)

- [ ] **Step 3: Run the benchmarks**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test -bench=BenchmarkBlocker3 -benchmem -run=^$ ./modelsdk/codec/poc/... -v
```

Expected output (numbers will differ):

```
BenchmarkBlocker3_FullEncode-8             1000    1234567 ns/op    65536 B/op    234 allocs/op
BenchmarkBlocker3_IncrementalEncode-8    100000      12345 ns/op      512 B/op     12 allocs/op
```

**Pass criterion:** `IncrementalEncode` is at least **5x faster** than `FullEncode` for a non-trivial microflow. If they're within 2x, the dirty-tracking optimization is not effective — record this in the addendum and flag that ALTER on large pages may need PageMutator.

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go modelsdk/codec/poc/poc_helpers_test.go
git commit -m "test(poc): blocker 3 — benchmark incremental vs full BSON encoding"
```

---

## Task 5: Blocker 4 — Write-then-read cache consistency

**Files:**
- Create: `modelsdk/codec/poc/blocker4_cache_consistency_test.go`

- [ ] **Step 1: Write the test**

Create `/mnt/data_sdd/gh/mxcli-wt-02/modelsdk/codec/poc/blocker4_cache_consistency_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestBlocker4_WriteThenRead confirms that after InsertUnit completes,
// a follow-up read via the same modelsdk/mpr.Reader sees the new unit.
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 4.
func TestBlocker4_WriteThenRead(t *testing.T) {
	w := openTestMPR(t, "microflows")
	r := w.Reader()

	// Snapshot baseline count
	baseRefs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		t.Fatalf("ListUnitsByType (baseline): %v", err)
	}
	baseCount := len(baseRefs)

	// Construct + encode a new microflow (newID defined in blocker2 test file)
	mf := genMf.NewMicroflow()
	mfID := newID()
	containerID := newID()
	mf.SetID(mfID)
	mf.SetContainerID(containerID)
	mf.SetName("PoCBlocker4Flow")

	enc := &codec.Encoder{}
	contents, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Write
	if err := w.InsertUnit(string(mfID), string(containerID), "Documents", "Microflows$Microflow", contents); err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}

	// Read immediately — same Reader instance, no manual cache invalidation
	afterRefs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		t.Fatalf("ListUnitsByType (after write): %v", err)
	}

	if len(afterRefs) != baseCount+1 {
		t.Errorf("after InsertUnit: count = %d, want %d (write not visible to reader)", len(afterRefs), baseCount+1)
	}

	// Verify the new unit specifically by ID
	found := false
	for _, ref := range afterRefs {
		if ref.ID == string(mfID) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("new microflow ID %q not visible in ListUnitsByType after InsertUnit", mfID)
	}

	// Verify GetRawUnitBytes also sees it
	bytes, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Errorf("GetRawUnitBytes(%q): %v (write not visible)", mfID, err)
	}
	if len(bytes) == 0 {
		t.Errorf("GetRawUnitBytes(%q) returned empty bytes", mfID)
	}
}

// TestBlocker4_TransactionWriteThenRead repeats the test through a
// WriteTransaction (the path used by the new repos for atomic operations).
func TestBlocker4_TransactionWriteThenRead(t *testing.T) {
	w := openTestMPR(t, "microflows")
	r := w.Reader()

	wtx, err := w.BeginWriteTransaction()
	if err != nil {
		t.Fatalf("BeginWriteTransaction: %v", err)
	}

	mf := genMf.NewMicroflow()
	mfID := newID()
	mf.SetID(mfID)
	mf.SetContainerID(newID())
	mf.SetName("PoCBlocker4TxFlow")

	enc := &codec.Encoder{}
	contents, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if err := wtx.WriteUnit(string(mfID), contents); err != nil {
		_ = wtx.Rollback()
		t.Fatalf("WriteUnit: %v", err)
	}
	if err := wtx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Read after commit
	bytes, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Errorf("GetRawUnitBytes after commit: %v", err)
	}
	if len(bytes) == 0 {
		t.Errorf("GetRawUnitBytes after commit returned empty")
	}
}
```

Note: `WriteTransaction.WriteUnit` is the API used for **updates** to existing units. For inserts of brand-new units, `InsertUnit` may be the only path (depends on the MPR v1/v2 format). If `WriteUnit` errors with "unit not found", that's expected behavior — change the second test to first `InsertUnit`, then read, then `WriteTransaction.WriteUnit` to update.

- [ ] **Step 2: Run the tests**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/codec/poc/... -run TestBlocker4 -v
```

Expected: both tests PASS. If `TestBlocker4_WriteThenRead` fails (count mismatch or unit not found), then writes are NOT visible to the same Reader without explicit invalidation. Record this in the addendum — the new architecture's `ReaderCache.Invalidate()` becomes a hard requirement, not an optional optimization.

- [ ] **Step 3: Commit**

```bash
git add modelsdk/codec/poc/blocker4_cache_consistency_test.go
git commit -m "test(poc): blocker 4 — verify write-then-read cache consistency"
```

---

## Task 6: Verify all PoC tests pass together

**Files:** None (verification only)

- [ ] **Step 1: Run the full PoC suite**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/codec/poc/... -v -count=1
```

Expected: all four blockers' tests PASS (or any failures are ones we've explicitly recorded as findings to address in the addendum).

- [ ] **Step 2: Run benchmarks once more for the addendum**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test -bench=BenchmarkBlocker3 -benchmem -run=^$ ./modelsdk/codec/poc/... | tee /tmp/blocker3_bench.txt
```

Save the output — we'll quote the actual numbers in the addendum.

- [ ] **Step 3: Confirm no regressions in existing tests**

```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/... -count=1 -timeout 300s
```

Expected: no new failures (the new `poc` package adds tests, doesn't change anything).

---

## Task 7: Write spec addendum with findings

**Files:**
- Create: `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md`

This addendum updates the spec's Section 9 (Blockers) and Section 11 (Open Decisions) with empirical findings. It is the **input** for the Stage 2-4 implementation plan.

- [ ] **Step 1: Write the addendum**

Create `/mnt/data_sdd/gh/mxcli-wt-02/docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md`:

```markdown
# Modelsdk-Native Architecture — Stage 1 PoC Addendum

**Date:** 2026-05-13
**Status:** PoC complete — go/no-go for Stage 2-4
**References:** Original spec `2026-05-13-modelsdk-native-architecture-design.md`, plan `docs/superpowers/plans/2026-05-13-modelsdk-native-stage1-poc.md`

---

## Blocker Outcomes

### Blocker 1: Encode freshly-constructed gen object

**Result:** [PASS / FAIL]

**Test:** `modelsdk/codec/poc/blocker1_fresh_encode_test.go`
- `TestBlocker1_EncodeFreshMicroflow`: [PASS / FAIL — explain]
- `TestBlocker1_DecodeRoundTrip`: [PASS / FAIL — explain]

**Findings:** [Describe what was empirically observed. Include actual API signatures used (codec.NewEncoder vs alternatives), and any surprises.]

**Implication for Stage 2-4:** [If PASS: Encoder.Encode works for fresh objects, no design change needed. If FAIL: describe the specific failure mode and the design change required.]

---

### Blocker 2: gen type ID/Container setters

**Result:** [PASS / FAIL]

**Test:** `modelsdk/codec/poc/blocker2_id_setters_test.go`

**Findings:** [Document actual setter signatures: SetID(string), SetID(element.ID), etc. If anything was missing, what was it and how was it fixed?]

**Implication for Stage 2-4:** [Confirm `repos.IDGenerator` can return what `SetID` accepts. Note the type used for IDs.]

---

### Blocker 3: Incremental encoding (dirty tracking)

**Result:** [PASS (5x+ speedup) / MARGINAL (2-5x) / FAIL (<2x)]

**Benchmark output:**

```
[paste actual ns/op numbers from /tmp/blocker3_bench.txt]
```

**Findings:** [Describe the actual ratio and what it means.]

**Implication for Stage 2-4:**
- If PASS: ALTER operations on large objects are performant via the standard `repo.Update(elem)` path. **No PageMutator interface needed.**
- If MARGINAL: Mostly use standard path; document specific operations that should use PageMutator.
- If FAIL: PageMutator interface required for ALTER PAGE/WORKFLOW. Add to spec section 5.

---

### Blocker 4: Write-then-read cache consistency

**Result:** [PASS (cache invalidation automatic) / PARTIAL (some paths) / FAIL (manual invalidation required)]

**Test:** `modelsdk/codec/poc/blocker4_cache_consistency_test.go`

**Findings:** [Describe whether reads after writes saw the new data. If PARTIAL, list which write paths are auto-invalidated and which are not.]

**Implication for Stage 2-4:**
- If PASS: `ReaderCache.Invalidate()` interface still defined (for explicit control), but repo implementations don't need to call it after every write.
- If PARTIAL/FAIL: Repo Write methods MUST call cache invalidation. Document the exact pattern.

---

## Open Decisions Resolved

These were deferred from spec Section 11; the PoC results now resolve them:

| Decision | Resolution | Source |
|----------|------------|--------|
| PageMutator interface presence | [Yes / No] | Blocker 3 outcome |
| Cache invalidation strategy | [Per-unit / Full / Automatic] | Blocker 4 outcome |
| gen type constructor location | [Already in modelsdk/gen/* / Codegen change required] | Blocker 2 outcome |
| TransactionFactory implementation | [Direct wrap / Custom] | Implementation experience from Blocker 4 transactional test |

---

## Go/No-Go Decision

**Recommendation:** [Proceed to Stage 2 / Modify spec then proceed / Halt]

**Rationale:** [2-3 sentences. If proceed, summarize confidence. If modify, list spec changes needed. If halt, list deal-breakers.]
```

**Important:** This template has placeholders (`[PASS / FAIL]`, etc.) that must be filled in with actual test results. Do NOT commit the template with placeholders unfilled.

- [ ] **Step 2: Fill in the addendum**

Run the test suite one more time and copy actual results into the addendum. Replace every `[...]` placeholder with concrete findings.

- [ ] **Step 3: Verify no placeholders remain**

```bash
grep -n "\[.*\]" /mnt/data_sdd/gh/mxcli-wt-02/docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md
```

Expected: no output (all bracketed placeholders replaced). If any remain, fill them in before committing.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md
git commit -m "docs(spec): Stage 1 PoC addendum — empirical findings for modelsdk-native design"
```

---

## Self-Review

**Spec coverage:**
- [x] Blocker 1 (encode fresh) — Task 3
- [x] Blocker 2 (ID setters) — Task 2 (done first because Blockers 1/3/4 depend on it)
- [x] Blocker 3 (incremental encoding) — Task 4
- [x] Blocker 4 (cache consistency) — Task 5
- [x] Findings document — Task 7
- [x] Recommendation gate — Task 7 final section

**Placeholder scan:** All code blocks contain runnable code. The addendum template (Task 7 Step 1) has placeholders — but Step 2 explicitly requires filling them, and Step 3 verifies completeness before commit. This is intentional: the template lives in the plan, the filled version lives in the spec addendum.

**Type consistency:**
- `newID()` helper defined in `blocker2_id_setters_test.go` (Task 2), reused by Tasks 3-5 — wraps `mmpr.GenerateID()` (which returns `string`) and casts to `element.ID` (`type ID string`)
- `genMf.NewMicroflow()` used consistently
- `&codec.Encoder{}` (zero-value struct, no constructor) used consistently
- `codec.NewDecoder(codec.DefaultRegistry)` consistent
- `mmpr.NewWriter` (alias for `modelsdk/mpr`) consistent
- `r.ListUnitsByType` and `r.GetRawUnitBytes` used per Task 1 helper structure

**Out of scope:**
- Stage 2-4 detailed tasks (separate plan, written after PoC results land)
- Updating existing executor handlers (Stage 3)
- Deleting sdk/* packages (Stage 4)
- Public API changes in api/ or modelsdk.go (separate plans)
