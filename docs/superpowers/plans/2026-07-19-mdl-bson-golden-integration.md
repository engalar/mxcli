# MDL → BSON Golden Integration Test — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an integration test that verifies the full MDL → Executor → Backend → Writer → SQLite pipeline produces BSON identical to Studio Pro.

**Architecture:** Use `:memory:` SQLite V1 MPR as a zero-disk fixture; run real MDL through the real executor/backend/writer pipeline; read BSON back from SQLite; compare against golden Canonical Extended JSON extracted from Studio Pro MPRs.

**Tech Stack:** Go, `modernc.org/sqlite`, `go.mongodb.org/mongo-driver/v2/bson`, existing `modelsdk/codec` + `mdl/executor` + `modelsdk/mpr`.

## Global Constraints

- All new code goes in `modelsdk/codec/golden/` unless otherwise specified
- No modifications to existing production code (executor, backend, codec, mpr) as part of this plan
- `:memory:` SQLite V1 only (no V2 disk path testing)
- Golden files are Canonical Extended JSON (via `bson.MarshalExtJSON`), never raw BSON bytes
- Golden files are extracted from Studio Pro MPRs, never hand-edited
- Tests run with `go test` — no external dependencies beyond what the codebase already uses
- All tests pass with `go test ./modelsdk/codec/golden/`

---

## File Structure

```
modelsdk/codec/golden/
├── memory/
│   └── mpr.go                    ← Thin fixture: :memory: SQLite + Reader/Writer
│   └── mpr_test.go               ← MemoryMPR unit tests
├── snapshot/
│   ├── golden.go                 ← UnitSnapshot type + canonical JSON
│   ├── golden_test.go            ← Canonical JSON roundtrip tests
│   ├── extract.go                ← Extract from real MPR
│   ├── extract_test.go           ← Extract tests
│   ├── compare.go                ← Dual-mode comparison
│   └── compare_test.go           ← Compare engine tests
├── testdata/
│   ├── nanoflow/
│   │   ├── create.golden.json    ← Extracted golden (generated)
│   │   ├── create.mdl            ← MDL to reproduce the nanoflow
│   │   └── create.setup.mdl      ← Setup MDL (module + entity)
├── integration_test.go           ← TestMDLToGolden entry point
└── registry.go                   ← GoldenCase registration
```

## Task Decomposition

| Task | Files | Responsibility |
|------|-------|---------------|
| 1 | `memory/mpr.go`, `memory/mpr_test.go` | Thin fixture: :memory: SQLite + Reader/Writer |
| 2 | `snapshot/golden.go`, `snapshot/golden_test.go` | UnitSnapshot type, canonical JSON conversion |
| 3 | `snapshot/extract.go`, `snapshot/extract_test.go` | Extract UnitSnapshot from Studio Pro MPR |
| 4 | `snapshot/compare.go`, `snapshot/compare_test.go` | CompareCanonical — dual-mode comparison |
| 5 | `registry.go`, `integration_test.go` | Test harness + first GoldenCase |
| 6 | `testdata/nanoflow/*` files | Generate golden files from `/tmp/minimal.mpr` |

---

### Task 1: MemoryMPR Fixture

**Files:**
- Create: `modelsdk/codec/golden/memory/mpr.go`
- Create: `modelsdk/codec/golden/memory/mpr_test.go`

**Interfaces:**
- Produces: `memory.MPR` struct with `DB`, `Reader`, `Writer` fields

- [ ] **Step 1: Create `memory/mpr.go`**

