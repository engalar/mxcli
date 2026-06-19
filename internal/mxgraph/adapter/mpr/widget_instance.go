package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// =============================================================================
// 反模式说明：modelsdk 类型系统覆盖不全，需 raw BSON 回退
// =============================================================================
//
// 根因：modelsdk 为 Forms$FormCallArgument 注册了类型，但它的 typed
// Properties 只暴露了 Parameter/Variable/Argument 三个字段——Widget 和
// Widgets 字段（承载实际的 widget 树）未在 generated type 中声明，因此
// 通过 typed 路径（elem.Properties() → ChildListProperty → ChildElements()
// → inspectWidget）永远无法到达 widget 树。
//
// 为什么 typed 路径不可用？
//   - modelsdk.LoadUnit(unitID) → 完整 typed codec 解码 → 返回 Forms$Page
//   - Forms$Page.layoutCall → Forms$LayoutCall (typed, Props=2)
//   - LayoutCall.arguments → Forms$FormCallArgument (typed, Props=3)
//   - FormCallArgument.Properties() → [Parameter, Variable, Argument] ← 没有 Widget!
//   - 实际 BSON 中还藏了 Widget/Widgets 字段，但 typed 不可见。
//
// 解决方案（三级策略）：
//   1. DIP（依赖倒置）：定义 RawUnitSource 接口取代直接依赖 modelsdk.Model，
//      使 adapter 不依赖底层 BSON 实现。
//   2. 绕过 codec（性能）：ModelsdkUnitSource 通过 modelsdk.GetRawUnitBytes()
//      直接获取 raw BSON bytes，跳过 typed codec 解码（codec 解码会递归
//      实例化整个 widget 树，拖慢 2s+）。
//   3. raw BSON 导航 + map 字段读取：用 bson.Unmarshal 一次把整页 BSON 展开为
//      map[string]any，然后走 knownContainerKeys → walkRawMap → 递归遍历。
//      对每个 widget，直接用 extractAppearanceFromMap(m) 从 map 中读取
//      Appearance.Class / .Style / .DesignProperties，不再经过 codec 的
//      map→BSON→element 来回编解码（decodeRawChild 的反模式）。
//
// 为什么不直接用 modelsdk 的 typed 路径？
//   - typed 路径不可到达 FormCallArgument 内的 Widgets（类型定义缺失）
//   - 即使修复类型定义，typed 路径仍然要经过 codec 解码整个 widget 树，
//     而 raw BSON 的 map 导航跳过 codec 直接访问字段，吞吐量高 1-2 个数量级。
//
// =============================================================================

// RawUnit 是 WidgetInstanceAdapter 需要的页面/片段单元的最小接口。
// 只暴露 ID、类型名、原始 BSON 数据——不依赖 modelsdk 的类型系统。
// 遵循 DIP：高层模块不应依赖低层模块，两者都应依赖抽象。
type RawUnit interface {
	ID() string
	TypeName() string
	Raw() []byte
}

// RawUnitSource 是页面/片段单元的来源抽象。
// WidgetInstanceAdapter 依赖此接口而非 modelsdk.Model。
type RawUnitSource interface {
	Units() []RawUnit
	ResolveModuleName(unitID string) string
}

// WidgetInstanceAdapter 遍历页面和片段中的 widget 树，提取 Appearance 数据。
// 每个带样式信息的 widget 实例生成一个 WidgetInstance 节点。
// 只依赖 RawUnitSource 接口，不依赖 modelsdk。
type WidgetInstanceAdapter struct {
	Source   RawUnitSource
	DocCache BsonDocCache // 可选：共享 BSON 解码缓存
}

func (a *WidgetInstanceAdapter) Name() string { return "widgetinstance" }

func (a *WidgetInstanceAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"WidgetInstance"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_WIDGET_INSTANCE", "Page", "WidgetInstance"},
			{"HAS_WIDGET_INSTANCE", "Snippet", "WidgetInstance"},
		},
	}
}

