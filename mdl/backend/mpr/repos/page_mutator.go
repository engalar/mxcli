// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"
	"reflect"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/mendixlabs/mxcli/modelsdk/property"
)

// pageMutator is the Stage 2 / 2.5 decode-edit-encode mutator (Option A,
// addendum Blocker 3).
//
// Stage 2.5 finishes the four widget-tree edits (Insert / Delete / Replace
// / SetLayout) by walking the cached *Page in memory via the generic
// PartList[element.Element] / Part[element.Element] property surface and
// re-encoding on Commit. We deliberately keep the same Option A discipline
// SetWidgetProperty already uses; a true Option B raw-BSON tree walker is
// a separate optimization deferred to a later stage.
type pageMutator struct {
	repo *pageRepo
	page *genPg.Page
}

// SetWidgetProperty mutates a property on the widget identified by
// widgetID. Stage 2 routes through reflection: on the located element
// we look for a method named "Set<Prop>" with one argument and call
// it with `value`. Setters that take element.Element / nested gen
// types are not yet supported (returns explicit error).
func (m *pageMutator) SetWidgetProperty(widgetID model.ID, prop string, value any) error {
	target := findElementByID(m.page, widgetID)
	if target == nil {
		return fmt.Errorf("SetWidgetProperty: widget %s not found in page %s", widgetID, m.page.ID())
	}
	rv := reflect.ValueOf(target)
	method := rv.MethodByName("Set" + prop)
	if !method.IsValid() {
		return fmt.Errorf("SetWidgetProperty: %T has no setter for %q", target, prop)
	}
	mt := method.Type()
	if mt.NumIn() != 1 {
		return fmt.Errorf("SetWidgetProperty: setter Set%s on %T expects %d args, want 1", prop, target, mt.NumIn())
	}
	argType := mt.In(0)
	if value == nil {
		method.Call([]reflect.Value{reflect.Zero(argType)})
		return nil
	}
	val := reflect.ValueOf(value)
	if !val.Type().AssignableTo(argType) {
		if val.Type().ConvertibleTo(argType) {
			val = val.Convert(argType)
		} else {
			return fmt.Errorf("SetWidgetProperty: %T.Set%s expects %s, got %s",
				target, prop, argType, val.Type())
		}
	}
	method.Call([]reflect.Value{val})
	return nil
}

// InsertWidget appends `widget` to the PartList named `slot` on the
// element identified by parentID. Examples of valid slot names:
// "Widgets" (LayoutCallArgument), "Arguments" (LayoutCall),
// container-shaped widget slots like "Items" / "Cells".
func (m *pageMutator) InsertWidget(parentID model.ID, slot string, widget element.Element) error {
	if widget == nil {
		return fmt.Errorf("InsertWidget: widget is nil (parent=%s slot=%s)", parentID, slot)
	}
	parent := findElementByID(m.page, parentID)
	if parent == nil {
		return fmt.Errorf("InsertWidget: parent %s not found in page %s", parentID, m.page.ID())
	}
	plist, ok := findPartListBySlot(parent, slot)
	if !ok {
		return fmt.Errorf("InsertWidget: %T has no PartList slot %q", parent, slot)
	}
	plist.Append(widget)
	return nil
}

// DeleteWidget removes the widget identified by widgetID from whichever
// PartList currently contains it.
func (m *pageMutator) DeleteWidget(widgetID model.ID) error {
	parent, plist, idx := findContainingPartList(m.page, widgetID)
	if parent == nil || plist == nil {
		return fmt.Errorf("DeleteWidget: widget %s not found in any PartList of page %s", widgetID, m.page.ID())
	}
	plist.Remove(idx)
	return nil
}

// ReplaceWidget swaps the widget identified by widgetID with `replacement`,
// preserving its index in the parent PartList.
func (m *pageMutator) ReplaceWidget(widgetID model.ID, replacement element.Element) error {
	if replacement == nil {
		return fmt.Errorf("ReplaceWidget: replacement is nil (widget=%s)", widgetID)
	}
	parent, plist, idx := findContainingPartList(m.page, widgetID)
	if parent == nil || plist == nil {
		return fmt.Errorf("ReplaceWidget: widget %s not found in any PartList of page %s", widgetID, m.page.ID())
	}
	plist.Remove(idx)
	plist.InsertAt(idx, replacement)
	return nil
}

// SetLayout points the page's LayoutCall at the layout identified by
// layoutQN. If the page has no LayoutCall yet (fresh page), one is
// constructed on the fly. Layout existence is verified through
// QualifiedNameResolver (Stage 2.5 task 3); the resolved kind must
// be "layout" — pointing at a microflow/page/etc surfaces an
// explicit wrong-kind error.
func (m *pageMutator) SetLayout(layoutQN string) error {
	if err := m.verifyLayoutExists(layoutQN); err != nil {
		return err
	}
	lcAny := m.page.LayoutCall()
	if lcAny == nil {
		lc := genPg.NewLayoutCall()
		m.page.SetLayoutCall(lc)
		lc.SetLayoutQualifiedName(layoutQN)
		return nil
	}
	lc, ok := lcAny.(*genPg.LayoutCall)
	if !ok {
		return fmt.Errorf("SetLayout: page LayoutCall is unexpected type %T", lcAny)
	}
	lc.SetLayoutQualifiedName(layoutQN)
	return nil
}

