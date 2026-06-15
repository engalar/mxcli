package mxgraph

import (
	"fmt"
	"testing"
)

// buildLargeGraph 创建 nodeCount 个节点，每节点有 edgesPerNode 条出边
func buildLargeGraph(nodeCount, edgesPerNode int) *Graph {
	g := New()
	for i := 0; i < nodeCount; i++ {
		g.AddNode(NodeID(fmt.Sprintf("n%d", i)), "Entity", map[string]any{"idx": i})
	}
	edgeIdx := 0
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < edgesPerNode; j++ {
			to := (i + j + 1) % nodeCount
			eid := NodeID(fmt.Sprintf("e%d", edgeIdx))
			g.AddEdge(eid, NodeID(fmt.Sprintf("n%d", i)), NodeID(fmt.Sprintf("n%d", to)), "HAS_ATTRIBUTE", nil)
			edgeIdx++
		}
	}
	return g
}

// BenchmarkEdges_Outbound 当前是 O(E) 全边扫描，修复后应为 O(degree)
func BenchmarkEdges_Outbound(b *testing.B) {
	g := buildLargeGraph(1000, 5) // 1000 节点，每节点 5 条出边，共 5000 条边
	target := NodeID("n500")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Edges(target, Outbound)
	}
}

// BenchmarkEdges_Both 测试双向查询
func BenchmarkEdges_Both(b *testing.B) {
	g := buildLargeGraph(1000, 5)
	target := NodeID("n500")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Edges(target, Both)
	}
}

// BenchmarkRemoveNode_Large 当前 RemoveNode 有 O(E) 全边扫描
func BenchmarkRemoveNode_Large(b *testing.B) {
	const nodeCount = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g := buildLargeGraph(nodeCount, 5)
		b.StartTimer()
		g.RemoveNode(NodeID("n500"))
	}
}
