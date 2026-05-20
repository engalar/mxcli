// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
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

// TestAstToEntityGen_IndexAttributePointerNonEmpty verifies that each
// IndexedAttribute produced by astToEntityGen references a real attribute
// ID, not an empty string.
//
// Regression: without pre-assigning UUIDs to attributes before building
// attrNameToID, IndexedAttribute.AttributePointer was serialised as BSON
// string "", causing mx check to throw InvalidCastException (String→Byte[]).
func TestAstToEntityGen_IndexAttributePointerNonEmpty(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Saiban"},
		Attributes: []ast.Attribute{
			{Name: "Id_Shubetu", Type: ast.DataType{Kind: ast.TypeString, Length: 10}},
			{Name: "Sub_Key", Type: ast.DataType{Kind: ast.TypeString, Length: 20}},
		},
		Indexes: []ast.Index{
			{Columns: []ast.IndexColumn{{Name: "Id_Shubetu"}, {Name: "Sub_Key"}}},
		},
	}
	entity := astToEntityGen(s)
	if entity == nil {
		t.Fatal("astToEntityGen returned nil")
	}
	if len(entity.IndexesItems()) != 1 {
		t.Fatalf("Indexes count = %d, want 1", len(entity.IndexesItems()))
	}
	idx, ok := entity.IndexesItems()[0].(*genDm.Index)
	if !ok {
		t.Fatalf("Indexes[0] type = %T, want *genDm.Index", entity.IndexesItems()[0])
	}
	if len(idx.AttributesItems()) != 2 {
		t.Fatalf("IndexedAttribute count = %d, want 2", len(idx.AttributesItems()))
	}
	for i, ia := range idx.AttributesItems() {
		indexed, ok := ia.(*genDm.IndexedAttribute)
		if !ok {
			t.Fatalf("Attributes[%d] type = %T, want *genDm.IndexedAttribute", i, ia)
		}
		if indexed.AttributeRefID() == "" {
			t.Errorf("Attributes[%d].AttributePointer is empty — serialised as BSON string, Mendix expects binary UUID", i)
		}
	}
}

// TestAstToEntityGen_IndexAttributePointerMatchesAttrID verifies that each
// IndexedAttribute.AttributePointer matches exactly the $ID of the
// corresponding attribute — so the cross-reference resolves correctly in
// Studio Pro.
func TestAstToEntityGen_IndexAttributePointerMatchesAttrID(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Order"},
		Attributes: []ast.Attribute{
			{Name: "Code", Type: ast.DataType{Kind: ast.TypeString, Length: 10}},
			{Name: "Status", Type: ast.DataType{Kind: ast.TypeString, Length: 10}},
		},
		Indexes: []ast.Index{
			{Columns: []ast.IndexColumn{{Name: "Code"}}},
			{Columns: []ast.IndexColumn{{Name: "Status"}, {Name: "Code"}}},
		},
	}
	entity := astToEntityGen(s)
	if entity == nil {
		t.Fatal("astToEntityGen returned nil")
	}

	// Build attr name→ID map from the constructed entity.
	attrIDs := make(map[string]element.ID)
	for _, a := range entity.AttributesItems() {
		type namer interface{ Name() string }
		type ider interface{ ID() element.ID }
		n, hasN := a.(namer)
		id, hasID := a.(ider)
		if hasN && hasID {
			attrIDs[n.Name()] = id.ID()
		}
	}
	// Precondition: attributes must have non-empty IDs — otherwise the
	// cross-reference check below would trivially pass (empty==empty).
	for name, id := range attrIDs {
		if id == "" {
			t.Fatalf("attribute %q has empty ID — UUID pre-assignment is missing", name)
		}
	}

	wantPointers := [][]element.ID{
		{attrIDs["Code"]},
		{attrIDs["Status"], attrIDs["Code"]},
	}
	for i, idxElem := range entity.IndexesItems() {
		idx, ok := idxElem.(*genDm.Index)
		if !ok {
			t.Fatalf("Indexes[%d] type = %T", i, idxElem)
		}
		for j, ia := range idx.AttributesItems() {
			indexed, ok := ia.(*genDm.IndexedAttribute)
			if !ok {
				t.Fatalf("Indexes[%d].Attributes[%d] type = %T", i, j, ia)
			}
			want := wantPointers[i][j]
			if indexed.AttributeRefID() != want {
				t.Errorf("Indexes[%d].Attributes[%d]: AttributePointer = %q, want %q (attr ID)", i, j, indexed.AttributeRefID(), want)
			}
		}
	}
}
