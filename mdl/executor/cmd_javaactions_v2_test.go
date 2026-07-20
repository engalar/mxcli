// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.2.A1-A5 tests: gen-typed read paths for Java/JavaScript actions.
// Stage 3.3.2.D1-D2 tests: AST→gen converters and execCreateJavaActionGen.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCA "github.com/mendixlabs/mxcli/modelsdk/gen/codeactions"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// ─────────────────────────────────────────────────────────────────────
// A1 — listJavaActionsGen
// ─────────────────────────────────────────────────────────────────────

func TestListJavaActionsGen_OutputsHeaderAndSummary(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	ctx.Deps = ctx.buildDeps()

	if err := listJavaActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Qualified Name") {
		t.Errorf("expected header 'Qualified Name' in output, got: %q", out)
	}
	if !strings.Contains(out, "java actions)") {
		t.Errorf("expected count summary in output, got: %q", out)
	}
}

func TestListJavaActionsGen_RowsPresent(t *testing.T) {
	// Fixture has 2 java actions (ValidateEmail, XSS_Sanitizer).
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	ctx.Deps = ctx.buildDeps()

	if err := listJavaActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ValidateEmail") {
		t.Errorf("expected ValidateEmail in output: %q", out)
	}
	if !strings.Contains(out, "XSS_Sanitizer") {
		t.Errorf("expected XSS_Sanitizer in output: %q", out)
	}
	if !strings.Contains(out, "(2 java actions)") {
		t.Errorf("expected '(2 java actions)' summary, got: %q", out)
	}
}

