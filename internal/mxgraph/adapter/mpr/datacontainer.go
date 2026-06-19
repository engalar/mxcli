package mpr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// jsonBufPool reduces allocations for JSON marshaling of ChildWidgets/ContextVariables.
var jsonBufPool = sync.Pool{
	New: func() any { return new(strings.Builder) },
}

// DataContainerAdapter 遍历页面 widget 树，为每个数据容器（DataView/DataGrid/
// ListView/TemplateGrid/Gallery/DataGrid2）创建 DataContainer 节点，包含：
//   - 嵌套层级（HAS_DATA_CONTAINER 边）
//   - DataSource 类型、实体引用、微流引用
//   - 子 widget 外观摘要（ChildWidgets JSON）
//   - 语境变量（ContextVariables JSON）
//
// 不依赖 typed modelsdk 路径，走 raw BSON 遍历（与 WidgetInstanceAdapter
// 和 PageRefAdapter 相同的模式）。
type DataContainerAdapter struct {
	Source RawUnitSource
	Model  *modelsdk.Model // 用于解析 module 和 entity QN
}

// childWidgetSummary 描述数据容器内一个子 widget 的外观和条件性配置。
type childWidgetSummary struct {
	Name                  string `json:"name"`
	WidgetType            string `json:"widgetType"`
	Class                 string `json:"class,omitempty"`
	Style                 string `json:"style,omitempty"`
	Caption               string `json:"caption,omitempty"`
	Attribute             string `json:"attribute,omitempty"`
	ConditionalVisibility string `json:"condVis,omitempty"`
	ConditionalEditability string `json:"condEdit,omitempty"`
}

// containerCtxVar 描述数据容器层级可用的一个语境变量。
type containerCtxVar struct {
	Name       string `json:"name"`       // 变量名，如 "$currentObject"
	EntityType string `json:"entityType"` // 实体 QN
	Source     string `json:"source"`     // "datasource" | "parameter" | "parent" | "selection"
}

// dcWalkState 是遍历 widget 树时的状态。
type dcWalkState struct {
	depth      int
	parentID   mxgraph.NodeID
	entityCtx  string // 当前层级的实体 QN（$currentObject）
	parentEntity string // 父层级的实体 QN
	module     string
	pageQN     string
	widgetNameMap map[string]string // 第一遍收集: widgetName → entity QN
}

// dataSourceInfo 描述从 DataSource 子文档提取的信息。
type dataSourceInfo struct {
	dsType     string // "database" | "microflow" | "nanoflow" | "parameter" | "association" | "selection" | "none"
	entity     string // 实体 QN（已解析）
	microflow  string // 微流 QN
	paramName  string // 参数名
	listenTarget string // 监听目标 widget 名
}

func (a *DataContainerAdapter) Name() string { return "datacontainer" }

func (a *DataContainerAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"DataContainer"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_DATA_CONTAINER", "Page", "DataContainer"},
			{"HAS_DATA_CONTAINER", "DataContainer", "DataContainer"},
			{"HAS_DATASOURCE_ENTITY", "DataContainer", "Entity"},
			{"HAS_DATASOURCE_MICROFLOW", "DataContainer", "Microflow"},
			{"HAS_SELECTION_CONTEXT", "DataContainer", "DataContainer"},
		},
	}
}

