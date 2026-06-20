package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// DocumentGrantAdapter 索引页面和微流的授权信息：
//   - HAS_ALLOWED_ROLE 边 (Page → ModuleRole)
//   - HAS_ALLOWED_ROLE 边 (Microflow → ModuleRole)
//   - Microflow 节点的 ApplyEntityAccess 属性
//
// 数据来源（typed modelsdk 路径）：
//
//	Forms$Page.AllowedRolesQualifiedNames()
//	Microflows$Microflow.AllowedModuleRolesQualifiedNames()
//	Microflows$Microflow.ApplyEntityAccess()
type DocumentGrantAdapter struct {
	Model *modelsdk.Model
}

func (a *DocumentGrantAdapter) Name() string { return "documentgrant" }

func (a *DocumentGrantAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_ALLOWED_ROLE", "Page", "ModuleRole"},
			{"HAS_ALLOWED_ROLE", "Microflow", "ModuleRole"},
		},
	}
}

func (a *DocumentGrantAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil || elem == nil {
			continue
		}

		typeName := elem.TypeName()
		switch {
		case strings.HasSuffix(typeName, "$Page") || strings.HasSuffix(typeName, "$Form"):
			evts := a.indexPage(elem, unit.ID)
			events = append(events, evts...)
		case strings.HasSuffix(typeName, "$Microflow"):
			evts := a.indexMicroflow(elem, unit.ID)
			events = append(events, evts...)
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *DocumentGrantAdapter) indexPage(elem element.Element, unitID interface{}) []mxgraph.Event {
	var events []mxgraph.Event
	pageID := mxgraph.NodeID(elem.ID())

	allowedRolesProp := findProperty(elem, "AllowedRoles")
	if allowedRolesProp == nil {
		return nil
	}

	// AllowedRoles 是 ByNameRefList，需要通过 WritableProperty 获取 BSONValue
	wp, ok := allowedRolesProp.(element.WritableProperty)
	if !ok {
		return nil
	}
	if v := wp.BSONValue(); v != nil {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if mrQN, ok := item.(string); ok && mrQN != "" {
					edgeID := mxgraph.NodeID(fmt.Sprintf("%s--HAS_ALLOWED_ROLE-->%s", pageID, mrQN))
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   edgeID,
							From: pageID,
							To:   mxgraph.NodeID(mrQN),
							Type: "HAS_ALLOWED_ROLE",
						},
					})
				}
			}
		}
		// ByNameRefList 也可能返回 []string
		if strs, ok := v.([]string); ok {
			for _, mrQN := range strs {
				if mrQN == "" {
					continue
				}
				edgeID := mxgraph.NodeID(fmt.Sprintf("%s--HAS_ALLOWED_ROLE-->%s", pageID, mrQN))
				events = append(events, mxgraph.Event{
					Type: mxgraph.EdgeCreated,
					Edge: &mxgraph.Edge{
						ID:   edgeID,
						From: pageID,
						To:   mxgraph.NodeID(mrQN),
						Type: "HAS_ALLOWED_ROLE",
					},
				})
			}
		}
	}
	return events
}

func (a *DocumentGrantAdapter) indexMicroflow(elem element.Element, unitID interface{}) []mxgraph.Event {
	var events []mxgraph.Event
	mfID := mxgraph.NodeID(elem.ID())

	// 提取 ApplyEntityAccess
	var applyEntityAccess bool
	for _, p := range elem.Properties() {
		if p.Name() == "ApplyEntityAccess" {
			if wp, ok := p.(element.WritableProperty); ok {
				if v := wp.BSONValue(); v != nil {
					if b, ok := v.(bool); ok {
						applyEntityAccess = b
					}
				}
			}
		}
	}
	if applyEntityAccess {
		events = append(events, mxgraph.Event{
			Type: mxgraph.NodeUpdated,
			Node: &mxgraph.Node{
				ID:    mfID,
				Label: "Microflow",
				Props: map[string]any{"ApplyEntityAccess": true},
			},
		})
	}

	// 提取 AllowedModuleRoles
	allowedRolesProp := findProperty(elem, "AllowedModuleRoles")
	if allowedRolesProp == nil {
		return events
	}
	wp, ok := allowedRolesProp.(element.WritableProperty)
	if !ok {
		return events
	}
	if v := wp.BSONValue(); v != nil {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if mrQN, ok := item.(string); ok && mrQN != "" {
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_ALLOWED_ROLE-->%s", mfID, mrQN)),
							From: mfID,
							To:   mxgraph.NodeID(mrQN),
							Type: "HAS_ALLOWED_ROLE",
						},
					})
				}
			}
		}
	}
	return events
}

func (a *DocumentGrantAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
