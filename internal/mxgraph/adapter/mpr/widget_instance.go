package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
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
func (a *WidgetInstanceAdapter) walkWidgets(
	elem element.Element,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
) []mxgraph.Event {
	var events []mxgraph.Event

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

	return events
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
		"Forms$LayoutCall":           true,
		"Forms$FormCall":             true,
		"Forms$Appearance":           true,
		"Forms$DesignPropertyValue":  true,
		"Forms$OptionDesignPropertyValue":  true,
		"Forms$ToggleDesignPropertyValue": true,
		"Forms$CustomDesignPropertyValue":  true,
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
