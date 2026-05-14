// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// newAction builds a gen ActionActivity with the given ID and inner
// action. element IDs are stamped via SetID so the deeply-nested
// "deep" assertion below stays meaningful.
func newAction(id string, action element.Element) *genMf.ActionActivity {
	a := genMf.NewActionActivity()
	a.SetID(element.ID(id))
	if action != nil {
		a.SetAction(action)
	}
	return a
}

// newLoop builds a gen LoopedActivity owning the supplied children.
func newLoop(id string, children ...element.Element) *genMf.LoopedActivity {
	loop := genMf.NewLoopedActivity()
	loop.SetID(element.ID(id))
	body := genMf.NewMicroflowObjectCollection()
	for _, child := range children {
		body.AddObjects(child)
	}
	loop.SetObjectCollection(body)
	return loop
}

func TestCollectActionActivities_TopLevelOnly(t *testing.T) {
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(newAction("a1", genMf.NewMicroflowCallAction()))
	oc.AddObjects(newAction("a2", genMf.NewCreateObjectAction()))

	result := collectActionActivities(oc)
	if len(result) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(result))
	}
}

func TestCollectActionActivities_InsideLoop(t *testing.T) {
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(newLoop("loop1",
		newAction("inner1", genMf.NewMicroflowCallAction()),
		newAction("inner2", genMf.NewShowPageAction()),
	))
	oc.AddObjects(newAction("outer1", genMf.NewRetrieveAction()))

	result := collectActionActivities(oc)
	if len(result) != 3 {
		t.Fatalf("expected 3 activities (2 inside loop + 1 outside), got %d", len(result))
	}
}

func TestCollectActionActivities_NestedLoops(t *testing.T) {
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(newLoop("outer-loop",
		newLoop("inner-loop",
			newAction("deep", genMf.NewMicroflowCallAction()),
		),
	))

	result := collectActionActivities(oc)
	if len(result) != 1 {
		t.Fatalf("expected 1 deeply nested activity, got %d", len(result))
	}
	if got := string(result[0].ID()); got != "deep" {
		t.Errorf("expected activity ID 'deep', got %q", got)
	}
}

func TestCollectActionActivities_NilCollection(t *testing.T) {
	result := collectActionActivities(nil)
	if result != nil {
		t.Fatalf("expected nil for nil collection, got %v", result)
	}
}

func TestCollectActionActivities_SkipsNilActions(t *testing.T) {
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(newAction("no-action", nil))
	oc.AddObjects(newAction("has-action", genMf.NewMicroflowCallAction()))

	result := collectActionActivities(oc)
	if len(result) != 1 {
		t.Fatalf("expected 1 activity (skipping nil action), got %d", len(result))
	}
}