func TestListJavaActionsGen_FilterByModule(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	ctx.Deps = ctx.buildDeps()

	if err := listJavaActionsGen(ctx, "NoSuchModule"); err != nil {
		t.Fatalf("listJavaActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(0 java actions)") {
		t.Errorf("expected '(0 java actions)' for unknown module, got: %q", out)
	}
}

// ─────────────────────────────────────────────────────────────────────
// A2 — formatJavaActionTypeGen + formatJavaActionReturnTypeGen
// ─────────────────────────────────────────────────────────────────────

func TestFormatJavaActionReturnTypeGen_NilReturnsVoid(t *testing.T) {
	if got := formatJavaActionReturnTypeGen(nil, nil); got != "Void" {
		t.Errorf("got %q, want Void", got)
	}
}

func TestFormatJavaActionTypeGen_NilReturnsObject(t *testing.T) {
	if got := formatJavaActionTypeGen(nil, nil); got != "Object" {
		t.Errorf("got %q, want Object", got)
	}
}

// TestFormatJavaActionTypeGen_LegacyStorageNames verifies that the
// formatter dispatches on the Studio-Pro-emitted "CodeActions$X"
// storage names (the gen registry uses "JavaActions$X" but the BSON
// from real fixtures uses the legacy namespace — see the schema-gap
// note at the head of cmd_javaactions_gen.go).
//
// Driven through real fixture data so we exercise the full
// decode→format chain that listJavaActionsGen / describeJavaActionGen
// run in production.
func TestFormatJavaActionTypeGen_FixtureReturnsBoolean(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	for _, p := range pairs {
		if p.Elem == nil || p.Elem.Name() != "ValidateEmail" {
			continue
		}
		rt := javaActionReturnTypeElement(p.Elem)
		got := formatJavaActionReturnTypeGen(rt, p.Elem.ActionTypeParametersItems())
		if got != "Boolean" {
			t.Errorf("ValidateEmail return type: got %q, want Boolean", got)
		}
		return
	}
	t.Fatal("ValidateEmail not in fixture")
}

func TestFormatJavaActionTypeGen_FixtureReturnsString(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	for _, p := range pairs {
		if p.Elem == nil || p.Elem.Name() != "XSS_Sanitizer" {
			continue
		}
		rt := javaActionReturnTypeElement(p.Elem)
		got := formatJavaActionReturnTypeGen(rt, p.Elem.ActionTypeParametersItems())
		if got != "String" {
			t.Errorf("XSS_Sanitizer return type: got %q, want String", got)
		}
		return
	}
	t.Fatal("XSS_Sanitizer not in fixture")
}

// ─────────────────────────────────────────────────────────────────────
// A3 — describeJavaActionGen
// ─────────────────────────────────────────────────────────────────────

func TestDescribeJavaActionGen_OutputsCreateStatement(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf

	// Fixture module name + action name; module discovered from
	// listJavaActionsWithContainerGen + container chain.
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaActionsWithContainerGen: %v", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		t.Fatalf("getHierarchy: %v", err)
	}
	var modName string
	for _, p := range pairs {
		if p.Elem != nil && p.Elem.Name() == "ValidateEmail" {
			modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
			modName = h.GetModuleName(modID)
			break
		}
	}
	if modName == "" {
		t.Fatal("could not resolve module for ValidateEmail")
	}

	if err := describeJavaActionGen(ctx, ast.QualifiedName{Module: modName, Name: "ValidateEmail"}); err != nil {
		t.Fatalf("describeJavaActionGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "create or modify java action ") {
		t.Errorf("missing create or modify statement: %q", out)
	}
	if !strings.Contains(out, ".ValidateEmail(") {
		t.Errorf("missing action name + paren: %q", out)
	}
	if !strings.Contains(out, "EmailAddress") {
		t.Errorf("missing parameter name: %q", out)
	}
	if !strings.Contains(out, "returns Boolean") {
		t.Errorf("missing 'returns Boolean': %q", out)
	}
}

func TestDescribeJavaActionGen_NotFound(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeJavaActionGen(ctx, ast.QualifiedName{Module: "Foo", Name: "Bar"})
	if err == nil {
		t.Fatal("expected error for missing action, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────
// A4 — listJavaScriptActionsGen
// ─────────────────────────────────────────────────────────────────────

func TestListJavaScriptActionsGen_OutputsPlatformColumn(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	ctx.Deps = ctx.buildDeps()

	if err := listJavaScriptActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaScriptActionsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Platform") {
		t.Errorf("expected 'Platform' header, got: %q", out)
	}
	if !strings.Contains(out, "javascript actions)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListJavaScriptActionsGen_RendersAllPlatformAsLabel(t *testing.T) {
	// Fixture has Platform="" actions which should render as "All".
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	ctx.Deps = ctx.buildDeps()

	if err := listJavaScriptActionsGen(ctx, ""); err != nil {
		t.Fatalf("listJavaScriptActionsGen: %v", err)
	}
	out := buf.String()
	// Fixture has 67 actions, mostly "All" but some "Web".
	if !strings.Contains(out, "Web") {
		t.Errorf("expected 'Web' platform in output: %q", out[:200])
	}
}

// ─────────────────────────────────────────────────────────────────────
// A5 — describeJavaScriptActionGen
// ─────────────────────────────────────────────────────────────────────

func TestDescribeJavaScriptActionGen_OutputsCreateStatement(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Deps = ctx.buildDeps()

	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listJavaScriptActionsWithContainerGen: %v", err)
	}
	if len(pairs) == 0 {
		t.Fatal("fixture has no JS actions")
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		t.Fatalf("getHierarchy: %v", err)
	}
	first := pairs[0].Elem
	modID := h.FindModuleID(modelIDFromElementID(pairs[0].ContainerID))
	modName := h.GetModuleName(modID)
	if modName == "" {
		t.Fatal("could not resolve module for first JS action")
	}

	if err := describeJavaScriptActionGen(ctx, ast.QualifiedName{Module: modName, Name: first.Name()}); err != nil {
		t.Fatalf("describeJavaScriptActionGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "create or modify javascript action ") {
		t.Errorf("missing create statement: %q", out)
	}
	if !strings.Contains(out, "."+first.Name()+"(") {
		t.Errorf("missing action name + paren for %s: %q", first.Name(), out)
	}
}

func TestDescribeJavaScriptActionGen_NotFound(t *testing.T) {
	ctx := newJavaActionsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeJavaScriptActionGen(ctx, ast.QualifiedName{Module: "Foo", Name: "Bar"})
	if err == nil {
		t.Fatal("expected error for missing JS action, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────
// D1 — astDataTypeToJavaActionParamTypeGen + ReturnTypeGen
// ─────────────────────────────────────────────────────────────────────

func TestAstDataTypeToJavaActionParamTypeGen_PrimitiveTypes(t *testing.T) {
	cases := []struct {
		name     string
		dt       ast.DataType
		wantType string
	}{
		{"Boolean", ast.DataType{Kind: ast.TypeBoolean}, "CodeActions$BooleanType"},
		{"Integer", ast.DataType{Kind: ast.TypeInteger}, "CodeActions$IntegerType"},
		{"Decimal", ast.DataType{Kind: ast.TypeDecimal}, "CodeActions$DecimalType"},
		{"String", ast.DataType{Kind: ast.TypeString}, "CodeActions$StringType"},
		{"DateTime", ast.DataType{Kind: ast.TypeDateTime}, "CodeActions$DateTimeType"},
		{"Date", ast.DataType{Kind: ast.TypeDate}, "CodeActions$DateTimeType"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			elem := astDataTypeToJavaActionParamTypeGen(c.dt, nil)
			if elem == nil {
				t.Fatal("got nil element")
			}
			if elem.TypeName() != c.wantType {
				t.Errorf("got %q, want %q", elem.TypeName(), c.wantType)
			}
		})
	}
}

func TestAstDataTypeToJavaActionParamTypeGen_ConcreteEntity(t *testing.T) {
	dt := ast.DataType{
		Kind:      ast.TypeEntity,
		EntityRef: &ast.QualifiedName{Module: "Sales", Name: "Order"},
	}
	elem := astDataTypeToJavaActionParamTypeGen(dt, nil)
	et, ok := elem.(*genCA.ConcreteEntityType)
	if !ok {
		t.Fatalf("got %T, want *genCA.ConcreteEntityType", elem)
	}
	if et.EntityQualifiedName() != "Sales.Order" {
		t.Errorf("got %q, want Sales.Order", et.EntityQualifiedName())
	}
}

func TestAstDataTypeToJavaActionParamTypeGen_EntityList(t *testing.T) {
	dt := ast.DataType{
		Kind:      ast.TypeListOf,
		EntityRef: &ast.QualifiedName{Module: "Sales", Name: "Order"},
	}
	elem := astDataTypeToJavaActionParamTypeGen(dt, nil)
	lt, ok := elem.(*genCA.ListType)
	if !ok {
		t.Fatalf("got %T, want *genCA.ListType", elem)
	}
	inner, ok := lt.Parameter().(*genCA.ConcreteEntityType)
	if !ok {
		t.Fatalf("inner = %T, want *genCA.ConcreteEntityType", lt.Parameter())
	}
	if inner.EntityQualifiedName() != "Sales.Order" {
		t.Errorf("inner entity = %q, want Sales.Order", inner.EntityQualifiedName())
	}
}

func TestAstDataTypeToJavaActionParamTypeGen_TypeParamRef(t *testing.T) {
	typeParamIDs := map[string]element.ID{"pEntity": element.ID("tp-uuid-123")}
	dt := ast.DataType{Kind: ast.TypeEntityTypeParam, TypeParamName: "pEntity"}
	elem := astDataTypeToJavaActionParamTypeGen(dt, typeParamIDs)
	etp, ok := elem.(*genCA.EntityTypeParameterType)
	if !ok {
		t.Fatalf("got %T, want *genCA.EntityTypeParameterType", elem)
	}
	if etp.TypeParameterRefID() != element.ID("tp-uuid-123") {
		t.Errorf("got %q, want tp-uuid-123", etp.TypeParameterRefID())
	}
}

// TestAstDataTypeToJavaActionParamTypeGen_BareEnumRefAsTypeParam verifies that
// a bare unqualified TypeEnumeration whose name matches a declared type parameter
// is emitted as ParameterizedEntityType (not ConcreteEntityType).
// This covers the pattern: InputObject: T  where T is a type parameter.
func TestAstDataTypeToJavaActionParamTypeGen_BareEnumRefAsTypeParam(t *testing.T) {
	typeParamIDs := map[string]element.ID{"T": element.ID("tp-T-uuid")}
	dt := ast.DataType{
		Kind:    ast.TypeEnumeration,
		EnumRef: &ast.QualifiedName{Module: "", Name: "T"},
	}
	elem := astDataTypeToJavaActionParamTypeGen(dt, typeParamIDs)
	pe, ok := elem.(*genCA.ParameterizedEntityType)
	if !ok {
		t.Fatalf("got %T, want *genCA.ParameterizedEntityType", elem)
	}
	if pe.TypeParameterRefID() != element.ID("tp-T-uuid") {
		t.Errorf("TypeParameterRefID = %q, want tp-T-uuid", pe.TypeParameterRefID())
	}
}

// TestAstDataTypeToJavaActionParamTypeGen_QualifiedEnumRefNotTypeParam verifies
// that a qualified TypeEnumeration (Module.Name) is NOT treated as a type param
// even if the Name segment happens to match a type param key.
func TestAstDataTypeToJavaActionParamTypeGen_QualifiedEnumRefNotTypeParam(t *testing.T) {
	typeParamIDs := map[string]element.ID{"Customer": element.ID("tp-uuid")}
	dt := ast.DataType{
		Kind:    ast.TypeEnumeration,
		EnumRef: &ast.QualifiedName{Module: "MyModule", Name: "Customer"},
	}
	elem := astDataTypeToJavaActionParamTypeGen(dt, typeParamIDs)
	if _, ok := elem.(*genCA.ParameterizedEntityType); ok {
		t.Fatal("got ParameterizedEntityType for qualified enum ref, want ConcreteEntityType")
	}
	et, ok := elem.(*genCA.ConcreteEntityType)
	if !ok {
		t.Fatalf("got %T, want *genCA.ConcreteEntityType", elem)
	}
	if et.EntityQualifiedName() != "MyModule.Customer" {
		t.Errorf("EntityQualifiedName = %q, want MyModule.Customer", et.EntityQualifiedName())
	}
}

func TestAstDataTypeToJavaActionReturnTypeGen_VoidAndPrimitives(t *testing.T) {
	cases := []struct {
		name     string
		dt       ast.DataType
		wantType string
	}{
		{"Void", ast.DataType{Kind: ast.TypeVoid}, "CodeActions$VoidType"},
		{"Boolean", ast.DataType{Kind: ast.TypeBoolean}, "CodeActions$BooleanType"},
		{"String", ast.DataType{Kind: ast.TypeString}, "CodeActions$StringType"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			elem := astDataTypeToJavaActionReturnTypeGen(c.dt, nil)
			if elem == nil {
				t.Fatal("got nil element")
			}
			if elem.TypeName() != c.wantType {
				t.Errorf("got %q, want %q", elem.TypeName(), c.wantType)
			}
		})
	}
}

func TestAstDataTypeToJavaActionReturnTypeGen_ConcreteEntity(t *testing.T) {
	dt := ast.DataType{
		Kind:      ast.TypeEntity,
		EntityRef: &ast.QualifiedName{Module: "M", Name: "Customer"},
	}
	elem := astDataTypeToJavaActionReturnTypeGen(dt, nil)
	et, ok := elem.(*genCA.ConcreteEntityType)
	if !ok {
		t.Fatalf("got %T, want *genCA.ConcreteEntityType", elem)
	}
	if et.EntityQualifiedName() != "M.Customer" {
		t.Errorf("got %q, want M.Customer", et.EntityQualifiedName())
	}
}

// ─────────────────────────────────────────────────────────────────────
// D2 — execCreateJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// newCreateJavaActionMockCtx wires a writable mock backend with a single
// module + an empty JavaActions list, capturing CreateJavaActionGen calls
// in the returned slice for assertion. CreateJavaAction routes to the
// gen path through ctx.Backend (per plan §7 D2 — no direct repo call).
func newCreateJavaActionMockCtx(t *testing.T, moduleName string) (*ExecContext, *bytes.Buffer, *[]*genJA.JavaAction, *[]*genJA.JavaAction) {
	t.Helper()
	mod := mkModule(moduleName)
	h := mkHierarchy(mod)
	created := []*genJA.JavaAction{}
	updated := []*genJA.JavaAction{}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		CreateJavaActionGenFunc: func(parentUUID, containmentName string, ja *genJA.JavaAction) error {
			created = append(created, ja)
			return nil
		},
		UpdateJavaActionGenFunc: func(ja *genJA.JavaAction) error {
			updated = append(updated, ja)
			return nil
		},
		WriteJavaSourceFileGenFunc: func(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error {
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	// JavaActions repo: empty by default (no existing actions).
	ctx.JavaActions = &emptyJavaActionRepo{}
	ctx.Deps = ctx.buildDeps()
	return ctx, buf, &created, &updated
}

// emptyJavaActionRepo is a minimal repo for tests that only need the
// existence-check path to find no matches.
type emptyJavaActionRepo struct {
	items []*genJA.JavaAction
}

func (r *emptyJavaActionRepo) Get(id model.ID) (*genJA.JavaAction, error)          { return nil, nil }
func (r *emptyJavaActionRepo) List(moduleID model.ID) ([]*genJA.JavaAction, error) { return nil, nil }
func (r *emptyJavaActionRepo) ListAll() ([]*genJA.JavaAction, error)               { return r.items, nil }
func (r *emptyJavaActionRepo) FindByQualifiedName(qn string) (*genJA.JavaAction, error) {
	return nil, nil
}
func (r *emptyJavaActionRepo) GetContainerUUID(id model.ID) (model.ID, error) { return "", nil }
func (r *emptyJavaActionRepo) Create(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	return nil
}
func (r *emptyJavaActionRepo) Update(ja *genJA.JavaAction) error { return nil }
func (r *emptyJavaActionRepo) Delete(id model.ID) error          { return nil }

func TestExecCreateJavaActionGen_BasicCreate(t *testing.T) {
	ctx, buf, created, _ := newCreateJavaActionMockCtx(t, "TestModule")
	stmt := &ast.CreateJavaActionStmt{
		Name:          ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
		Documentation: "test docs",
		ReturnType:    ast.DataType{Kind: ast.TypeBoolean},
	}
	if err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execCreateJavaActionGen: %v", err)
	}
	if len(*created) != 1 {
		t.Fatalf("expected 1 created action, got %d", len(*created))
	}
	ja := (*created)[0]
	if ja.Name() != "MyAction" {
		t.Errorf("Name = %q, want MyAction", ja.Name())
	}
	if ja.Documentation() != "test docs" {
		t.Errorf("Documentation = %q, want 'test docs'", ja.Documentation())
	}
	if ja.ExportLevel() != "Public" {
		t.Errorf("ExportLevel = %q, want Public", ja.ExportLevel())
	}
	out := buf.String()
	if !strings.Contains(out, "Created java action: TestModule.MyAction") {
		t.Errorf("expected success message, got: %q", out)
	}
}

func TestExecCreateJavaActionGen_WithParameters(t *testing.T) {
	ctx, _, created, _ := newCreateJavaActionMockCtx(t, "TestModule")
	stmt := &ast.CreateJavaActionStmt{
		Name: ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
		Parameters: []ast.JavaActionParam{
			{Name: "p1", Type: ast.DataType{Kind: ast.TypeString}, IsRequired: true},
			{Name: "p2", Type: ast.DataType{Kind: ast.TypeInteger}, IsRequired: false},
		},
		ReturnType: ast.DataType{Kind: ast.TypeBoolean},
	}
	if err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execCreateJavaActionGen: %v", err)
	}
	if len(*created) != 1 {
		t.Fatalf("expected 1 created action, got %d", len(*created))
	}
	ja := (*created)[0]
	// Executor writes to Parameters (old API, version marker 2), not ActionParameters.
	params := ja.ParametersItems()
	if len(params) != 2 {
		t.Fatalf("Parameters count = %d, want 2", len(params))
	}
	p1, ok := params[0].(*genJA.JavaActionParameter)
	if !ok {
		t.Fatalf("params[0] = %T, want *JavaActionParameter", params[0])
	}
	if p1.Name() != "p1" {
		t.Errorf("p1.Name = %q, want p1", p1.Name())
	}
	if !p1.IsRequired() {
		t.Errorf("p1.IsRequired = false, want true")
	}
	// ParameterType wraps the actual type in CodeActions$BasicParameterType.
	pt1 := p1.ParameterType()
	if pt1 == nil || pt1.TypeName() != "CodeActions$BasicParameterType" {
		t.Errorf("p1.ParameterType = %v, want CodeActions$BasicParameterType", pt1)
	}
	bpt1, ok := pt1.(*genCA.BasicParameterType)
	if !ok {
		t.Fatalf("p1.ParameterType = %T, want *genCA.BasicParameterType", pt1)
	}
	if inner := bpt1.Type(); inner == nil || inner.TypeName() != "CodeActions$StringType" {
		t.Errorf("p1 inner type = %v, want CodeActions$StringType", bpt1.Type())
	}
	p2 := params[1].(*genJA.JavaActionParameter)
	if p2.Name() != "p2" {
		t.Errorf("p2.Name = %q, want p2", p2.Name())
	}
	if p2.IsRequired() {
		t.Errorf("p2.IsRequired = true, want false")
	}
}

func TestExecCreateJavaActionGen_ConcreteEntityReturnType(t *testing.T) {
	ctx, _, created, _ := newCreateJavaActionMockCtx(t, "TestModule")
	stmt := &ast.CreateJavaActionStmt{
		Name: ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
		ReturnType: ast.DataType{
			Kind:      ast.TypeEntity,
			EntityRef: &ast.QualifiedName{Module: "Sales", Name: "Order"},
		},
	}
	if err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execCreateJavaActionGen: %v", err)
	}
	if len(*created) != 1 {
		t.Fatalf("expected 1 created action, got %d", len(*created))
	}
	ja := (*created)[0]
	// Executor writes JavaReturnType (old API), not ActionReturnType.
	rt := ja.JavaReturnType()
	if rt == nil {
		t.Fatal("JavaReturnType is nil")
	}
	et, ok := rt.(*genCA.ConcreteEntityType)
	if !ok {
		t.Fatalf("JavaReturnType = %T, want *genCA.ConcreteEntityType", rt)
	}
	if et.EntityQualifiedName() != "Sales.Order" {
		t.Errorf("entity = %q, want Sales.Order", et.EntityQualifiedName())
	}
}

func TestExecCreateJavaActionGen_AlreadyExists(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)
	// Pre-populate JavaActions repo with one item that the existence
	// check will resolve to TestModule.MyAction. The cache-helper path
	// reads container UUIDs via GetContainerUUID; we return the module
	// ID so the hierarchy resolves the module name.
	existing := genJA.NewJavaAction()
	existing.SetID(element.ID(nextID("ja")))
	existing.SetName("MyAction")
	repo := &existingJavaActionRepo{
		items:       []*genJA.JavaAction{existing},
		containerOf: map[model.ID]model.ID{model.ID(existing.ID()): mod.ID},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.JavaActions = repo
	ctx.Deps = ctx.buildDeps()

	stmt := &ast.CreateJavaActionStmt{
		Name:           ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
		ReturnType:     ast.DataType{Kind: ast.TypeBoolean},
		CreateOrModify: false,
	}
	err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps)
	if err == nil {
		t.Fatal("expected error when action already exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "MyAction") {
		t.Errorf("expected 'already exists' or action name in error, got: %v", err)
	}
}

func TestExecCreateJavaActionGen_ModuleNotFound(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy()))
	ctx.JavaActions = &emptyJavaActionRepo{}
	ctx.Deps = ctx.buildDeps()

	stmt := &ast.CreateJavaActionStmt{
		Name:       ast.QualifiedName{Module: "NoSuchModule", Name: "MyAction"},
		ReturnType: ast.DataType{Kind: ast.TypeBoolean},
	}
	err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps)
	if err == nil {
		t.Fatal("expected error when module not found, got nil")
	}
}

func TestExecCreateJavaActionGen_OrModifyOverwrites(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)
	existing := genJA.NewJavaAction()
	existingID := element.ID(nextID("ja"))
	existing.SetID(existingID)
	existing.SetName("MyAction")
	repo := &existingJavaActionRepo{
		items:       []*genJA.JavaAction{existing},
		containerOf: map[model.ID]model.ID{model.ID(existingID): mod.ID},
	}
	created := []*genJA.JavaAction{}
	updated := []*genJA.JavaAction{}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		CreateJavaActionGenFunc: func(parentUUID, containmentName string, ja *genJA.JavaAction) error {
			created = append(created, ja)
			return nil
		},
		UpdateJavaActionGenFunc: func(ja *genJA.JavaAction) error {
			updated = append(updated, ja)
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.JavaActions = repo
	ctx.Deps = ctx.buildDeps()

	stmt := &ast.CreateJavaActionStmt{
		Name:           ast.QualifiedName{Module: "TestModule", Name: "MyAction"},
		Documentation:  "updated docs",
		ReturnType:     ast.DataType{Kind: ast.TypeBoolean},
		CreateOrModify: true,
	}
	if err := execCreateJavaActionGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execCreateJavaActionGen: %v", err)
	}
	if len(created) != 0 {
		t.Errorf("expected 0 created (existing path → update), got %d", len(created))
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 updated action, got %d", len(updated))
	}
	if updated[0].ID() != existingID {
		t.Errorf("updated action ID = %q, want %q (must reuse existing)", updated[0].ID(), existingID)
	}
	if updated[0].Documentation() != "updated docs" {
		t.Errorf("updated Documentation = %q, want 'updated docs'", updated[0].Documentation())
	}
	out := buf.String()
	if !strings.Contains(out, "Modified java action") {
		t.Errorf("expected 'Modified java action' message, got: %q", out)
	}
}

