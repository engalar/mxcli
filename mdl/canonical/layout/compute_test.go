package layout_test

import (
	"image"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/canonical/layout"
)

func TestCompute_SingleNode(t *testing.T) {
	nodes := []layout.Node{{ID: "a", AttrCount: 3}}
	got := layout.Compute(nodes, nil)
	if got["a"] != (image.Point{X: 50, Y: 50}) {
		t.Errorf("single node: want (50,50), got %v", got["a"])
	}
}

func TestCompute_LinearChain(t *testing.T) {
	// A → B → C  should produce 3 different columns
	nodes := []layout.Node{
		{ID: "a", AttrCount: 2},
		{ID: "b", AttrCount: 2},
		{ID: "c", AttrCount: 2},
	}
	edges := []layout.Edge{{From: "a", To: "b"}, {From: "b", To: "c"}}
	got := layout.Compute(nodes, edges)
	if len(got) != 3 {
		t.Fatalf("want 3 positions, got %d", len(got))
	}
	// All three X values must be distinct (different columns)
	xs := map[int]bool{got["a"].X: true, got["b"].X: true, got["c"].X: true}
	if len(xs) != 3 {
		t.Errorf("linear chain: expected 3 distinct X columns, got %v", got)
	}
}

func TestCompute_Star(t *testing.T) {
	// hub connects to 4 spokes; hub should be in its own column
	nodes := []layout.Node{
		{ID: "hub", AttrCount: 5},
		{ID: "s1", AttrCount: 1},
		{ID: "s2", AttrCount: 1},
		{ID: "s3", AttrCount: 1},
		{ID: "s4", AttrCount: 1},
	}
	edges := []layout.Edge{
		{From: "hub", To: "s1"}, {From: "hub", To: "s2"},
		{From: "hub", To: "s3"}, {From: "hub", To: "s4"},
	}
	got := layout.Compute(nodes, edges)
	hubX := got["hub"].X
	for _, spoke := range []string{"s1", "s2", "s3", "s4"} {
		if got[spoke].X == hubX {
			t.Errorf("spoke %s should be in a different column from hub", spoke)
		}
	}
}

func TestCompute_TwoComponents(t *testing.T) {
	// Two disconnected pairs; they must not overlap
	nodes := []layout.Node{
		{ID: "a1", AttrCount: 2}, {ID: "a2", AttrCount: 2},
		{ID: "b1", AttrCount: 2}, {ID: "b2", AttrCount: 2},
	}
	edges := []layout.Edge{
		{From: "a1", To: "a2"},
		{From: "b1", To: "b2"},
	}
	got := layout.Compute(nodes, edges)
	// All four positions must be distinct
	seen := map[image.Point]string{}
	for id, pt := range got {
		if prev, exists := seen[pt]; exists {
			t.Errorf("collision: %s and %s both at %v", id, prev, pt)
		}
		seen[pt] = id
	}
}

func TestCompute_EntityHeight(t *testing.T) {
	// Two entities in same column; larger one above means more Y gap below it
	nodes := []layout.Node{
		{ID: "big", AttrCount: 10},  // height = 40 + 10*20 + 30 = 270
		{ID: "small", AttrCount: 1}, // height = max(90, 40 + 1*20 + 30) = 90
	}
	got := layout.Compute(nodes, nil)
	// Both are disconnected, so same column. The second entity's Y must be
	// at least startY + height(first) + gap above it.
	yBig := got["big"].Y
	ySmall := got["small"].Y
	if yBig == ySmall {
		t.Errorf("entities in same column must have different Y values")
	}
	// Gap between them must be at least min-height (90) + 20 gap
	diff := abs(ySmall - yBig)
	if diff < 90+20 {
		t.Errorf("Y gap too small: %d (want >= 110)", diff)
	}
}

func TestCompute_Cycle(t *testing.T) {
	// A ↔ B cycle; algorithm must not hang or produce wrong output
	nodes := []layout.Node{
		{ID: "x", AttrCount: 2},
		{ID: "y", AttrCount: 2},
	}
	edges := []layout.Edge{{From: "x", To: "y"}, {From: "y", To: "x"}}
	got := layout.Compute(nodes, edges)
	if len(got) != 2 {
		t.Fatalf("cycle: want 2 positions, got %d", len(got))
	}
}

func TestCompute_Empty(t *testing.T) {
	got := layout.Compute(nil, nil)
	if len(got) != 0 {
		t.Errorf("empty input: want empty map, got %v", got)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