// verifyLayoutExists delegates to QualifiedNameResolver. The resolver
// already returns "invalid qualified name" / "not found" with the
// layoutQN substring; we wrap the error with a SetLayout: prefix so
// callers don't have to special-case those classes.
func (m *pageMutator) verifyLayoutExists(layoutQN string) error {
	res := NewQualifiedNameResolver(m.repo.w)
	_, kind, err := res.ResolveQualifiedName(layoutQN)
	if err != nil {
		return fmt.Errorf("SetLayout: %w", err)
	}
	if kind != "layout" {
		return fmt.Errorf("SetLayout: %q is a %s, not a layout", layoutQN, kind)
	}
	return nil
}

// Commit persists the mutated page via pageRepo.Update — the canonical
// decode-edit-encode round-trip from Option A.
func (m *pageMutator) Commit() error { return m.repo.Update(m.page) }

var _ repos.PageMutator = (*pageMutator)(nil)

// findElementByID walks the page recursively and returns the first
// element whose ID matches.
func findElementByID(page *genPg.Page, want model.ID) element.Element {
	if string(page.ID()) == string(want) {
		return page
	}
	if lc := page.LayoutCall(); lc != nil {
		if found := walkElement(lc, want); found != nil {
			return found
		}
	}
	return nil
}

// walkElement performs a generic depth-first probe using reflection on
// any *Items() method that returns []element.Element. This avoids
// hard-coding every widget container shape (~50 widget types) and is
// sufficient for Stage 2 tests; Stage 2.5 may replace with explicit
// per-type walkers if reflection proves a hotspot.
func walkElement(elem element.Element, want model.ID) element.Element {
	if elem == nil {
		return nil
	}
	if string(elem.ID()) == string(want) {
		return elem
	}
	rv := reflect.ValueOf(elem)
	for i := 0; i < rv.NumMethod(); i++ {
		mt := rv.Type().Method(i)
		if mt.Type.NumIn() != 1 || mt.Type.NumOut() != 1 {
			continue
		}
		out := mt.Type.Out(0)
		if out == elementSliceType {
			items := rv.Method(i).Call(nil)[0].Interface().([]element.Element)
			for _, child := range items {
				if found := walkElement(child, want); found != nil {
					return found
				}
			}
			continue
		}
		if out == elementInterfaceType {
			child, _ := rv.Method(i).Call(nil)[0].Interface().(element.Element)
			if child == nil {
				continue
			}
			if found := walkElement(child, want); found != nil {
				return found
			}
		}
	}
	return nil
}

// findPartListBySlot looks at the element's exported Properties() and
// returns the first *property.PartList[element.Element] whose Name()
// matches `slot` (case-sensitive — slot names follow the BSON storage
// convention, e.g. "Widgets", "Arguments").
func findPartListBySlot(elem element.Element, slot string) (*property.PartList[element.Element], bool) {
	for _, p := range elem.Properties() {
		pl, ok := p.(*property.PartList[element.Element])
		if !ok {
			continue
		}
		if pl.Name() == slot {
			return pl, true
		}
	}
	return nil, false
}

// findContainingPartList walks the page and returns (parentElement,
// partList, indexInList) for the PartList that currently contains
// childID. Returns (nil, nil, -1) if not found.
func findContainingPartList(page *genPg.Page, childID model.ID) (element.Element, *property.PartList[element.Element], int) {
	// The Page itself can never be inside its own PartList; start the
	// traversal from its child slots.
	return walkForContainingPartList(page, childID)
}

// walkForContainingPartList visits each Property on `cur`. For PartList
// children: scans every item — if found, returns this parent + the
// PartList + the index. Otherwise descends into each item. For Part
// children: descends. The walk is depth-first.
func walkForContainingPartList(cur element.Element, want model.ID) (element.Element, *property.PartList[element.Element], int) {
	for _, p := range cur.Properties() {
		switch pp := p.(type) {
		case *property.PartList[element.Element]:
			items := pp.Items()
			for i, child := range items {
				if child != nil && string(child.ID()) == string(want) {
					return cur, pp, i
				}
			}
			for _, child := range items {
				if child == nil {
					continue
				}
				if parent, plist, idx := walkForContainingPartList(child, want); plist != nil {
					return parent, plist, idx
				}
			}
		case *property.Part[element.Element]:
			child := pp.Get()
			if child == nil {
				continue
			}
			if parent, plist, idx := walkForContainingPartList(child, want); plist != nil {
				return parent, plist, idx
			}
		}
	}
	return nil, nil, -1
}

var (
	elementInterfaceType = reflect.TypeOf((*element.Element)(nil)).Elem()
	elementSliceType     = reflect.TypeOf([]element.Element(nil))
)
