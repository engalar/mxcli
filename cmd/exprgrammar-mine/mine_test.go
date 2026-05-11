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
BEGIN
    DECLARE $ok Boolean = false;
    IF $ok = false THEN
        SET $ok = true;
    END IF;
    RETURN $ok;
END;
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
