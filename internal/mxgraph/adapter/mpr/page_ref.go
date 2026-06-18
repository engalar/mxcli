package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// PageRefAdapter 遍历页面 widget 树，提取实体和微流引用。
//
// 已检测的模式（raw BSON 路径）：
//   - Widget 的 DataSource 子文档中 entity 或 entityId 字段 → READS_ENTITY
//   - Widget 的 OnClickMicroflow 或 OnClickAction 子文档中 microflow 字段 → CALLS_MICROFLOW
//   - DataGrid 的 DataSource 子文档中 entity 字段 → READS_ENTITY
type PageRefAdapter struct {
	Model *modelsdk.Model
}

func (a *PageRefAdapter) Name() string { return "pageref" }

func (a *PageRefAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"READS_ENTITY", "Page", "Entity"},
			{"CALLS_MICROFLOW", "Page", "Microflow"},
		},
	}
}

func (a *PageRefAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
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
		if !strings.HasSuffix(typeName, "$Page") && !strings.HasSuffix(typeName, "$Form") {
			continue
		}

		raw := elem.Raw()
		if len(raw) == 0 {
			continue
		}

		pageID := mxgraph.NodeID(elem.ID())
		module := a.Model.ResolveModuleName(unit.ID)

		evts := a.walkPageRaw(raw, pageID, module)
		events = append(events, evts...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *PageRefAdapter) walkPageRaw(raw bson.Raw, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

	var doc map[string]any
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil
	}

	// 找到 FormCall/LayoutCall → Arguments → Widgets/Widget 树
	for _, key := range []string{"FormCall", "LayoutCall"} {
		val, ok := doc[key]
		if !ok {
			continue
		}
		formCall := toMap(val)
		if formCall == nil {
			continue
		}
		args := arrayVal(formCall, "Arguments")
		for _, arg := range args {
			argMap := toMap(arg)
			if argMap == nil {
				continue
			}
			widgets := arrayVal(argMap, "Widgets")
			if len(widgets) == 0 {
				if w := toMap(argMap["Widget"]); w != nil {
					widgets = []any{w}
				}
			}
			for _, w := range widgets {
				if wMap := toMap(w); wMap != nil {
					evts := a.walkWidgetMap(wMap, pageID, module)
					events = append(events, evts...)
				}
			}
		}
	}

	return events
}

func (a *PageRefAdapter) walkWidgetMap(w map[string]any, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

	// 1. 提取 DataSource 中的 entity 引用
	if ds := toMap(w["DataSource"]); ds != nil {
		if entityQN, ok := ds["entity"].(string); ok && entityQN != "" {
			qn := qualifyName(entityQN, module)
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--READS_ENTITY-->%s", pageID, qn)),
					From: pageID,
					To:   mxgraph.NodeID(qn),
					Type: "READS_ENTITY",
				},
			})
		}
	}

	// 2. 提取 OnClick 中的 microflow 引用
	if action := toMap(w["OnClickMicroflow"]); action != nil {
		if mfQN, ok := action["microflow"].(string); ok && mfQN != "" {
			qn := qualifyName(mfQN, module)
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--CALLS_MICROFLOW-->%s", pageID, qn)),
					From: pageID,
					To:   mxgraph.NodeID(qn),
					Type: "CALLS_MICROFLOW",
				},
			})
		}
	}
	if action := toMap(w["OnClickAction"]); action != nil {
		if mfQN, ok := action["microflow"].(string); ok && mfQN != "" {
			qn := qualifyName(mfQN, module)
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--CALLS_MICROFLOW-->%s", pageID, qn)),
					From: pageID,
					To:   mxgraph.NodeID(qn),
					Type: "CALLS_MICROFLOW",
				},
			})
		}
	}

	// 3. 递归子 widget
	for _, key := range []string{"Widgets", "widgets"} {
		for _, child := range arrayVal(w, key) {
			if childMap := toMap(child); childMap != nil {
				evts := a.walkWidgetMap(childMap, pageID, module)
				events = append(events, evts...)
			}
		}
	}
	// 单个 Widget（LayoutGrid child 等）
	if childMap := toMap(w["Widget"]); childMap != nil {
		evts := a.walkWidgetMap(childMap, pageID, module)
		events = append(events, evts...)
	}

	return events
}

// qualifyName 确保名称包含模块前缀。
func qualifyName(name, module string) string {
	if strings.Contains(name, ".") {
		return name
	}
	if module != "" {
		return module + "." + name
	}
	return name
}

// toMap 尝试将任意值转为 map[string]any。
func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch m := v.(type) {
	case map[string]any:
		return m
	case bson.M:
		return m
	case bson.D:
		result := make(map[string]any, len(m))
		for _, e := range m {
			result[e.Key] = e.Value
		}
		return result
	}
	return nil
}

// arrayVal 尝试从 map 中读取数组值，跳过版本前缀。
func arrayVal(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case bson.A:
		return dropVersionPrefix(arr)
	case []any:
		if len(arr) > 0 {
			if _, ok := arr[0].(int32); ok {
				return arr[1:]
			}
		}
		return arr
	}
	return nil
}

func (a *PageRefAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
