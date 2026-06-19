package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// NavigationAdapter 索引导航文档树：Profile → HomePage/MenuItem → Page/Microflow。
//
// 数据来源：raw BSON 解析，与 mdl/backend/mpr/navigation_legacy.go 相同路径。
// 使用 ModelsdkUnitSource 而不是 typed gen/navigation 路径，因为 typed 路径
// 的 MenuItemCollection 内部 Action/Item 树未在生成类型中完整声明。
type NavigationAdapter struct {
	Source RawUnitSource
}

func (a *NavigationAdapter) Name() string { return "navigation" }

func (a *NavigationAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"NavigationProfile", "NavigationMenuItem"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_PROFILE", "NavigationProfile", "NavigationProfile"},
			{"HAS_MENU_ITEM", "NavigationProfile", "NavigationMenuItem"},
			{"HAS_CHILD_ITEM", "NavigationMenuItem", "NavigationMenuItem"},
			{"TARGETS_PAGE", "NavigationMenuItem", "Page"},
			{"TARGETS_PAGE", "NavigationProfile", "Page"},
			{"TARGETS_MICROFLOW", "NavigationMenuItem", "Microflow"},
			{"TARGETS_MICROFLOW", "NavigationProfile", "Microflow"},
			{"HAS_LOGIN_PAGE", "NavigationProfile", "Page"},
			{"HAS_NOT_FOUND_PAGE", "NavigationProfile", "Page"},
			{"HAS_OFFLINE_ENTITY", "NavigationProfile", "Entity"},
		},
	}
}

