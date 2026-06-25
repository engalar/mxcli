//go:build poc
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestNewMiner(t *testing.T) {
	m := NewMiner()
	if m == nil {
		t.Fatal("NewMiner returned nil")
	}
	if m.Records == nil {
		t.Fatal("Miner.Records is nil — must be allocated")
	}
}

func TestWalker_RecordsIfCondition(t *testing.T) {
	mdl := `
CREATE MICROFLOW Mod.Foo ()
RETURNS Boolean AS $ok
{
    DECLARE $ok Boolean = false;
    IF $ok = false THEN
        SET $ok = true;
    END IF;
    RETURN $ok;
}
`
	m := NewMiner()
	if err := WalkMDL(m, "Mod.Foo", mdl); err != nil {
		t.Fatalf("WalkMDL: %v", err)
	}
	var ifCondSeen bool
	for _, r := range m.Records {
		if r.SlotPath == "IfStmt.Condition" && r.SourceText == "$ok = false" {
			ifCondSeen = true
		}
	}
	if !ifCondSeen {
		t.Fatalf("expected IfStmt.Condition record with source '$ok = false'; got %+v", m.Records)
	}
}

func TestWalker_CoversAllSlots(t *testing.T) {
	mdl := `
CREATE MICROFLOW Mod.Foo ($p Integer)
RETURNS Integer AS $r
{
    DECLARE $r Integer = 0;
    SET $r = $p + 1;
    WHILE $r < 10 {
        SET $r = $r + 1;
    }
    RETRIEVE $list FROM Mod.Entity LIMIT 5 OFFSET 1;
    LOG INFO 'count=' + toString($r);
    RETURN $r * 2;
}
`
	m := NewMiner()
	if err := WalkMDL(m, "Mod.Foo", mdl); err != nil {
		t.Fatalf("WalkMDL: %v", err)
	}
	want := map[string]bool{
		"DeclareStmt.InitialValue": false,
		"MfSetStmt.Value":          false,
		"WhileStmt.Condition":      false,
		"RetrieveStmt.LimitExpr":   false,
		"RetrieveStmt.OffsetExpr":  false,
		"LogStmt.Message":          false,
		"ReturnStmt.Value":         false,
	}
	for _, r := range m.Records {
		if _, ok := want[r.SlotPath]; ok {
			want[r.SlotPath] = true
		}
	}
	for slot, hit := range want {
		if !hit {
			t.Errorf("slot %s not recorded; records: %+v", slot, m.Records)
		}
	}
}
