// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/model/layout"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RelayoutDomainModel reads the domain model, computes optimal entity positions
// using the layout algorithm, updates each entity's Location in-memory, and
// writes the entire domain model back in a single transaction.
func (b *MprBackend) RelayoutDomainModel(domainModelID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return err
	}
	positions := layout.Compute(collectLayoutNodes(dm), collectLayoutEdges(dm))
	if len(positions) == 0 {
		return nil
	}
	for _, elem := range dm.EntitiesItems() {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		if pt, found := positions[string(e.ID())]; found {
			// Mendix Location format: "x;y" — see executor.layoutPos.
			e.SetLocation(fmt.Sprintf("%d;%d", pt.X, pt.Y))
		}
	}
	return b.UpdateDomainModelGen(dm)
}

// collectLayoutNodes extracts layout.Node values from a gen DomainModel.
// This is the only place that touches gen types for layout purposes.
func collectLayoutNodes(dm *genDm.DomainModel) []layout.Node {
	items := dm.EntitiesItems()
	nodes := make([]layout.Node, 0, len(items))
	for _, elem := range items {
		e, ok := elem.(*genDm.Entity)
		if !ok {
			continue
		}
		nodes = append(nodes, layout.Node{
			ID:        string(e.ID()),
			AttrCount: len(e.AttributesItems()),
		})
	}
	return nodes
}

// collectLayoutEdges extracts layout.Edge values from a gen DomainModel.
// Both Associations and CrossAssociations are included.
func collectLayoutEdges(dm *genDm.DomainModel) []layout.Edge {
	var edges []layout.Edge
	for _, elem := range dm.AssociationsItems() {
		a, ok := elem.(*genDm.Association)
		if !ok {
			continue
		}
		from := string(a.ParentRefID())
		to := string(a.ChildRefID())
		if from != "" && to != "" {
			edges = append(edges, layout.Edge{From: from, To: to})
		}
	}
	return edges
}
