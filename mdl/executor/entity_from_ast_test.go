// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestBuildEntityFromAST_Basic(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Customer"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Name", Type: ast.DataType{Kind: ast.TypeString, Length: 200}},
		},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	if got := e.Name(); got != "Customer" {
		t.Errorf("Name() = %q, want %q", got, "Customer")
	}

	attrs := e.AttributesItems()
	if len(attrs) != 1 {
		t.Fatalf("AttributesItems() len = %d, want 1", len(attrs))
	}
	attr, ok := attrs[0].(*genDm.Attribute)
	if !ok {
		t.Fatalf("attribute[0] is %T, want *genDm.Attribute", attrs[0])
	}
	if got := attr.Name(); got != "Name" {
		t.Errorf("attr Name() = %q, want %q", got, "Name")
	}
	st, ok := attr.Type().(*genDm.StringAttributeType)
	if !ok {
		t.Fatalf("attr type is %T, want *genDm.StringAttributeType", attr.Type())
	}
	if got := st.Length(); got != 200 {
		t.Errorf("String length = %d, want 200", got)
	}

	ng, ok := e.Generalization().(*genDm.NoGeneralization)
	if !ok {
		t.Fatalf("generalization is %T, want *genDm.NoGeneralization", e.Generalization())
	}
	if !ng.Persistable() {
		t.Errorf("Persistable() = false, want true for persistent entity")
	}
}

func TestBuildEntityFromAST_BooleanDefaultInjected(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Flagged"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			// Boolean attribute with NO explicit default.
			{Name: "Active", Type: ast.DataType{Kind: ast.TypeBoolean}, HasDefault: true, DefaultValue: false},
		},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	attr := e.AttributesItems()[0].(*genDm.Attribute)
	sv, ok := attr.Value().(*genDm.StoredValue)
	if !ok {
		t.Fatalf("attr value is %T, want *genDm.StoredValue", attr.Value())
	}
	if got := sv.DefaultValue(); got != "false" {
		t.Errorf("Boolean DefaultValue() = %q, want %q", got, "false")
	}
}

func TestBuildEntityFromAST_AutoNumberSeedInjected(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Counter"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			// AutoNumber with no explicit seed -> must default to "1".
			{Name: "Number", Type: ast.DataType{Kind: ast.TypeAutoNumber}},
		},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	attr := e.AttributesItems()[0].(*genDm.Attribute)
	sv, ok := attr.Value().(*genDm.StoredValue)
	if !ok {
		t.Fatalf("attr value is %T, want *genDm.StoredValue", attr.Value())
	}
	if got := sv.DefaultValue(); got != "1" {
		t.Errorf("AutoNumber DefaultValue() = %q, want %q", got, "1")
	}
}

func TestBuildEntityFromAST_EnumDefaultTruncated(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{
				Name: "Status",
				Type: ast.DataType{
					Kind:    ast.TypeEnumeration,
					EnumRef: &ast.QualifiedName{Module: "MyModule", Name: "OrderStatus"},
				},
				HasDefault:   true,
				DefaultValue: "MyModule.OrderStatus.Draft",
			},
		},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	attr := e.AttributesItems()[0].(*genDm.Attribute)
	sv := attr.Value().(*genDm.StoredValue)
	if got := sv.DefaultValue(); got != "Draft" {
		t.Errorf("Enum DefaultValue() = %q, want %q (trailing segment only)", got, "Draft")
	}
}

func TestBuildEntityFromAST_Location(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name:     ast.QualifiedName{Module: "MyModule", Name: "Placed"},
		Kind:     ast.EntityPersistent,
		Position: &ast.Position{X: 120, Y: 340},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	if got := e.Location(); got != "120;340" {
		t.Errorf("Location() = %q, want %q (semicolon-separated)", got, "120;340")
	}
}

func TestBuildEntityFromAST_EventHandler(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "Audited"},
		Kind: ast.EntityPersistent,
		EventHandlers: []ast.EventHandlerDef{
			{
				Moment:          "Before",
				Event:           "Commit",
				Microflow:       ast.QualifiedName{Module: "MyModule", Name: "ACT_Validate"},
				PassEventObject: true,
			},
		},
	}

	e, err := buildEntityFromAST("MyModule", s)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	handlers := e.EventHandlersItems()
	if len(handlers) != 1 {
		t.Fatalf("EventHandlersItems() len = %d, want 1", len(handlers))
	}
	h, ok := handlers[0].(*genDm.EventHandler)
	if !ok {
		t.Fatalf("handler[0] is %T, want *genDm.EventHandler", handlers[0])
	}
	if h.Moment() != "Before" || h.Event() != "Commit" {
		t.Errorf("handler = (%q,%q), want (Before,Commit)", h.Moment(), h.Event())
	}
}

func TestCanonicalDataTypeToGenAttrType(t *testing.T) {
	cases := []struct {
		name string
		dt   canonical.DataType
		want any
	}{
		{"String", canonical.DataType{Kind: canonical.KindString, Length: 50}, (*genDm.StringAttributeType)(nil)},
		{"Integer", canonical.DataType{Kind: canonical.KindInteger}, (*genDm.IntegerAttributeType)(nil)},
		{"Boolean", canonical.DataType{Kind: canonical.KindBoolean}, (*genDm.BooleanAttributeType)(nil)},
		{"AutoNumber", canonical.DataType{Kind: canonical.KindAutoNumber}, (*genDm.AutoNumberAttributeType)(nil)},
		{"Enum", canonical.DataType{Kind: canonical.KindEnumRef, Ref: "M.E"}, (*genDm.EnumerationAttributeType)(nil)},
		{"UnresolvedRef", canonical.DataType{Kind: canonical.KindUnresolvedRef, Ref: "M.E"}, (*genDm.EnumerationAttributeType)(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			el, err := canonicalDataTypeToGenAttrType(tc.dt)
			if err != nil {
				t.Fatalf("canonicalDataTypeToGenAttrType(%v): %v", tc.dt, err)
			}
			switch tc.want.(type) {
			case *genDm.StringAttributeType:
				if _, ok := el.(*genDm.StringAttributeType); !ok {
					t.Errorf("got %T, want *StringAttributeType", el)
				}
			case *genDm.IntegerAttributeType:
				if _, ok := el.(*genDm.IntegerAttributeType); !ok {
					t.Errorf("got %T, want *IntegerAttributeType", el)
				}
			case *genDm.BooleanAttributeType:
				if _, ok := el.(*genDm.BooleanAttributeType); !ok {
					t.Errorf("got %T, want *BooleanAttributeType", el)
				}
			case *genDm.AutoNumberAttributeType:
				if _, ok := el.(*genDm.AutoNumberAttributeType); !ok {
					t.Errorf("got %T, want *AutoNumberAttributeType", el)
				}
			case *genDm.EnumerationAttributeType:
				ea, ok := el.(*genDm.EnumerationAttributeType)
				if !ok {
					t.Fatalf("got %T, want *EnumerationAttributeType", el)
				}
				if ea.EnumerationQualifiedName() != tc.dt.Ref {
					t.Errorf("EnumerationQualifiedName() = %q, want %q", ea.EnumerationQualifiedName(), tc.dt.Ref)
				}
			}
		})
	}

	// Entity / list-of types are rejected as entity attributes.
	if _, err := canonicalDataTypeToGenAttrType(canonical.DataType{Kind: canonical.KindEntityRef, Ref: "M.E"}); err == nil {
		t.Error("expected error for KindEntityRef attribute, got nil")
	}
}
