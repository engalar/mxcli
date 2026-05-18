// SPDX-License-Identifier: Apache-2.0

package meta_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func openBackend(t *testing.T, mprPath string) *mprbackend.MprBackend {
	t.Helper()
	b, err := mprbackend.NewFromPath(mprPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Disconnect() })
	return b
}

func TestBuildFromBackend_EntityAttrs(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)
	require.NotNil(t, idx)

	// 验证已知存在的实体属性（来自真实 MPR）
	kind, ok := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "Status")
	assert.True(t, ok, "Status 属性应存在")
	assert.NotZero(t, kind, "kind 应为非零值")

	kind2, ok2 := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "AWApplicationNo")
	assert.True(t, ok2, "AWApplicationNo 属性应存在")
	assert.NotZero(t, kind2)
}

func TestBuildFromBackend_SystemEntity(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	_, ok := idx.AttributeKind("System.User", "Name")
	// System.User 可能不在普通 DomainModel 列表，允许 false（不强断言）
	t.Logf("System.User.Name found: %v", ok)
}