func (a *DataContainerAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Source.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		typeName := unit.TypeName()
		if !strings.HasSuffix(typeName, "$Page") && !strings.HasSuffix(typeName, "$Form") {
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

		pageName, _ := doc["Name"].(string)
		pageQN := pageName
		if module != "" && pageName != "" {
			pageQN = module + "." + pageName
		}

		pageID := mxgraph.NodeID(unit.ID())

		// 单遍遍历: 同时构建 name→entity 映射和 DataContainer 事件
		// 先收集映射，再遍历，避免两次全量映射转换
		var walkBuf []mxgraph.Event

		nameMap := make(map[string]string, 128)
		state := &dcWalkState{
			depth:        0,
			parentID:     pageID,
			module:       module,
			pageQN:       pageQN,
			widgetNameMap: nameMap,
		}

		// 快速名称收集（不转换整棵 widget 树的 map，只取 Name 和 DataSource.entity）
		a.collectNamesFast(doc, state, nameMap)

		// 主遍历（复用已收集 nameMap）
		state2 := &dcWalkState{
			depth:         0,
			parentID:      pageID,
			module:        module,
			pageQN:        pageQN,
			widgetNameMap: nameMap,
		}

		a.walkPageWidgets(doc, state2, &walkBuf)
		events = append(events, walkBuf...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// collectNamesFast 轻量名称收集：只取 Name 和 DataSource，不转整棵 widget 树 map。
func (a *DataContainerAdapter) collectNamesFast(doc map[string]any, state *dcWalkState, nameMap map[string]string) {
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
			a.collectNamesFromWidgets(argMap, state, nameMap)
		}
	}
}

func (a *DataContainerAdapter) collectNamesFromWidgets(m map[string]any, state *dcWalkState, nameMap map[string]string) {
	widgets := arrayVal(m, "Widgets")
	if len(widgets) == 0 {
		if w := toMap(m["Widget"]); w != nil {
			widgets = []any{w}
		}
	}
	for _, w := range widgets {
		wMap := toMap(w)
		if wMap == nil {
			continue
		}
		a.collectNameOne(wMap, state, nameMap)
	}
}

func (a *DataContainerAdapter) collectNameOne(w map[string]any, state *dcWalkState, nameMap map[string]string) {
	typeName, _ := w["$Type"].(string)
	if typeName == "" {
		return
	}
	name, _ := w["Name"].(string)

	if isDataContainerType(typeName) || isListContainerType(typeName) || isWidgetType(typeName) {
		entityCtx := ""
		if ds := a.extractDataSource(w); ds.entity != "" {
			entityCtx = ds.entity
			if name != "" && entityCtx != "" {
				nameMap[name] = entityCtx
			}
		} else if name != "" && state.entityCtx != "" {
			nameMap[name] = state.entityCtx
		}

		subCtx := entityCtx
		if subCtx == "" {
			subCtx = state.entityCtx
		}

		a.collectNamesDeep(w, typeName, subCtx, nameMap)
	}
}

func (a *DataContainerAdapter) collectNamesDeep(w map[string]any, typeName string, entityCtx string, nameMap map[string]string) {
	for _, child := range arrayVal(w, "Widgets") {
		if cMap := toMap(child); cMap != nil {
			sub := &dcWalkState{entityCtx: entityCtx, widgetNameMap: nameMap}
			a.collectNameOne(cMap, sub, nameMap)
		}
	}

	if typeName == "Forms$TabControl" || typeName == "Pages$TabControl" {
		for _, tp := range arrayVal(w, "TabPages") {
			if tpMap := toMap(tp); tpMap != nil {
				for _, tw := range arrayVal(tpMap, "Widgets") {
					if twMap := toMap(tw); twMap != nil {
						sub := &dcWalkState{entityCtx: entityCtx, widgetNameMap: nameMap}
						a.collectNameOne(twMap, sub, nameMap)
					}
				}
			}
		}
	}

	if typeName == "Forms$ScrollContainer" || typeName == "Pages$ScrollContainer" {
		if center := toMap(w["CenterRegion"]); center != nil {
			sub := &dcWalkState{entityCtx: entityCtx, widgetNameMap: nameMap}
			a.collectNamesFromWidgets(center, sub, nameMap)
		}
	}

	if typeName == "Forms$LayoutGrid" || typeName == "Pages$LayoutGrid" {
		for _, row := range arrayVal(w, "Rows") {
			rowMap := toMap(row)
			if rowMap == nil {
				continue
			}
			for _, col := range arrayVal(rowMap, "Columns") {
				colMap := toMap(col)
				if colMap == nil {
					continue
				}
				for _, cw := range arrayVal(colMap, "Widgets") {
					if cwMap := toMap(cw); cwMap != nil {
						sub := &dcWalkState{entityCtx: entityCtx, widgetNameMap: nameMap}
						a.collectNameOne(cwMap, sub, nameMap)
					}
				}
			}
		}
	}
}



// walkPageWidgets 主要遍历路径，生成 DataContainer 节点。
func (a *DataContainerAdapter) walkPageWidgets(doc map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	for _, key := range []string{"FormCall", "LayoutCall"} {
		val, ok := doc[key]
		if !ok {
			continue
		}
		formCall := toMap(val)
		if formCall == nil {
			continue
		}
		for _, arg := range arrayVal(formCall, "Arguments") {
			argMap := toMap(arg)
			if argMap == nil {
				continue
			}
			a.walkWidgetsForDataContainer(argMap, state, events)
		}
	}
}

// walkWidgetsForDataContainer 遍历一批 widgets，生成 DataContainer 事件。
func (a *DataContainerAdapter) walkWidgetsForDataContainer(m map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	widgets := arrayVal(m, "Widgets")
	if len(widgets) == 0 {
		if w := toMap(m["Widget"]); w != nil {
			widgets = []any{w}
		}
	}
	for _, w := range widgets {
		wMap := toMap(w)
		if wMap == nil {
			continue
		}
		a.processWidgetNode(wMap, state, events)
	}
}

// processWidgetNode 处理一个 widget，如果是数据容器则创建节点并递归子 widget。
func (a *DataContainerAdapter) processWidgetNode(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	typeName, _ := w["$Type"].(string)
	if typeName == "" {
		return
	}

	switch {
	case typeName == "Forms$TabControl" || typeName == "Pages$TabControl":
		a.processTabControl(w, state, events)

	case typeName == "Forms$ScrollContainer" || typeName == "Pages$ScrollContainer":
		a.processScrollContainer(w, state, events)

	case typeName == "Forms$LayoutGrid" || typeName == "Pages$LayoutGrid":
		a.processLayoutGrid(w, state, events)

	case isDataContainerType(typeName) || isListContainerType(typeName):
		a.emitDataContainer(w, state, events)

	case isPluggableDataWidget(typeName):
		a.emitDataContainer(w, state, events)

	case typeName == "Forms$DivContainer" || typeName == "Pages$DivContainer" ||
		typeName == "Forms$GroupBox" || typeName == "Pages$GroupBox" ||
		typeName == "Forms$NavigationList" || typeName == "Pages$NavigationList":
		for _, child := range arrayVal(w, "Widgets") {
			if cMap := toMap(child); cMap != nil {
				a.processWidgetNode(cMap, state, events)
			}
		}
	}
}

func (a *DataContainerAdapter) emitDataContainer(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	typeName, _ := w["$Type"].(string)
	name, _ := w["Name"].(string)

	ds := a.extractDataSource(w)

	// 解析 TargetEntity
	targetEntity := ds.entity
	if ds.dsType == "selection" && ds.listenTarget != "" {
		if ent, ok := state.widgetNameMap[ds.listenTarget]; ok {
			targetEntity = ent
		}
	}
	if ds.dsType == "association" && ds.entity != "" {
		parts := strings.Split(ds.entity, "/")
		if len(parts) > 1 {
			targetEntity = parts[len(parts)-1]
		} else {
			targetEntity = ds.entity
		}
	}

	hasSelection, selectionName := a.extractSelectionSetting(w)

	dcID := mxgraph.NodeID(fmt.Sprintf("DC:%s/%s/%d", state.pageQN, name, state.depth))
	containerQN := state.pageQN + "." + name

	// 子 widget 收集（使用池化 builder）
	children := a.collectChildWidgets(w)
	ctxVars := a.computeContextVars(ds, targetEntity, state, hasSelection, selectionName)

	buf := jsonBufPool.Get().(*strings.Builder)
	buf.Reset()
	json.NewEncoder(buf).Encode(ctxVars)
	ctxVarsStr := buf.String()
	buf.Reset()
	json.NewEncoder(buf).Encode(children)
	childrenStr := buf.String()
	jsonBufPool.Put(buf)

	props := map[string]any{
		"$Type":            "DataContainer",
		"WidgetType":       a.shortType(typeName),
		"WidgetName":       name,
		"DataSourceType":   ds.dsType,
		"EntityPath":       ds.entity,
		"TargetEntity":     targetEntity,
		"DataSourceMicroflow": ds.microflow,
		"ParameterName":    ds.paramName,
		"ListenTargetWidget": ds.listenTarget,
		"HasSelection":     hasSelection,
		"SelectionName":    selectionName,
		"Depth":            state.depth,
		"PageQN":           state.pageQN,
		"Module":           state.module,
		"QualifiedName":    containerQN,
		"ChildWidgets":     childrenStr,
		"ContextVariables": ctxVarsStr,
	}

	*events = append(*events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{ID: dcID, Label: "DataContainer", Props: props},
	})

	*events = append(*events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_DATA_CONTAINER-->%s", state.parentID, dcID)),
			From: state.parentID,
			To:   dcID,
			Type: "HAS_DATA_CONTAINER",
		},
	})

	if targetEntity != "" && ds.dsType != "selection" {
		qn := qualifyName(targetEntity, state.module)
		*events = append(*events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_DATASOURCE_ENTITY-->%s", dcID, qn)),
				From: dcID,
				To:   mxgraph.NodeID(qn),
				Type: "HAS_DATASOURCE_ENTITY",
			},
		})
	}

	if ds.microflow != "" {
		qn := qualifyName(ds.microflow, state.module)
		*events = append(*events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_DATASOURCE_MICROFLOW-->%s", dcID, qn)),
				From: dcID,
				To:   mxgraph.NodeID(qn),
				Type: "HAS_DATASOURCE_MICROFLOW",
			},
		})
	}

	if ds.dsType == "selection" && ds.listenTarget != "" {
		srcDCID := mxgraph.NodeID(fmt.Sprintf("DC:%s/%s/0", state.pageQN, ds.listenTarget))
		*events = append(*events, mxgraph.Event{
			Type: mxgraph.EdgeCreated,
			Edge: &mxgraph.Edge{
				ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_SELECTION_CONTEXT-->%s", dcID, srcDCID)),
				From: dcID,
				To:   srcDCID,
				Type: "HAS_SELECTION_CONTEXT",
			},
		})
	}

	// 递归子 widget
	subState := &dcWalkState{
		depth:         state.depth + 1,
		parentID:      dcID,
		entityCtx:     targetEntity,
		parentEntity:  state.entityCtx,
		module:        state.module,
		pageQN:        state.pageQN,
		widgetNameMap: state.widgetNameMap,
	}

	for _, child := range arrayVal(w, "Widgets") {
		if cMap := toMap(child); cMap != nil {
			a.processWidgetNode(cMap, subState, events)
		}
	}

	a.walkExtraChildren(w, subState, events)
}