func (a *WidgetInstanceAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Source.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		typeName := unit.TypeName()
		var containerLabel mxgraph.Label
		switch {
		case strings.HasSuffix(typeName, "$Page") || strings.HasSuffix(typeName, "$Form"):
			containerLabel = "Page"
		case strings.HasSuffix(typeName, "$Snippet"):
			containerLabel = "Snippet"
		default:
			continue
		}

		module := a.Source.ResolveModuleName(unit.ID())
		containerID := mxgraph.NodeID(unit.ID())
		uid := unit.ID()

		// 尝试从缓存获取已解码文档
		var doc map[string]any
		useCache := a.DocCache != nil
		if useCache {
			if cached, ok := a.DocCache.Get(uid); ok {
				doc = cached
			}
		}
		if doc == nil {
			if raw := unit.Raw(); len(raw) > 0 {
				if err := bson.Unmarshal(raw, &doc); err != nil {
					continue
				}
				if useCache {
					a.DocCache.Set(uid, doc)
				}
			} else {
				continue
			}
		}

		wiEvents := a.walkDoc(doc, containerID, containerLabel, module)
		events = append(events, wiEvents...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// walkDoc 从已解码的 map 遍历 widget 树，
// 直接从 map 中读取 Appearance（不经过 modelsdk codec 解码）。
func (a *WidgetInstanceAdapter) walkDoc(
	doc map[string]any,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	if doc == nil {
		return nil
	}
	return a.walkRawMap(doc, containerID, containerLabel, module)
}

// walkRawMap 递归遍历 raw map 中的 widget 容器 key。
func (a *WidgetInstanceAdapter) walkRawMap(
	m map[string]any,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	var events []mxgraph.Event

	for _, rawKey := range knownContainerKeys(m) {
		val := m[rawKey]

		if d, ok := val.(bson.D); ok {
			if len(d) == 0 {
				continue
			}
			evts := a.processRawDoc(dToMap(d), containerID, containerLabel, module)
			events = append(events, evts...)
		}

		if arr, ok := val.(bson.A); ok {
			items := dropVersionPrefix(arr)
			for _, item := range items {
				switch v := item.(type) {
				case bson.D:
					evts := a.processRawDoc(dToMap(v), containerID, containerLabel, module)
					events = append(events, evts...)
				case bson.M:
					evts := a.processRawDoc(v, containerID, containerLabel, module)
					events = append(events, evts...)
				}
			}
		}
	}

	return events
}

// knownContainerKeys 从 raw BSON doc 中找出已知的 widget 容器 key。
func knownContainerKeys(doc map[string]any) []string {
	candidates := []string{"FormCall", "LayoutCall", "Widget", "Widgets", "Arguments"}
	var keys []string
	for _, k := range candidates {
		if _, ok := doc[k]; ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// processRawDoc 处理单个 raw BSON sub-document。
// widget 类型 → inspectRawWidget（直接从 map 读 Appearance）。
// 容器类型 → 递归 walkRawMap。
func (a *WidgetInstanceAdapter) processRawDoc(
	m map[string]any,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	typeName, _ := m["$Type"].(string)
	if typeName == "" {
		return nil
	}

	var events []mxgraph.Event

	if isWidgetType(typeName) {
		// 直接从 map 提取 Appearance，不经过 modelsdk
		evts := a.inspectRawWidget(m, containerID, containerLabel, module)
		events = append(events, evts...)
		// 递归遍历子 widget（Widgets 数组）
		events = append(events, a.walkRawMap(m, containerID, containerLabel, module)...)
	} else {
		// 容器类型，递归进去
		events = append(events, a.walkRawMap(m, containerID, containerLabel, module)...)
	}

	return events
}

// inspectRawWidget 直接从 raw map 读取 widget 的 Name, Class, Style, DesignProperties。
// 不经过 modelsdk codec，无需 decodeRawChild 的 map→BSON→element 来回。
func (a *WidgetInstanceAdapter) inspectRawWidget(
	m map[string]any,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	typeName, _ := m["$Type"].(string)
	if typeName == "" || !isWidgetType(typeName) {
		return nil
	}

	widgetName, _ := m["Name"].(string)
	if widgetName == "" {
		return nil
	}

	// 直接从 map 提取 Appearance（不经过 codec）
	class, style, dps := extractAppearanceFromMap(m)
	if class == "" && style == "" && len(dps) == 0 {
		return nil
	}

	// 用 $ID 或 Name 作为节点 ID
	nodeID := mxgraph.NodeID(widgetName)
	if id, ok := m["$ID"].(string); ok && id != "" {
		nodeID = mxgraph.NodeID(id)
	}
	if bin, ok := m["$ID"].(bson.Binary); ok {
		nodeID = mxgraph.NodeID(fmt.Sprintf("%x", bin.Data))
	}

	qn := fmt.Sprintf("%s.%s", module, widgetName)
	if module == "" {
		qn = widgetName
	}

	condVis := extractConditionalVisibility(m)
	condEdit := extractConditionalEditability(m)

	props := map[string]any{
		"$Type":             "WidgetInstance",
		"Name":              widgetName,
		"WidgetType":        shortTypeName(typeName),
		"Class":             class,
		"Style":             style,
		"DesignProperties":  dps,
		"ConditionalVisibility":  condVis,
		"ConditionalEditability": condEdit,
		"QualifiedName":     qn,
		"Module":            module,
	}

	var events []mxgraph.Event
	events = append(events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{ID: nodeID, Label: "WidgetInstance", Props: props},
	})
	events = append(events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_WIDGET_INSTANCE-->%s", containerID, nodeID)),
			From: containerID,
			To:   nodeID,
			Type: "HAS_WIDGET_INSTANCE",
		},
	})

	return events
}

// extractAppearanceFromMap 直接从 raw map 的 Appearance 子文档提取 Class, Style, DesignProperties。
// 不经过 modelsdk codec 解码，纯 map 访问。
func extractAppearanceFromMap(m map[string]any) (class, style string, designProps map[string]string) {
	appRaw, ok := m["Appearance"].(bson.D)
	if !ok {
		appMap, ok2 := m["Appearance"].(map[string]any)
		if !ok2 {
			// 也可能根本不存在
			return
		}
		class, _ = appMap["Class"].(string)
		style, _ = appMap["Style"].(string)
		if dps, ok := appMap["DesignProperties"].(bson.A); ok {
			designProps = extractRawDesignProps(dps)
		}
		return
	}
	app := dToMap(appRaw)
	class, _ = app["Class"].(string)
	style, _ = app["Style"].(string)
	if dps, ok := app["DesignProperties"].(bson.A); ok {
		designProps = extractRawDesignProps(dps)
	}
	return
}

// extractRawDesignProps 直接遍历 BSON 数组中的 DesignPropertyValue。
func extractRawDesignProps(arr bson.A) map[string]string {
	result := make(map[string]string)
	items := dropVersionPrefix(arr)
	for _, item := range items {
		var dpMap map[string]any
		switch v := item.(type) {
		case bson.D:
			dpMap = dToMap(v)
		case map[string]any:
			dpMap = v
		default:
			continue
		}

		key, _ := dpMap["Key"].(string)
		if key == "" {
			continue
		}

		// Value 子文档
		valDoc, ok := dpMap["Value"].(bson.D)
		if !ok {
			valMap, ok2 := dpMap["Value"].(map[string]any)
			if !ok2 {
				result[key] = "ON"
				continue
			}
			innerType, _ := valMap["$Type"].(string)
			innerType = normalizeTypeName(innerType)
			switch {
			case strings.Contains(innerType, "ToggleDesignPropertyValue"):
				result[key] = "ON"
			case strings.Contains(innerType, "OptionDesignPropertyValue"):
				if opt, ok := valMap["Option"].(string); ok {
					result[key] = opt
				}
			default:
				result[key] = "ON"
			}
			continue
		}
		valMap := dToMap(valDoc)
		innerType, _ := valMap["$Type"].(string)
		innerType = normalizeTypeName(innerType)
		switch {
		case strings.Contains(innerType, "ToggleDesignPropertyValue"):
			result[key] = "ON"
		case strings.Contains(innerType, "OptionDesignPropertyValue"):
			if opt, ok := valMap["Option"].(string); ok {
				result[key] = opt
			}
		default:
			result[key] = "ON"
		}
	}
	return result
}

// dToMap 将 bson.D 转为 map[string]any。
func dToMap(d bson.D) map[string]any {
	m := make(map[string]any, len(d))
	for _, e := range d {
		m[e.Key] = e.Value
	}
	return m
}

// dropVersionPrefix 移除 Mendix BSON 数组开头的版本整数。
func dropVersionPrefix(arr bson.A) []any {
	if len(arr) == 0 {
		return nil
	}
	// Mendix 数组格式：[int32(version), elem1, elem2, ...]
	if _, ok := arr[0].(int32); ok {
		return arr[1:]
	}
	return arr
}

// isWidgetType 判断是否为可渲染的 widget 类型。
func isWidgetType(typeName string) bool {
	skipTypes := map[string]bool{
		"Forms$LayoutCall":                true,
		"Forms$LayoutCallArgument":        true,
		"Forms$FormCall":                  true,
		"Forms$FormCallArgument":          true,
		"Forms$Appearance":                true,
		"Forms$DesignPropertyValue":       true,
		"Forms$OptionDesignPropertyValue": true,
		"Forms$ToggleDesignPropertyValue":  true,
		"Forms$CustomDesignPropertyValue":  true,
		"Forms$ConditionalVisibilityWidget": true,
	}
	if skipTypes[typeName] {
		return false
	}
	return strings.HasPrefix(typeName, "Forms$") || strings.HasPrefix(typeName, "Pages$")
}

func shortTypeName(bsonType string) string {
	parts := strings.Split(bsonType, "$")
	if len(parts) < 2 {
		return bsonType
	}
	return parts[1]
}

func normalizeTypeName(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.Split(name, "$")
	if len(parts) < 2 {
		return name
	}
	return "$" + parts[len(parts)-1]
}

// extractConditionalVisibility 从 raw widget map 提取条件可见性表达式。
func extractConditionalVisibility(m map[string]any) string {
	if cvs := toMap(m["ConditionalVisibilitySettings"]); cvs != nil {
		if expr, ok := cvs["Expression"].(string); ok && expr != "" {
			return expr
		}
	}
	return ""
}

// extractConditionalEditability 从 raw widget map 提取条件可编辑性表达式。
func extractConditionalEditability(m map[string]any) string {
	if ces := toMap(m["ConditionalEditabilitySettings"]); ces != nil {
		if expr, ok := ces["Expression"].(string); ok && expr != "" {
			return expr
		}
	}
	return ""
}

func (a *WidgetInstanceAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
