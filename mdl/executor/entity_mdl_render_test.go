// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestRenderEntityMDLBasic(t *testing.T) {
	spec := entityMDLSpec{
		module: "MyModule",
		name:   "Customer",
		kind:   "persistent",
		attributes: []attrMDLSpec{
			{name: "Name", dataType: canonical.DataType{Kind: canonical.KindString, Length: 200}, notNull: true},
			{name: "Age", dataType: canonical.DataType{Kind: canonical.KindInteger}},
		},
	}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "create persistent entity MyModule.Customer") {
		t.Errorf("missing create header:\n%s", out)
	}
	if !strings.Contains(out, "Name: String(200) not null") {
		t.Errorf("missing Name attr line:\n%s", out)
	}
	if !strings.Contains(out, "Age: Integer") {
		t.Errorf("missing Age attr line:\n%s", out)
	}
}

func TestRenderEntityMDLCreateOrModify(t *testing.T) {
	spec := entityMDLSpec{module: "M", name: "E", kind: "persistent"}
	out := renderEntityMDL(spec, true)
	if !strings.HasPrefix(strings.TrimSpace(out), "create or modify persistent entity M.E") {
		t.Errorf("expected 'create or modify' prefix:\n%s", out)
	}
	plain := renderEntityMDL(spec, false)
	if strings.Contains(plain, "create or modify") {
		t.Errorf("createOrModify=false should not contain 'create or modify':\n%s", plain)
	}
}

func TestRenderEntityMDLNonPersistent(t *testing.T) {
	spec := entityMDLSpec{module: "M", name: "Temp", kind: "non-persistent"}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "create non-persistent entity M.Temp") {
		t.Errorf("expected non-persistent kind:\n%s", out)
	}
}

func TestRenderEntityMDLDocAndPosition(t *testing.T) {
	spec := entityMDLSpec{
		module:        "M",
		name:          "E",
		kind:          "persistent",
		documentation: "An example entity",
		hasPosition:   true,
		positionX:     120,
		positionY:     80,
	}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "/**\n * An example entity\n */") {
		t.Errorf("missing documentation block:\n%s", out)
	}
	if !strings.Contains(out, "@Position(120, 80)") {
		t.Errorf("missing @Position annotation:\n%s", out)
	}
}

func TestRenderEntityMDLExtendsAndConstraints(t *testing.T) {
	spec := entityMDLSpec{
		module:    "M",
		name:      "Child",
		kind:      "persistent",
		extendsQN: "System.User",
		attributes: []attrMDLSpec{
			{
				name:         "Email",
				dataType:     canonical.DataType{Kind: canonical.KindString},
				notNull:      true,
				notNullError: "it's required",
				unique:       true,
			},
		},
	}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "extends System.User") {
		t.Errorf("missing extends clause:\n%s", out)
	}
	// Single quotes in error messages are doubled, not backslash-escaped.
	if !strings.Contains(out, "not null error 'it''s required'") {
		t.Errorf("error message quoting wrong:\n%s", out)
	}
	if !strings.Contains(out, "unique") {
		t.Errorf("missing unique constraint:\n%s", out)
	}
}

