// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// --------------------------------------------------------------------------
// Lift
// --------------------------------------------------------------------------

func TestLift_PersistentEntity(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "Sales", Name: "Customer"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{
				Name:    "Name",
				Type:    ast.DataType{Kind: ast.TypeString, Length: 100},
				NotNull: true,
			},
			{
				Name:         "Active",
				Type:         ast.DataType{Kind: ast.TypeBoolean},
				HasDefault:   true,
				DefaultValue: "true",
			},
		},
		Position: &ast.Position{X: 100, Y: 200},
	}

	m, err := Lift(stmt)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "Sales", m.Name.Module)
	assert.Equal(t, "Customer", m.Name.Name)
	assert.Equal(t, EntityPersistent, m.Kind)
	require.NotNil(t, m.Position)
	assert.Equal(t, 100, m.Position.X)
	assert.Equal(t, 200, m.Position.Y)
	require.Len(t, m.Attributes, 2)
	assert.Equal(t, "Name", m.Attributes[0].Name)
	assert.Equal(t, canonical.KindString, m.Attributes[0].Type.Kind)
	assert.Equal(t, 100, m.Attributes[0].Type.Length)
	assert.True(t, m.Attributes[0].NotNull)
	assert.Equal(t, "Active", m.Attributes[1].Name)
	assert.True(t, m.Attributes[1].HasDefault)
	assert.Equal(t, "true", m.Attributes[1].DefaultValue)
}

func TestLift_NonPersistent(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "Tmp", Name: "Buffer"},
		Kind: ast.EntityNonPersistent,
	}
	m, err := Lift(stmt)
	require.NoError(t, err)
	assert.Equal(t, EntityNonPersistent, m.Kind)
}

func TestLift_WithExtends(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name:           ast.QualifiedName{Module: "App", Name: "MyImage"},
		Kind:           ast.EntityPersistent,
		Generalization: &ast.QualifiedName{Module: "System", Name: "Image"},
	}
	m, err := Lift(stmt)
	require.NoError(t, err)
	require.NotNil(t, m.Extends)
	assert.Equal(t, "System", m.Extends.Module)
	assert.Equal(t, "Image", m.Extends.Name)
}

func TestLift_EnumAttribute(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "App", Name: "Order"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{
				Name: "Status",
				Type: ast.DataType{
					Kind:    ast.TypeEnumeration,
					EnumRef: &ast.QualifiedName{Module: "App", Name: "OrderStatus"},
				},
			},
		},
	}
	m, err := Lift(stmt)
	require.NoError(t, err)
	require.Len(t, m.Attributes, 1)
	assert.Equal(t, canonical.KindUnresolvedRef, m.Attributes[0].Type.Kind)
	assert.Equal(t, "App.OrderStatus", m.Attributes[0].Type.Ref)
}

func TestLift_NilStatement(t *testing.T) {
	_, err := Lift(nil)
	assert.Error(t, err)
}

// --------------------------------------------------------------------------
// ToMDL
// --------------------------------------------------------------------------

func TestToMDL_MinimalEntity(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "Sales", Name: "Customer"},
		Kind: EntityPersistent,
	}
	out := m.ToMDL()
	assert.Contains(t, out, "create persistent entity Sales.Customer")
	assert.Contains(t, out, "(")
	assert.Contains(t, out, ")")
}

func TestToMDL_WithAttributes(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "Sales", Name: "Customer"},
		Kind: EntityPersistent,
		Attributes: []AttributeModel{
			{
				Name:    "Name",
				Type:    canonical.DataType{Kind: canonical.KindString, Length: 100},
				NotNull: true,
			},
			{
				Name:         "Active",
				Type:         canonical.DataType{Kind: canonical.KindBoolean},
				HasDefault:   true,
				DefaultValue: "true",
			},
		},
	}
	out := m.ToMDL()
	assert.Contains(t, out, "Name: String(100) not null")
	assert.Contains(t, out, "Active: Boolean default true")
	// Last attribute has no trailing comma.
	assert.True(t, strings.Contains(out, "Active: Boolean default true\n)"),
		"expected last attribute to have no comma; got:\n%s", out)
}

func TestToMDL_NonPersistent(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "Tmp", Name: "Buffer"},
		Kind: EntityNonPersistent,
	}
	out := m.ToMDL()
	assert.Contains(t, out, "create non-persistent entity Tmp.Buffer")
}

