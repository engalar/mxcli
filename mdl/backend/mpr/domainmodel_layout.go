// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"image"

	"github.com/mendixlabs/mxcli/mdl/model/layout"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RelayoutDomainModel reads the domain model, computes optimal entity positions
// using the layout algorithm, updates each entity's Location in-memory, and
// writes the entire domain model back in a single transaction.
//
// It also updates each association's ParentConnection and ChildConnection so
// the line exits from the correct edge of each entity rectangle.  Without
// this, Studio Pro defaults to X=0 (left edge) for empty connection strings,
// causing every association line to emerge from the left side regardless of
// which entity is on which side.
func (b *MprBackend) RelayoutDomainModel(domainModelID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return err
	}
	nodes, nameToID := collectLayoutNodes(dm)
	// Build reverse map (GUID → name) for edge lookup.
	idToName := make(map[string]string, len(nameToID))
	for name, id := range nameToID {
		idToName[id] = name
	}
	// positions is keyed by entity name (stable across runs — see collectLayoutNodes).
	positions := layout.Compute(nodes, collectLayoutEdges(dm, idToName))
	if len(positions) == 0 {
		return nil
	}
	for _, elem := range dm.EntitiesItems() {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		if pt, found := positions[e.Name()]; found {
			// Mendix Location format: "x;y" — see executor.layoutPos.
			e.SetLocation(fmt.Sprintf("%d;%d", pt.X, pt.Y))
		}
	}
	// Update association connection points based on relative entity positions.
	updateAssociationConnections(dm, positions, idToName)
	return b.UpdateDomainModelGen(dm)
}

// updateAssociationConnections sets ParentConnection and ChildConnection on
// every association so the line exits the entity from the nearest horizontal
// edge.
//
// Connection point format: "X;Y" percentage strings (0–100).
//
//	"100;50" = right edge midpoint   "0;50" = left edge midpoint
//
// Naming convention (counter-intuitive — see CLAUDE.md):
//
//	ParentRefID → FROM entity (FK owner, MDL "from" keyword)
//	ChildRefID  → TO entity   (referenced, MDL "to" keyword)
//
// Without this reset, Studio Pro defaults to X=0 (left edge) for empty
// connection strings, so every line emerges from the left side of both
// entities regardless of their relative positions.
func updateAssociationConnections(dm *genDm.DomainModel, positions map[string]image.Point, idToName map[string]string) {
	for _, elem := range dm.AssociationsItems() {
		a, ok := elem.(*genDm.Association)
		if !ok {
			continue
		}
		fromName := idToName[string(a.ParentRefID())]
		toName := idToName[string(a.ChildRefID())]
		fromPt, fromOK := positions[fromName]
		toPt, toOK := positions[toName]
		if !fromOK || !toOK {
			continue
		}
		if fromPt.X <= toPt.X {
			// FROM is left of (or same column as) TO: exit right, enter left.
			a.SetParentConnection("100;50")
			a.SetChildConnection("0;50")
		} else {
			// FROM is right of TO: exit left, enter right.
			a.SetParentConnection("0;50")
			a.SetChildConnection("100;50")
		}
	}
}

// collectLayoutNodes extracts layout.Node values from a gen DomainModel.
// Node IDs are entity names (not GUIDs) so the layout is deterministic across
// runs that create the same entities with different auto-assigned GUIDs.
// Returns nodes and a nameToID map for mapping results back to entities.
func collectLayoutNodes(dm *genDm.DomainModel) ([]layout.Node, map[string]string) {
	items := dm.EntitiesItems()
	nodes := make([]layout.Node, 0, len(items))
	nameToID := make(map[string]string, len(items)) // entityName → GUID
	for _, elem := range items {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		nameToID[e.Name()] = string(e.ID())
		nodes = append(nodes, layout.Node{
			ID:        e.Name(),
			AttrCount: len(e.AttributesItems()),
		})
	}
	return nodes, nameToID
}

// collectLayoutEdges extracts layout.Edge values from a gen DomainModel using
// entity names (not GUIDs) to match the node IDs produced by collectLayoutNodes.
// idToName maps GUID → entity name for the lookup.
func collectLayoutEdges(dm *genDm.DomainModel, idToName map[string]string) []layout.Edge {
	var edges []layout.Edge
	for _, elem := range dm.AssociationsItems() {
		a, ok := elem.(*genDm.Association)
		if !ok {
			continue
		}
		fromName := idToName[string(a.ParentRefID())]
		toName := idToName[string(a.ChildRefID())]
		if fromName != "" && toName != "" {
			edges = append(edges, layout.Edge{From: fromName, To: toName})
		}
	}
	return edges
}
