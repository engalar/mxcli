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

// repoRootForTest resolves the repo root from this file's location so the
// test works regardless of `go test` cwd.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// thisFile = .../mdl/model/entity/roundtrip_test.go → up 3 → repo root.
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func gitRestoreRoundTrip(t *testing.T) {
	t.Helper()
	root := repoRootForTest(t)
	cmd := exec.Command("git", "restore", "testdata/expr-checker/")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("git restore testdata failed: %v\n%s", err, out)
	}
}

// TestRoundTrip_Entity proves the canonical model is lossless across
// Persist → Hydrate → ToMDL: building an EntityModel, writing it via the
// canonical pipeline, reading it back from the gen Entity, and rendering
// it again must produce byte-identical MDL.
func TestRoundTrip_Entity(t *testing.T) {
	t.Cleanup(func() { gitRestoreRoundTrip(t) })

	root := repoRootForTest(t)
	mprPath := filepath.Join(root, "testdata", "expr-checker", "minimal.mpr")

	b := mprbackend.New()
	require.NoError(t, b.Connect(mprPath))
	t.Cleanup(func() { _ = b.Disconnect() })

	mod, err := b.GetModuleByName("MyFirstModule")
	require.NoError(t, err)
	require.NotNil(t, mod)

	dmGen, err := b.GetDomainModelGen(mod.ID)
	require.NoError(t, err)
	require.NotNil(t, dmGen)
	dmID := modelID.ID(dmGen.ID())
	require.NotEmpty(t, dmID)

	cases := []struct {
		name  string
		model *entity.EntityModel
	}{
		{
			name: "persistent with constraints",
			model: &entity.EntityModel{
				Name: entity.QualifiedName{Module: "MyFirstModule", Name: "RTTicket"},
				Kind: entity.EntityPersistent,
				Attributes: []entity.AttributeModel{
					{
						Name:         "Title",
						Type:         model.DataType{Kind: model.KindString, Length: 500},
						NotNull:      true,
						NotNullError: "Title required",
					},
					{
						Name:         "Priority",
						Type:         model.DataType{Kind: model.KindInteger},
						HasDefault:   true,
						DefaultValue: "1",
					},
				},
			},
		},
		{
			name: "non-persistent",
			model: &entity.EntityModel{
				Name: entity.QualifiedName{Module: "MyFirstModule", Name: "RTHelper"},
				Kind: entity.EntityNonPersistent,
				Attributes: []entity.AttributeModel{
					{
						Name: "Value",
						Type: model.DataType{Kind: model.KindDecimal},
					},
				},
			},
		},
	}

	ctx := model.PersistContext{DomainModelID: dmID, Backend: b}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			originalMDL := tc.model.ToMDL()
			require.NoError(t, tc.model.Persist(ctx))

			// Read back through the same backend handle. Locate by name —
			// the backend assigns a UUID on create so we cannot key by ID.
			dmAfter, err := b.GetDomainModelByIDGen(dmID)
			require.NoError(t, err)
			require.NotNil(t, dmAfter)

			var found *genDm.Entity
			for _, item := range dmAfter.EntitiesItems() {
				if e, ok := item.(*genDm.Entity); ok && e.Name() == tc.model.Name.Name {
					found = e
					break
				}
			}
			require.NotNil(t, found, "%s not found after Persist", tc.model.Name.Name)

			readBack, warns, err := entity.Hydrate(tc.model.Name.Module, found)
			require.NoError(t, err)
			assert.Empty(t, warns, "unexpected hydrate warnings: %+v", warns)

			assert.Equal(t, originalMDL, readBack.ToMDL(),
				"round-trip MDL must be stable for %s", tc.name)
		})
	}
}