func TestToMDLStatement_CreateOrModifyPreservesDocumentation(t *testing.T) {
	// Regression: previously executor did `strings.Replace(m.ToMDL(), "create ",
	// "create or modify ", 1)` which incorrectly rewrote the first `create ` it
	// found — including inside the `/** ... */` doc block when the doc happened
	// to contain that token. ToMDLStatement(true) must emit the `create or
	// modify` prefix at the statement line, not via post-hoc substitution.
	m := &EntityModel{
		Name:          QualifiedName{Module: "Sales", Name: "Customer"},
		Kind:          EntityPersistent,
		Documentation: "Use create when inserting a new customer.",
	}

	out := m.ToMDLStatement(true)

	lines := strings.Split(out, "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	assert.Equal(t, "/**", lines[0],
		"documentation block must come first; got:\n%s", out)
	assert.Contains(t, out, "create or modify persistent entity Sales.Customer",
		"prefix must be 'create or modify' at the statement line, not via string replace; got:\n%s", out)
	// Documentation body must be preserved verbatim — no accidental rewrite of
	// the word 'create' inside the doc.
	assert.Contains(t, out, "Use create when inserting a new customer.",
		"documentation body must not be rewritten; got:\n%s", out)
}

func TestToMDLStatement_DefaultIsCreate(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "Sales", Name: "Customer"},
		Kind: EntityPersistent,
	}
	// ToMDL() and ToMDLStatement(false) must produce identical output —
	// ToMDL() is the back-compat alias.
	assert.Equal(t, m.ToMDL(), m.ToMDLStatement(false))
	assert.Contains(t, m.ToMDLStatement(false), "create persistent entity")
	assert.NotContains(t, m.ToMDLStatement(false), "create or modify")
}

func TestToMDL_LiftRoundTrip(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name:           ast.QualifiedName{Module: "Sales", Name: "Customer"},
		Kind:           ast.EntityPersistent,
		Generalization: &ast.QualifiedName{Module: "System", Name: "Image"},
		Attributes: []ast.Attribute{
			{
				Name:    "Name",
				Type:    ast.DataType{Kind: ast.TypeString, Length: 200},
				NotNull: true,
			},
		},
		Documentation: "A customer.",
	}
	m, err := Lift(stmt)
	require.NoError(t, err)
	out := m.ToMDL()
	assert.Contains(t, out, "create persistent entity Sales.Customer extends System.Image")
	assert.Contains(t, out, "Name: String(200) not null")
	assert.Contains(t, out, "A customer.")
}

// --------------------------------------------------------------------------
// Hydrate
// --------------------------------------------------------------------------

func TestHydrate_BasicEntity(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Customer")
	e.SetDocumentation("A customer.")
	e.SetLocation("100 200")

	// Make it non-persistent via NoGeneralization with Persistable=false.
	gen := genDm.NewNoGeneralization()
	gen.SetPersistable(false)
	e.SetGeneralization(gen)

	attr := genDm.NewAttribute()
	attr.SetName("Name")
	st := genDm.NewStringAttributeType()
	st.SetLength(100)
	attr.SetType(st)
	e.AddAttributes(attr)

	m, warns, err := Hydrate("Sales", e)
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, "Sales", m.Name.Module)
	assert.Equal(t, "Customer", m.Name.Name)
	assert.Equal(t, EntityNonPersistent, m.Kind)
	assert.Equal(t, "A customer.", m.Documentation)
	require.NotNil(t, m.Position)
	assert.Equal(t, 100, m.Position.X)
	assert.Equal(t, 200, m.Position.Y)
	require.Len(t, m.Attributes, 1)
	assert.Equal(t, "Name", m.Attributes[0].Name)
	assert.Equal(t, canonical.KindString, m.Attributes[0].Type.Kind)
	assert.Equal(t, 100, m.Attributes[0].Type.Length)
}

func TestHydrate_NotNullConstraint(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Customer")

	attr := genDm.NewAttribute()
	attr.SetName("Email")
	attr.SetType(genDm.NewStringAttributeType())
	e.AddAttributes(attr)

	vr := genDm.NewValidationRule()
	vr.SetAttributeQualifiedName("Sales.Customer.Email")
	vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
	e.AddValidationRules(vr)

	m, warns, err := Hydrate("Sales", e)
	require.NoError(t, err)
	assert.Empty(t, warns)
	require.Len(t, m.Attributes, 1)
	assert.True(t, m.Attributes[0].NotNull, "expected Email to be NotNull from RequiredRuleInfo")
}

func TestHydrate_EnumAttribute(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Order")

	attr := genDm.NewAttribute()
	attr.SetName("Status")
	enumType := genDm.NewEnumerationAttributeType()
	enumType.SetEnumerationQualifiedName("App.OrderStatus")
	attr.SetType(enumType)
	e.AddAttributes(attr)

	m, warns, err := Hydrate("App", e)
	require.NoError(t, err)
	assert.Empty(t, warns)
	require.Len(t, m.Attributes, 1)
	assert.Equal(t, canonical.KindEnumRef, m.Attributes[0].Type.Kind)
	assert.Equal(t, "App.OrderStatus", m.Attributes[0].Type.Ref)
}

func TestHydrate_CalculatedAttribute(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("E")
	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(true)
	e.SetGeneralization(noGen)

	attr := genDm.NewAttribute()
	attr.SetName("TotalPrice")
	attr.SetType(genDm.NewDecimalAttributeType())
	cv := genDm.NewCalculatedValue()
	cv.SetMicroflowQualifiedName("MyModule.ComputePrice")
	attr.SetValue(cv)
	e.AddAttributes(attr)

	m, warns, err := Hydrate("MyModule", e)
	require.NoError(t, err)
	assert.Empty(t, warns)
	require.Len(t, m.Attributes, 1)
	assert.True(t, m.Attributes[0].Calculated)
	assert.False(t, m.Attributes[0].HasDefault)
	require.NotNil(t, m.Attributes[0].CalculatedMicroflow)
	assert.Equal(t, "MyModule.ComputePrice", m.Attributes[0].CalculatedMicroflow.String())
}

// --------------------------------------------------------------------------
// Location format regression tests (CE-bug: space format "X Y" rejected by
// Studio Pro 11.6.6; correct format is semicolon "X;Y").
// --------------------------------------------------------------------------

func TestParseLocation_SemicolonFormat(t *testing.T) {
	x, y, ok := parseLocation("280;100")
	require.True(t, ok)
	assert.Equal(t, 280, x)
	assert.Equal(t, 100, y)
}

func TestParseLocation_SpaceFormat_BackwardCompat(t *testing.T) {
	x, y, ok := parseLocation("1000 100")
	require.True(t, ok)
	assert.Equal(t, 1000, x)
	assert.Equal(t, 100, y)
}

func TestBuildGenEntity_LocationUsesSemicolon(t *testing.T) {
	m := &EntityModel{
		Name:     QualifiedName{Module: "Test", Name: "Ent"},
		Kind:     EntityPersistent,
		Position: &Position{X: 280, Y: 100},
	}
	e, err := buildGenEntity(m)
	require.NoError(t, err)
	assert.Equal(t, "280;100", e.Location(),
		"persist must use semicolon format accepted by Studio Pro 11.6.6+")
	assert.NotContains(t, e.Location(), " ",
		"space-separated format is rejected by Studio Pro 11.6.6+")
}

// --------------------------------------------------------------------------
// Event handlers
// --------------------------------------------------------------------------

func TestLift_WithEventHandler(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		EventHandlers: []ast.EventHandlerDef{
			{
				Moment:            "Before",
				Event:             "Commit",
				Microflow:         ast.QualifiedName{Module: "M", Name: "BeforeCommit"},
				RaiseErrorOnFalse: true,
				PassEventObject:   true,
			},
		},
	}
	m, err := Lift(stmt)
	require.NoError(t, err)
	require.Len(t, m.EventHandlers, 1)
	eh := m.EventHandlers[0]
	assert.Equal(t, "Before", eh.Moment)
	assert.Equal(t, "Commit", eh.Event)
	assert.Equal(t, "M.BeforeCommit", eh.Microflow.String())
	assert.True(t, eh.RaiseErrorOnFalse)
	assert.True(t, eh.PassEventObject)
}

func TestToMDL_WithEventHandler(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "M", Name: "E"},
		Kind: EntityPersistent,
		EventHandlers: []EventHandlerModel{
			{Moment: "Before", Event: "Commit", Microflow: QualifiedName{Module: "M", Name: "ACT_BeforeCommit"}, RaiseErrorOnFalse: true, PassEventObject: true},
		},
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "on before commit call M.ACT_BeforeCommit($currentObject) raise error")
}

func TestToMDL_EventHandler_AfterCreate_NoPassObject(t *testing.T) {
	m := &EntityModel{
		Name: QualifiedName{Module: "M", Name: "E"},
		Kind: EntityPersistent,
		EventHandlers: []EventHandlerModel{
			{Moment: "After", Event: "Create", Microflow: QualifiedName{Module: "M", Name: "ACT_After"}, PassEventObject: false},
		},
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "on after create call M.ACT_After()")
	assert.NotContains(t, got, "$currentObject")
	assert.NotContains(t, got, "raise error")
}