func (a *DataContainerAdapter) walkExtraChildren(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	typeName, _ := w["$Type"].(string)

	if typeName == "Forms$DataView" || typeName == "Pages$DataView" {
		for _, fw := range arrayVal(w, "FooterWidgets") {
			if fwMap := toMap(fw); fwMap != nil {
				a.processWidgetNode(fwMap, state, events)
			}
		}
	}

	if isPluggableDataWidget(typeName) {
		for _, cb := range arrayVal(w, "ControlBar") {
			if cbMap := toMap(cb); cbMap != nil {
				a.processWidgetNode(cbMap, state, events)
			}
		}
	}

	if typeName == "Forms$TabControl" || typeName == "Pages$TabControl" {
		for _, tp := range arrayVal(w, "TabPages") {
			if tpMap := toMap(tp); tpMap != nil {
				for _, tw := range arrayVal(tpMap, "Widgets") {
					if twMap := toMap(tw); twMap != nil {
						a.processWidgetNode(twMap, state, events)
					}
				}
			}
		}
	}

	if typeName == "Forms$LayoutGrid" || typeName == "Pages$LayoutGrid" {
		for _, row := range arrayVal(w, "Rows") {
			rowMap := toMap(row)
			if rowMap == nil {
				continue
			}
			for _, col := range arrayVal(rowMap, "Columns") {
				colMap := toMap(col)
				if colMap == nil {
					continue
				}
				for _, cw := range arrayVal(colMap, "Widgets") {
					if cwMap := toMap(cw); cwMap != nil {
						a.processWidgetNode(cwMap, state, events)
					}
				}
			}
		}
	}

	if typeName == "Forms$ScrollContainer" || typeName == "Pages$ScrollContainer" {
		if center := toMap(w["CenterRegion"]); center != nil {
			for _, cw := range arrayVal(center, "Widgets") {
				if cwMap := toMap(cw); cwMap != nil {
					a.processWidgetNode(cwMap, state, events)
				}
			}
		}
	}
}

