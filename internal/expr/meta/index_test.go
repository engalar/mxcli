// SPDX-License-Identifier: Apache-2.0

package meta_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
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

func TestBuildFromBackend_EnumValues(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	assert.Greater(t, idx.EnumCount(), 0, "应该索引到至少一个枚举")

	// ENUM_MailStatus 是已知枚举 (见诊断输出)
	vals, ok := idx.EnumCases("MailInquiry.ENUM_MailStatus")
	assert.True(t, ok, "MailInquiry.ENUM_MailStatus 应存在")
	assert.NotEmpty(t, vals, "枚举值列表不应为空")
}

func TestBuildFromBackend_Constants(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	assert.Greater(t, idx.ConstantsCount(), 0, "应有常量记录")
	// FeedbackModule.LocalStorageKey 是已知常量 (见诊断输出)
	assert.True(t, idx.HasConstant("@FeedbackModule.LocalStorageKey"),
		"FeedbackModule.LocalStorageKey 应被索引")
}

func TestBuildFromBackend_MissingEnum(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	_, ok := idx.EnumCases("NonExistent.Module.FakeEnum")
	assert.False(t, ok, "不存在的枚举应返回 false")
}

func TestIndex_ImplementsCatalogReader(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	var _ exprcheck.CatalogReader = idx
	t.Logf("Index implements CatalogReader: entities=%d enums=%d constants=%d",
		idx.EntityCount(), idx.EnumCount(), idx.ConstantsCount())
}

func TestMockIndex_Basics(t *testing.T) {
	m := meta.NewMockIndex(map[string][]string{
		"M.E": {"A", "B"},
	})
	m.AddConstant("@M.K")
	m.AddEntityAttr("M.Ent", "Field1", exprcheck.KindString)

	vals, ok := m.EnumCases("M.E")
	assert.True(t, ok)
	assert.Equal(t, []string{"A", "B"}, vals)

	assert.True(t, m.HasConstant("@M.K"))
	assert.False(t, m.HasConstant("@X.Y"))

	k, ok := m.AttributeKind("M.Ent", "Field1")
	assert.True(t, ok)
	assert.Equal(t, exprcheck.KindString, k)

	assert.True(t, m.HasEntity("M.Ent"))
	assert.False(t, m.HasEntity("X.Y"))
}

func TestBuildFromBackend_SystemEntity(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	_, ok := idx.AttributeKind("System.User", "Name")
	// System.User 可能不在普通 DomainModel 列表，允许 false（不强断言）
	t.Logf("System.User.Name found: %v", ok)
}
