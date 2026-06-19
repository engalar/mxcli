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
//   - Widget 的 OnClickMicroflow/OnClickAction/Action 中 microflow 字段 → CALLS_MICROFLOW
//   - Widget 的 Action 中 FormAction → SHOWS_PAGE（页面级导航）
//   - DataGrid 的 DataSource 子文档中 entity 字段 → READS_ENTITY
type PageRefAdapter struct {
	Model    *modelsdk.Model
	DocCache BsonDocCache // 可选：共享 BSON 解码缓存
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
			{"SHOWS_PAGE", "Page", "Page"},
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
		uid := string(elem.ID())

		// 优先用缓存
		var doc map[string]any
		useCache := a.DocCache != nil
		if useCache {
			if cached, ok := a.DocCache.Get(uid); ok {
				doc = cached
			}
		}
		if doc == nil {
			if err := bson.Unmarshal(raw, &doc); err != nil {
				continue
			}
			if useCache {
				a.DocCache.Set(uid, doc)
			}
		}

		evts := a.walkPageDoc(doc, pageID, module)
		events = append(events, evts...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *PageRefAdapter) walkPageDoc(doc map[string]any, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

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

	// 1. 提取 DataSource 中的 entity/EntityRef 引用
	if ds := toMap(w["DataSource"]); ds != nil {
		entityQN := a.extractEntityRef(ds, module)
		if entityQN != "" {
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

	// 2. 提取 OnClick / Action 中的 microflow/page 引用
	events = append(events, a.extractActionRefs(w, pageID, module)...)

	// 3. 提取插拔式 widget（DataGrid2/Gallery 等）的 Object.Properties 中的操作引用
	evts3 := a.extractPluggableActions(w, pageID, module)
	events = append(events, evts3...)

	// 4. 递归子 widget
	for _, key := range []string{"Widgets", "widgets"} {
		for _, child := range arrayVal(w, key) {
			if childMap := toMap(child); childMap != nil {
				evts := a.walkWidgetMap(childMap, pageID, module)
				events = append(events, evts...)
			}
		}
	}
	if childMap := toMap(w["Widget"]); childMap != nil {
		evts := a.walkWidgetMap(childMap, pageID, module)
		events = append(events, evts...)
	}
	// DataGrid columns
	if cols := arrayVal(w, "Columns"); len(cols) > 0 {
		for _, col := range cols {
			if colMap := toMap(col); colMap != nil {
				evts := a.walkWidgetMap(colMap, pageID, module)
				events = append(events, evts...)
			}
		}
	}
	// TabControl sub-widgets
	for _, tp := range arrayVal(w, "TabPages") {
		if tpMap := toMap(tp); tpMap != nil {
			for _, tw := range arrayVal(tpMap, "Widgets") {
				if twMap := toMap(tw); twMap != nil {
					evts := a.walkWidgetMap(twMap, pageID, module)
					events = append(events, evts...)
				}
			}
		}
	}
	// LayoutGrid sub-widgets
	for _, row := range arrayVal(w, "Rows") {
		if rowMap := toMap(row); rowMap != nil {
			for _, col := range arrayVal(rowMap, "Columns") {
				if colMap := toMap(col); colMap != nil {
					for _, cw := range arrayVal(colMap, "Widgets") {
						if cwMap := toMap(cw); cwMap != nil {
							evts := a.walkWidgetMap(cwMap, pageID, module)
							events = append(events, evts...)
						}
					}
				}
			}
		}
	}

	return events
}

// extractActionRefs 提取 widget 的操作引用（OnClick + Action）。
// 返回的事件包含 CALLS_MICROFLOW（调微流）和 SHOWS_PAGE（开页面）。
func (a *PageRefAdapter) extractActionRefs(w map[string]any, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

	// 综合检查所有可能的操作字段（从现代到遗留）
	actionSources := []struct {
		sourceKey string      // BSON 字段名
		innerKey  string      // 字段内部的操作类型 key（"" 表示直接是操作）
	}{
		{"Action", ""},              // 现代 ActionButton/LinkButton
		{"OnClickMicroflow", "microflow"},
		{"OnClickAction", "microflow"},
	}

	for _, src := range actionSources {
		action := toMap(w[src.sourceKey])
		if action == nil {
			continue
		}

		actionType, _ := action["$Type"].(string)
		mfQN := ""
		pageQN := ""

		switch {
		case actionType == "" && src.innerKey != "":
			// 遗留格式：OnClickMicroflow → { microflow: "..." }
			mfQN, _ = action[src.innerKey].(string)

		case strings.HasSuffix(actionType, "MicroflowAction"),
			strings.HasSuffix(actionType, "MicroflowClientAction"):
			if ms := toMap(action["MicroflowSettings"]); ms != nil {
				mfQN, _ = ms["Microflow"].(string)
			}

		case strings.HasSuffix(actionType, "FormAction"),
			strings.HasSuffix(actionType, "PageClientAction"):
			if fs := toMap(action["FormSettings"]); fs != nil {
				pageQN, _ = fs["Form"].(string)
			}

		case strings.HasSuffix(actionType, "NanoflowAction"),
			strings.HasSuffix(actionType, "NanoflowClientAction"):
			if ns := toMap(action["NanoflowSettings"]); ns != nil {
				mfQN, _ = ns["Nanoflow"].(string)
			}

		case strings.HasSuffix(actionType, "OpenLinkAction"),
			strings.HasSuffix(actionType, "OpenLinkClientAction"):
			// URL 打开，不产生边

		case strings.HasSuffix(actionType, "NoAction"),
			strings.HasSuffix(actionType, "NoClientAction"):
			// 无操作
		}

		if mfQN != "" {
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
		if pageQN != "" {
			qn := qualifyName(pageQN, module)
			events = append(events, mxgraph.Event{
				Type: mxgraph.EdgeCreated,
				Edge: &mxgraph.Edge{
					ID:   mxgraph.NodeID(fmt.Sprintf("%s--SHOWS_PAGE-->%s", pageID, qn)),
					From: pageID,
					To:   mxgraph.NodeID(qn),
					Type: "SHOWS_PAGE",
				},
			})
		}
	}

	return events
}

// extractEntityRef 从 DataSource 中提取实体引用。
func (a *PageRefAdapter) extractEntityRef(ds map[string]any, module string) string {
	// 1. 直接 entity 字段（遗留格式）
	if e, ok := ds["entity"].(string); ok && e != "" {
		return e
	}
	// 2. EntityRef.Entity（DatabaseSource 等现代格式）
	if ref := toMap(ds["EntityRef"]); ref != nil {
		if e, ok := ref["Entity"].(string); ok && e != "" {
			return e
		}
	}
	// 3. EntityPath（AssociationSource）
	if e, ok := ds["EntityPath"].(string); ok && e != "" {
		parts := strings.Split(e, "/")
		return parts[len(parts)-1]
	}
	return ""
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

// extractPluggableActions 从插拔式 widget（DataGrid2/Gallery 等）的
// Object.Properties 中提取操作引用。
//
// 使用 TypePointer → PropertyKey 映射找到正确属性，再遍历属性值中的操作引用。
func (a *PageRefAdapter) extractPluggableActions(w map[string]any, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	obj := toMap(w["Object"])
	if obj == nil {
		return nil
	}

	propKeyMap := a.buildPropTypeKeyMap(w)
	if len(propKeyMap) == 0 {
		return nil
	}

	var events []mxgraph.Event
	for _, prop := range arrayVal(obj, "Properties") {
		propMap := toMap(prop)
		if propMap == nil {
			continue
		}
		typePtr := extractBinaryIDFromMap(propMap["TypePointer"])
		key := propKeyMap[typePtr]
		if key == "" {
			continue
		}
		value := toMap(propMap["Value"])
		if value == nil {
			continue
		}

		switch {
		case key == "onClick" || key == "onChange" || key == "action":
			// 直接操作属性
			evts := a.extractActionRefsFromMap(value, pageID, module)
			events = append(events, evts...)

		case key == "columns" || key == "DataGridColumns":
			// 列定义，每列可能有 onClick
			for _, col := range arrayVal(value, "columns") {
				if colMap := toMap(col); colMap != nil {
					if onClick := toMap(colMap["onClick"]); onClick != nil {
						evts := a.extractActionRefsFromMap(onClick, pageID, module)
						events = append(events, evts...)
					}
				}
			}

		case key == "filtersPlaceholder" || key == "content":
			// 嵌套 widget 区域
			for _, wgt := range arrayVal(value, "Widgets") {
				if wgtMap := toMap(wgt); wgtMap != nil {
					evts := a.walkWidgetMap(wgtMap, pageID, module)
					events = append(events, evts...)
				}
			}
		}
	}
	return events
}

// buildPropTypeKeyMap 构建 TypePointer ID → 属性名的映射。
// 用于解析插拔式 widget 的 Object.Properties。
func (a *PageRefAdapter) buildPropTypeKeyMap(w map[string]any) map[string]string {
	result := make(map[string]string)
	typeObj := toMap(w["Type"])
	if typeObj == nil {
		return result
	}
	objType := toMap(typeObj["ObjectType"])
	if objType == nil {
		return result
	}
	for _, pt := range arrayVal(objType, "PropertyTypes") {
		ptMap := toMap(pt)
		if ptMap == nil {
			continue
		}
		key, _ := ptMap["PropertyKey"].(string)
		if key == "" {
			continue
		}
		id := extractBinaryIDFromMap(ptMap["$ID"])
		if id != "" {
			result[id] = key
		}
	}
	return result
}

// extractBinaryIDFromMap 从 BSON map 值中提取二进制 ID 的字符串表示。
func extractBinaryIDFromMap(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return fmt.Sprintf("%x", val)
	case bson.Binary:
		return fmt.Sprintf("%x", val.Data)
	case map[string]any:
		if id, ok := val["$ID"].(string); ok {
			return id
		}
		if b, ok := val["$ID"].(bson.Binary); ok {
			return fmt.Sprintf("%x", b.Data)
		}
	}
	return ""
}

// extractActionRefsFromMap 从 map 中提取微流/页面引用并返回事件。
// 与 extractActionRefs 逻辑相同但直接操作 map。
func (a *PageRefAdapter) extractActionRefsFromMap(m map[string]any, pageID mxgraph.NodeID, module string) []mxgraph.Event {
	var events []mxgraph.Event

	actionType, _ := m["$Type"].(string)

	// 直接 microflow 字段（简化格式）
	if mfQN, ok := m["microflow"].(string); ok && mfQN != "" {
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

	// 按 $Type 解析
	switch {
	case strings.HasSuffix(actionType, "MicroflowAction"),
		strings.HasSuffix(actionType, "MicroflowClientAction"):
		if ms := toMap(m["MicroflowSettings"]); ms != nil {
			if mfQN, ok := ms["Microflow"].(string); ok && mfQN != "" {
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

	case strings.HasSuffix(actionType, "FormAction"),
		strings.HasSuffix(actionType, "PageClientAction"):
		if fs := toMap(m["FormSettings"]); fs != nil {
			if pageQN, ok := fs["Form"].(string); ok && pageQN != "" {
				qn := qualifyName(pageQN, module)
				events = append(events, mxgraph.Event{
					Type: mxgraph.EdgeCreated,
					Edge: &mxgraph.Edge{
						ID:   mxgraph.NodeID(fmt.Sprintf("%s--SHOWS_PAGE-->%s", pageID, qn)),
						From: pageID,
						To:   mxgraph.NodeID(qn),
						Type: "SHOWS_PAGE",
					},
				})
			}
		}

	case strings.HasSuffix(actionType, "NanoflowAction"),
		strings.HasSuffix(actionType, "NanoflowClientAction"):
		if ns := toMap(m["NanoflowSettings"]); ns != nil {
			if nfQN, ok := ns["Nanoflow"].(string); ok && nfQN != "" {
				qn := qualifyName(nfQN, module)
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
	}

	return events
}

func (a *PageRefAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
