package bson

import (
	"strings"
	"testing"
)

func TestCompareIdentical(t *testing.T) {
	doc := map[string]any{
		"$Type": "Workflows$Workflow",
		"Name":  "MyWorkflow",
		"Title": "Hello",
	}
	diffs := Compare(doc, doc, CompareOptions{})
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical maps, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareOnlyInLeft(t *testing.T) {
	left := map[string]any{
		"Name":  "MyWorkflow",
		"Title": "Hello",
		"Extra": "leftonly",
	}
	right := map[string]any{
		"Name":  "MyWorkflow",
		"Title": "Hello",
	}
	diffs := Compare(left, right, CompareOptions{})

	found := false
	for _, d := range diffs {
		if d.Path == "Extra" && d.Type == OnlyInLeft {
			found = true
		}
	}
	if !found {
		t.Errorf("expected OnlyInLeft diff for 'Extra', got: %v", diffs)
	}
}

func TestCompareOnlyInRight(t *testing.T) {
	left := map[string]any{
		"Name": "MyWorkflow",
	}
	right := map[string]any{
		"Name":  "MyWorkflow",
		"Extra": "rightonly",
	}
	diffs := Compare(left, right, CompareOptions{})

	found := false
	for _, d := range diffs {
		if d.Path == "Extra" && d.Type == OnlyInRight {
			found = true
		}
	}
	if !found {
		t.Errorf("expected OnlyInRight diff for 'Extra', got: %v", diffs)
	}
}

func TestCompareValueMismatch(t *testing.T) {
	left := map[string]any{
		"Title": "Hello",
	}
	right := map[string]any{
		"Title": "World",
	}
	diffs := Compare(left, right, CompareOptions{})

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Type != ValueMismatch {
		t.Errorf("expected ValueMismatch, got %v", diffs[0].Type)
	}
	if diffs[0].Path != "Title" {
		t.Errorf("expected path 'Title', got %q", diffs[0].Path)
	}
}

func TestCompareSkipsStructural(t *testing.T) {
	left := map[string]any{
		"$ID":                 "abc-123",
		"PersistentId":        "pid-1",
		"RelativeMiddlePoint": map[string]any{"X": 0},
		"Size":                map[string]any{"Width": 100},
		"Name":                "Same",
	}
	right := map[string]any{
		"$ID":                 "xyz-789",
		"PersistentId":        "pid-2",
		"RelativeMiddlePoint": map[string]any{"X": 999},
		"Size":                map[string]any{"Width": 200},
		"Name":                "Same",
	}

	diffs := Compare(left, right, CompareOptions{IncludeAll: false})
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs (structural fields skipped), got %d: %v", len(diffs), diffs)
	}
}

func TestCompareWithAll(t *testing.T) {
	left := map[string]any{
		"$ID":  "abc-123",
		"Name": "Same",
	}
	right := map[string]any{
		"$ID":  "xyz-789",
		"Name": "Same",
	}

	diffs := Compare(left, right, CompareOptions{IncludeAll: true})
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff ($ID), got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "$ID" {
		t.Errorf("expected path '$ID', got %q", diffs[0].Path)
	}
}

func TestCompareArraySmartMatch(t *testing.T) {
	left := map[string]any{
		"Activities": []any{
			map[string]any{"$Type": "Workflows$UserTask", "Name": "Approve", "Caption": "Approve it"},
			map[string]any{"$Type": "Workflows$UserTask", "Name": "Review", "Caption": "Review it"},
		},
	}
	right := map[string]any{
		"Activities": []any{
			// Same elements, reversed order, with a caption change
			map[string]any{"$Type": "Workflows$UserTask", "Name": "Review", "Caption": "Review updated"},
			map[string]any{"$Type": "Workflows$UserTask", "Name": "Approve", "Caption": "Approve it"},
		},
	}

	diffs := Compare(left, right, CompareOptions{})

	// Should find exactly 1 diff: Caption change on Review, matched by identity
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (Caption on Review), got %d: %v", len(diffs), diffs)
	}
	if !strings.Contains(diffs[0].Path, "Review") {
		t.Errorf("expected diff path to reference Review, got %q", diffs[0].Path)
	}
	if diffs[0].Type != ValueMismatch {
		t.Errorf("expected ValueMismatch, got %v", diffs[0].Type)
	}
}

func TestCompareArraySmartMatchOnlyInLeft(t *testing.T) {
	left := map[string]any{
		"Items": []any{
			map[string]any{"$Type": "T", "Name": "A"},
			map[string]any{"$Type": "T", "Name": "B"},
		},
	}
	right := map[string]any{
		"Items": []any{
			map[string]any{"$Type": "T", "Name": "A"},
		},
	}

	diffs := Compare(left, right, CompareOptions{})

	found := false
	for _, d := range diffs {
		if d.Type == OnlyInLeft && strings.Contains(d.Path, "B") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected OnlyInLeft for element B, got: %v", diffs)
	}
}

func TestCompareNestedMaps(t *testing.T) {
	left := map[string]any{
		"Outer": map[string]any{
			"Inner": "leftval",
		},
	}
	right := map[string]any{
		"Outer": map[string]any{
			"Inner": "rightval",
		},
	}

	diffs := Compare(left, right, CompareOptions{})
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "Outer.Inner" {
		t.Errorf("expected path 'Outer.Inner', got %q", diffs[0].Path)
	}
}

func TestFormatDiffsOutput(t *testing.T) {
	diffs := []Diff{
		{Path: "Title", Type: ValueMismatch, LeftValue: `"A"`, RightValue: `"B"`},
		{Path: "Extra", Type: OnlyInLeft, LeftValue: `"x"`},
		{Path: "New", Type: OnlyInRight, RightValue: `"y"`},
	}

	output := FormatDiffs(diffs)
	if !strings.Contains(output, "Summary: 3 differences") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
	if !strings.Contains(output, "1 only-in-left") {
		t.Errorf("expected only-in-left count, got:\n%s", output)
	}
}

func TestFormatDiffsEmpty(t *testing.T) {
	output := FormatDiffs(nil)
	if output != "No differences found." {
		t.Errorf("expected 'No differences found.', got: %q", output)
	}
}
