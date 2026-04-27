package property

import (
	"testing"
)

func TestByNameRef(t *testing.T) {
	ref := NewByNameRef[*testElement]("superClass", "DomainModels$Entity")

	if ref.Name() != "superClass" {
		t.Errorf("expected name 'superClass', got %q", ref.Name())
	}
	if ref.TargetType() != "DomainModels$Entity" {
		t.Errorf("expected targetType 'DomainModels$Entity', got %q", ref.TargetType())
	}
	if ref.QualifiedName() != "" {
		t.Errorf("expected empty qname before set, got %q", ref.QualifiedName())
	}
	if ref.Dirty() {
		t.Error("expected Dirty() == false before SetQualifiedName()")
	}

	ref.SetQualifiedName("MyModule.Customer")

	if ref.QualifiedName() != "MyModule.Customer" {
		t.Errorf("expected 'MyModule.Customer', got %q", ref.QualifiedName())
	}
	if !ref.Dirty() {
		t.Error("expected Dirty() == true after SetQualifiedName()")
	}
}

func TestByNameRef_SetFromDecode(t *testing.T) {
	ref := NewByNameRef[*testElement]("superClass", "DomainModels$Entity")
	ref.SetFromDecode("MyModule.Base")

	if ref.QualifiedName() != "MyModule.Base" {
		t.Errorf("expected 'MyModule.Base', got %q", ref.QualifiedName())
	}
	if ref.Dirty() {
		t.Error("expected Dirty() == false after SetFromDecode()")
	}
}

func TestByIdRef(t *testing.T) {
	ref := NewByIdRef[*testElement]("entityRef")

	if ref.Name() != "entityRef" {
		t.Errorf("expected name 'entityRef', got %q", ref.Name())
	}
	if ref.RefID() != "" {
		t.Errorf("expected empty RefID before SetID, got %q", ref.RefID())
	}
	if ref.Dirty() {
		t.Error("expected Dirty() == false before SetID()")
	}

	ref.SetID("abc-123")

	if ref.RefID() != "abc-123" {
		t.Errorf("expected 'abc-123', got %q", ref.RefID())
	}
	if !ref.Dirty() {
		t.Error("expected Dirty() == true after SetID()")
	}
}

func TestEnum(t *testing.T) {
	type Visibility string

	e := NewEnum[Visibility]("visibility")

	if e.Name() != "visibility" {
		t.Errorf("expected name 'visibility', got %q", e.Name())
	}
	if e.Get() != "" {
		t.Errorf("expected empty value before Set, got %q", e.Get())
	}
	if e.Dirty() {
		t.Error("expected Dirty() == false before Set()")
	}

	e.Set("Hidden")

	if e.Get() != "Hidden" {
		t.Errorf("expected 'Hidden', got %q", e.Get())
	}
	if !e.Dirty() {
		t.Error("expected Dirty() == true after Set()")
	}
}

func TestEnum_SetFromDecode(t *testing.T) {
	type Visibility string

	e := NewEnum[Visibility]("visibility")
	e.SetFromDecode("Visible")

	if e.Get() != "Visible" {
		t.Errorf("expected 'Visible', got %q", e.Get())
	}
	if e.Dirty() {
		t.Error("expected Dirty() == false after SetFromDecode()")
	}
}