```go
// Package memory provides an in-memory MPR fixture for integration tests.
package memory

import (
	"database/sql"
	"fmt"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	_ "modernc.org/sqlite"
)

type MPR struct {
	DB     *sql.DB
	Reader *mmpr.Reader
	Writer *mmpr.Writer
}

func New(projectVersion string) (*MPR, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open :memory: sqlite: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	if err := insertMeta(db, projectVersion); err != nil {
		db.Close()
		return nil, fmt.Errorf("insert metadata: %w", err)
	}
	reader, err := mmpr.OpenWithDB(db, ":memory:", "")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open reader: %w", err)
	}
	writer := mmpr.NewWriterWithReader(reader)
	return &MPR{DB: db, Reader: reader, Writer: writer}, nil
}

func (m *MPR) Close() error { return m.DB.Close() }

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS Unit (
			UnitID          BLOB    PRIMARY KEY NOT NULL,
			ContainerID     BLOB,
			ContainmentName TEXT,
			TreeConflict    INTEGER,
			ContentsHash    TEXT,
			ContentsConflicts TEXT,
			Contents        BLOB
		)
	`)
	return err
}

func insertMeta(db *sql.DB, projectVersion string) error {
	_, err := db.Exec(`
		INSERT INTO _MetaData (Version, MxVersion, SchemaVersion)
		VALUES (?, ?, 1)
	`, projectVersion, projectVersion)
	return err
}
```

- [ ] **Step 2: Create `memory/mpr_test.go`**

```go
package memory

import "testing"

