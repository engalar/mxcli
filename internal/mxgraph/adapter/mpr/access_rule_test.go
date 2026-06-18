package mpr

import (
	"testing"
)

func TestAccessRuleAdapter_Name(t *testing.T) {
	a := &AccessRuleAdapter{}
	if a.Name() != "accessrule" {
		t.Errorf("Name() = %q, want accessrule", a.Name())
	}
}

func TestAccessRuleAdapter_Schema(t *testing.T) {
	a := &AccessRuleAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, l := range s.NodeLabels {
		if l == "AccessRule" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing AccessRule label")
	}
	hasEdge := false
	for _, et := range s.EdgeTypes {
		if et.Type == "HAS_ACCESS_RULE" {
			hasEdge = true
			break
		}
	}
	if !hasEdge {
		t.Error("Schema missing HAS_ACCESS_RULE edge type")
	}
}

func TestParseAccessRuleRights(t *testing.T) {
	tests := []struct {
		rights         string
		wantRead, wantWrite bool
	}{
		{"ReadOnly", true, false},
		{"ReadWrite", true, true},
		{"", false, false},
		{"None", false, false},
	}
	for _, tt := range tests {
		r, w := parseAccessRuleRights(tt.rights)
		if r != tt.wantRead || w != tt.wantWrite {
			t.Errorf("parseAccessRuleRights(%q) = (%v,%v), want (%v,%v)",
				tt.rights, r, w, tt.wantRead, tt.wantWrite)
		}
	}
}
