package mpr

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/nanoflows"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/security"

	"github.com/mendixlabs/mxcli/modelsdk"
)

func findTestMPR(t testing.TB) string {
	t.Helper()
	patterns := []string{
		"testdata/corpus-a/app.mpr",
		"testdata/*/app.mpr",
	}
	root := filepath.Join("..", "..", "..", "..")
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(root, p))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

type recordingSink struct {
	events []mxgraph.Event
}

func (s *recordingSink) Emit(events []mxgraph.Event) error {
	s.events = append(s.events, events...)
	return nil
}

func TestMprAdapterFindPath(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	mg := mxgraph.NewIndexManager()
	mg.RegisterAdapter(&DomainModelAdapter{Model: m})
	ctx := context.Background()
	if err := mg.BuildAll(ctx); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}

	graph := mg.Query()
	t.Logf("Graph: %d nodes", len(graph.AllNodes()))

	entities := graph.FindNodes("Entity", nil)
	if len(entities) == 0 {
		t.Skip("no entities in graph")
	}
	t.Logf("Found %d entities", len(entities))
	for _, e := range entities {
		t.Logf("  Entity: id=%s", e.ID)
	}

	entityID := entities[0].ID
	attributes := graph.Neighbors(entityID, "HAS_ATTRIBUTE")
	if len(attributes) == 0 {
		t.Skip("entity has no attributes")
	}
	t.Logf("Entity %s has %d attributes", entityID, len(attributes))

	schemas := graph.FindPathSchemas(entityID, attributes[0].ID, 5)
	t.Logf("Found %d path schemas from entity to its first attribute", len(schemas))
	for i, s := range schemas {
		t.Logf("  Schema[%d]: %s", i, s.Label)
	}

	if len(schemas) == 0 {
		t.Fatal("expected at least one path schema from entity to its attribute")
	}

	path := graph.ExplorePath(entityID, schemas[0])
	if len(path) == 0 {
		t.Fatal("ExplorePath returned empty path")
	}
	t.Logf("Path has %d steps", len(path))
}

func TestMprAdapterBuild(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &DomainModelAdapter{Model: m}
	sink := &recordingSink{}

	ctx := context.Background()
	if err := a.Build(ctx, sink); err != nil {
		t.Fatalf("Build: %v", err)
	}

	t.Logf("MPR adapter produced %d events", len(sink.events))
	if len(sink.events) == 0 {
		t.Fatal("expected at least some events from MPR build")
	}

	hasDomainModel := false
	for _, ev := range sink.events {
		if ev.Type == mxgraph.NodeCreated && ev.Node.Label == "DomainModel" {
			hasDomainModel = true
			break
		}
	}
	if !hasDomainModel {
		t.Error("expected at least one DomainModel node")
	}
}