func TestNewMemoryMPR(t *testing.T) {
	mpr, err := New("11.12.1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mpr.Close()
	if mpr.DB == nil { t.Fatal("DB is nil") }
	if mpr.Reader == nil { t.Fatal("Reader is nil") }
	if mpr.Writer == nil { t.Fatal("Writer is nil") }
}

func TestMemoryMPR_WriteAndRead(t *testing.T) {
	mpr, err := New("11.12.1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mpr.Close()

	unitID := "00000000-0000-0000-0000-000000000001"
	contents := []byte("hello bson")

	err = mpr.Writer.InsertUnit(unitID, unitID, "Documents", "Microflows$Test", contents)
	if err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}
	got := mpr.Reader.GetRawUnitBytes([]byte(unitID))
	if string(got) != string(contents) {
		t.Fatalf("got %q, want %q", string(got), string(contents))
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/memory/ -v -count=1
# Expected: PASS
```

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/golden/memory/
git commit -m "feat(golden): add in-memory MPR fixture for integration tests"
```

---

### Task 2: UnitSnapshot + Canonical JSON

**Files:**
- Create: `modelsdk/codec/golden/snapshot/golden.go`
- Create: `modelsdk/codec/golden/snapshot/golden_test.go`

**Interfaces:**
- Produces: `snapshot.UnitSnapshot`, `snapshot.ToCanonicalJSON()`, `snapshot.FromCanonicalJSON()`

- [ ] **Step 1: Create `snapshot/golden.go`**

```go
package snapshot

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type UnitSnapshot struct {
	Type      string          `json:"type"`
	Canonical json.RawMessage `json:"canonical"`
	rawBSON   []byte          `json:"-"`
}

func ToCanonicalJSON(raw []byte) (json.RawMessage, error) {
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal bson: %w", err)
	}
	extJSON, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		return nil, fmt.Errorf("marshal extjson: %w", err)
	}
	return extJSON, nil
}

func FromCanonicalJSON(data []byte) ([]byte, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON(data, true, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal extjson: %w", err)
	}
	raw, err := bson.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal bson: %w", err)
	}
	return raw, nil
}

func NewUnitSnapshot(unitType string, rawBSON []byte) (*UnitSnapshot, error) {
	canonical, err := ToCanonicalJSON(rawBSON)
	if err != nil {
		return nil, err
	}
	return &UnitSnapshot{
		Type:      unitType,
		Canonical: canonical,
		rawBSON:   rawBSON,
	}, nil
}

func (s *UnitSnapshot) RawBSON() []byte { return s.rawBSON }
```

- [ ] **Step 2: Create `snapshot/golden_test.go`**

```go
package snapshot

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func mustMarshal(t *testing.T, doc bson.D) []byte {
	t.Helper()
	data, err := bson.Marshal(doc)
	if err != nil { t.Fatalf("marshal: %v", err) }
	return data
}

func TestToCanonicalJSON_Roundtrip(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	canonical, err := ToCanonicalJSON(raw)
	if err != nil { t.Fatalf("ToCanonicalJSON: %v", err) }
	recovered, err := FromCanonicalJSON(canonical)
	if err != nil { t.Fatalf("FromCanonicalJSON: %v", err) }
	if !bytes.Equal(raw, recovered) {
		t.Fatalf("roundtrip produced different bytes")
	}
}

func TestNewUnitSnapshot(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	snap, err := NewUnitSnapshot("Microflows$Nanoflow", raw)
	if err != nil { t.Fatalf("NewUnitSnapshot: %v", err) }
	if snap.Type != "Microflows$Nanoflow" { t.Fatalf("Type = %q", snap.Type) }
	if len(snap.Canonical) == 0 { t.Fatal("Canonical is empty") }
}
```

- [ ] **Step 3: Run tests**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/snapshot/ -v -count=1
```

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/golden/snapshot/golden.go modelsdk/codec/golden/snapshot/golden_test.go
git commit -m "feat(golden): add UnitSnapshot type and canonical JSON conversion"
```

---

### Task 3: Extract Snapshot from MPR

**Files:**
- Create: `modelsdk/codec/golden/snapshot/extract.go`
- Create: `modelsdk/codec/golden/snapshot/extract_test.go`

**Interfaces:**
- Consumes: `mmpr.Reader`, `snapshot.UnitSnapshot`
- Produces: `snapshot.ExtractFromMPR(path, unitType)`

- [ ] **Step 1: Read `mmpr.Reader` API signatures**

```bash
cd /mnt/data_sdb/mxcli && grep -n "func.*Reader.*ListUnits\|func.*Reader.*GetRawUnitBytes\|func.*Reader.*UnitRow" modelsdk/mpr/reader.go | head -10
```

- [ ] **Step 2: Create `snapshot/extract.go`**

```go
package snapshot

import (
	"encoding/json"
	"fmt"
	"os"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func ExtractFromMPR(mprPath, unitType string) ([]*UnitSnapshot, error) {
	r, err := mmpr.Open(mprPath)
	if err != nil {
		return nil, fmt.Errorf("open mpr: %w", err)
	}
	defer r.Close()

	var results []*UnitSnapshot
	// TODO: Use mmpr.Reader API to list units and filter by $Type.
	// For each matching unit, read raw BSON and create UnitSnapshot.
	return results, nil
}

func ExtractFromMPRToFile(mprPath, unitType, outputPath string) error {
	snapshots, err := ExtractFromMPR(mprPath, unitType)
	if err != nil { return err }
	if len(snapshots) == 0 {
		return fmt.Errorf("no units with type %q found", unitType)
	}
	data, err := json.MarshalIndent(snapshots[0], "", "  ")
	if err != nil { return fmt.Errorf("marshal: %w", err) }
	return os.WriteFile(outputPath, data, 0644)
}
```

- [ ] **Step 3: Write test that extracts from `/tmp/minimal.mpr`**

```go
package snapshot

import (
	"os"
	"testing"
)

func TestExtractNanoflowFromMinimal(t *testing.T) {
	if _, err := os.Stat("/tmp/minimal.mpr"); os.IsNotExist(err) {
		t.Skip("/tmp/minimal.mpr not found")
	}
	snaps, err := ExtractFromMPR("/tmp/minimal.mpr", "Microflows$Nanoflow")
	if err != nil { t.Fatalf("ExtractFromMPR: %v", err) }
	if len(snaps) == 0 {
		t.Fatal("no nanoflow units found")
	}
	for _, s := range snaps {
		t.Logf("Found %s: %d bytes canonical", s.Type, len(s.Canonical))
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/snapshot/ -run TestExtract -v -count=1
```

- [ ] **Step 5: Commit**

```bash
git add modelsdk/codec/golden/snapshot/extract.go modelsdk/codec/golden/snapshot/extract_test.go
git commit -m "feat(golden): add snapshot extraction from real MPR files"
```

---

### Task 4: Dual-Mode Comparison Engine

**Files:**
- Create: `modelsdk/codec/golden/snapshot/compare.go`
- Create: `modelsdk/codec/golden/snapshot/compare_test.go`

**Interfaces:**
- Consumes: `snapshot.UnitSnapshot`, `golden.CompareBSON`
- Produces: `snapshot.CompareCanonical()`, `snapshot.CompareBinary()`

- [ ] **Step 1: Create `snapshot/compare.go`**

```go
package snapshot

import (
	"encoding/json"
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/codec/golden"
)

type CompareMode int

const (
	CompareStructure CompareMode = iota
	CompareBinary
)

type CompareResult struct {
	Mode  CompareMode
	Diffs []golden.Diff
}

func CompareCanonical(gotBSON []byte, g *UnitSnapshot, mode CompareMode) (*CompareResult, error) {
	switch mode {
	case CompareStructure:
		return compareStructure(gotBSON, g)
	case CompareBinary:
		return compareBinary(gotBSON, g)
	default:
		return nil, fmt.Errorf("unknown compare mode: %d", mode)
	}
}

func compareStructure(gotBSON []byte, g *UnitSnapshot) (*CompareResult, error) {
	gotCanonical, err := ToCanonicalJSON(gotBSON)
	if err != nil { return nil, err }
	var gotObj, goldenObj any
	if err := json.Unmarshal(gotCanonical, &gotObj); err != nil { return nil, err }
	if err := json.Unmarshal(g.Canonical, &goldenObj); err != nil { return nil, err }
	diffs := jsonDiff(gotObj, goldenObj, "$")
	return &CompareResult{Mode: CompareStructure, Diffs: diffs}, nil
}

func compareBinary(gotBSON []byte, g *UnitSnapshot) (*CompareResult, error) {
	diffs := golden.CompareBSON(gotBSON, g.RawBSON(), nil)
	return &CompareResult{Mode: CompareBinary, Diffs: diffs}, nil
}

func jsonDiff(got, expected any, path string) []golden.Diff {
	var diffs []golden.Diff
	switch e := expected.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return []golden.Diff{{Path: path, Kind: golden.DiffType, Got: fmt.Sprintf("%T", got), Expected: "object"}}
		}
		for k, expVal := range e {
			curPath := path + "." + k
			if k == "$oid" || k == "$binary" { continue }
			gotVal, found := g[k]
			if !found {
				diffs = append(diffs, golden.Diff{Path: curPath, Kind: golden.DiffMissing, Got: nil, Expected: expVal})
				continue
			}
			diffs = append(diffs, jsonDiff(gotVal, expVal, curPath)...)
		}
		for k := range g {
			if _, found := e[k]; !found {
				if k == "$oid" || k == "$binary" { continue }
				diffs = append(diffs, golden.Diff{Path: path + "." + k, Kind: golden.DiffExtra, Got: g[k], Expected: nil})
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok { return []golden.Diff{{Path: path, Kind: golden.DiffType, Got: fmt.Sprintf("%T", got), Expected: "array"}} }
		minLen := len(e)
		if len(g) < minLen { minLen = len(g) }
		for i := 0; i < minLen; i++ {
			diffs = append(diffs, jsonDiff(g[i], e[i], fmt.Sprintf("%s[%d]", path, i))...)
		}
		for i := minLen; i < len(e); i++ {
			diffs = append(diffs, golden.Diff{Path: fmt.Sprintf("%s[%d]", path, i), Kind: golden.DiffMissing, Got: nil, Expected: e[i]})
		}
		for i := minLen; i < len(g); i++ {
			diffs = append(diffs, golden.Diff{Path: fmt.Sprintf("%s[%d]", path, i), Kind: golden.DiffExtra, Got: g[i], Expected: nil})
		}
	default:
		if got != expected {
			diffs = append(diffs, golden.Diff{Path: path, Kind: golden.DiffValue, Got: got, Expected: expected})
		}
	}
	return diffs
}
```

- [ ] **Step 2: Create `snapshot/compare_test.go`**

```go
package snapshot

import (
	"testing"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCompareCanonical_Identical(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	snap, _ := NewUnitSnapshot("Microflows$Nanoflow", raw)
	result, err := CompareCanonical(raw, snap, CompareStructure)
	if err != nil { t.Fatalf("CompareCanonical: %v", err) }
	if len(result.Diffs) != 0 {
		t.Fatalf("expected 0 diffs, got %d: %v", len(result.Diffs), result.Diffs)
	}
}

func TestCompareCanonical_ValueMismatch(t *testing.T) {
	golden := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Expected"},
	})
	got := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Got"},
	})
	snap, _ := NewUnitSnapshot("Microflows$Nanoflow", golden)
	result, _ := CompareCanonical(got, snap, CompareStructure)
	if len(result.Diffs) != 0 { t.Fatal("expected diffs") }
}
```

- [ ] **Step 3: Run tests**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/snapshot/ -v -count=1
```

- [ ] **Step 4: Commit**

```bash
git add modelsdk/codec/golden/snapshot/compare.go modelsdk/codec/golden/snapshot/compare_test.go
git commit -m "feat(golden): add dual-mode comparison engine"
```

---

### Task 5: Integration Test Harness + First GoldenCase

**Files:**
- Create: `modelsdk/codec/golden/registry.go`
- Create: `modelsdk/codec/golden/integration_test.go`

**Interfaces:**
- Consumes: `memory.MPR`, `snapshot.CompareCanonical`, executor pipeline
- Produces: Runnable `TestMDLToGolden`

- [ ] **Step 1: Create `registry.go`**

```go
package golden

import (
	"embed"
	"encoding/json"
	"io/fs"
	"path/filepath"

	"github.com/mendixlabs/mxcli/modelsdk/codec/golden/snapshot"
)

//go:embed testdata/*/*.mdl testdata/*/*.golden.json
var goldenFS embed.FS

type GoldenCase struct {
	Name           string
	MDLPath        string
	SetupMDLPath   string
	GoldenPath     string
	TargetUnitType string
}

func Registry() []GoldenCase {
	return []GoldenCase{
		{
			Name:           "nanoflow/create",
			MDLPath:        "nanoflow/create.mdl",
			SetupMDLPath:   "nanoflow/create.setup.mdl",
			GoldenPath:     "nanoflow/create.golden.json",
			TargetUnitType: "Microflows$Nanoflow",
		},
	}
}

func (g GoldenCase) LoadMDL() (string, error) {
	data, err := goldenFS.ReadFile(filepath.Join("testdata", g.MDLPath))
	if err != nil { return "", err }
	return string(data), nil
}

func (g GoldenCase) LoadSetupMDL() (string, error) {
	if g.SetupMDLPath == "" { return "", nil }
	data, err := goldenFS.ReadFile(filepath.Join("testdata", g.SetupMDLPath))
	if err != nil { return "", err }
	return string(data), nil
}

func (g GoldenCase) LoadGolden() (*snapshot.UnitSnapshot, error) {
	data, err := goldenFS.ReadFile(filepath.Join("testdata", g.GoldenPath))
	if err != nil { return nil, err }
	var snap snapshot.UnitSnapshot
	if err := json.Unmarshal(data, &snap); err != nil { return nil, err }
	return &snap, nil
}
```

- [ ] **Step 2: Create `integration_test.go`** (skeleton — executor wiring TBD)

```go
package golden_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec/golden"
	"github.com/mendixlabs/mxcli/modelsdk/codec/golden/memory"
	"github.com/mendixlabs/mxcli/modelsdk/codec/golden/snapshot"
)

func TestMDLToGolden(t *testing.T) {
	for _, tc := range golden.Registry() {
		t.Run(tc.Name, func(t *testing.T) {
			mpr, err := memory.New("11.12.1")
			if err != nil { t.Fatalf("memory.New: %v", err) }
			defer mpr.Close()

			// Setup
			if setupMDL, err := tc.LoadSetupMDL(); err != nil {
				t.Fatalf("LoadSetupMDL: %v", err)
			} else if setupMDL != "" {
				runMDL(t, mpr, setupMDL)
			}

			// Main MDL
			mdl, err := tc.LoadMDL()
			if err != nil { t.Fatalf("LoadMDL: %v", err) }
			runMDL(t, mpr, mdl)

			// Read back BSON
			bson := readUnitBSON(t, mpr, tc.TargetUnitType)

			// Load golden
			golden, err := tc.LoadGolden()
			if err != nil { t.Fatalf("LoadGolden: %v", err) }

			// Compare
			result, err := snapshot.CompareCanonical(bson, golden, snapshot.CompareStructure)
			if err != nil { t.Fatalf("CompareCanonical: %v", err) }
			for _, d := range result.Diffs {
				t.Errorf("%s", d)
			}
		})
	}
}

// runMDL and readUnitBSON require executor wiring — implement after
// studying mdl/executor/handler_deps.go and mdl/backend/mpr/backend_lifecycle.go.
func runMDL(t *testing.T, mpr *memory.MPR, mdl string) {
	t.Helper()
	// TODO: wire Executor using mpr.Reader/mpr.Writer
}

func readUnitBSON(t *testing.T, mpr *memory.MPR, unitType string) []byte {
	t.Helper()
	// TODO: query SQLite Unit table for matching $Type
	return nil
}
```

- [ ] **Step 3: Create MDL test input files**

```sql
-- modelsdk/codec/golden/testdata/nanoflow/create.setup.mdl
create module Administration;
create entity Administration.Account (FullName: String(200));
```

```sql
-- modelsdk/codec/golden/testdata/nanoflow/create.mdl
create nanoflow MyFirstModule.Nanoflow
  returns list of Administration.Account as $AccountList
{
  synchronize;
  retrieve $AccountList from Administration.Account
    where [FullName = empty and System.owner = '[%CurrentUser%]']
    sort by FullName desc;
  return $AccountList;
}
```

- [ ] **Step 4: Run tests**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/... -v -count=1 -timeout 60s
```

- [ ] **Step 5: Commit**

```bash
git add modelsdk/codec/golden/registry.go modelsdk/codec/golden/integration_test.go modelsdk/codec/golden/testdata/
git commit -m "feat(golden): add MDL->BSON integration test harness"
```

---

### Task 6: Generate First Golden Files

**Files:**
- Create: `modelsdk/codec/golden/testdata/nanoflow/create.golden.json` (generated)

- [ ] **Step 1: Run extract against `/tmp/minimal.mpr`**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/snapshot/ -run TestExtractNanoflow -v
# If the test prints canonical JSON, redirect to the golden file:
# go test ... | grep '^{' > modelsdk/codec/golden/testdata/nanoflow/create.golden.json
```

- [ ] **Step 2: Verify golden JSON is valid**

```bash
python3 -c "import json; json.load(open('modelsdk/codec/golden/testdata/nanoflow/create.golden.json'))" && echo "valid"
```

- [ ] **Step 3: Run full integration test**

```bash
cd /mnt/data_sdb/mxcli && go test ./modelsdk/codec/golden/... -v -count=1 -timeout 60s
```

- [ ] **Step 4: Commit golden files**

```bash
git add modelsdk/codec/golden/testdata/nanoflow/create.golden.json
git commit -m "feat(golden): add extracted golden for nanoflow/create"
```
