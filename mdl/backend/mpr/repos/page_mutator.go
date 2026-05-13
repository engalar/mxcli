// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"
	"reflect"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// pageMutator is the Stage 2 decode-edit-encode mutator (Option A,
// addendum Blocker 3).
//
// Stage 2 scope: Commit (encode + persist via pageRepo.Update) is fully
// wired and exercised by tests. SetWidgetProperty is implemented via
// reflection-dispatched Setter calls on the gen Element. The widget
// tree-walk helpers (find / insert / delete / replace) ship as TODO
// stubs returning explicit errors; SetLayout is also stubbed. Stage
// 2.5 may flesh these out when fixture-with-pages is available and
// throughput targets are measured.
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

// InsertWidget / DeleteWidget / ReplaceWidget / SetLayout are Stage 2.5
// follow-ups: they require widget-tree walkers that synthesise the
// correct AddXxx / RemoveXxx / SetXxx call on the parent. Stage 2 ships
// the explicit error so the executor's Stage 3 cutover plan can pick
// these up rather than hit a silent no-op.

func (m *pageMutator) InsertWidget(parentID model.ID, slot string, widget element.Element) error {
	return fmt.Errorf("pageMutator.InsertWidget: Stage 2.5 follow-up (parent=%s slot=%s)", parentID, slot)
}

func (m *pageMutator) DeleteWidget(widgetID model.ID) error {
	return fmt.Errorf("pageMutator.DeleteWidget: Stage 2.5 follow-up (widget=%s)", widgetID)
}

func (m *pageMutator) ReplaceWidget(widgetID model.ID, replacement element.Element) error {
	return fmt.Errorf("pageMutator.ReplaceWidget: Stage 2.5 follow-up (widget=%s)", widgetID)
}

func (m *pageMutator) SetLayout(layoutQN string) error {
	return fmt.Errorf("pageMutator.SetLayout: Stage 2.5 follow-up (layoutQN=%s)", layoutQN)
}

// Commit persists the mutated page via pageRepo.Update — the canonical
// decode-edit-encode round-trip from Option A.
func (m *pageMutator) Commit() error { return m.repo.Update(m.page) }

var _ repos.PageMutator = (*pageMutator)(nil)

// findElementByID walks the page recursively (LayoutCall.ArgumentsItems
// → LayoutCallArgument.Widget(s)Items, plus generic getter probing) and
// returns the first element whose ID matches.
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

var (
	elementInterfaceType = reflect.TypeOf((*element.Element)(nil)).Elem()
	elementSliceType     = reflect.TypeOf([]element.Element(nil))
)