func (a *DataContainerAdapter) processTabControl(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	for _, tp := range arrayVal(w, "TabPages") {
		if tpMap := toMap(tp); tpMap != nil {
			for _, tw := range arrayVal(tpMap, "Widgets") {
				if twMap := toMap(tw); twMap != nil {
					a.processWidgetNode(twMap, state, events)
				}
			}
		}
	}
}

func (a *DataContainerAdapter) processScrollContainer(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	if center := toMap(w["CenterRegion"]); center != nil {
		for _, cw := range arrayVal(center, "Widgets") {
			if cwMap := toMap(cw); cwMap != nil {
				a.processWidgetNode(cwMap, state, events)
			}
		}
	}
}

func (a *DataContainerAdapter) processLayoutGrid(w map[string]any, state *dcWalkState, events *[]mxgraph.Event) {
	for _, row := range arrayVal(w, "Rows") {
		rowMap := toMap(row)
		if rowMap == nil {
			continue
		}
		for _, col := range arrayVal(rowMap, "Columns") {
			colMap := toMap(col)
			if colMap == nil {
				continue
			}
			for _, cw := range arrayVal(colMap, "Widgets") {
				if cwMap := toMap(cw); cwMap != nil {
					a.processWidgetNode(cwMap, state, events)
				}
			}
		}
	}
}

