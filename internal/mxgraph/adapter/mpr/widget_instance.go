package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// WidgetInstanceAdapter 遍历页面和片段中的 widget 树，提取 Appearance 数据。
// 每个带样式信息的 widget 实例生成一个 WidgetInstance 节点。
type WidgetInstanceAdapter struct {
	Model *modelsdk.Model
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

		typeName := elem.TypeName()
		var containerLabel mxgraph.Label
		switch {
		case strings.HasSuffix(typeName, "$Page") || strings.HasSuffix(typeName, "$Form"):
			containerLabel = "Page"
		case strings.HasSuffix(typeName, "$Snippet"):
			containerLabel = "Snippet"
		default:
			continue
		}

		module := a.Model.ResolveModuleName(unit.ID)
		containerID := mxgraph.NodeID(elem.ID())

		wiEvents := a.walkWidgets(elem, containerID, containerLabel, module)
		events = append(events, wiEvents...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// walkWidgets 递归遍历 widget 树，提取 Appearance 数据。
// 优先使用 typed Properties() 遍历。
// 额外扫描 raw BSON 查找 typed 路径未暴露的 widget 容器 key
//（如 Forms$FormCallArgument 的 Widget/Widgets 字段）。
func (a *WidgetInstanceAdapter) walkWidgets(
	elem element.Element,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	var events []mxgraph.Event

	// 1. Typed 路径
	for _, prop := range elem.Properties() {
		switch p := prop.(type) {
		case element.ChildProperty:
			if child := p.ChildElement(); child != nil {
				evts := a.inspectWidget(child, containerID, containerLabel, module)
				events = append(events, evts...)
				events = append(events, a.walkWidgets(child, containerID, containerLabel, module)...)
			}
		case element.ChildListProperty:
			for _, child := range p.ChildElements() {
				if child == nil {
					continue
				}
				evts := a.inspectWidget(child, containerID, containerLabel, module)
				events = append(events, evts...)
				events = append(events, a.walkWidgets(child, containerID, containerLabel, module)...)
			}
		}
	}

	// 2. Raw BSON 回退：查找 typed 路径可能遗漏的 widget 容器 key
	//    （如 Forms$FormCallArgument 的 Widget/Widgets 字段在 typed 中不可见）
	if len(elem.Raw()) > 0 {
		rawEvents := a.walkRawBSON(elem.Raw(), containerID, containerLabel, module)
		events = append(events, rawEvents...)
	}

	return events
}

// walkRawBSON 从 raw BSON 解码子元素，处理 FormCall/LayoutCall/Arguments 等未注册类型。
func (a *WidgetInstanceAdapter) walkRawBSON(
	raw bson.Raw,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	var events []mxgraph.Event

	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil
	}

	// 遍历已知的 widget 容器 key
	for _, rawKey := range knownContainerKeys(doc) {
		val := doc[rawKey]
		_ = val

		// bson.D 是 mongo driver v2 解码 sub-document 时的类型
		if d, ok := val.(bson.D); ok {
			if len(d) == 0 {
				continue
			}
			a.processRawSubDoc(dToMap(d), containerID, containerLabel, module, &events)
		}

		// bson.A 是数组
		if arr, ok := val.(bson.A); ok {
			items := dropVersionPrefix(arr)
			for _, item := range items {
				switch v := item.(type) {
				case bson.D:
					a.processRawSubDoc(dToMap(v), containerID, containerLabel, module, &events)
				case bson.M:
					a.processRawSubDoc(v, containerID, containerLabel, module, &events)
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

// processRawSubDoc 处理单个 raw BSON sub-document。
// 如果是已知 widget 类型 → typed decode → inspectWidget + 递归 walkWidgets。
// 如果是未知容器类型（如 FormCall）→ 直接递归 walkRawBSON。
func (a *WidgetInstanceAdapter) processRawSubDoc(
	m map[string]any,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
	events *[]mxgraph.Event,
) {
	typeName, _ := m["$Type"].(string)
	if typeName == "" {
		return
	}
	if isWidgetType(typeName) {
		// 已知 widget 类型，通过 codec 解码获取类型化 Properties
		if child := a.decodeRawChild(m); child != nil {
			evts := a.inspectWidget(child, containerID, containerLabel, module)
			*events = append(*events, evts...)
			*events = append(*events, a.walkWidgets(child, containerID, containerLabel, module)...)
		}
	} else {
		// 未知容器类型（FormCall 等），将 raw BSON 重新编码后递归
		raw, err := bson.Marshal(m)
		if err != nil {
			return
		}
		evts := a.walkRawBSON(raw, containerID, containerLabel, module)
		*events = append(*events, evts...)
	}
}

// decodeRawChild 通过 codec 解码 raw BSON map 为 typed element。
// 仅用于 isWidgetType 返回 true 的类型。
func (a *WidgetInstanceAdapter) decodeRawChild(m map[string]any) element.Element {
	typeName, _ := m["$Type"].(string)
	if typeName == "" {
		return nil
	}
	// 跳过非 widget 类型
	if !isWidgetType(typeName) {
		return nil
	}
	// 用 codec 解码以获得类型化 Properties（包括 Appearance）
	raw, err := bson.Marshal(m)
	if err != nil {
		return nil
	}
	decoder := codec.NewDecoder(codec.DefaultRegistry)
	child, err := decoder.Decode(raw)
	if err != nil {
		return nil
	}
	return child
}

// dToMap 将 bson.D 转为 map[string]any（ToMap 在 v2 中不存在）。
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

// inspectWidget 检查单个元素，如果是 widget 且有 Appearance 则创建 WidgetInstance 节点。
func (a *WidgetInstanceAdapter) inspectWidget(
	widget element.Element,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	typeName := widget.TypeName()
	if !isWidgetType(typeName) {
		return nil
	}

	widgetName := getWidgetNameFromElement(widget)
	if widgetName == "" {
		return nil
	}

	class, style, dps := extractAppearanceData(widget)
	if class == "" && style == "" && len(dps) == 0 {
		return nil
	}

	widgetID := mxgraph.NodeID(widget.ID())
	qn := fmt.Sprintf("%s.%s", module, widgetName)
	if module == "" {
		qn = widgetName
	}

	props := map[string]any{
		"$Type":            "WidgetInstance",
		"Name":             widgetName,
		"WidgetType":       shortTypeName(typeName),
		"Class":            class,
		"Style":            style,
		"DesignProperties": dps,
		"ElementID":        string(widget.ID()),
		"QualifiedName":    qn,
		"Module":           module,
	}

	var events []mxgraph.Event
	events = append(events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{ID: widgetID, Label: "WidgetInstance", Props: props},
	})
	events = append(events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_WIDGET_INSTANCE-->%s", containerID, widgetID)),
			From: containerID,
			To:   widgetID,
			Type: "HAS_WIDGET_INSTANCE",
		},
	})

	return events
}

// extractAppearanceData 从 widget 元素中提取 Class, Style, DesignProperties。
func extractAppearanceData(widget element.Element) (class, style string, designProps map[string]string) {
	for _, prop := range widget.Properties() {
		if prop.Name() != "Appearance" {
			continue
		}
		cp, ok := prop.(element.ChildProperty)
		if !ok {
			continue
		}
		app := cp.ChildElement()
		if app == nil {
			return
		}

		// 提取标量属性
		for _, ap := range app.Properties() {
			wp, ok := ap.(element.WritableProperty)
			if !ok {
				continue
			}
			v := wp.BSONValue()
			if v == nil {
				continue
			}
			switch ap.Name() {
			case "Class":
				class, _ = v.(string)
			case "Style":
				style, _ = v.(string)
			}
		}

		// 提取 DesignProperties
		for _, ap := range app.Properties() {
			if ap.Name() != "DesignProperties" {
				continue
			}
			cl, ok := ap.(element.ChildListProperty)
			if !ok {
				continue
			}
			designProps = extractDesignPropsFromList(cl)
		}
		return
	}
	return
}

// extractDesignPropsFromList 从 DesignProperties PartList 中提取 key→value map。
func extractDesignPropsFromList(cl element.ChildListProperty) map[string]string {
	result := make(map[string]string)
	for _, dpv := range cl.ChildElements() {
		if dpv == nil {
			continue
		}

		var key string
		var valueMap map[string]any

		// 读取 Key
		for _, dpProp := range dpv.Properties() {
			if dpProp.Name() == "Key" {
				if wp, ok := dpProp.(element.WritableProperty); ok {
					if v := wp.BSONValue(); v != nil {
						key, _ = v.(string)
					}
				}
			}
			// 读取嵌套 Value 子元素
			if dpProp.Name() == "Value" {
				cp, ok := dpProp.(element.ChildProperty)
				if !ok {
					continue
				}
				val := cp.ChildElement()
				if val == nil {
					continue
				}
				valueMap = make(map[string]any)
				for _, vp := range val.Properties() {
					if wp, ok := vp.(element.WritableProperty); ok {
						if bv := wp.BSONValue(); bv != nil {
							valueMap[vp.Name()] = bv
						}
					}
				}
			}
		}

		if key == "" {
			continue
		}

		innerType, _ := valueMap["$Type"].(string)
		innerType = normalizeTypeName(innerType)

		switch {
		case strings.Contains(innerType, "ToggleDesignPropertyValue"):
			result[key] = "ON"
		case strings.Contains(innerType, "OptionDesignPropertyValue"):
			if opt, ok := valueMap["Option"].(string); ok {
				result[key] = opt
			}
		default:
			// Fallback: use "ON" for any unrecognized inner type
			result[key] = "ON"
		}
	}
	return result
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

func getWidgetNameFromElement(elem element.Element) string {
	for _, p := range elem.Properties() {
		if p.Name() == "Name" {
			if wp, ok := p.(element.WritableProperty); ok {
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

func (a *WidgetInstanceAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
