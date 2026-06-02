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

// TestUpdateAssociationConnections_SpreadRange verifies that for N associations
// sharing a side, Y values are strictly spread (no duplicates, no corners).
func TestUpdateAssociationConnections_SpreadRange(t *testing.T) {
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
