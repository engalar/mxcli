// SPDX-License-Identifier: Apache-2.0

package mpr

import "github.com/mendixlabs/mxcli/internal/mxgraph"

// GetMxGraph returns the in-memory mxgraph index, or nil if no snapshot
// was available when the Reader was opened. The graph is read-only and
// shared across all consumers.
func (r *Reader) GetMxGraph() *mxgraph.Graph {
	return r.mxGraph
}

// SetMxGraph replaces the in-memory graph (e.g. after a fresh build).
// The caller is responsible for persisting the snapshot to disk.
func (r *Reader) SetMxGraph(g *mxgraph.Graph) {
	r.mxGraph = g
}