// collectChildWidgets 收集数据容器内的子 widget 外观摘要。
func (a *DataContainerAdapter) collectChildWidgets(w map[string]any) []childWidgetSummary {
	var children []childWidgetSummary

	for _, child := range arrayVal(w, "Widgets") {
		if cMap := toMap(child); cMap != nil {
			children = append(children, a.extractChildSummary(cMap))
		}
	}

	// Extra child paths
	typeName, _ := w["$Type"].(string)

	if typeName == "Forms$DataView" || typeName == "Pages$DataView" {
		for _, fw := range arrayVal(w, "FooterWidgets") {
			if fwMap := toMap(fw); fwMap != nil {
				children = append(children, a.extractChildSummary(fwMap))
			}
		}
	}

	if isPluggableDataWidget(typeName) {
		for _, cb := range arrayVal(w, "ControlBar") {
			if cbMap := toMap(cb); cbMap != nil {
				children = append(children, a.extractChildSummary(cbMap))
			}
		}
	}

	return children
}

// extractChildSummary 从 widget raw map 提取外观摘要。
func (a *DataContainerAdapter) extractChildSummary(w map[string]any) childWidgetSummary {
	typeName, _ := w["$Type"].(string)
	name, _ := w["Name"].(string)

	s := childWidgetSummary{
		Name:       name,
		WidgetType: a.shortType(typeName),
	}

	if app := toMap(w["Appearance"]); app != nil {
		s.Class, _ = app["Class"].(string)
		s.Style, _ = app["Style"].(string)
		if dps := arrayVal(app, "DesignProperties"); len(dps) > 0 {
			// 简化：只标记有 design props 但不保存具体值
		}
	}

	if cvs := toMap(w["ConditionalVisibilitySettings"]); cvs != nil {
		if expr, ok := cvs["Expression"].(string); ok && expr != "" {
			s.ConditionalVisibility = expr
		}
	}
	if ces := toMap(w["ConditionalEditabilitySettings"]); ces != nil {
		if expr, ok := ces["Expression"].(string); ok && expr != "" {
			s.ConditionalEditability = expr
		}
	}

	// AttributeRef
	if attrRef := toMap(w["AttributeRef"]); attrRef != nil {
		if attr, ok := attrRef["Attribute"].(string); ok && attr != "" {
			s.Attribute = a.shortAttrName(attr)
		}
	}

	// Caption
	if capTemplate := toMap(w["CaptionTemplate"]); capTemplate != nil {
		s.Caption = a.extractText(capTemplate)
	}
	if s.Caption == "" {
		if caption, ok := w["Caption"].(string); ok {
			s.Caption = caption
		}
	}

	return s
}

