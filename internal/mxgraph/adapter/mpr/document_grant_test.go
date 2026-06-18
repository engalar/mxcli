package mpr

import (
	"testing"
)

func TestDocumentGrantAdapter_Name(t *testing.T) {
	a := &DocumentGrantAdapter{}
	if a.Name() != "documentgrant" {
		t.Errorf("Name() = %q, want documentgrant", a.Name())
	}
}

func TestDocumentGrantAdapter_Schema(t *testing.T) {
	a := &DocumentGrantAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, et := range s.EdgeTypes {
		if et.Type == "HAS_ALLOWED_ROLE" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing HAS_ALLOWED_ROLE edge type")
	}
}
