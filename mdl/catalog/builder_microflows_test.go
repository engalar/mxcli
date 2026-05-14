// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// unknownObject satisfies element.Element but is not in any of the
// type switches below — used to verify the default branch.
type unknownObject struct {
	element.Base
}

func TestGetMicroflowObjectType(t *testing.T) {
	tests := []struct {
		name string
		obj  element.Element
		want string
	}{
		{"ActionActivity", genMf.NewActionActivity(), "ActionActivity"},
		{"StartEvent", genMf.NewStartEvent(), "StartEvent"},
		{"EndEvent", genMf.NewEndEvent(), "EndEvent"},
		{"ExclusiveSplit", genMf.NewExclusiveSplit(), "ExclusiveSplit"},
		{"InheritanceSplit", genMf.NewInheritanceSplit(), "InheritanceSplit"},
		{"ExclusiveMerge", genMf.NewExclusiveMerge(), "ExclusiveMerge"},
		{"LoopedActivity", genMf.NewLoopedActivity(), "LoopedActivity"},
		{"Annotation", genMf.NewAnnotation(), "Annotation"},
		{"BreakEvent", genMf.NewBreakEvent(), "BreakEvent"},
		{"ContinueEvent", genMf.NewContinueEvent(), "ContinueEvent"},
		{"ErrorEvent", genMf.NewErrorEvent(), "ErrorEvent"},
		{"unknown object falls to default", &unknownObject{}, "MicroflowObject"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMicroflowObjectType(tt.obj); got != tt.want {
				t.Errorf("getMicroflowObjectType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetMicroflowActionType(t *testing.T) {
	// The legacy class names are the catalog contract, so a few gen
	// renames are intentionally exercised: CloseFormAction maps to
	// ClosePageAction; DeleteAction maps to DeleteObjectAction;
	// CommitAction maps to CommitObjectsAction; RollbackAction maps to
	// RollbackObjectAction.
	tests := []struct {
		name   string
		action element.Element
		want   string
	}{
		{"CreateObjectAction", genMf.NewCreateObjectAction(), "CreateObjectAction"},
		{"ChangeObjectAction", genMf.NewChangeObjectAction(), "ChangeObjectAction"},
		{"RetrieveAction", genMf.NewRetrieveAction(), "RetrieveAction"},
		{"MicroflowCallAction", genMf.NewMicroflowCallAction(), "MicroflowCallAction"},
		{"JavaActionCallAction", genMf.NewJavaActionCallAction(), "JavaActionCallAction"},
		{"ShowMessageAction", genMf.NewShowMessageAction(), "ShowMessageAction"},
		{"LogMessageAction", genMf.NewLogMessageAction(), "LogMessageAction"},
		{"ValidationFeedbackAction", genMf.NewValidationFeedbackAction(), "ValidationFeedbackAction"},
		{"ChangeVariableAction", genMf.NewChangeVariableAction(), "ChangeVariableAction"},
		{"CreateVariableAction", genMf.NewCreateVariableAction(), "CreateVariableAction"},
		{"AggregateListAction", genMf.NewAggregateListAction(), "AggregateListAction"},
		{"ListOperationAction", genMf.NewListOperationAction(), "ListOperationAction"},
		{"CastAction", genMf.NewCastAction(), "CastAction"},
		{"DownloadFileAction", genMf.NewDownloadFileAction(), "DownloadFileAction"},
		{"CloseFormAction → ClosePageAction", genMf.NewCloseFormAction(), "ClosePageAction"},
		{"ShowPageAction", genMf.NewShowPageAction(), "ShowPageAction"},
		{"CallExternalAction", genMf.NewCallExternalAction(), "CallExternalAction"},
		{"DeleteAction → DeleteObjectAction", genMf.NewDeleteAction(), "DeleteObjectAction"},
		{"CommitAction → CommitObjectsAction", genMf.NewCommitAction(), "CommitObjectsAction"},
		{"RollbackAction → RollbackObjectAction", genMf.NewRollbackAction(), "RollbackObjectAction"},
		{"unknown action falls to default", &unknownObject{}, "MicroflowAction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMicroflowActionType(tt.action); got != tt.want {
				t.Errorf("getMicroflowActionType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountFlowParameters(t *testing.T) {
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(genMf.NewMicroflowParameter())
	oc.AddObjects(genMf.NewMicroflowParameter())
	oc.AddObjects(genMf.NewActionActivity())
	oc.AddObjects(genMf.NewMicroflowParameterObject())
	if got, want := countFlowParameters(oc), 3; got != want {
		t.Errorf("countFlowParameters = %d, want %d", got, want)
	}
	if got := countFlowParameters(nil); got != 0 {
		t.Errorf("countFlowParameters(nil) = %d, want 0", got)
	}
}

func TestCountFlowActivities(t *testing.T) {
	tests := []struct {
		name string
		oc   *genMf.MicroflowObjectCollection
		want int
	}{
		{
			name: "nil collection",
			oc:   nil,
			want: 0,
		},
		{
			name: "empty collection",
			oc:   genMf.NewMicroflowObjectCollection(),
			want: 0,
		},
		{
			name: "excludes start/end/merge/parameter",
			oc: collectionWith(
				genMf.NewStartEvent(),
				genMf.NewActionActivity(),
				genMf.NewExclusiveSplit(),
				genMf.NewEndEvent(),
				genMf.NewExclusiveMerge(),
				genMf.NewMicroflowParameter(),
			),
			want: 2, // ActionActivity + ExclusiveSplit
		},
		{
			name: "counts loops and annotations",
			oc: collectionWith(
				genMf.NewLoopedActivity(),
				genMf.NewAnnotation(),
				genMf.NewErrorEvent(),
			),
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countFlowActivities(tt.oc); got != tt.want {
				t.Errorf("countFlowActivities() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCalculateFlowComplexity(t *testing.T) {
	tests := []struct {
		name string
		oc   *genMf.MicroflowObjectCollection
		want int
	}{
		{
			name: "nil collection — base complexity",
			oc:   nil,
			want: 1,
		},
		{
			name: "no decision points",
			oc: collectionWith(
				genMf.NewStartEvent(),
				genMf.NewActionActivity(),
				genMf.NewEndEvent(),
			),
			want: 1,
		},
		{
			name: "exclusive split adds 1",
			oc: collectionWith(
				genMf.NewExclusiveSplit(),
			),
			want: 2,
		},
		{
			name: "inheritance split adds 1",
			oc: collectionWith(
				genMf.NewInheritanceSplit(),
			),
			want: 2,
		},
		{
			name: "loop adds 1 plus nested decisions",
			oc: collectionWithLoop(
				genMf.NewExclusiveSplit(),
			),
			want: 3, // 1 base + 1 loop + 1 nested split
		},
		{
			name: "error event adds 1",
			oc: collectionWith(
				genMf.NewErrorEvent(),
			),
			want: 2,
		},
		{
			name: "complex flow",
			oc: collectionWith(
				genMf.NewExclusiveSplit(),
				genMf.NewExclusiveSplit(),
				genMf.NewInheritanceSplit(),
				genMf.NewLoopedActivity(),
				genMf.NewErrorEvent(),
			),
			want: 6, // 1 + 2 splits + 1 inheritance + 1 loop + 1 error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateFlowComplexity(tt.oc); got != tt.want {
				t.Errorf("calculateFlowComplexity() = %d, want %d", got, tt.want)
			}
		})
	}
}

// collectionWith builds a populated MicroflowObjectCollection from the
// supplied elements. Used by the gen-typed test cases above to keep
// the table-driven style readable.
func collectionWith(items ...element.Element) *genMf.MicroflowObjectCollection {
	oc := genMf.NewMicroflowObjectCollection()
	for _, it := range items {
		oc.AddObjects(it)
	}
	return oc
}

// collectionWithLoop wraps the given items inside a single LoopedActivity
// nested inside a fresh top-level collection. Mirrors the legacy
// "loop adds 1 plus nested decisions" complexity test case.
func collectionWithLoop(inner ...element.Element) *genMf.MicroflowObjectCollection {
	body := genMf.NewMicroflowObjectCollection()
	for _, it := range inner {
		body.AddObjects(it)
	}
	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(body)
	return collectionWith(loop)
}
