// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// splitWith builds a fresh ExclusiveSplit with the given caption.
func splitWith(caption string) *genMf.ExclusiveSplit {
	s := genMf.NewExclusiveSplit()
	s.SetCaption(caption)
	return s
}

func TestFindEmptySplitCaptions_EmptyCaption(t *testing.T) {
	objects := []element.Element{splitWith("")}

	var violations []linter.Violation
	r := NewExclusiveSplitCaptionRule()
	findEmptySplitCaptions(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "CONV012" {
		t.Errorf("expected CONV012, got %s", violations[0].RuleID)
	}
}

func TestFindEmptySplitCaptions_WithCaption(t *testing.T) {
	objects := []element.Element{splitWith("Is order valid?")}

	var violations []linter.Violation
	r := NewExclusiveSplitCaptionRule()
	findEmptySplitCaptions(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestFindEmptySplitCaptions_WhitespaceOnly(t *testing.T) {
	objects := []element.Element{splitWith("   ")}

	var violations []linter.Violation
	r := NewExclusiveSplitCaptionRule()
	findEmptySplitCaptions(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation for whitespace caption, got %d", len(violations))
	}
}

func TestFindEmptySplitCaptions_InsideLoop(t *testing.T) {
	loopBody := genMf.NewMicroflowObjectCollection()
	loopBody.AddObjects(splitWith(""))

	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(loopBody)

	objects := []element.Element{loop}

	var violations []linter.Violation
	r := NewExclusiveSplitCaptionRule()
	findEmptySplitCaptions(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation inside loop, got %d", len(violations))
	}
}

func TestExclusiveSplitCaptionRule_Metadata(t *testing.T) {
	r := NewExclusiveSplitCaptionRule()
	if r.ID() != "CONV012" {
		t.Errorf("ID = %q, want CONV012", r.ID())
	}
	if r.Category() != "quality" {
		t.Errorf("Category = %q, want quality", r.Category())
	}
}
