package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// AccessRuleAdapter 遍历 DomainModels → Entities → AccessRules，
// 为每个实体访问规则创建 AccessRule 节点和 HAS_ACCESS_RULE 边。
//
// 数据来源（typed modelsdk 路径，domain model 类型注册完整）：
//   DomainModels$DomainModel
//     → Entities[] (ChildListProperty)
//       → Entity
//         → AccessRules[] (ChildListProperty)
//           → AccessRule
//             → ModuleRolesQualifiedNames()
//             → DefaultMemberAccessRights()  ("ReadOnly"|"ReadWrite"|"" )
//             → AllowCreate()
//             → AllowDelete()
//             → XPathConstraint()
type AccessRuleAdapter struct {
	Model *modelsdk.Model
}

func (a *AccessRuleAdapter) Name() string { return "accessrule" }

func (a *AccessRuleAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"AccessRule"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_ACCESS_RULE", "Entity", "AccessRule"},
		},
	}
}

func (a *AccessRuleAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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
		if elem.TypeName() != "DomainModels$DomainModel" {
			continue
		}

		module := a.Model.ResolveModuleName(unit.ID)

		entitiesProp := findProperty(elem, "Entities")
		if entitiesProp == nil {
			continue
		}
		cl, ok := entitiesProp.(element.ChildListProperty)
		if !ok {
			continue
		}

		for _, entItem := range cl.ChildElements() {
			if entItem == nil {
				continue
			}
			ent, ok := entItem.(*genDm.Entity)
			if !ok {
				continue
			}
			entityQN := module + "." + ent.Name()
			entityID := mxgraph.NodeID(ent.ID())

			for _, arItem := range ent.AccessRulesItems() {
				ar, ok := arItem.(*genDm.AccessRule)
				if !ok || ar == nil {
					continue
				}

				rights := ar.DefaultMemberAccessRights()
				canRead, canWrite := parseAccessRuleRights(rights)
				xpath := extractXPath(ar)

				for _, mrQN := range ar.ModuleRolesQualifiedNames() {
					nodeID := mxgraph.NodeID(fmt.Sprintf("%s.rule.%s", entityQN, mrQN))
					props := map[string]any{
						"$Type":          "AccessRule",
						"EntityQN":       entityQN,
						"ModuleRoleQN":   mrQN,
						"CanRead":        canRead,
						"CanWrite":       canWrite,
						"CanCreate":      ar.AllowCreate(),
						"CanDelete":      ar.AllowDelete(),
						"XPathConstraint": xpath,
						"QualifiedName":  fmt.Sprintf("%s→%s", entityQN, mrQN),
					}
					events = append(events, mxgraph.Event{
						Type: mxgraph.NodeCreated,
						Node: &mxgraph.Node{ID: nodeID, Label: "AccessRule", Props: props},
					})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_ACCESS_RULE-->%s", entityID, nodeID)),
							From: entityID,
							To:   nodeID,
							Type: "HAS_ACCESS_RULE",
						},
					})
				}
			}
		}
	}
	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *AccessRuleAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

// parseAccessRuleRights 将 DefaultMemberAccessRights 转为读写布尔值。
func parseAccessRuleRights(rights string) (canRead, canWrite bool) {
	switch rights {
	case "ReadOnly":
		return true, false
	case "ReadWrite":
		return true, true
	default:
		return false, false
	}
}

// extractXPath 从 AccessRule 中提取 XPathConstraint 字符串。
func extractXPath(ar *genDm.AccessRule) string {
	for _, prop := range ar.Properties() {
		if prop.Name() == "XPathConstraint" {
			if wp, ok := prop.(element.WritableProperty); ok {
				if v := wp.BSONValue(); v != nil {
					if s, ok := v.(string); ok {
						return s
					}
				}
			}
		}
	}
	return ""
}

// findProperty 在 element.Properties() 中按名查找。
func findProperty(elem element.Element, name string) element.Property {
	for _, p := range elem.Properties() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
