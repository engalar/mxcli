// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.D1 — GenCustomWidgetElem bridges the transition from sdk-typed
// *pages.CustomWidget to gen-typed element.Element in the pluggable widget path.
//
// During the D1 window cmd_pages_builder_v3.go still builds sdk-typed widget
// trees (backend.Widget), so the pluggable engine must return a value that
// satisfies backend.Widget while internally carrying a gen element.  Once
// Cat-B migration replaces the V3 builder, this adapter is deleted and
// PluggableWidgetEngine.Build returns element.Element directly.

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// GenCustomWidgetElem wraps a gen-decoded custom widget element so it satisfies
// the backend.Widget interface (= sdk/pages.Widget) during the D1 transition.
type GenCustomWidgetElem struct {
	elem element.Element
}

// NewGenCustomWidgetElem wraps e as a backend.Widget-compatible value.
func NewGenCustomWidgetElem(e element.Element) *GenCustomWidgetElem {
	return &GenCustomWidgetElem{elem: e}
}

// AsElement returns the underlying gen element for codec-based serialization.
func (g *GenCustomWidgetElem) AsElement() element.Element { return g.elem }

// GetID implements backend.Widget (= pages.Widget).
func (g *GenCustomWidgetElem) GetID() model.ID { return model.ID(g.elem.ID()) }

// GetTypeName implements backend.Widget (= pages.Widget).
func (g *GenCustomWidgetElem) GetTypeName() string { return g.elem.TypeName() }

// GetName implements backend.Widget (= pages.Widget).
func (g *GenCustomWidgetElem) GetName() string {
	if named, ok := g.elem.(interface{ Name() string }); ok {
		return named.Name()
	}
	return ""
}
