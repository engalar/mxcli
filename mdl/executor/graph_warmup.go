// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"os"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

// MxGraphProvider is an optional interface backends can implement to expose
// a pre-loaded mxgraph snapshot, avoiding a file read in tryLoadGraphSnapshot.
type MxGraphProvider interface {
	GetMxGraph() *mxgraph.Graph
}

// warmCacheFromGraph seeds executorCache name-lookup maps from the in-memory
// graph, avoiding cold-start backend scans.
//
// Only fills maps that are nil — existing cache entries (from a previous
// warm-up or manual population) are never overwritten. This function is
// intentionally O(nodes) over the graph, not O(N²) over backend calls.
//
// It consumes the graphcatalog typed-node accessors (Entities/Microflows/Pages)
// with an empty module filter (= all modules), so it never touches mxgraph's
// raw Node.Props maps directly.
//
// SOLID:
//   - S: single job — graph → cache translation, no backend I/O
//   - O: adds a fast path; existing backend fallback untouched
//   - D: depends on *graphcatalog.ProjectGraph abstraction, not concrete mxgraph types
func warmCacheFromGraph(cache *executorCache, pg *graphcatalog.ProjectGraph) {
	if cache == nil || pg == nil {
		return
	}

	// ── Entity names ──────────────────────────────────────────────────
	if cache.entityNames == nil {
		nodes := pg.Entities("") // empty module = all modules
		if len(nodes) > 0 {
			m := make(map[model.ID]string, len(nodes))
			for _, n := range nodes {
				if n.QualifiedName != "" {
					m[model.ID(n.ID)] = n.QualifiedName
				}
			}
			if len(m) > 0 {
				cache.entityNames = m
			}
		}
	}

	// ── Microflow + nanoflow names ─────────────────────────────────────
	if cache.microflowNames == nil {
		nodes := pg.Microflows("") // includes both Microflow and Nanoflow labels
		if len(nodes) > 0 {
			m := make(map[model.ID]string, len(nodes))
			for _, n := range nodes {
				if n.QualifiedName != "" {
					m[model.ID(n.ID)] = n.QualifiedName
				}
			}
			if len(m) > 0 {
				cache.microflowNames = m
			}
		}
	}

	// ── Page names ─────────────────────────────────────────────────────
	if cache.pageNames == nil {
		nodes := pg.Pages("")
		if len(nodes) > 0 {
			m := make(map[model.ID]string, len(nodes))
			for _, n := range nodes {
				if n.QualifiedName != "" {
					m[model.ID(n.ID)] = n.QualifiedName
				}
			}
			if len(m) > 0 {
				cache.pageNames = m
			}
		}
	}
}

// tryLoadGraphSnapshot 尝试从项目目录的 .mxcli/graph.gob 加载图快照。
// 如果 backend 实现了 MxGraphProvider，优先从 backend 的缓存读取。
// 成功时预热 cache 并设置 *out；失败时静默返回（graph 是可选加速器）。
//
// 设计为独立函数（而非 execConnect 内联）方便单独测试。
func tryLoadGraphSnapshot(projectDir string, cache *executorCache, out **graphcatalog.ProjectGraph, providers ...MxGraphProvider) {
	if projectDir == "" || cache == nil || out == nil {
		return
	}

	// Fast path: get cached graph from a provider (e.g. MprBackend)
	for _, p := range providers {
		if p == nil {
			continue
		}
		if g := p.GetMxGraph(); g != nil {
			pg := graphcatalog.NewProjectGraph(mxgraph.NewIndexManagerFromGraph(g))
			*out = pg
			warmCacheFromGraph(cache, pg)
			return
		}
	}

	// Fallback: read snapshot file directly.
	snapPath := graphcatalog.SnapshotPath(projectDir)
	data, err := os.ReadFile(snapPath)
	if err != nil {
		return // 文件不存在或无权限——静默跳过
	}
	pg, err := graphcatalog.UnmarshalSnapshot(data)
	if err != nil {
		return // 快照损坏——静默跳过，不影响连接
	}
	*out = pg
	warmCacheFromGraph(cache, pg)
}