// extractDataSource 从 widget 的 DataSource 子文档提取信息。
func (a *DataContainerAdapter) extractDataSource(w map[string]any) dataSourceInfo {
	ds := toMap(w["DataSource"])
	if ds == nil {
		return dataSourceInfo{dsType: "none"}
	}

	dsType, _ := ds["$Type"].(string)
	info := dataSourceInfo{}

	switch {
	case strings.HasSuffix(dsType, "DatabaseSource"):
		info.dsType = "database"
		if entityRef := toMap(ds["EntityRef"]); entityRef != nil {
			info.entity, _ = entityRef["Entity"].(string)
		}
		if info.entity == "" {
			info.entity, _ = ds["entity"].(string)
		}

	case strings.HasSuffix(dsType, "MicroflowSource"):
		info.dsType = "microflow"
		if settings := toMap(ds["MicroflowSettings"]); settings != nil {
			info.microflow, _ = settings["Microflow"].(string)
		}
		if info.microflow == "" {
			info.microflow, _ = ds["microflow"].(string)
		}

	case strings.HasSuffix(dsType, "NanoflowSource"):
		info.dsType = "nanoflow"
		if settings := toMap(ds["NanoflowSettings"]); settings != nil {
			info.microflow, _ = settings["Nanoflow"].(string)
		}

	case strings.HasSuffix(dsType, "DataViewSource"):
		info.dsType = "parameter"
		if srcVar := toMap(ds["SourceVariable"]); srcVar != nil {
			info.paramName, _ = srcVar["PageParameter"].(string)
		}
		if entityRef := toMap(ds["EntityRef"]); entityRef != nil {
			info.entity, _ = entityRef["Entity"].(string)
		}
		if info.entity == "" {
			info.entity, _ = ds["EntityPath"].(string)
		}

	case strings.HasSuffix(dsType, "EntityPathSource"):
		info.dsType = "association"
		info.entity, _ = ds["EntityPath"].(string)

	case strings.HasSuffix(dsType, "ListenTargetSource"):
		info.dsType = "selection"
		info.listenTarget, _ = ds["ListenTarget"].(string)

	case strings.HasSuffix(dsType, "ListViewXPathSource"):
		info.dsType = "database"
		if entityRef := toMap(ds["EntityRef"]); entityRef != nil {
			info.entity, _ = entityRef["Entity"].(string)
		}

	default:
		info.dsType = "none"
	}

	return info
}

