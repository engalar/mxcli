// SPDX-License-Identifier: Apache-2.0

// Package executor - Gen-typed widget tree walking for page introspection.
package executor

import (
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// walkPageWidgetsGen walks all widgets in a gen-typed Page via the visitor function.
// The visitor receives each widget as element.Element; use type assertions for concrete types.
func walkPageWidgetsGen(page *genPg.Page, visitor func(widget element.Element) error) error {
	if page == nil {
		return nil
	}
	lcElem := page.LayoutCall()
	if lcElem == nil {
		return nil
	}
	lc, ok := lcElem.(*genPg.LayoutCall)
	if !ok {
		return nil
	}
	for _, argElem := range lc.ArgumentsItems() {
		arg, ok := argElem.(*genPg.LayoutCallArgument)
		if !ok {
			continue
		}
		if w := arg.Widget(); w != nil {
			if err := walkWidgetGen(w, visitor); err != nil {
				return err
			}
		}
		for _, w := range arg.WidgetsItems() {
			if err := walkWidgetGen(w, visitor); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkSnippetWidgetsGen walks all widgets in a gen-typed Snippet via the visitor function.
func walkSnippetWidgetsGen(snippet *genPg.Snippet, visitor func(widget element.Element) error) error {
	if snippet == nil {
		return nil
	}
	for _, w := range snippet.WidgetsItems() {
		if err := walkWidgetGen(w, visitor); err != nil {
			return err
		}
	}
	return nil
}

// walkWidgetGen recursively walks a gen-typed widget element and its children.
func walkWidgetGen(widget element.Element, visitor func(widget element.Element) error) error {
	if widget == nil {
		return nil
	}
	if err := visitor(widget); err != nil {
		return err
	}
	switch w := widget.(type) {
	case *genPg.LayoutGrid:
		for _, rowElem := range w.RowsItems() {
			row, ok := rowElem.(*genPg.LayoutGridRow)
			if !ok {
				continue
			}
			for _, colElem := range row.ColumnsItems() {
				col, ok := colElem.(*genPg.LayoutGridColumn)
				if !ok {
					continue
				}
				for _, child := range col.WidgetsItems() {
					if err := walkWidgetGen(child, visitor); err != nil {
						return err
					}
				}
			}
		}
	case *genPg.DataView:
		for _, child := range w.WidgetsItems() {
			if err := walkWidgetGen(child, visitor); err != nil {
				return err
			}
		}
		for _, child := range w.FooterWidgetsItems() {
			if err := walkWidgetGen(child, visitor); err != nil {
				return err
			}
		}
	case *genPg.ListView:
		for _, child := range w.WidgetsItems() {
			if err := walkWidgetGen(child, visitor); err != nil {
				return err
			}
		}
	case *genPg.DivContainer:
		for _, child := range w.WidgetsItems() {
			if err := walkWidgetGen(child, visitor); err != nil {
				return err
			}
		}
	case *genPg.GroupBox:
		for _, child := range w.WidgetsItems() {
			if err := walkWidgetGen(child, visitor); err != nil {
				return err
			}
		}
	case *genPg.TabContainer:
		for _, tabElem := range w.TabPagesItems() {
			tab, ok := tabElem.(*genPg.TabPage)
			if !ok {
				continue
			}
			for _, child := range tab.WidgetsItems() {
				if err := walkWidgetGen(child, visitor); err != nil {
					return err
				}
			}
		}
	case *genPg.ScrollContainer:
		// ScrollContainer uses named regions (Center, Left, Right, Top, Bottom).
		for _, regionElem := range []element.Element{w.Center(), w.Left(), w.Right(), w.Top(), w.Bottom()} {
			if regionElem == nil {
				continue
			}
			region, ok := regionElem.(*genPg.ScrollContainerRegion)
			if !ok {
				continue
			}
			for _, child := range region.WidgetsItems() {
				if err := walkWidgetGen(child, visitor); err != nil {
					return err
				}
			}
		}
	}
	// gen/pages has no CustomWidget type (codegen gap).
	// Pluggable widget child traversal deferred to Phase 4.3 E3.
	return nil
}
