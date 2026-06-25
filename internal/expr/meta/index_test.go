// SPDX-License-Identifier: Apache-2.0

package meta_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/stretchr/testify/assert"
)

// sharedIdx is built once per test binary run (TestMain) and reused by all
// TestBuildFromBackend_* tests. Saving ~5s per test that previously opened
// and indexed corpus-a independently.
var sharedIdx *meta.Index

func TestMain(m *testing.M) {
	mprPath := findCorpusAMPR()
	if mprPath == "" {
		// corpus-a not available — individual tests will skip via testutil.FindMPR.
		os.Exit(m.Run())
	}
	b, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "meta_test: open corpus-a: %v\n", err)
		os.Exit(1)
	}
	defer b.Disconnect() //nolint:errcheck
	idx, err := meta.BuildFromBackend(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "meta_test: build index: %v\n", err)
		os.Exit(1)
	}
	sharedIdx = idx
	os.Exit(m.Run())
}

func findCorpusAMPR() string {
	if p := os.Getenv("CORPUS_A_MPR"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "testdata/corpus-a/app.mpr")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// requireSharedIdx returns sharedIdx or skips the test if corpus-a was absent.
func requireSharedIdx(t *testing.T) *meta.Index {
	t.Helper()
	if sharedIdx == nil {
		// Trigger the normal skip path so the test reports "SKIP".
		testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	}
	return sharedIdx
}

func TestBuildFromBackend_EntityAttrs(t *testing.T) {
	t.Parallel()
	idx := requireSharedIdx(t)

	// 验证已知存在的实体属性（来自真实 MPR）
	kind, ok := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "Status")
	assert.True(t, ok, "Status 属性应存在")
	assert.NotZero(t, kind, "kind 应为非零值")

	kind2, ok2 := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "AWApplicationNo")
	assert.True(t, ok2, "AWApplicationNo 属性应存在")
	assert.NotZero(t, kind2)
}

func TestBuildFromBackend_EnumValues(t *testing.T) {
	t.Parallel()
	idx := requireSharedIdx(t)

	assert.Greater(t, idx.EnumCount(), 0, "应该索引到至少一个枚举")

	// ENUM_MailStatus 是已知枚举 (见诊断输出)
	vals, ok := idx.EnumCases("MailInquiry.ENUM_MailStatus")
	assert.True(t, ok, "MailInquiry.ENUM_MailStatus 应存在")
	assert.NotEmpty(t, vals, "枚举值列表不应为空")
}

func TestBuildFromBackend_Constants(t *testing.T) {
	t.Parallel()
	idx := requireSharedIdx(t)

	assert.Greater(t, idx.ConstantsCount(), 0, "应有常量记录")
	// FeedbackModule.LocalStorageKey 是已知常量 (见诊断输出)
	assert.True(t, idx.HasConstant("@FeedbackModule.LocalStorageKey"),
		"FeedbackModule.LocalStorageKey 应被索引")
}

func TestBuildFromBackend_MissingEnum(t *testing.T) {
	t.Parallel()
	idx := requireSharedIdx(t)

	_, ok := idx.EnumCases("NonExistent.Module.FakeEnum")
	assert.False(t, ok, "不存在的枚举应返回 false")
}

func TestIndex_ImplementsCatalogReader(t *testing.T) {
	t.Parallel()
	idx := requireSharedIdx(t)

	var _ exprcheck.CatalogReader = idx
	t.Logf("Index implements CatalogReader: entities=%d enums=%d constants=%d",
		idx.EntityCount(), idx.EnumCount(), idx.ConstantsCount())
}

func TestMockIndex_Basics(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	idx := requireSharedIdx(t)

	_, ok := idx.AttributeKind("System.User", "Name")
	// System.User 可能不在普通 DomainModel 列表，允许 false（不强断言）
	t.Logf("System.User.Name found: %v", ok)
}
