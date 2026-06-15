package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// SecurityAdapter emits UserRole / ModuleRole nodes.
//
// Verified BSON structure (modelsdk/gen/security/descriptors.go) — roles are not
// top-level units; they are nested inside the two security container units:
//
//	Security$ProjectSecurity → UserRoles   (PartList of Security$UserRole)
//	Security$ModuleSecurity  → ModuleRoles (PartList of Security$ModuleRole)
type SecurityAdapter struct {
	Model *modelsdk.Model
}

func (a *SecurityAdapter) Name() string { return "security" }

func (a *SecurityAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"UserRole", "ModuleRole"},
	}
}

func (a *SecurityAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}

		switch elem.TypeName() {
		case "Security$ProjectSecurity":
			for _, role := range childList(elem, "UserRoles") {
				if role == nil {
					continue
				}
				node := nodeForElement(role, "UserRole")
				events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
			}
		case "Security$ModuleSecurity":
			for _, role := range childList(elem, "ModuleRoles") {
				if role == nil {
					continue
				}
				node := nodeForElement(role, "ModuleRole")
				events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
			}
		}
	}

	if len(events) > 0 {
		if err := sink.Emit(events); err != nil {
			return fmt.Errorf("emit events: %w", err)
		}
	}
	return nil
}

func (a *SecurityAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