// existingJavaActionRepo is a minimal repo that returns a pre-populated
// list of JavaActions and resolves their container via a name-keyed map.
type existingJavaActionRepo struct {
	items       []*genJA.JavaAction
	containerOf map[model.ID]model.ID
}

func (r *existingJavaActionRepo) Get(id model.ID) (*genJA.JavaAction, error) {
	for _, ja := range r.items {
		if model.ID(ja.ID()) == id {
			return ja, nil
		}
	}
	return nil, nil
}
func (r *existingJavaActionRepo) List(moduleID model.ID) ([]*genJA.JavaAction, error) {
	return nil, nil
}
func (r *existingJavaActionRepo) ListAll() ([]*genJA.JavaAction, error) { return r.items, nil }
func (r *existingJavaActionRepo) FindByQualifiedName(qn string) (*genJA.JavaAction, error) {
	return nil, nil
}
func (r *existingJavaActionRepo) GetContainerUUID(id model.ID) (model.ID, error) {
	if c, ok := r.containerOf[id]; ok {
		return c, nil
	}
	return "", nil
}
func (r *existingJavaActionRepo) Create(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	return nil
}
func (r *existingJavaActionRepo) Update(ja *genJA.JavaAction) error { return nil }
func (r *existingJavaActionRepo) Delete(id model.ID) error          { return nil }
