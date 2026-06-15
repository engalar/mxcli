package mxgraph

import (
	"context"
	"fmt"
)

type GraphSchema struct {
	NodeLabels []Label
	EdgeTypes  []struct {
		Type RelType
		From Label
		To   Label
	}
}

type EventSink interface {
	Emit(events []Event) error
}

type IndexAdapter interface {
	Name() string
	Schema() *GraphSchema
	Build(ctx context.Context, sink EventSink) error
	Watch(ctx context.Context, sink EventSink) (func(), error)
}

type IndexManager struct {
	graph    *Graph
	adapters map[string]IndexAdapter
}

func NewIndexManager() *IndexManager {
	return &IndexManager{
		graph:    New(),
		adapters: map[string]IndexAdapter{},
	}
}

// NewIndexManagerFromGraph 从已有图创建只读 IndexManager（用于从 snapshot 恢复）。
func NewIndexManagerFromGraph(g *Graph) *IndexManager {
	return &IndexManager{
		graph:    g,
		adapters: map[string]IndexAdapter{},
	}
}

func (m *IndexManager) RegisterAdapter(a IndexAdapter) {
	m.adapters[a.Name()] = a
}

func (m *IndexManager) BuildAll(ctx context.Context) error {
	for name, a := range m.adapters {
		if err := a.Build(ctx, m); err != nil {
			return fmt.Errorf("adapter %q Build: %w", name, err)
		}
	}
	return nil
}

func (m *IndexManager) BuildOne(ctx context.Context, name string) error {
	a, ok := m.adapters[name]
	if !ok {
		return fmt.Errorf("adapter %q not registered", name)
	}
	return a.Build(ctx, m)
}

func (m *IndexManager) Emit(events []Event) error {
	m.graph.Apply(events)
	return nil
}

func (m *IndexManager) Query() *Graph {
	return m.graph
}
