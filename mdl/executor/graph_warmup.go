// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

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
