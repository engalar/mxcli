// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"log"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// widgetBuilderFn builds a gen widget element from a WidgetV3 AST node.
// All type strings are lowercase (matched via strings.ToLower in buildWidgetV3).
type widgetBuilderFn func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error)

// widgetBuilders maps each built-in widget type (lowercase) to its builder.
// Pluggable/unknown types fall through to the pluggable widget engine in
// buildWidgetV3's fallback path — they are NOT listed here.
//
// "legacydatagrid", "tabpage", and "item" are intentionally absent:
// they return validation errors rather than elements.
//
// Populated in init() rather than as a package-level literal: some builders
// (e.g. buildLayoutGridV3) recurse into buildWidgetV3, which reads this map,
// forming an initialization cycle the compiler rejects for direct var init.
var widgetBuilders map[string]widgetBuilderFn

func init() {
	widgetBuilders = map[string]widgetBuilderFn{
		"dataview":   func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDataViewV3(w) },
		"datagrid":   func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDataGridV3(w) },
		"listview":   func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildListViewV3(w) },
		"layoutgrid": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildLayoutGridV3(w) },
		"row":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerWithRowV3(w) },
		"column": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) {
			return pb.buildContainerWithColumnV3(w)
		},
		"container":       func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerV3(w) },
		"customcontainer": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerV3(w) },
		"textbox":         func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextBoxV3(w) },
		"textarea":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextAreaV3(w) },
		"datepicker":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDatePickerV3(w) },
		"dropdown":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDropdownV3(w) },
		"checkbox":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildCheckBoxV3(w) },
		"fileinput":       func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFileManagerV3(w) },
		"text":            func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextWidgetV3(w) },
		"statictext":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextWidgetV3(w) },
		"dynamictext":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDynamicTextV3(w) },
		"title":           func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTitleV3(w) },
		"button":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildButtonV3(w) },
		"actionbutton":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildButtonV3(w) },
		"tabcontainer":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTabContainerV3(w) },
		"groupbox":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildGroupBoxV3(w) },
		"radiobuttons":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildRadioButtonsV3(w) },
		"navigationlist":  func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildNavigationListV3(w) },
		"snippetcall":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildSnippetCallV3(w) },
		"footer":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFooterV3(w) },
		"header":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildHeaderV3(w) },
		"controlbar":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildControlBarV3(w) },
		"template":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTemplateV3(w) },
		"filter":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFilterV3(w) },
		"staticimage":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildStaticImageV3(w) },
		"dynamicimage":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDynamicImageV3(w) },
		// image: tries pluggable widget first, falls back to static image
		"image": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) {
			pb.initPluggableEngine()
			if pb.widgetRegistry != nil {
				if def, ok := pb.widgetRegistry.Get("image"); ok {
					cw, err := pb.pluggableEngine.Build(def, w)
					if err != nil {
						return nil, err
					}
					return pb.customWidgetToElement(cw)
				}
			}
			log.Printf("warning: pluggable image widget not found, using static image fallback")
			return pb.buildStaticImageV3(w)
		},
	}
}
