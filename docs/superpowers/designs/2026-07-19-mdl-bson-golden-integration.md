# MDL → BSON Golden Integration Test — Architecture Design

> **Status:** Design complete, awaiting implementation
> **Driving principle:** Test the full MDL → Executor → Backend → Writer → SQLite pipeline against Studio Pro golden BSON, using only in-memory fixtures.

---

## Architecture Overview

```
                 Studio Pro
                     │
                     ▼
               ExtractSnapshot
                     │
                     ▼
          testdata/*.golden.json    ← Canonical Extended JSON (not raw BSON)
                     ▲
                     │
        CompareCanonical (dual-mode)
                     ▲
                     │
      Read Unit.Content from :memory: SQLite
                     ▲
                     │
              mmpr.Writer (real, not mocked)
                     ▲
                     │
                 Backend (MprBackend, real)
                     ▲
                     │
                 Executor (real)
                     ▲
                     │
                    MDL
```

## Key Design Decisions

### 1. Real Pipeline, No Shortcuts

The test MUST go through the real `mmpr.Writer` → SQLite path. No `RecordingSink`, no `SetSessionBuf` interception for production tests:

```
MDL → Executor → Backend → Repo → mmpr.Writer → SQLite(:memory:)
```

`SetSessionBuf` is reserved as a `GOLDEN_DEBUG=1` debugging tool only.

### 2. :memory: SQLite V1 as Test Fixture

- Use `sql.Open("sqlite", ":memory:")` — already supported by existing `mmpr.OpenWithDB()` and `mmpr.NewWriterFromDB()`
- V1 format (single `Unit` table, no `mprcontents/` directory)
- No disk I/O for the test MPR itself
- Fixture is thin — only creates DB + tables + Reader/Writer, no extra methods

### 3. Canonical Extended JSON as Golden Format

Golden files are MongoDB Canonical Extended JSON (`bson.MarshalExtJSON` with `canonical=true`), NOT raw BSON bytes:

```
golden.nanoflow.create.json:
{
  "canonical": {
    "$ID": {"$binary":{"base64":"...","subType":"00"}},
    "$Type": "Microflows$Nanoflow",
    "Name": "Nanoflow",
    ...
  }
}
```

**Rationale:**
- Git diff is human-readable
- Type information preserved (`{"$numberInt":"42"}` stays int64)
- Bidirectional conversion with `bson.UnmarshalExtJSON`
- Reviewable in PRs

### 4. Dual-Mode Comparison

| Mode | Scope | When | Sensitivity |
|------|-------|------|-------------|
| `CompareStructure` | Fields, order, types, values (skip $ID) | Every PR | Medium |
| `CompareBinary` | Byte-for-byte exact (including $ID) | Nightly | High |

### 5. Snapshot Source Discipline

- **Only one source of truth**: Studio Pro MPR → `ExtractSnapshot` → `golden.json`
- **Never hand-edit golden files**
- **Update via `GOLDEN_UPDATE=1`** environment variable

---

## Package Structure

```
modelsdk/codec/golden/
├── memory/
│   └── mpr.go              ← Thin fixture: :memory: SQLite + Reader/Writer
├── snapshot/
│   ├── golden.go            ← UnitSnapshot + canonical JSON conversion
│   ├── extract.go           ← Extract UnitSnapshot from real MPR
│   └── compare.go           ← CompareCanonical, CompareBinary
├── testdata/
│   └── nanoflow/
│       ├── create.golden.json
│       ├── create.mdl
│       └── create.setup.mdl
├── registry.go              ← GoldenCase definitions
├── integration_test.go      ← TestMDLToGolden entry point
```

## Key Types

```go
// memory.MPR — in-memory MPR fixture
type MPR struct {
    DB     *sql.DB
    Reader *mmpr.Reader
    Writer *mmpr.Writer
}

// snapshot.UnitSnapshot — golden snapshot of one BSON document
type UnitSnapshot struct {
    Type      string          `json:"type"`
    Canonicl  json.RawMessage `json:"canonical"`
    rawBSON   []byte
}

// snapshot.CompareMode
type CompareMode int
const (
    CompareStructure CompareMode = iota
    CompareBinary
)
```

## Test Flow

```go
func TestMDLToGolden(t *testing.T) {
    mpr := memory.New("11.12.1")
    defer mpr.Close()

    // 1. Run setup MDL (create module, entities)
    runMDL(t, mpr, setupMDL)

    // 2. Run main MDL (create nanoflow)
    runMDL(t, mpr, mainMDL)

    // 3. Read BSON from SQLite
    bson := readUnitBSON(mpr.DB, "Microflows$Nanoflow")

    // 4. Load golden snapshot
    golden := loadGolden("testdata/nanoflow/create.golden.json")

    // 5. Compare
    result := CompareCanonical(bson, golden, CompareStructure)
}
```

## Updating Golden Files

```bash
# From Studio Pro MPR
GOLDEN_EXTRACT=mpr=/tmp/minimal.mpr,type=Microflows$Nanoflow \
    go test ./modelsdk/codec/golden -run TestExtract

# From test output (when intentional changes are verified)
GOLDEN_UPDATE=1 go test ./modelsdk/codec/golden -run TestMDLToGolden
```

## CI Integration

| Stage | Compare Mode | Frequency |
|-------|-------------|-----------|
| PR | `CompareStructure` | Every push |
| Nightly | `CompareBinary` | Daily |

## Principles Applied

| Decision | Principle |
|----------|-----------|
| Real mmpr.Writer + SQLite path | DIP: depend on abstractions, not shortcuts |
| SetSessionBuf = debug only | YAGNI: don't add complexity until needed |
| Canonical JSON golden files | Ubiquitous Language: Golden = Studio Pro output |
| Dual-mode comparison | OCP: extend comparison without modifying core |
| Snapshot extracted, not hand-edited | DRY: single source of truth |
| Thin MemoryMPR fixture | SRP: one responsibility per component |
