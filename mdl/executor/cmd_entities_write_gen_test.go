// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestAstToAttributeTypeGen_Primitives(t *testing.T) {
	cases := []struct {
		kind     string
		dt       ast.DataType
		wantType string
	}{
		{"Integer", ast.DataType{Kind: ast.TypeInteger}, "DomainModels$IntegerAttributeType"},
		{"Long", ast.DataType{Kind: ast.TypeLong}, "DomainModels$LongAttributeType"},
		{"Decimal", ast.DataType{Kind: ast.TypeDecimal}, "DomainModels$DecimalAttributeType"},
		{"Boolean", ast.DataType{Kind: ast.TypeBoolean}, "DomainModels$BooleanAttributeType"},
		{"AutoNumber", ast.DataType{Kind: ast.TypeAutoNumber}, "DomainModels$AutoNumberAttributeType"},
		{"Binary", ast.DataType{Kind: ast.TypeBinary}, "DomainModels$BinaryAttributeType"},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			got := astToAttributeTypeGen(c.dt)
			if got == nil {
				t.Fatal("got nil")
			}
			if got.TypeName() != c.wantType {
				t.Errorf("TypeName = %q, want %q", got.TypeName(), c.wantType)
			}
		})
	}
}

func TestAstToAttributeTypeGen_StringWithLength(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{Kind: ast.TypeString, Length: 250})
	s, ok := got.(*genDm.StringAttributeType)
	if !ok {
		t.Fatalf("got %T, want *StringAttributeType", got)
	}
	if s.Length() != 250 {
		t.Errorf("Length = %d, want 250", s.Length())
	}
}

func TestAstToAttributeTypeGen_StringDefaultLength(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{Kind: ast.TypeString})
	s, ok := got.(*genDm.StringAttributeType)
	if !ok {
		t.Fatalf("got %T, want *StringAttributeType", got)
	}
	if s.Length() != 0 {
		t.Errorf("Length = %d, want 0 (unlimited)", s.Length())
	}
}

func TestAstToAttributeTypeGen_DateTimeLocalized(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{Kind: ast.TypeDateTime})
	dt, ok := got.(*genDm.DateTimeAttributeType)
	if !ok {
		t.Fatalf("got %T, want *DateTimeAttributeType", got)
	}
	if !dt.LocalizeDate() {
		t.Error("DateTime should have LocalizeDate=true (matches legacy convertDataType)")
	}
}

func TestAstToAttributeTypeGen_DateGap(t *testing.T) {
	// R7: gen has no Date; falls through to DateTimeAttributeType{}.
	got := astToAttributeTypeGen(ast.DataType{Kind: ast.TypeDate})
	dt, ok := got.(*genDm.DateTimeAttributeType)
	if !ok {
		t.Fatalf("Date kind: got %T, want *DateTimeAttributeType (R7 schema gap fallback)", got)
	}
	if dt.LocalizeDate() {
		t.Error("Date kind should not set LocalizeDate (distinguishes from DateTime)")
	}
}

func TestAstToAttributeTypeGen_EnumerationWithRef(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{
		Kind:    ast.TypeEnumeration,
		EnumRef: &ast.QualifiedName{Module: "MyMod", Name: "MyEnum"},
	})
	e, ok := got.(*genDm.EnumerationAttributeType)
	if !ok {
		t.Fatalf("got %T, want *EnumerationAttributeType", got)
	}
	if e.EnumerationQualifiedName() != "MyMod.MyEnum" {
		t.Errorf("EnumerationQualifiedName = %q, want MyMod.MyEnum", e.EnumerationQualifiedName())
	}
}

func TestAstToAttributeTypeGen_EnumerationNilRef(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{Kind: ast.TypeEnumeration})
	e, ok := got.(*genDm.EnumerationAttributeType)
	if !ok {
		t.Fatalf("got %T, want *EnumerationAttributeType", got)
	}
	if e.EnumerationQualifiedName() != "" {
		t.Errorf("EnumerationQualifiedName = %q, want empty", e.EnumerationQualifiedName())
	}
}

func TestAstToAttributeGen_NameType(t *testing.T) {
	a := &ast.Attribute{
		Name:          "Email",
		Documentation: "user email",
		Type:          ast.DataType{Kind: ast.TypeString, Length: 200},
	}
	got := astToAttributeGen(a)
	if got == nil {
		t.Fatal("nil attr")
	}
	if got.Name() != "Email" {
		t.Errorf("Name = %q", got.Name())
	}
	if got.Documentation() != "user email" {
		t.Errorf("Documentation = %q", got.Documentation())
	}
	if got.Type() == nil {
		t.Error("Type is nil")
	}
}

func TestAstToAttributeGen_Calculated(t *testing.T) {
	a := &ast.Attribute{
		Name:                "Total",
		Type:                ast.DataType{Kind: ast.TypeDecimal},
		Calculated:          true,
		CalculatedMicroflow: &ast.QualifiedName{Module: "M", Name: "CalcTotal"},
	}
	got := astToAttributeGen(a)
	cv, ok := got.Value().(*genDm.CalculatedValue)
	if !ok {
		t.Fatalf("Value should be *CalculatedValue, got %T", got.Value())
	}
	if cv.MicroflowQualifiedName() != "M.CalcTotal" {
		t.Errorf("MicroflowQualifiedName = %q", cv.MicroflowQualifiedName())
	}
}

func TestAstToAttributeGen_DefaultStringValue(t *testing.T) {
	a := &ast.Attribute{
		Name:         "Status",
		Type:         ast.DataType{Kind: ast.TypeString},
		HasDefault:   true,
		DefaultValue: "active",
	}
	got := astToAttributeGen(a)
	sv, ok := got.Value().(*genDm.StoredValue)
	if !ok {
		t.Fatalf("Value should be *StoredValue, got %T", got.Value())
	}
	if sv.DefaultValue() != "active" {
		t.Errorf("DefaultValue = %q", sv.DefaultValue())
	}
}

func TestAstToAttributeGen_EnumDefaultStripsPrefix(t *testing.T) {
	a := &ast.Attribute{
		Name:         "Status",
		Type:         ast.DataType{Kind: ast.TypeEnumeration, EnumRef: &ast.QualifiedName{Module: "M", Name: "Status"}},
		HasDefault:   true,
		DefaultValue: "M.Status.Open",
	}
	got := astToAttributeGen(a)
	sv := got.Value().(*genDm.StoredValue)
	if sv.DefaultValue() != "Open" {
		t.Errorf("Enum default should strip prefix; got %q", sv.DefaultValue())
	}
}

func TestAstToAttributeGen_NilInput(t *testing.T) {
	if got := astToAttributeGen(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
}

func TestAstToAttributeTypeGen_DefaultFallback(t *testing.T) {
	got := astToAttributeTypeGen(ast.DataType{Kind: 999})
	s, ok := got.(*genDm.StringAttributeType)
	if !ok {
		t.Fatalf("unknown kind: got %T, want *StringAttributeType", got)
	}
	if s.Length() != 200 {
		t.Errorf("default fallback Length = %d, want 200", s.Length())
	}
}