func TestEntitySpecFromASTBasic(t *testing.T) {
	s := &ast.CreateEntityStmt{
		Name:           ast.QualifiedName{Module: "Sales", Name: "Order"},
		Kind:           ast.EntityPersistent,
		Documentation:  "order doc",
		Generalization: &ast.QualifiedName{Module: "System", Name: "FileDocument"},
		Position:       &ast.Position{X: 10, Y: 20},
		Attributes: []ast.Attribute{
			{Name: "Total", Type: ast.DataType{Kind: ast.TypeDecimal, Precision: 10, Scale: 2}, NotNull: true},
			{Name: "Note", Type: ast.DataType{Kind: ast.TypeString, Length: 500}},
		},
		Indexes: []ast.Index{
			{Columns: []ast.IndexColumn{{Name: "Total"}, {Name: "Note", Descending: true}}},
		},
		SystemMembers: []string{"createdDate"},
	}
	spec := entitySpecFromAST(s)
	if spec.module != "Sales" || spec.name != "Order" {
		t.Fatalf("unexpected name: %q.%q", spec.module, spec.name)
	}
	if spec.kind != "persistent" {
		t.Errorf("expected persistent, got %q", spec.kind)
	}
	if spec.documentation != "order doc" {
		t.Errorf("documentation not carried: %q", spec.documentation)
	}
	if spec.extendsQN != "System.FileDocument" {
		t.Errorf("extends not carried: %q", spec.extendsQN)
	}
	if !spec.hasPosition || spec.positionX != 10 || spec.positionY != 20 {
		t.Errorf("position not carried: %+v", spec)
	}
	if len(spec.attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(spec.attributes))
	}
	if spec.attributes[0].dataType.Kind != canonical.KindDecimal || spec.attributes[0].dataType.Precision != 10 {
		t.Errorf("decimal type not converted: %+v", spec.attributes[0].dataType)
	}
	if len(spec.indexes) != 1 || len(spec.indexes[0].columns) != 2 {
		t.Fatalf("index not converted: %+v", spec.indexes)
	}
	if !spec.indexes[0].columns[0].ascending || spec.indexes[0].columns[1].ascending {
		t.Errorf("index column direction wrong: %+v", spec.indexes[0].columns)
	}
	if len(spec.systemMembers) != 1 || spec.systemMembers[0] != "createdDate" {
		t.Errorf("system members not carried: %+v", spec.systemMembers)
	}

	// Round-trip through the renderer.
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "create persistent entity Sales.Order extends System.FileDocument") {
		t.Errorf("rendered header wrong:\n%s", out)
	}
	if !strings.Contains(out, "index (Total, Note desc)") {
		t.Errorf("rendered index wrong:\n%s", out)
	}
}

func TestEntitySpecFromASTNoBooleanDefaultInjection(t *testing.T) {
	// The renderer must NOT inject `default false` for booleans — that is the
	// write path's job. A Boolean attribute with HasDefault=false stays bare.
	s := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Active", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	spec := entitySpecFromAST(s)
	if spec.attributes[0].hasDefault {
		t.Errorf("renderer injected a default; HasDefault should stay false")
	}
	out := renderEntityMDL(spec, false)
	if strings.Contains(out, "default") {
		t.Errorf("rendered output injected a default:\n%s", out)
	}
}

func TestEntityKindStr(t *testing.T) {
	cases := map[ast.EntityKind]string{
		ast.EntityPersistent:    "persistent",
		ast.EntityNonPersistent: "non-persistent",
		ast.EntityView:          "view",
		ast.EntityExternal:      "external",
	}
	for k, want := range cases {
		if got := entityKindStr(k); got != want {
			t.Errorf("entityKindStr(%v) = %q, want %q", k, got, want)
		}
	}
}

func TestEntitySpecFromGenBasic(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Product")
	e.SetDocumentation("a product")
	e.SetLocation("100;200")

	attr := genDm.NewAttribute()
	attr.SetName("Title")
	attr.SetType(genDm.NewStringAttributeType())
	e.AddAttributes(attr)

	spec := entitySpecFromGen("Catalog", e)
	if spec.module != "Catalog" || spec.name != "Product" {
		t.Fatalf("unexpected name: %q.%q", spec.module, spec.name)
	}
	if spec.documentation != "a product" {
		t.Errorf("documentation not carried: %q", spec.documentation)
	}
	if !spec.hasPosition || spec.positionX != 100 || spec.positionY != 200 {
		t.Errorf("position not parsed: %+v", spec)
	}
	if spec.kind != "persistent" {
		t.Errorf("expected persistent, got %q", spec.kind)
	}
	if len(spec.attributes) != 1 || spec.attributes[0].name != "Title" {
		t.Fatalf("attributes not converted: %+v", spec.attributes)
	}
	if spec.attributes[0].dataType.Kind != canonical.KindString {
		t.Errorf("string type not converted: %+v", spec.attributes[0].dataType)
	}

	out := renderEntityMDL(spec, true)
	if !strings.Contains(out, "create or modify persistent entity Catalog.Product") {
		t.Errorf("rendered header wrong:\n%s", out)
	}
	if !strings.Contains(out, "Title: String") {
		t.Errorf("rendered attribute wrong:\n%s", out)
	}
}

func TestLastAttrSegment(t *testing.T) {
	if got := lastAttrSegment("Mod.Ent.Attr"); got != "Attr" {
		t.Errorf("lastAttrSegment = %q, want Attr", got)
	}
	if got := lastAttrSegment("Bare"); got != "Bare" {
		t.Errorf("lastAttrSegment = %q, want Bare", got)
	}
}
