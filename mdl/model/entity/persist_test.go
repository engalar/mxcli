//go:build integration

// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
	modelID "github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// testdataMPR returns the absolute path of testdata/expr-checker/minimal.mpr
// from this file's location, so the test works regardless of `go test` cwd.
func testdataMPR(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not resolve test file location")
	// thisFile = .../mdl/model/entity/persist_test.go
	// repo root = ../../../  from the file's directory (entity → model → mdl → repo).
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "testdata", "expr-checker", "minimal.mpr")
}

// gitRestoreTestdata reverts in-place mutations made by the test.
func gitRestoreTestdata(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	cmd := exec.Command("git", "restore", "testdata/expr-checker/")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git restore testdata failed: %v\n%s", err, out)
	}
}

// TestPersist_CreateEntity opens the bundled MyFirstModule project, writes a
// new entity through the canonical model -> Persist path, and reads it back
// via the same backend interface. All in-place writes are reverted on
// completion via `git restore`.
func TestPersist_CreateEntity(t *testing.T) {
	t.Cleanup(func() { gitRestoreTestdata(t) })

	b := mprbackend.New()
	require.NoError(t, b.Connect(testdataMPR(t)))
	t.Cleanup(func() { _ = b.Disconnect() })

	mod, err := b.GetModuleByName("MyFirstModule")
	require.NoError(t, err)
	require.NotNil(t, mod)
	// model.Module.DomainModelID is not populated by the modelsdk read path;
	// fetch the DM by module ID and use its element ID instead.
	dmGen, err := b.GetDomainModelGen(mod.ID)
	require.NoError(t, err)
	require.NotNil(t, dmGen, "MyFirstModule has no DomainModel unit")
	dmID := modelID.ID(dmGen.ID())
	require.NotEmpty(t, dmID, "DomainModel element ID is empty")

	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "MyFirstModule", Name: "PersistPocCustomer"},
		Kind: entity.EntityPersistent,
		Attributes: []entity.AttributeModel{
			{
				Name:    "Name",
				Type:    model.DataType{Kind: model.KindString, Length: 100},
				NotNull: true,
			},
			{
				Name: "Active",
				Type: model.DataType{Kind: model.KindBoolean},
			},
		},
	}

	require.NoError(t, m.Persist(model.PersistContext{
		DomainModelID: dmID,
		Backend:       b,
	}))

	// Read back via the same backend handle (in-process mutations are visible).
	dm, err := b.GetDomainModelByIDGen(dmID)
	require.NoError(t, err)
	require.NotNil(t, dm)

	var found *genDm.Entity
	for _, item := range dm.EntitiesItems() {
		if e, ok := item.(*genDm.Entity); ok && e.Name() == "PersistPocCustomer" {
			found = e
			break
		}
	}
	require.NotNil(t, found, "PersistPocCustomer not found after Persist")
	assert.Equal(t, "PersistPocCustomer", found.Name())
	require.Len(t, found.AttributesItems(), 2)

	// First attribute: Name as String(100).
	a0, ok := found.AttributesItems()[0].(*genDm.Attribute)
	require.True(t, ok)
	assert.Equal(t, "Name", a0.Name())
	st, ok := a0.Type().(*genDm.StringAttributeType)
	require.True(t, ok, "Name should be StringAttributeType, got %T", a0.Type())
	assert.Equal(t, int32(100), st.Length())

	// Second attribute: Active as Boolean.
	a1, ok := found.AttributesItems()[1].(*genDm.Attribute)
	require.True(t, ok)
	assert.Equal(t, "Active", a1.Name())
	_, ok = a1.Type().(*genDm.BooleanAttributeType)
	assert.True(t, ok, "Active should be BooleanAttributeType, got %T", a1.Type())

	// ValidationRule for NotNull was emitted.
	hasRequired := false
	for _, item := range found.ValidationRulesItems() {
		vr, ok := item.(*genDm.ValidationRule)
		if !ok {
			continue
		}
		if vr.AttributeQualifiedName() == "MyFirstModule.PersistPocCustomer.Name" {
			if ri := vr.RuleInfo(); ri != nil && ri.TypeName() == "DomainModels$RequiredRuleInfo" {
				hasRequired = true
				break
			}
		}
	}
	assert.True(t, hasRequired, "expected RequiredRuleInfo for Name attribute")
}
