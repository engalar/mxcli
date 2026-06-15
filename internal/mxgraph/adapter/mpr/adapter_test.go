package mpr

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func findTestMPR(t testing.TB) string {
	t.Helper()
	patterns := []string{
		"testdata/corpus-a/app.mpr",
		"testdata/*/app.mpr",
	}
	root := filepath.Join("..", "..", "..", "..", "..")
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

	a := &Adapter{Model: m}
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
