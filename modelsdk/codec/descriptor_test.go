package codec

import (
	"testing"
)

func TestDescRegistryRegisterAndLookup(t *testing.T) {
	reg := NewDescRegistry()
	reg.Register("DomainModels$Entity", &TypeDesc{
		TypeName: "DomainModels$Entity",
		Properties: []PropDesc{
			{Name: "Name", BSONKey: "Name", Kind: PropKindString},
		},
	})

	td, ok := reg.Lookup("DomainModels$Entity")
	if !ok {
		t.Fatal("Lookup failed for registered type")
	}
	if td.TypeName != "DomainModels$Entity" {
		t.Errorf("TypeName = %q", td.TypeName)
	}
	if len(td.Properties) != 1 {
		t.Fatalf("got %d properties, want 1", len(td.Properties))
	}
	if td.Properties[0].Name != "Name" {
		t.Errorf("property Name = %q", td.Properties[0].Name)
	}
}

func TestDescRegistryAll(t *testing.T) {
	reg := NewDescRegistry()
	reg.Register("A", &TypeDesc{TypeName: "A"})
	reg.Register("B", &TypeDesc{TypeName: "B"})
	all := reg.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d, want 2", len(all))
	}
}

func TestDefaultDescRegistry(t *testing.T) {
	if DefaultDescRegistry == nil {
		t.Fatal("DefaultDescRegistry is nil")
	}
}
