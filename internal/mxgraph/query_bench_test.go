package mxgraph

import (
	"fmt"
	"testing"
)

// buildChainGraph 创建线性链：n0→n1→n2→...→nN，全部同 label 同 relType
func buildChainGraph(length int) *Graph {
	g := New()
	for i := 0; i <= length; i++ {
		g.AddNode(NodeID(fmt.Sprintf("n%d", i)), "Node", nil)
	}
	for i := 0; i < length; i++ {
		from := NodeID(fmt.Sprintf("n%d", i))
		to := NodeID(fmt.Sprintf("n%d", i+1))
		g.AddEdge(NodeID(fmt.Sprintf("e%d", i)), from, to, "NEXT", nil)
	}
	return g
}

func BenchmarkFindPathSchemas_Chain20(b *testing.B) {
	g := buildChainGraph(20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.FindPathSchemas("n0", "n20", 25)
	}
}