// extractSelectionSetting 检测 DataGrid/Gallery 的行选择设置。
func (a *DataContainerAdapter) extractSelectionSetting(w map[string]any) (hasSelection bool, selectionName string) {
	typeName, _ := w["$Type"].(string)

	// 原生 Forms 类型
	if typeName == "Forms$DataGrid" || typeName == "Forms$ListView" || typeName == "Forms$TemplateGrid" {
		if sel := toMap(w["Selection"]); sel != nil {
			if name, ok := sel["Name"].(string); ok && name != "" {
				return true, name
			}
		}
	}

	// Pluggable widgets (DataGrid2/Gallery) — Selection 在 Object.Properties 中
	obj := toMap(w["Object"])
	if obj == nil {
		return false, ""
	}
	for _, prop := range arrayVal(obj, "Properties") {
		propMap := toMap(prop)
		if propMap == nil {
			continue
		}
		if key, _ := propMap["Key"].(string); key != "selection" {
			continue
		}
		val := toMap(propMap["Value"])
		if val == nil {
			continue
		}
		selMap := toMap(val["Selection"])
		if selMap == nil {
			continue
		}
		if name, ok := selMap["name"].(string); ok && name != "" {
			return true, name
		}
	}

	return false, ""
}

func (a *DataContainerAdapter) computeContextVars(ds dataSourceInfo, targetEntity string, state *dcWalkState, hasSelection bool, selectionName string) []containerCtxVar {
	var vars []containerCtxVar

	// $currentObject
	if targetEntity != "" {
		vars = append(vars, containerCtxVar{
			Name:       "$currentObject",
			EntityType: targetEntity,
			Source:     "datasource",
		})
	}

	// $parent
	if state.parentEntity != "" {
		vars = append(vars, containerCtxVar{
			Name:       "$parent",
			EntityType: state.parentEntity,
			Source:     "parent",
		})
	}

	// $paramName
	if ds.paramName != "" {
		vars = append(vars, containerCtxVar{
			Name:       "$" + ds.paramName,
			EntityType: targetEntity,
			Source:     "parameter",
		})
	}

	// $selectionName
	if hasSelection && selectionName != "" {
		vars = append(vars, containerCtxVar{
			Name:       "$" + selectionName,
			EntityType: targetEntity,
			Source:     "selection",
		})
	}

	// 继承上级的 selection 变量（传递但不改名）
	// 注意：在当前简化实现中，上级的 selection 变量在上级 context 中管理，
	// 这里我们只传递 $parent 信息

	return vars
}

func (a *DataContainerAdapter) shortType(typeName string) string {
	parts := strings.Split(typeName, "$")
	if len(parts) > 1 {
		return parts[1]
	}
	return typeName
}

func (a *DataContainerAdapter) shortAttrName(attr string) string {
	if idx := strings.LastIndex(attr, "."); idx >= 0 {
		return attr[idx+1:]
	}
	return attr
}

func (a *DataContainerAdapter) extractText(tmpl map[string]any) string {
	if inner := toMap(tmpl["Template"]); inner != nil {
		for _, item := range arrayVal(inner, "Items") {
			if m := toMap(item); m != nil {
				if text, ok := m["Text"].(string); ok && text != "" {
					return text
				}
			}
		}
	}
	for _, item := range arrayVal(tmpl, "Items") {
		if m := toMap(item); m != nil {
			if text, ok := m["Text"].(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}

func (a *DataContainerAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

// isDataContainerType 判断是否为单对象数据容器。
func isDataContainerType(typeName string) bool {
	return typeName == "Forms$DataView" || typeName == "Pages$DataView"
}

// isListContainerType 判断是否为列表数据容器。
func isListContainerType(typeName string) bool {
	return typeName == "Forms$DataGrid" || typeName == "Forms$ListView" ||
		typeName == "Forms$TemplateGrid" || typeName == "Pages$DataGrid" ||
		typeName == "Pages$ListView"
}

// isPluggableDataWidget 判断是否为 pluggable 数据 widget（DataGrid2/Gallery）。
func isPluggableDataWidget(typeName string) bool {
	// CustomWidgets 类型
	if !strings.HasPrefix(typeName, "CustomWidgets$") {
		return false
	}
	return true // 所有 CustomWidget 都可能包含数据源；具体类型在 extractDataSource 中区分
}
