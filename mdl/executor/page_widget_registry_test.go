// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"testing"
)

// TestWidgetBuilderRegistryCoverage 确认所有内置 widget 类型都有注册 handler。
// legacydatagrid/tabpage/item 是错误 case，不在此列表。
// pluggable widget（unknown 类型）由 widgetRegistry fallback 处理，不在此列表。
func TestWidgetBuilderRegistryCoverage(t *testing.T) {
	knownTypes := []string{
		"dataview", "datagrid", "listview", "layoutgrid",
		"row", "column", "container", "customcontainer",
		"textbox", "textarea", "datepicker", "dropdown",
		"checkbox", "fileinput", "text", "statictext",
		"dynamictext", "title", "button", "actionbutton",
		"tabcontainer", "groupbox", "radiobuttons",
		"navigationlist", "snippetcall",
		"footer", "header", "controlbar",
		"template", "filter", "staticimage", "dynamicimage", "image",
	}
	for _, typ := range knownTypes {
		if _, ok := widgetBuilders[typ]; !ok {
			t.Errorf("widget type %q has no handler in widgetBuilders — add it to page_widget_registry.go", typ)
		}
	}
}
