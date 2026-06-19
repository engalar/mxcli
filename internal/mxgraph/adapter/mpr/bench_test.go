package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

func BenchmarkFullGraphBuild(b *testing.B) {
	mprPath := findTestMPR(b)
	if mprPath == "" {
		b.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer m.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		// Simulate full build: all adapters in sequence
		sink := &countingSink{}

		a := &DomainModelAdapter{Model: m}
		if err := a.Build(ctx, sink); err != nil {
			b.Fatalf("DomainModelAdapter: %v", err)
		}

		a2 := &PageAdapter{Model: m}
		if err := a2.Build(ctx, sink); err != nil {
			b.Fatalf("PageAdapter: %v", err)
		}

		a3 := &MicroflowAdapter{Model: m}
		if err := a3.Build(ctx, sink); err != nil {
			b.Fatalf("MicroflowAdapter: %v", err)
		}

		a4 := &SecurityAdapter{Model: m}
		if err := a4.Build(ctx, sink); err != nil {
			b.Fatalf("SecurityAdapter: %v", err)
		}

		src := &ModelsdkUnitSource{Model: m}

		a5 := &NavigationAdapter{Source: src}
		if err := a5.Build(ctx, sink); err != nil {
			b.Fatalf("NavigationAdapter: %v", err)
		}

		a6 := &DataContainerAdapter{Source: src, Model: m}
		if err := a6.Build(ctx, sink); err != nil {
			b.Fatalf("DataContainerAdapter: %v", err)
		}

		a7 := &DocumentGrantAdapter{Model: m}
		if err := a7.Build(ctx, sink); err != nil {
			b.Fatalf("DocumentGrantAdapter: %v", err)
		}

		_ = sink
	}
}

func BenchmarkDataContainerAdapter(b *testing.B) {
	mprPath := findTestMPR(b)
	if mprPath == "" {
		b.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &DataContainerAdapter{
		Source: &ModelsdkUnitSource{Model: m},
		Model:  m,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink := &countingSink{}
		if err := a.Build(context.Background(), sink); err != nil {
			b.Fatalf("Build: %v", err)
		}
	}
}

func BenchmarkNavigationAdapter(b *testing.B) {
	mprPath := findTestMPR(b)
	if mprPath == "" {
		b.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer m.Close()

	a := &NavigationAdapter{
		Source: &ModelsdkUnitSource{Model: m},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink := &countingSink{}
		if err := a.Build(context.Background(), sink); err != nil {
			b.Fatalf("Build: %v", err)
		}
	}
}

type countingSink struct {
	nodeCount int
	edgeCount int
}

func (s *countingSink) Emit(events []mxgraph.Event) error {
	for _, ev := range events {
		if ev.Type == mxgraph.NodeCreated || ev.Type == mxgraph.NodeUpdated {
			s.nodeCount++
		}
		if ev.Type == mxgraph.EdgeCreated {
			s.edgeCount++
		}
	}
	return nil
}


