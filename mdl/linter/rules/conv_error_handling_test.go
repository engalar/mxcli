// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// activityWithEH builds an ActionActivity owning the given inner
// action with its ErrorHandlingType set. Mirrors the legacy test
// pattern of "activity{ErrorHandlingType: X, Action: Y}" — but in
// gen the property lives on the inner action, not the activity.
func activityWithEH(inner ehSetter, eh string) *genMf.ActionActivity {
	inner.SetErrorHandlingType(eh)
	a := genMf.NewActionActivity()
	a.SetAction(inner.(element.Element))
	return a
}

// ehSetter is the small interface every gen action with an
// ErrorHandlingType property satisfies — used by activityWithEH so
// the test cases stay readable.
type ehSetter interface {
	element.Element
	SetErrorHandlingType(v string)
}

func TestFindUnhandledCalls_RestCallNoCustom(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewRestCallAction(), genMf.ErrorHandlingTypeAbort),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "CONV013" {
		t.Errorf("expected CONV013, got %s", violations[0].RuleID)
	}
}

func TestFindUnhandledCalls_RestCallCustom(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewRestCallAction(), genMf.ErrorHandlingTypeCustom),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations with Custom handling, got %d", len(violations))
	}
}

func TestFindUnhandledCalls_CustomWithoutRollback(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewRestCallAction(), genMf.ErrorHandlingTypeCustomWithoutRollBack),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations with CustomWithoutRollBack, got %d", len(violations))
	}
}

func TestFindUnhandledCalls_JavaAction(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewJavaActionCallAction(), genMf.ErrorHandlingTypeAbort),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for Java action, got %d", len(violations))
	}
}

func TestFindUnhandledCalls_WebServiceCall(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewWebServiceCallAction(), genMf.ErrorHandlingTypeContinue),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for WS call, got %d", len(violations))
	}
	if violations[0].RuleID != "CONV013" {
		t.Errorf("expected CONV013, got %s", violations[0].RuleID)
	}
}

func TestFindUnhandledCalls_NonExternalAction(t *testing.T) {
	// CommitAction is the gen rename of legacy CommitObjectsAction.
	objects := []element.Element{
		activityWithEH(genMf.NewCommitAction(), genMf.ErrorHandlingTypeAbort),
	}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-external action, got %d", len(violations))
	}
}

func TestFindUnhandledCalls_InsideLoop(t *testing.T) {
	loopBody := genMf.NewMicroflowObjectCollection()
	loopBody.AddObjects(activityWithEH(genMf.NewRestCallAction(), genMf.ErrorHandlingTypeAbort))

	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(loopBody)

	objects := []element.Element{loop}

	var violations []linter.Violation
	r := NewErrorHandlingOnCallsRule()
	findUnhandledCalls(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation inside loop, got %d", len(violations))
	}
}

// --- CONV014 tests ---

func TestFindContinueErrorHandling_Activity(t *testing.T) {
	commit := genMf.NewCommitAction()
	commit.SetErrorHandlingType(genMf.ErrorHandlingTypeContinue)
	a := genMf.NewActionActivity()
	a.SetAction(commit)
	a.SetCaption("Do something")

	objects := []element.Element{a}

	var violations []linter.Violation
	r := NewNoContinueErrorHandlingRule()
	findContinueErrorHandling(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "CONV014" {
		t.Errorf("expected CONV014, got %s", violations[0].RuleID)
	}
}

func TestFindContinueErrorHandling_Loop(t *testing.T) {
	loop := genMf.NewLoopedActivity()
	loop.SetErrorHandlingType(genMf.ErrorHandlingTypeContinue)
	// Gen LoopedActivity has no Caption; set Documentation so the
	// violation message uses something descriptive.
	loop.SetDocumentation("Process items")

	objects := []element.Element{loop}

	var violations []linter.Violation
	r := NewNoContinueErrorHandlingRule()
	findContinueErrorHandling(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for loop, got %d", len(violations))
	}
}

func TestFindContinueErrorHandling_AbortIsOk(t *testing.T) {
	objects := []element.Element{
		activityWithEH(genMf.NewCommitAction(), genMf.ErrorHandlingTypeAbort),
	}

	var violations []linter.Violation
	r := NewNoContinueErrorHandlingRule()
	findContinueErrorHandling(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for Abort, got %d", len(violations))
	}
}

func TestErrorHandlingOnCallsRule_Metadata(t *testing.T) {
	r := NewErrorHandlingOnCallsRule()
	if r.ID() != "CONV013" {
		t.Errorf("ID = %q, want CONV013", r.ID())
	}
}

func TestNoContinueErrorHandlingRule_Metadata(t *testing.T) {
	r := NewNoContinueErrorHandlingRule()
	if r.ID() != "CONV014" {
		t.Errorf("ID = %q, want CONV014", r.ID())
	}
}
