// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"image"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func makeTestAssoc(t *testing.T, name, fromID, toID string) *genDm.Association {
	t.Helper()
	a := genDm.NewAssociation()
	a.SetName(name)
	a.SetParentID(element.ID(fromID))
	a.SetChildID(element.ID(toID))
	return a
}

func makeTestEntity(t *testing.T, name, id string) *genDm.Entity {
	t.Helper()
	e := genDm.NewEntity()
	e.SetName(name)
	e.SetID(element.ID(id))
	return e
}

func buildTestDM(entities []*genDm.Entity, assocs []*genDm.Association) *genDm.DomainModel {
	dm := genDm.NewDomainModel()
	for _, e := range entities {
		dm.AddEntities(e)
	}
	for _, a := range assocs {
		dm.AddAssociations(a)
	}
	return dm
}

// parseConn splits "X;Y" into (x, y) as ints.
func parseConn(t *testing.T, conn string) (x, y int) {
	t.Helper()
	parts := strings.SplitN(conn, ";", 2)
	if len(parts) != 2 {
		t.Fatalf("bad connection string %q", conn)
	}
	parseInt := func(s string) int {
		n := 0
		for _, ch := range s {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		return n
	}
	return parseInt(parts[0]), parseInt(parts[1])
}

// TestUpdateAssociationConnections_SingleAssoc verifies direction is correct
// for a simple left→right pair.
func TestUpdateAssociationConnections_SingleAssoc(t *testing.T) {
	t.Parallel()
	eA := makeTestEntity(t, "A", "id-a")
	eB := makeTestEntity(t, "B", "id-b")
	assoc := makeTestAssoc(t, "A_B", "id-a", "id-b")
	dm := buildTestDM([]*genDm.Entity{eA, eB}, []*genDm.Association{assoc})

	positions := map[string]image.Point{"A": {X: 50, Y: 100}, "B": {X: 400, Y: 100}}
	idToName := map[string]string{"id-a": "A", "id-b": "B"}

	updateAssociationConnections(dm, positions, idToName)

	// A is FROM (parent), B is TO (child). A is left of B: A exits right (X=100), B enters left (X=0).
	px, _ := parseConn(t, assoc.ParentConnection())
	cx, _ := parseConn(t, assoc.ChildConnection())
	if px != 100 {
		t.Errorf("ParentConnection X: want 100 (right), got %d (%q)", px, assoc.ParentConnection())
	}
	if cx != 0 {
		t.Errorf("ChildConnection X: want 0 (left), got %d (%q)", cx, assoc.ChildConnection())
	}
}

// TestUpdateAssociationConnections_MultipleAssocsSameSide verifies that
// when multiple associations exit from the same side, Y coordinates are distinct.
func TestUpdateAssociationConnections_MultipleAssocsSameSide(t *testing.T) {
	t.Parallel()
	eA := makeTestEntity(t, "A", "id-a")
	eB := makeTestEntity(t, "B", "id-b")
	eC := makeTestEntity(t, "C", "id-c")
	eD := makeTestEntity(t, "D", "id-d")

	ab := makeTestAssoc(t, "A_B", "id-a", "id-b")
	ac := makeTestAssoc(t, "A_C", "id-a", "id-c")
	ad := makeTestAssoc(t, "A_D", "id-a", "id-d")

	dm := buildTestDM(
		[]*genDm.Entity{eA, eB, eC, eD},
		[]*genDm.Association{ab, ac, ad},
	)

	positions := map[string]image.Point{
		"A": {X: 50, Y: 100},
		"B": {X: 400, Y: 50},
		"C": {X: 400, Y: 200},
		"D": {X: 400, Y: 350},
	}
	idToName := map[string]string{"id-a": "A", "id-b": "B", "id-c": "C", "id-d": "D"}

	updateAssociationConnections(dm, positions, idToName)

	_, yAB := parseConn(t, ab.ParentConnection())
	_, yAC := parseConn(t, ac.ParentConnection())
	_, yAD := parseConn(t, ad.ParentConnection())

	// All three exit from A's right side — Y values must be distinct.
	if yAB == yAC || yAB == yAD || yAC == yAD {
		t.Errorf("Y values must be distinct when 3 assocs share a side: A_B=%d A_C=%d A_D=%d (all clustered!)", yAB, yAC, yAD)
	}

	// All should exit A's right side and enter B/C/D's left side.
	for _, a := range []*genDm.Association{ab, ac, ad} {
		px, _ := parseConn(t, a.ParentConnection())
		cx, _ := parseConn(t, a.ChildConnection())
		if px != 100 {
			t.Errorf("%s ParentConnection X: want 100, got %d", a.Name(), px)
		}
		if cx != 0 {
			t.Errorf("%s ChildConnection X: want 0, got %d", a.Name(), cx)
		}
	}
}

// TestUpdateAssociationConnections_NoCrossings verifies that Y positions on an
// entity side are ordered by the Y position of the connected entity so that
// lines do not cross each other.
//
// Layout:
//
//	A (left) ──→ B (top-right)   A right Y must be < A right Y for C
//	A (left) ──→ C (mid-right)   same logic on the entry side
//	A (left) ──→ D (bot-right)   i.e. entry Y on B < entry Y on B for other assocs
//
// The assocs are created in reverse order (D first) so non-sorted output
// would assign the smallest Y to A_D and the largest to A_B — crossing every line.
func TestUpdateAssociationConnections_NoCrossings(t *testing.T) {
	t.Parallel()
	eA := makeTestEntity(t, "A", "id-a")
	eB := makeTestEntity(t, "B", "id-b")
	eC := makeTestEntity(t, "C", "id-c")
	eD := makeTestEntity(t, "D", "id-d")

	// Intentionally created in reverse Y order to expose ordering bugs.
	ad := makeTestAssoc(t, "A_D", "id-a", "id-d") // D is lowest (Y=350)
	ac := makeTestAssoc(t, "A_C", "id-a", "id-c") // C is middle (Y=200)
	ab := makeTestAssoc(t, "A_B", "id-a", "id-b") // B is highest (Y=50)

	dm := buildTestDM(
		[]*genDm.Entity{eA, eB, eC, eD},
		[]*genDm.Association{ad, ac, ab},
	)

	positions := map[string]image.Point{
		"A": {X: 50, Y: 200},
		"B": {X: 400, Y: 50},
		"C": {X: 400, Y: 200},
		"D": {X: 400, Y: 350},
	}
	idToName := map[string]string{"id-a": "A", "id-b": "B", "id-c": "C", "id-d": "D"}

	updateAssociationConnections(dm, positions, idToName)

	_, yAB := parseConn(t, ab.ParentConnection()) // exit from A toward B (Y=50)
	_, yAC := parseConn(t, ac.ParentConnection()) // exit from A toward C (Y=200)
	_, yAD := parseConn(t, ad.ParentConnection()) // exit from A toward D (Y=350)

	// B is topmost → should use smallest Y on A's right side.
	// C is middle  → should use middle Y.
	// D is bottom  → should use largest Y.
	if !(yAB < yAC && yAC < yAD) {
		t.Errorf("exit Y on A must be ordered top→bottom to avoid crossings: A_B=%d A_C=%d A_D=%d (want A_B < A_C < A_D)", yAB, yAC, yAD)
	}

	// Entry side: B, C, D each have one connection from A entering their left.
	// Each has only one entry so it gets Y=50 (midpoint) — just verify X=0.
	for _, a := range []*genDm.Association{ab, ac, ad} {
		cx, _ := parseConn(t, a.ChildConnection())
		if cx != 0 {
			t.Errorf("%s ChildConnection X: want 0 (left), got %d", a.Name(), cx)
		}
	}
}

// TestUpdateAssociationConnections_NoCrossings_EntryOrdering verifies that when
// multiple associations enter the SAME entity's left side from sources at
// different Y positions, the entry Y values are ordered to avoid crossings.
func TestUpdateAssociationConnections_NoCrossings_EntryOrdering(t *testing.T) {
	t.Parallel()
	// B, C, D (left column) all connect to Hub (right column).
	// Assocs created in reverse Y order to expose ordering bugs.
	eHub := makeTestEntity(t, "Hub", "id-hub")
	eB := makeTestEntity(t, "B", "id-b")
	eC := makeTestEntity(t, "C", "id-c")
	eD := makeTestEntity(t, "D", "id-d")

	dHub := makeTestAssoc(t, "D_Hub", "id-d", "id-hub") // D is lowest (Y=350)
	cHub := makeTestAssoc(t, "C_Hub", "id-c", "id-hub") // C is middle (Y=200)
	bHub := makeTestAssoc(t, "B_Hub", "id-b", "id-hub") // B is highest (Y=50)

	dm := buildTestDM(
		[]*genDm.Entity{eHub, eB, eC, eD},
		[]*genDm.Association{dHub, cHub, bHub},
	)

	positions := map[string]image.Point{
		"Hub": {X: 400, Y: 200},
		"B":   {X: 50, Y: 50},
		"C":   {X: 50, Y: 200},
		"D":   {X: 50, Y: 350},
	}
	idToName := map[string]string{"id-hub": "Hub", "id-b": "B", "id-c": "C", "id-d": "D"}

	updateAssociationConnections(dm, positions, idToName)

	_, yBHub := parseConn(t, bHub.ChildConnection()) // entry from B (Y=50) → smallest entry Y
	_, yCHub := parseConn(t, cHub.ChildConnection()) // entry from C (Y=200)
	_, yDHub := parseConn(t, dHub.ChildConnection()) // entry from D (Y=350) → largest entry Y

	// B is topmost source → should enter Hub at smallest Y.
	if !(yBHub < yCHub && yCHub < yDHub) {
		t.Errorf("entry Y on Hub must be ordered top→bottom to avoid crossings: B_Hub=%d C_Hub=%d D_Hub=%d (want B < C < D)", yBHub, yCHub, yDHub)
	}
}

// TestUpdateAssociationConnections_SpreadRange verifies that for N associations
// sharing a side, Y values are strictly spread (no duplicates, no corners).
func TestUpdateAssociationConnections_SpreadRange(t *testing.T) {
	t.Parallel()
	names := []string{"B", "C", "D", "E"}
	eA := makeTestEntity(t, "A", "id-a")
	entities := []*genDm.Entity{eA}
	for _, n := range names {
		entities = append(entities, makeTestEntity(t, n, "id-"+strings.ToLower(n)))
	}
	var assocs []*genDm.Association
	for _, n := range names {
		assocs = append(assocs, makeTestAssoc(t, "A_"+n, "id-a", "id-"+strings.ToLower(n)))
	}

	dm := buildTestDM(entities, assocs)
	positions := map[string]image.Point{
		"A": {X: 50, Y: 200},
		"B": {X: 400, Y: 50},
		"C": {X: 400, Y: 150},
		"D": {X: 400, Y: 250},
		"E": {X: 400, Y: 350},
	}
	idToName := map[string]string{
		"id-a": "A", "id-b": "B", "id-c": "C", "id-d": "D", "id-e": "E",
	}

	updateAssociationConnections(dm, positions, idToName)

	seen := make(map[int]bool)
	for _, a := range assocs {
		_, y := parseConn(t, a.ParentConnection())
		if seen[y] {
			t.Errorf("duplicate Y=%d — endpoints still cluster at same point (assoc %s)", y, a.Name())
		}
		seen[y] = true
		if y <= 0 || y >= 100 {
			t.Errorf("Y=%d is at entity corner; should be strictly inside (1-99)", y)
		}
	}
}
