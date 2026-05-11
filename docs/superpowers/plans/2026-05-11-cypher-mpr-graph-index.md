# Cypher MPR Graph Index Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `mdl/catalog/` with a PuppyGraph-backed Cypher graph layer that indexes MPR elements into a temp SQLite file, enabling multi-hop graph queries and in-memory batch mutations with single-Flush write-back via modelsdk.

**Architecture:** modelsdk walks all MPR units at open time → writes nodes/edges to a temp SQLite file → PuppyGraph reads the temp file via JDBC and exposes Cypher → results (UUIDs) are fed back to modelsdk mutators → single `Flush()` writes all dirty elements to the MPR atomically. MDL remains the 0→1 creation tool; Cypher handles query+bulk-modify.

**Tech Stack:** Go, `modernc.org/sqlite` (driver name `"sqlite"`), PuppyGraph HTTP REST API (Bolt-compatible, runs as Docker container on port 8080/7687), standard `net/http` for adapter (no extra dependencies).

---

## File Structure

| File | Role |
|------|------|
| `mdl/graphdb/interface.go` | `GraphDB` interface + `Record` type (read-only contract) |
| `mdl/graphdb/schema.go` | DDL constants + `SetupDB(db)` to create tables + `LabelFromType(unitType)` |
| `mdl/graphdb/indexer.go` | `Indexer`: walks `modelsdk.Model` → populates temp SQLite |
| `mdl/graphdb/indexer_test.go` | Unit test: open `testdata/expr-checker/minimal.mpr`, assert node + edge counts |
| `mdl/graphdb/puppygraph.go` | `PuppyGraphAdapter`: registers SQLite file as PuppyGraph catalog, runs Cypher via HTTP |
| `mdl/graphdb/puppygraph_test.go` | Interface compliance + nil-safety tests (no real PuppyGraph needed) |
| `mdl/graphdb/mutator.go` | `BatchMutator`: Cypher find → modelsdk mutate → Flush |
| `mdl/graphdb/mutator_test.go` | Unit test with stub `GraphDB` |
| `cmd/mxcli/cmd_cypher.go` | `mxcli cypher -p app.mpr "MATCH ..."` CLI command |

---

## Task 1: GraphDB Interface + Record Type

**Files:**
- Create: `mdl/graphdb/interface.go`

- [ ] **Step 1: Write the interface**

```go
// File: mdl/graphdb/interface.go
package graphdb

import "context"

// Record is one row from a Cypher RETURN clause.
// Keys are the return variable names (e.g. "e.name", "mf.id").
type Record map[string]any

// GraphDB is the unified Cypher execution interface.
// The concrete implementation (PuppyGraphAdapter) connects to PuppyGraph.
// Tests use stub implementations.
type GraphDB interface {
	// RunCypher executes a Cypher query and returns all matching rows.
	RunCypher(ctx context.Context, cypher string, params map[string]any) ([]Record, error)
	// Close releases any resources held by the adapter.
	Close() error
}
```

- [ ] **Step 2: Verify the package compiles**

```bash
go build ./mdl/graphdb/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add mdl/graphdb/interface.go
git commit -m "feat(graphdb): GraphDB interface + Record type"
```

---

## Task 2: SQLite Schema + Label Mapping

**Files:**
- Create: `mdl/graphdb/schema.go`

- [ ] **Step 1: Write the failing test**

Create `mdl/graphdb/schema_test.go`:

```go
package graphdb

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSetupDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := SetupDB(db); err != nil {
		t.Fatalf("SetupDB: %v", err)
	}

	// Verify tables exist
	for _, tbl := range []string{"graph_nodes", "graph_edges"} {
		var count int
		err := db.QueryRow("SELECT count(*) FROM " + tbl).Scan(&count)
		if err != nil {
			t.Errorf("table %s not found: %v", tbl, err)
		}
	}
}

func TestLabelFromType(t *testing.T) {
	cases := []struct {
		unitType string
		want     string
	}{
		{"DomainModels$Entity", "Entity"},
		{"Microflows$Microflow", "Microflow"},
		{"Pages$Page", "Page"},
		{"Projects$Module", "Module"},
		{"DomainModels$Association", "Association"},
		{"Enumerations$Enumeration", "Enumeration"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := LabelFromType(tc.unitType)
		if got != tc.want {
			t.Errorf("LabelFromType(%q) = %q, want %q", tc.unitType, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/graphdb/... -run TestSetupDB -v
```

Expected: FAIL with "undefined: SetupDB".

