// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// ============================================================================
// Fragment Expansion (V3 builder)
// ============================================================================

// expandFragments processes a widget list, expanding any USE_FRAGMENT sentinels
// into their referenced fragment widgets. Non-fragment widgets pass through unchanged.
func (pb *pageBuilder) expandFragments(widgets []*ast.WidgetV3) ([]*ast.WidgetV3, error) {
	var result []*ast.WidgetV3
	for _, w := range widgets {
		expanded, err := pb.expandIfFragment(w)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}
	return result, nil
}

// expandIfFragment returns the widget as-is if it's not a USE_FRAGMENT sentinel,
// or expands it into cloned fragment widgets with optional prefix.
func (pb *pageBuilder) expandIfFragment(w *ast.WidgetV3) ([]*ast.WidgetV3, error) {
	if w.Type != "USE_FRAGMENT" {
		return []*ast.WidgetV3{w}, nil
	}

	if pb.fragments == nil {
		return nil, mdlerrors.NewNotFound("fragment", w.Name)
	}
	frag, ok := pb.fragments[w.Name]
	if !ok {
		return nil, mdlerrors.NewNotFound("fragment", w.Name)
	}

	widgets := cloneWidgets(frag.Widgets)
	if prefix, ok := w.Properties["Prefix"].(string); ok && prefix != "" {
		prefixWidgetNames(widgets, prefix)
	}
	return widgets, nil
}

// cloneWidgets deep-copies a widget tree to avoid mutating the fragment definition.
func cloneWidgets(widgets []*ast.WidgetV3) []*ast.WidgetV3 {
	if widgets == nil {
		return nil
	}
	result := make([]*ast.WidgetV3, len(widgets))
	for i, w := range widgets {
		result[i] = cloneWidget(w)
	}
	return result
}

func cloneWidget(w *ast.WidgetV3) *ast.WidgetV3 {
	clone := &ast.WidgetV3{
		Type:       w.Type,
		Name:       w.Name,
		Properties: make(map[string]interface{}, len(w.Properties)),
		Children:   cloneWidgets(w.Children),
	}
	for k, v := range w.Properties {
		clone.Properties[k] = v
	}
	return clone
}

// prefixWidgetNames recursively prepends a prefix to all widget names.
func prefixWidgetNames(widgets []*ast.WidgetV3, prefix string) {
	for _, w := range widgets {
		if w.Name != "" {
			w.Name = prefix + w.Name
		}
		prefixWidgetNames(w.Children, prefix)
	}
}