func (a *NavigationAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Source.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if unit.TypeName() != "Navigation$NavigationDocument" {
			continue
		}

		raw := unit.Raw()
		if len(raw) == 0 {
			continue
		}

		module := a.Source.ResolveModuleName(unit.ID())

		var doc map[string]any
		if err := bson.Unmarshal(raw, &doc); err != nil {
			continue
		}

		id := unit.ID()

		navName, _ := doc["Name"].(string)
		if navName == "" {
			navName = "Navigation"
		}

		for _, item := range arrayVal(doc, "Profiles") {
			profMap := toMap(item)
			if profMap == nil {
				continue
			}
			evts := a.indexProfile(profMap, id, module, navName)
			events = append(events, evts...)
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *NavigationAdapter) indexProfile(raw map[string]any, docID, module, navName string) []mxgraph.Event {
	var events []mxgraph.Event

	typeName, _ := raw["$Type"].(string)
	profileName, _ := raw["Name"].(string)
	if profileName == "" {
		return nil
	}
	kind, _ := raw["Kind"].(string)
	isNative := typeName == "Navigation$NativeNavigationProfile"

	profileID := mxgraph.NodeID(fmt.Sprintf("NavigationProfile:%s", profileName))

	events = append(events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    profileID,
			Label: "NavigationProfile",
			Props: map[string]any{
				"$Type":      typeName,
				"Name":       profileName,
				"Kind":       kind,
				"IsNative":   isNative,
				"Module":     module,
				"DocName":    navName,
				"DocID":      docID,
				"QualifiedName": profileName,
			},
		},
	})

	if !isNative {
		if hp := toMap(raw["HomePage"]); hp != nil {
			page, _ := hp["Page"].(string)
			mf, _ := hp["Microflow"].(string)
			evts := a.indexHomeTarget(profileID, profileName, page, mf, module)
			events = append(events, evts...)
		}

		for _, item := range arrayVal(raw, "RoleBasedHomePages") {
			rbMap := toMap(item)
			if rbMap == nil {
				continue
			}
			role, _ := rbMap["UserRole"].(string)
			page, _ := rbMap["Page"].(string)
			mf, _ := rbMap["Microflow"].(string)
			if role == "" {
				continue
			}
			rbProfileID := mxgraph.NodeID(fmt.Sprintf("NavigationProfile:%s/role:%s", profileName, role))
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: &mxgraph.Node{
					ID:    rbProfileID,
					Label: "NavigationProfile",
					Props: map[string]any{
						"$Type":        typeName,
						"Name":         profileName + "/" + role,
						"Kind":         kind,
						"IsNative":     false,
						"Module":       module,
						"ParentProfile": profileName,
						"UserRole":     role,
						"QualifiedName": profileName + "/" + role,
					},
				},
			})
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_MENU_ITEM-->%s", profileID, rbProfileID)),
					From: profileID,
					To:   rbProfileID,
					Type: "HAS_MENU_ITEM",
				},
			})
			evts := a.indexHomeTarget(rbProfileID, profileName, page, mf, module)
			events = append(events, evts...)
		}

		pageVal := ""
		if lps := toMap(raw["LoginPageSettings"]); lps != nil {
			pageVal, _ = lps["Form"].(string)
		}
		if pageVal != "" {
			qn := qualifyName(pageVal, module)
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_LOGIN_PAGE-->%s", profileID, qn)),
					From: profileID,
					To:   mxgraph.NodeID(qn),
					Type: "HAS_LOGIN_PAGE",
				},
			})
		}

		if nfp := toMap(raw["NotFoundHomepage"]); nfp != nil {
			nfPage, _ := nfp["Page"].(string)
			nfMF, _ := nfp["Microflow"].(string)
			if nfPage != "" {
				qn := qualifyName(nfPage, module)
				events = append(events, mxgraph.Event{
					Type: mxgraph.EdgeCreated,
					Edge: &mxgraph.Edge{
						ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_NOT_FOUND_PAGE-->%s", profileID, qn)),
						From: profileID,
						To:   mxgraph.NodeID(qn),
						Type: "HAS_NOT_FOUND_PAGE",
					},
				})
			}
			if nfMF != "" {
				qn := qualifyName(nfMF, module)
				events = append(events, mxgraph.Event{
					Type: mxgraph.EdgeCreated,
					Edge: &mxgraph.Edge{
						ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_MICROFLOW-->%s", profileID, qn)),
						From: profileID,
						To:   mxgraph.NodeID(qn),
						Type: "TARGETS_MICROFLOW",
					},
				})
			}
		}

		if menu := toMap(raw["Menu"]); menu != nil {
			for _, item := range arrayVal(menu, "Items") {
				miMap := toMap(item)
				if miMap == nil {
					continue
				}
				evts := a.indexMenuItem(miMap, profileID, module, 0)
				events = append(events, evts...)
			}
		}
	} else {
		if hp := toMap(raw["NativeHomePage"]); hp != nil {
			page, _ := hp["HomePagePage"].(string)
			nanoflow, _ := hp["HomePageNanoflow"].(string)
			evts := a.indexHomeTarget(profileID, profileName, page, nanoflow, module)
			events = append(events, evts...)
		}

		for _, item := range arrayVal(raw, "RoleBasedNativeHomePages") {
			rbMap := toMap(item)
			if rbMap == nil {
				continue
			}
			role, _ := rbMap["UserRole"].(string)
			page, _ := rbMap["HomePagePage"].(string)
			nanoflow, _ := rbMap["HomePageNanoflow"].(string)
			if role == "" {
				continue
			}
			rbProfileID := mxgraph.NodeID(fmt.Sprintf("NavigationProfile:%s/role:%s", profileName, role))
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: &mxgraph.Node{
					ID:    rbProfileID,
					Label: "NavigationProfile",
					Props: map[string]any{
						"$Type":         typeName,
						"Name":          profileName + "/" + role,
						"Kind":          kind,
						"IsNative":      true,
						"Module":        module,
						"ParentProfile": profileName,
						"UserRole":      role,
						"QualifiedName": profileName + "/" + role,
					},
				},
			})
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_MENU_ITEM-->%s", profileID, rbProfileID)),
					From: profileID,
					To:   rbProfileID,
					Type: "HAS_MENU_ITEM",
				},
			})
			evts := a.indexHomeTarget(rbProfileID, profileName, page, nanoflow, module)
			events = append(events, evts...)
		}

		for _, item := range arrayVal(raw, "BottomBarItems") {
			barMap := toMap(item)
			if barMap == nil {
				continue
			}
			evts := a.indexBottomBarItem(barMap, profileID, module)
			events = append(events, evts...)
		}
	}

	for _, item := range arrayVal(raw, "OfflineEntityConfigs") {
		oeMap := toMap(item)
		if oeMap == nil {
			continue
		}
		entity, _ := oeMap["Entity"].(string)
		if entity == "" {
			continue
		}
		qn := qualifyName(entity, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_OFFLINE_ENTITY-->%s", profileID, qn)),
				From: profileID,
				To:   mxgraph.NodeID(qn),
				Type: "HAS_OFFLINE_ENTITY",
			},
		})
	}

	return events
}

func (a *NavigationAdapter) indexHomeTarget(profileID mxgraph.NodeID, profileName, page, microflow, module string) []mxgraph.Event {
	var events []mxgraph.Event
	if page != "" {
		qn := qualifyName(page, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_PAGE-->%s", profileID, qn)),
				From: profileID,
				To:   mxgraph.NodeID(qn),
				Type: "TARGETS_PAGE",
			},
		})
	}
	if microflow != "" {
		qn := qualifyName(microflow, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_MICROFLOW-->%s", profileID, qn)),
				From: profileID,
				To:   mxgraph.NodeID(qn),
				Type: "TARGETS_MICROFLOW",
			},
		})
	}
	return events
}

