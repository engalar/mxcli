package bsoncompare_test

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

type mockT struct {
	failed  bool
	message string
}

func (m *mockT) Helper() {}
func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func TestAssertEqual_SelfComparePasses(t *testing.T) {
	mt := &mockT{}
	bsoncompare.AssertEqual(mt,
		"../../testdata/corpus-b/app.mpr",
		"../../testdata/corpus-b/app.mpr",
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
	if mt.failed {
		t.Errorf("self-compare must pass, got: %s", mt.message)
	}
}

func TestExpectAdded_Matches(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.ACT_New", Kind: bsoncompare.DiffAdded},
	}
	matcher := bsoncompare.ExpectAdded("MyFirstModule.ACT_New")
	if err := matcher.Match(diffs); err != nil {
		t.Errorf("ExpectAdded should match: %v", err)
	}
}

func TestExpectAdded_NotFound(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{}
	matcher := bsoncompare.ExpectAdded("MyFirstModule.ACT_New")
	if err := matcher.Match(diffs); err == nil {
		t.Error("ExpectAdded should fail when unit not in diffs")
	}
}

func TestExpectNoOtherChanges_ExtraUnit(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.ACT_New", Kind: bsoncompare.DiffAdded},
		{QualifiedName: "MyFirstModule.ACT_Unexpected", Kind: bsoncompare.DiffChanged},
	}
	// Claim ACT_New via ExpectAdded
	bsoncompare.ExpectAdded("MyFirstModule.ACT_New").Match(diffs)
	// Now ExpectNoOtherChanges should fail (ACT_Unexpected is unclaimed)
	err := bsoncompare.ExpectNoOtherChanges().Match(diffs)
	if err == nil {
		t.Error("ExpectNoOtherChanges should fail when unexpected diffs remain")
	}
}