- [ ] **Step 3: Implement schema.go**

```go
// File: mdl/graphdb/schema.go
package graphdb

import (
	"database/sql"
	"strings"
)

const ddl = `
CREATE TABLE IF NOT EXISTS graph_nodes (
    id        TEXT PRIMARY KEY,
    label     TEXT NOT NULL,
    name      TEXT,
    module    TEXT
);
CREATE TABLE IF NOT EXISTS graph_edges (
    from_id   TEXT NOT NULL,
    to_id     TEXT NOT NULL,
    rel_type  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_nodes_label ON graph_nodes(label);
CREATE INDEX IF NOT EXISTS idx_edges_from  ON graph_edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to    ON graph_edges(to_id);
`

// SetupDB creates the graph_nodes and graph_edges tables in db.
// Safe to call on an already-initialized database (uses IF NOT EXISTS).
func SetupDB(db *sql.DB) error {
	_, err := db.Exec(ddl)
	return err
}

// allowedLabels is the set of MPR unit types that become graph nodes.
// Keys are unit type strings (after the "$"); values are the Cypher labels.
var allowedLabels = map[string]string{
	"Module":      "Module",
	"Entity":      "Entity",
	"Attribute":   "Attribute",
	"Association": "Association",
	"Microflow":   "Microflow",
	"Nanoflow":    "NanoFlow",
	"Page":        "Page",
	"Enumeration": "Enumeration",
	"EnumValue":   "EnumValue",
}

// LabelFromType maps a modelsdk unit type (e.g. "DomainModels$Entity")
// to a Cypher node label (e.g. "Entity"). Returns "" for unknown types.
func LabelFromType(unitType string) string {
	idx := strings.LastIndex(unitType, "$")
	if idx < 0 {
		return ""
	}
	suffix := unitType[idx+1:]
	return allowedLabels[suffix]
}

// containmentRelType returns the HAS_* relationship type for a child node label.
// E.g. "Entity" → "HAS_ENTITY".
func containmentRelType(label string) string {
	return "HAS_" + strings.ToUpper(label)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./mdl/graphdb/... -run TestSetupDB -run TestLabelFromType -v
```

Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
git add mdl/graphdb/schema.go mdl/graphdb/schema_test.go
git commit -m "feat(graphdb): SQLite schema DDL + label mapper"
```

---

## Task 3: Indexer — Build Graph Index from modelsdk

**Files:**
- Create: `mdl/graphdb/indexer.go`
- Create: `mdl/graphdb/indexer_test.go`

The Indexer does two passes:
1. **Node pass**: iterate `model.Units()` → write `graph_nodes` rows (uses `codec.UnitInfo.Type`, `.Name`; resolves module via `model.ResolveModuleName`)
2. **Edge pass**: containment edges from `UnitInfo.ContainerID`; association cross-reference edges from BSON scan of `DomainModels$Association` units

- [ ] **Step 1: Write the failing test**

```go
// File: mdl/graphdb/indexer_test.go
package graphdb

import (
	"database/sql"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk"
	_ "modernc.org/sqlite"
)

func TestIndexer_Build(t *testing.T) {
	// Uses the checked-in minimal MPR from the expression-checker testdata.
	m, err := modelsdk.Open("../../testdata/expr-checker/minimal.mpr")
	if err != nil {
		t.Fatalf("open MPR: %v", err)
	}
	defer m.Close()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := SetupDB(db); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer(db)
	if err := idx.Build(m); err != nil {
		t.Fatalf("Build: %v", err)
	}

	var nodeCount int
	if err := db.QueryRow("SELECT count(*) FROM graph_nodes").Scan(&nodeCount); err != nil {
		t.Fatal(err)
	}
	if nodeCount == 0 {
		t.Error("expected graph_nodes to have rows, got 0")
	}

	var edgeCount int
	if err := db.QueryRow("SELECT count(*) FROM graph_edges").Scan(&edgeCount); err != nil {
		t.Fatal(err)
	}
	if edgeCount == 0 {
		t.Error("expected graph_edges to have rows, got 0")
	}

	t.Logf("indexed %d nodes, %d edges", nodeCount, edgeCount)

	// Spot-check: containment edge should exist (Module → Entity)
	var hasContainment int
	err = db.QueryRow(`
		SELECT count(*) FROM graph_edges
		WHERE rel_type LIKE 'HAS_%'
	`).Scan(&hasContainment)
	if err != nil {
		t.Fatal(err)
	}
	if hasContainment == 0 {
		t.Error("expected HAS_* containment edges, got none")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/graphdb/... -run TestIndexer_Build -v
```

Expected: FAIL with "undefined: NewIndexer".

- [ ] **Step 3: Implement indexer.go**

```go
// File: mdl/graphdb/indexer.go
package graphdb

import (
	"database/sql"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// Indexer populates graph_nodes and graph_edges in a SQLite database
// from a modelsdk.Model. The database must already have the schema set up
// via SetupDB.
type Indexer struct {
	db *sql.DB
}

// NewIndexer creates an Indexer that writes to db.
func NewIndexer(db *sql.DB) *Indexer {
	return &Indexer{db: db}
}

// Build performs two passes over the model:
//  1. Writes a graph_nodes row for every unit whose type maps to a Cypher label.
//  2. Writes graph_edges rows for containment (from ContainerID) and
//     association cross-references (ParentPointer / ChildPointer BSON fields).
func (idx *Indexer) Build(m *modelsdk.Model) error {
	units := m.Units()

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	nodeStmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO graph_nodes(id, label, name, module) VALUES(?,?,?,?)`)
	if err != nil {
		return err
	}
	defer nodeStmt.Close()

	edgeStmt, err := tx.Prepare(
		`INSERT INTO graph_edges(from_id, to_id, rel_type) VALUES(?,?,?)`)
	if err != nil {
		return err
	}
	defer edgeStmt.Close()

	// Pass 1: nodes
	for _, u := range units {
		label := LabelFromType(u.Type)
		if label == "" {
			continue
		}
		module := m.ResolveModuleName(u.ContainerID)
		if _, err := nodeStmt.Exec(string(u.ID), label, u.Name, module); err != nil {
			return err
		}
	}

	// Pass 2: edges — containment + association cross-refs
	for _, u := range units {
		label := LabelFromType(u.Type)
		if label == "" {
			continue
		}

		// Containment edge: container → this unit
		if u.ContainerID != "" {
			relType := containmentRelType(label)
			if _, err := edgeStmt.Exec(string(u.ContainerID), string(u.ID), relType); err != nil {
				return err
			}
		}

		// Cross-reference edges for associations
		if strings.HasSuffix(u.Type, "$Association") {
			if err := idx.addAssociationEdges(m, u, edgeStmt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// addAssociationEdges scans the BSON of an association unit for
// ParentPointer and ChildPointer UUID fields and writes FROM_ENTITY / TO_ENTITY edges.
func (idx *Indexer) addAssociationEdges(m *modelsdk.Model, u codec.UnitInfo, stmt *sql.Stmt) error {
	m.ScanUnitStrings(u.ID, func(fieldPath, value string) bool {
		// UUID-shaped values that look like references (36 chars, hyphens at positions 8,13,18,23)
		if !looksLikeUUID(value) {
			return true
		}
		switch fieldPath {
		case "ParentPointer":
			stmt.Exec(string(u.ID), value, "FROM_ENTITY") //nolint:errcheck
		case "ChildPointer":
			stmt.Exec(string(u.ID), value, "TO_ENTITY") //nolint:errcheck
		}
		return true
	})
	return nil
}

// looksLikeUUID returns true if s is a standard 8-4-4-4-12 UUID string.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// elementIDStr extracts the string form of an element ID.
func elementIDStr(id element.ID) string { return string(id) }
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./mdl/graphdb/... -run TestIndexer_Build -v
```

Expected: PASS. Log line shows node/edge counts > 0.

- [ ] **Step 5: Commit**

```bash
git add mdl/graphdb/indexer.go mdl/graphdb/indexer_test.go
git commit -m "feat(graphdb): Indexer — walk modelsdk units into graph SQLite"
```

---

## Task 4: PuppyGraph HTTP Adapter

**Files:**
- Create: `mdl/graphdb/puppygraph.go`
- Create: `mdl/graphdb/puppygraph_test.go`

PuppyGraph exposes:
- `PUT http://HOST:8080/schema` — register catalogs + vertex/edge mappings
- `POST http://HOST:8080/cypher` — run Cypher, returns JSON rows

No extra Go dependencies — uses `net/http` and `encoding/json` (already in stdlib).

- [ ] **Step 1: Write the failing test**

```go
// File: mdl/graphdb/puppygraph_test.go
package graphdb

import (
	"context"
	"testing"
)

// TestPuppyGraphAdapter_implements verifies the adapter satisfies the interface
// at compile time (no PuppyGraph server needed).
func TestPuppyGraphAdapter_implements(t *testing.T) {
	var _ GraphDB = (*PuppyGraphAdapter)(nil)
}

// TestNewPuppyGraphAdapter_defaults verifies the constructor sets expected fields.
func TestNewPuppyGraphAdapter_defaults(t *testing.T) {
	a := NewPuppyGraphAdapter("http://localhost:8080", "")
	if a.endpoint != "http://localhost:8080" {
		t.Errorf("endpoint = %q, want %q", a.endpoint, "http://localhost:8080")
	}
}

// TestPuppyGraphAdapter_Close_noError verifies Close() is safe to call.
func TestPuppyGraphAdapter_Close_noError(t *testing.T) {
	a := NewPuppyGraphAdapter("http://localhost:8080", "")
	if err := a.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

// stubGraphDB is an in-test implementation for BatchMutator tests.
type stubGraphDB struct {
	rows []Record
	err  error
}

func (s *stubGraphDB) RunCypher(_ context.Context, _ string, _ map[string]any) ([]Record, error) {
	return s.rows, s.err
}
func (s *stubGraphDB) Close() error { return nil }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/graphdb/... -run TestPuppyGraphAdapter -v
```

Expected: FAIL with "undefined: PuppyGraphAdapter".

- [ ] **Step 3: Implement puppygraph.go**

```go
// File: mdl/graphdb/puppygraph.go
package graphdb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PuppyGraphAdapter connects to a running PuppyGraph instance via its HTTP API.
// PuppyGraph exposes full native Cypher and can read from SQLite via JDBC.
//
// Typical setup:
//  1. Call RegisterSchema to point PuppyGraph at the graph SQLite temp file.
//  2. Call RunCypher to execute Cypher queries.
type PuppyGraphAdapter struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

// NewPuppyGraphAdapter creates an adapter for the PuppyGraph instance at endpoint.
// apiKey is sent as a Bearer token if non-empty.
func NewPuppyGraphAdapter(endpoint, apiKey string) *PuppyGraphAdapter {
	return &PuppyGraphAdapter{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		client:   &http.Client{},
	}
}

// puppySchema is the JSON body sent to PuppyGraph's PUT /schema endpoint.
// See PuppyGraph documentation for the full schema format.
type puppySchema struct {
	Catalogs []puppyCatalog `json:"catalogs"`
	Graph    puppyGraph     `json:"graph"`
}

type puppyCatalog struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	JdbcDriver string `json:"jdbcDriver"`
	JdbcURI    string `json:"jdbcUri"`
}

type puppyGraph struct {
	Name     string        `json:"name"`
	Vertices []puppyVertex `json:"vertices"`
	Edges    []puppyEdge   `json:"edges"`
}

type puppyVertex struct {
	Label      string            `json:"label"`
	Catalog    string            `json:"catalog"`
	Table      string            `json:"table"`
	Filter     string            `json:"filter,omitempty"`
	Properties []puppyProperty   `json:"properties"`
}

type puppyEdge struct {
	Label      string          `json:"label"`
	Catalog    string          `json:"catalog"`
	Table      string          `json:"table"`
	Filter     string          `json:"filter,omitempty"`
	FromVertex string          `json:"fromVertex"`
	ToVertex   string          `json:"toVertex"`
	From       string          `json:"from"`
	To         string          `json:"to"`
}

type puppyProperty struct {
	Name   string `json:"name"`
	Column string `json:"column"`
	Type   string `json:"type"`
}

// RegisterSchema registers the temp SQLite file as a PuppyGraph catalog,
// defining vertex and edge mappings. Must be called before RunCypher.
// dbPath is the absolute path to the temp SQLite file built by Indexer.
func (p *PuppyGraphAdapter) RegisterSchema(ctx context.Context, dbPath string) error {
	labels := []string{"Module", "Entity", "Attribute", "Association", "Microflow", "NanoFlow", "Page", "Enumeration", "EnumValue"}
	vertices := make([]puppyVertex, 0, len(labels))
	for _, lbl := range labels {
		vertices = append(vertices, puppyVertex{
			Label:   lbl,
			Catalog: "mxcli",
			Table:   "graph_nodes",
			Filter:  fmt.Sprintf("label = '%s'", lbl),
			Properties: []puppyProperty{
				{Name: "id", Column: "id", Type: "String"},
				{Name: "name", Column: "name", Type: "String"},
				{Name: "module", Column: "module", Type: "String"},
			},
		})
	}

	edgeLabels := []struct {
		label    string
		from, to string
		filter   string
	}{
		{"HAS_ENTITY", "Module", "Entity", "rel_type = 'HAS_ENTITY'"},
		{"HAS_ATTRIBUTE", "Entity", "Attribute", "rel_type = 'HAS_ATTRIBUTE'"},
		{"HAS_ASSOCIATION", "Entity", "Association", "rel_type = 'HAS_ASSOCIATION'"},
		{"HAS_MICROFLOW", "Module", "Microflow", "rel_type = 'HAS_MICROFLOW'"},
		{"HAS_NANOFLOW", "Module", "NanoFlow", "rel_type = 'HAS_NANOFLOW'"},
		{"HAS_PAGE", "Module", "Page", "rel_type = 'HAS_PAGE'"},
		{"FROM_ENTITY", "Association", "Entity", "rel_type = 'FROM_ENTITY'"},
		{"TO_ENTITY", "Association", "Entity", "rel_type = 'TO_ENTITY'"},
	}
	edges := make([]puppyEdge, 0, len(edgeLabels))
	for _, el := range edgeLabels {
		edges = append(edges, puppyEdge{
			Label:      el.label,
			Catalog:    "mxcli",
			Table:      "graph_edges",
			Filter:     el.filter,
			FromVertex: el.from,
			ToVertex:   el.to,
			From:       "from_id",
			To:         "to_id",
		})
	}

	schema := puppySchema{
		Catalogs: []puppyCatalog{{
			Name:       "mxcli",
			Type:       "jdbc",
			JdbcDriver: "org.sqlite.JDBC",
			JdbcURI:    "jdbc:sqlite:" + dbPath,
		}},
		Graph: puppyGraph{
			Name:     "mxcli_graph",
			Vertices: vertices,
			Edges:    edges,
		},
	}

	body, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, p.endpoint+"/schema", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("PuppyGraph schema registration: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PuppyGraph schema registration failed (HTTP %d): %s", resp.StatusCode, b)
	}
	return nil
}

type puppyQueryRequest struct {
	Query      string         `json:"query"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type puppyQueryResponse struct {
	Data struct {
		Rows    []map[string]any `json:"rows"`
		Columns []string         `json:"columns"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// RunCypher executes a Cypher query against PuppyGraph and returns matching rows.
// Each Record maps return variable names (e.g. "e.name") to their values.
func (p *PuppyGraphAdapter) RunCypher(ctx context.Context, cypher string, params map[string]any) ([]Record, error) {
	payload := puppyQueryRequest{Query: cypher, Parameters: params}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/cypher", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PuppyGraph query: %w", err)
	}
	defer resp.Body.Close()

	var pr puppyQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decode PuppyGraph response: %w", err)
	}
	if pr.Error != "" {
		return nil, fmt.Errorf("PuppyGraph Cypher error: %s", pr.Error)
	}

	records := make([]Record, len(pr.Data.Rows))
	for i, row := range pr.Data.Rows {
		records[i] = Record(row)
	}
	return records, nil
}

// Close is a no-op — PuppyGraph is a standalone service.
func (p *PuppyGraphAdapter) Close() error { return nil }

// OpenGraphDB is a convenience constructor that:
//  1. Creates a temp SQLite file.
//  2. Runs SetupDB + Indexer.Build on the model.
//  3. Registers the schema with PuppyGraph.
//
// Caller must call CloseGraphDB(db, tempPath) when done.
func OpenGraphDB(ctx context.Context, m *modelsdk.Model, puppyEndpoint, apiKey string) (*PuppyGraphAdapter, *sql.DB, string, error) {
	tmpFile, err := os.CreateTemp("", "mxcli-graph-*.db")
	if err != nil {
		return nil, nil, "", fmt.Errorf("create temp graph DB: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	db, err := sql.Open("sqlite", tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return nil, nil, "", err
	}

	if err := SetupDB(db); err != nil {
		db.Close()
		os.Remove(tmpPath)
		return nil, nil, "", err
	}

	if err := NewIndexer(db).Build(m); err != nil {
		db.Close()
		os.Remove(tmpPath)
		return nil, nil, "", err
	}

	adapter := NewPuppyGraphAdapter(puppyEndpoint, apiKey)
	if err := adapter.RegisterSchema(ctx, tmpPath); err != nil {
		db.Close()
		os.Remove(tmpPath)
		return nil, nil, "", fmt.Errorf("register schema with PuppyGraph: %w", err)
	}

	return adapter, db, tmpPath, nil
}

// CloseGraphDB closes the SQLite db and removes the temp file.
func CloseGraphDB(db *sql.DB, tmpPath string) {
	if db != nil {
		db.Close()
	}
	if tmpPath != "" {
		os.Remove(tmpPath)
	}
}

func (p *PuppyGraphAdapter) setAuth(req *http.Request) {
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
}
```

- [ ] **Step 4: Add missing import**

The file needs `"os"` and `"github.com/mendixlabs/mxcli/modelsdk"`. Add to the import block:

```go
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk"
)
```

- [ ] **Step 5: Run tests**

```bash
go test ./mdl/graphdb/... -run TestPuppyGraphAdapter -v
```

Expected: PASS (all 3 tests).

- [ ] **Step 6: Commit**

```bash
git add mdl/graphdb/puppygraph.go mdl/graphdb/puppygraph_test.go
git commit -m "feat(graphdb): PuppyGraphAdapter — HTTP schema registration + Cypher query"
```

---

## Task 5: BatchMutator — Cypher Find → modelsdk Mutate → Flush

**Files:**
- Create: `mdl/graphdb/mutator.go`
- Create: `mdl/graphdb/mutator_test.go`

- [ ] **Step 1: Write the failing test**

```go
// File: mdl/graphdb/mutator_test.go
package graphdb

import (
	"context"
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// mockModel is a minimal stub for mutator tests — tracks which IDs were requested.
type mockModel struct {
	loadedIDs []element.ID
	flushN    int
	flushErr  error
}

func (m *mockModel) LoadUnit(id element.ID) (element.Element, error) {
	m.loadedIDs = append(m.loadedIDs, id)
	return nil, nil // mutate fn receives nil element in this stub
}

func (m *mockModel) Flush() (int, error) {
	m.flushN++
	return 1, m.flushErr
}

func TestBatchMutator_RunCypherMutation(t *testing.T) {
	stub := &stubGraphDB{
		rows: []Record{
			{"a.id": "uuid-001"},
			{"a.id": "uuid-002"},
		},
	}
	mock := &mockModel{}
	mut := NewBatchMutator(mock, stub)

	ctx := context.Background()
	count, err := mut.RunCypherMutation(ctx,
		`MATCH (a:Attribute {name:'X'}) RETURN a.id`,
		func(id element.ID, elem element.Element) error { return nil },
	)
	if err != nil {
		t.Fatalf("RunCypherMutation: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(mock.loadedIDs) != 2 {
		t.Errorf("loadedIDs = %v, want 2 entries", mock.loadedIDs)
	}
}

func TestBatchMutator_Commit(t *testing.T) {
	stub := &stubGraphDB{}
	mock := &mockModel{}
	mut := NewBatchMutator(mock, stub)

	if _, err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if mock.flushN != 1 {
		t.Errorf("Flush called %d times, want 1", mock.flushN)
	}
}

func TestBatchMutator_CommitPropagatesFlushError(t *testing.T) {
	stub := &stubGraphDB{}
	mock := &mockModel{flushErr: errors.New("disk full")}
	mut := NewBatchMutator(mock, stub)

	_, err := mut.Commit()
	if err == nil || err.Error() != "disk full" {
		t.Errorf("expected disk full error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/graphdb/... -run TestBatchMutator -v
```

Expected: FAIL with "undefined: NewBatchMutator".

- [ ] **Step 3: Implement mutator.go**

```go
// File: mdl/graphdb/mutator.go
package graphdb

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// modelInterface is the subset of modelsdk.Model used by BatchMutator.
// Using an interface instead of *modelsdk.Model makes the mutator testable
// without a real MPR file.
type modelInterface interface {
	LoadUnit(id element.ID) (element.Element, error)
	Flush() (int, error)
}

// MutateFn is called for each matched element. The element may be nil
// if modelsdk returns an error loading it (e.g. unknown unit type).
// Return a non-nil error to abort the mutation batch.
type MutateFn func(id element.ID, elem element.Element) error

// BatchMutator finds elements via Cypher and applies in-memory mutations.
// Mutations accumulate in modelsdk's UnitCache; call Commit() once to
// write all dirty elements to the MPR in a single Flush.
type BatchMutator struct {
	model modelInterface
	db    GraphDB
}

// NewBatchMutator creates a mutator backed by model and db.
func NewBatchMutator(model modelInterface, db GraphDB) *BatchMutator {
	return &BatchMutator{model: model, db: db}
}

// RunCypherMutation executes findCypher to locate elements, then calls
// mutateFn for each. The first return value in each Record that is a string
// is treated as the element ID. All mutations are accumulated in memory —
// call Commit() to flush.
func (bm *BatchMutator) RunCypherMutation(ctx context.Context, findCypher string, mutateFn MutateFn) (int, error) {
	records, err := bm.db.RunCypher(ctx, findCypher, nil)
	if err != nil {
		return 0, fmt.Errorf("graph query: %w", err)
	}

	count := 0
	for _, rec := range records {
		id, ok := extractID(rec)
		if !ok {
			continue
		}
		elem, loadErr := bm.model.LoadUnit(id)
		if loadErr != nil {
			// Log but continue — don't abort the batch for one missing unit.
			continue
		}
		if err := mutateFn(id, elem); err != nil {
			return count, fmt.Errorf("mutation for %s: %w", id, err)
		}
		count++
	}
	return count, nil
}

// Commit writes all dirty in-memory elements to the MPR. This is the
// single BSON write-back at the end of a Cypher mutation session.
func (bm *BatchMutator) Commit() (int, error) {
	return bm.model.Flush()
}

// extractID returns the first string value in rec that looks like a UUID,
// as an element.ID. Accepts both bare UUID strings and "uuid" keyed values.
func extractID(rec Record) (element.ID, bool) {
	// Prefer an explicit "id" key
	for _, k := range []string{"id", "e.id", "a.id", "mf.id", "n.id"} {
		if v, ok := rec[k]; ok {
			if s, ok := v.(string); ok && looksLikeUUID(s) {
				return element.ID(s), true
			}
		}
	}
	// Fallback: first string value that looks like a UUID
	for _, v := range rec {
		if s, ok := v.(string); ok && looksLikeUUID(s) {
			return element.ID(s), true
		}
	}
	return "", false
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./mdl/graphdb/... -run TestBatchMutator -v
```

Expected: PASS for all 3 tests.

- [ ] **Step 5: Commit**

```bash
git add mdl/graphdb/mutator.go mdl/graphdb/mutator_test.go
git commit -m "feat(graphdb): BatchMutator — Cypher find + in-memory mutate + single Flush"
```

---

## Task 6: `mxcli cypher` CLI Command

**Files:**
- Create: `cmd/mxcli/cmd_cypher.go`
- Modify: `cmd/mxcli/main.go` (register `cypherCmd`)

- [ ] **Step 1: Write the failing test**

Create `cmd/mxcli/cmd_cypher_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestCypherCmd_requiresProject(t *testing.T) {
	// cypherCmd should error without -p flag
	cypherCmd.ResetFlags()
	cypherCmd.Flags().String("project", "", "")
	err := cypherCmd.RunE(cypherCmd, []string{"MATCH (e:Entity) RETURN e.name"})
	if err == nil {
		t.Fatal("expected error when no project specified, got nil")
	}
	if !strings.Contains(err.Error(), "project") {
		t.Errorf("error should mention 'project', got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/... -run TestCypherCmd_requiresProject -v
```

Expected: FAIL with "undefined: cypherCmd".

- [ ] **Step 3: Implement cmd_cypher.go**

```go
// File: cmd/mxcli/cmd_cypher.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mendixlabs/mxcli/mdl/graphdb"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/spf13/cobra"
)

var cypherCmd = &cobra.Command{
	Use:   "cypher <query>",
	Short: "Run a Cypher query against the MPR project graph",
	Long: `Run a Cypher query against the project's element graph using PuppyGraph.

The project is indexed into a temporary graph on startup. Nodes represent
Mendix elements (Entity, Attribute, Microflow, Page, ...). Edges represent
containment and cross-references.

Requires a running PuppyGraph instance (default: http://localhost:8080).
Start with: docker run -p 8080:8080 -p 7687:7687 puppygraph/puppygraph:latest

Examples:
  mxcli cypher -p app.mpr "MATCH (e:Entity) RETURN e.name, e.module"
  mxcli cypher -p app.mpr "MATCH (mf:Microflow)-[:USES_ENTITY]->(e:Entity {name:'Customer'}) RETURN mf.name"
  mxcli cypher -p app.mpr "MATCH (x)-[r]->(e:Entity {name:'Order'}) RETURN type(r), x.name"`,
	Args: cobra.ExactArgs(1),
	RunE: runCypher,
}

func init() {
	cypherCmd.Flags().String("puppygraph", "http://localhost:8080", "PuppyGraph endpoint")
	cypherCmd.Flags().String("api-key", "", "PuppyGraph API key (if required)")
}

func runCypher(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	if projectPath == "" {
		return fmt.Errorf("--project (-p) is required for cypher command")
	}

	puppyEndpoint, _ := cmd.Flags().GetString("puppygraph")
	apiKey, _ := cmd.Flags().GetString("api-key")
	query := args[0]

	m, err := modelsdk.Open(projectPath)
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer m.Close()

	ctx := context.Background()
	adapter, db, tmpPath, err := graphdb.OpenGraphDB(ctx, m, puppyEndpoint, apiKey)
	if err != nil {
		return fmt.Errorf("initialize graph index: %w", err)
	}
	defer graphdb.CloseGraphDB(db, tmpPath)
	defer adapter.Close()

	records, err := adapter.RunCypher(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("cypher query: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	for _, rec := range records {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "(no results)")
	}
	return nil
}
```

- [ ] **Step 4: Register the command in main.go**

In `cmd/mxcli/main.go`, find the `rootCmd.AddCommand` block (around line 315) and add:

```go
rootCmd.AddCommand(cypherCmd)
```

Add it after:
```go
rootCmd.AddCommand(searchCmd)
```

So it reads:
```go
rootCmd.AddCommand(searchCmd)
rootCmd.AddCommand(cypherCmd)     // ← add this line
rootCmd.AddCommand(lintCmd)
```

- [ ] **Step 5: Build and verify help text**

```bash
go build ./cmd/mxcli/ && ./bin/mxcli cypher --help 2>/dev/null || go run ./cmd/mxcli/ cypher --help
```

Expected output includes "Run a Cypher query against the MPR project graph" and usage examples.

- [ ] **Step 6: Run the CLI test**

```bash
go test ./cmd/mxcli/... -run TestCypherCmd_requiresProject -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli/cmd_cypher.go cmd/mxcli/cmd_cypher_test.go cmd/mxcli/main.go
git commit -m "feat(mxcli): cypher command — Cypher queries against MPR project graph"
```

---

## Task 7: End-to-End Build Verification

- [ ] **Step 1: Run the full test suite for the new package**

```bash
go test ./mdl/graphdb/... -v
```

Expected: All tests PASS. Note: `TestIndexer_Build` may emit a log line like `indexed 47 nodes, 52 edges` — the exact numbers depend on `testdata/expr-checker/minimal.mpr` content.

- [ ] **Step 2: Build the CLI binary**

```bash
make build
```

Expected: `bin/mxcli` produced, exit code 0.

- [ ] **Step 3: Verify the cypher subcommand appears in help**

```bash
./bin/mxcli --help | grep cypher
```

Expected: line containing `cypher` and a short description.

- [ ] **Step 4: Final commit**

```bash
git commit --allow-empty -m "chore(graphdb): verify build + tests green"
```

---

## Self-Review Checklist

**Spec coverage:**
- ✅ GraphDB interface with `RunCypher` + `Close` — Task 1
- ✅ PuppyGraph HTTP adapter (no extra deps) — Task 4
- ✅ Temp SQLite file (not in-process `:memory:`) — Task 4 `OpenGraphDB`
- ✅ Indexer: node + edge pass from modelsdk — Task 3
- ✅ BatchMutator: Cypher find → mutate → single Flush — Task 5
- ✅ `mxcli cypher` CLI command — Task 6
- ✅ NebulaAdapter: listed as optional (spec says "fallback"), not implemented in this plan — acceptable MVP scope
- ✅ Catalog removal: spec says "superseded" — existing catalog is NOT deleted in this plan; deletion is a follow-up PR to avoid breaking existing `show callers/callees/references` commands before Cypher replacements are validated

**Placeholder scan:** None found. All steps have complete code blocks.

**Type consistency:**
- `GraphDB` interface defined in Task 1, used in Tasks 4, 5, 6 — consistent
- `Record` type defined in Task 1, used in Tasks 4, 5 — consistent  
- `MutateFn` defined in Task 5, used in Task 5 only — consistent
- `OpenGraphDB` / `CloseGraphDB` defined in Task 4 (puppygraph.go), called in Task 6 — consistent
- `LabelFromType` / `containmentRelType` defined in Task 2, used in Task 3 — consistent
- `looksLikeUUID` defined in Task 3, used in Tasks 3 and 5 — consistent (both in package `graphdb`)
- `stubGraphDB` defined in `puppygraph_test.go` (Task 4), used in `mutator_test.go` (Task 5) — both are `package graphdb`, so accessible