func (a *NavigationAdapter) indexMenuItem(raw map[string]any, parentID mxgraph.NodeID, module string, depth int) []mxgraph.Event {
	var events []mxgraph.Event

	caption := ""
	if capMap := toMap(raw["Caption"]); capMap != nil {
		caption = extractTextFromMap(capMap)
	}

	menuID := mxgraph.NodeID(fmt.Sprintf("NavMenuItem:%s/%s/%d", parentID, caption, depth))

	pageTarget := ""
	mfTarget := ""

	if action := toMap(raw["Action"]); action != nil {
		actionType, _ := action["$Type"].(string)
		switch {
		case strings.HasSuffix(actionType, "FormAction") || strings.HasSuffix(actionType, "PageClientAction"):
			if fs := toMap(action["FormSettings"]); fs != nil {
				pageTarget, _ = fs["Form"].(string)
			}
		case strings.HasSuffix(actionType, "MicroflowAction") || strings.HasSuffix(actionType, "MicroflowClientAction"):
			if ms := toMap(action["MicroflowSettings"]); ms != nil {
				mfTarget, _ = ms["Microflow"].(string)
			}
		}
	}

	node := &mxgraph.Node{
		ID:    menuID,
		Label: "NavigationMenuItem",
		Props: map[string]any{
			"$Type":   "NavigationMenuItem",
			"Name":    menuID,
			"Caption": caption,
			"Depth":   depth,
			"Module":  module,
			"QualifiedName": menuID,
		},
	}

	if pageTarget != "" {
		qn := qualifyName(pageTarget, module)
		node.Props["Page"] = qn
	}
	if mfTarget != "" {
		qn := qualifyName(mfTarget, module)
		node.Props["Microflow"] = qn
	}

	events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})

	events = append(events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--%s", parentID, menuID)),
			From: parentID,
			To:   menuID,
			Type: "HAS_MENU_ITEM",
		},
	})

	if pageTarget != "" {
		qn := qualifyName(pageTarget, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_PAGE-->%s", menuID, qn)),
				From: menuID,
				To:   mxgraph.NodeID(qn),
				Type: "TARGETS_PAGE",
			},
		})
	}
	if mfTarget != "" {
		qn := qualifyName(mfTarget, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_MICROFLOW-->%s", menuID, qn)),
				From: menuID,
				To:   mxgraph.NodeID(qn),
				Type: "TARGETS_MICROFLOW",
			},
		})
	}

	for _, item := range arrayVal(raw, "Items") {
		subMap := toMap(item)
		if subMap == nil {
			continue
		}
		evts := a.indexMenuItem(subMap, menuID, module, depth+1)
		events = append(events, evts...)
	}

	return events
}

func (a *NavigationAdapter) indexBottomBarItem(raw map[string]any, parentID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

	caption := ""
	if capMap := toMap(raw["Caption"]); capMap != nil {
		caption = extractTextFromMap(capMap)
	}
	page, _ := raw["Page"].(string)

	itemID := mxgraph.NodeID(fmt.Sprintf("NavMenuItem:%s/%s", parentID, caption))

	events = append(events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    itemID,
			Label: "NavigationMenuItem",
			Props: map[string]any{
				"$Type":   "NavigationMenuItem",
				"Name":    itemID,
				"Caption": caption,
				"Depth":   0,
				"Module":  module,
				"QualifiedName": itemID,
			},
		},
	})

	events = append(events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--%s", parentID, itemID)),
			From: parentID,
			To:   itemID,
			Type: "HAS_MENU_ITEM",
		},
	})

	if page != "" {
		qn := qualifyName(page, module)
		events = append(events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--TARGETS_PAGE-->%s", itemID, qn)),
				From: itemID,
				To:   mxgraph.NodeID(qn),
				Type: "TARGETS_PAGE",
			},
		})
	}

	return events
}

// extractTextFromMap 从 Caption 的 ClientTemplate 中提取文本。
func extractTextFromMap(raw map[string]any) string {
	if items := arrayVal(raw, "Items"); items != nil {
		for _, item := range items {
			if m := toMap(item); m != nil {
				if text, ok := m["Text"].(string); ok && text != "" {
					return text
				}
			}
		}
	}
	if items := arrayVal(raw, "Translations"); items != nil {
		for _, item := range items {
			if m := toMap(item); m != nil {
				if text, ok := m["Text"].(string); ok && text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func (a *NavigationAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
