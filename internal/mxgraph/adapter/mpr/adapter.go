// Package mpr provides mxgraph IndexAdapter implementations for Mendix .mpr projects.
// Each adapter covers one domain and can be registered independently with IndexManager.
//
// Available adapters:
//   - DomainModelAdapter — entities, attributes, associations
//   - MicroflowAdapter   — microflows, nanoflows, call/create/retrieve/show-page refs
//   - PageAdapter        — pages, layouts, snippets, layout refs
//   - SecurityAdapter    — user roles, module roles
//   - EnumerationAdapter — enumerations, enum values
package mpr

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// nodeForElement builds an mxgraph.Node from an element.Element, copying every
// writable scalar property into Props (keyed by BSON key) plus the $Type marker.
// Part / PartList properties return nil from BSONValue() and are skipped here —
// child elements become their own nodes via the owning adapter.
func nodeForElement(elem element.Element, label mxgraph.Label) *mxgraph.Node {
	props := map[string]any{"$Type": elem.TypeName()}
	for _, p := range elem.Properties() {
		if wp, ok := p.(element.WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				props[p.Name()] = v
			}
		}
	}
	return &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: label, Props: props}
}

// setDerived stamps the derived "Module" and "QualifiedName" props onto a node.
// These are not stored in the element's BSON — Module comes from the owning unit's
// container hierarchy (resolved by the adapter) and QualifiedName is "<Module>.<Name>".
// graphcatalog.ProjectGraph filters/looks up nodes by these keys, so every queryable
// node must carry them. Falls back gracefully when module or name is empty.
func setDerived(node *mxgraph.Node, module string) {
	if module != "" {
		node.Props["Module"] = module
	}
	name, _ := node.Props["Name"].(string)
	if name == "" {
		return
	}
	if module != "" {
		node.Props["QualifiedName"] = module + "." + name
	} else {
		node.Props["QualifiedName"] = name
	}
}
