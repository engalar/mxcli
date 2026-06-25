package mpr

import (
	"testing"
)

func TestPageRefAdapter_Name(t *testing.T) {
	t.Parallel()
	a := &PageRefAdapter{}
	if a.Name() != "pageref" {
		t.Errorf("Name() = %q, want pageref", a.Name())
	}
}

func TestPageRefAdapter_Schema(t *testing.T) {
	t.Parallel()
	a := &PageRefAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, et := range s.EdgeTypes {
		if et.Type == "READS_ENTITY" || et.Type == "CALLS_MICROFLOW" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing READS_ENTITY or CALLS_MICROFLOW edge type")
	}
}

func TestQualifyName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, module, want string
	}{
		{"HD.Ticket", "HD", "HD.Ticket"},
		{"Ticket", "HD", "HD.Ticket"},
		{"System.User", "", "System.User"},
	}
	for _, tt := range tests {
		got := qualifyName(tt.name, tt.module)
		if got != tt.want {
			t.Errorf("qualifyName(%q, %q) = %q, want %q", tt.name, tt.module, got, tt.want)
		}
	}
}
