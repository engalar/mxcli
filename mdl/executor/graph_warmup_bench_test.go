// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mpr "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// findCorpusMPR 查找可用的测试 MPR 文件。
// 如果找不到则跳过 benchmark（CI 无 testdata 时合理）。
func findCorpusMPR(b *testing.B) string {
	b.Helper()
	patterns := []string{
		"../../testdata/corpus-b/app.mpr",
		"../../testdata/expr-checker/minimal.mpr",
	}
	for _, p := range patterns {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	b.Skip("no test MPR found; set MXCLI_TEST_MPR to provide one")
	return ""
}

// openBenchExecutor 创建连接到 mprPath 的 Executor，不带 graph。
func openBenchExecutor(b *testing.B, mprPath string) *Executor {
	b.Helper()
	be := mpr.New()
	if err := be.Connect(mprPath); err != nil {
		b.Fatalf("backend.Connect: %v", err)
	}
	b.Cleanup(func() { _ = be.Disconnect() })
	e := New(os.Stdout)
	e.SetBackend(be)
	return e
}

// BenchmarkEntityNamesFromBackend 测试冷缓存时 getEntityNames 的 backend 路径。
// 这是我们要加速的操作。
func BenchmarkEntityNamesFromBackend(b *testing.B) {
	mprPath := findCorpusMPR(b)
	e := openBenchExecutor(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次重置缓存以测量冷启动
		e.cache = nil
		ctx := e.newExecContext(context.Background())
		ctx.initRoles() // mirror the registry dispatch path that wires role interfaces
		h := &ContainerHierarchy{} // 空 hierarchy 够用于此基准
		_ = getEntityNames(ctx, h)
	}
}

// BenchmarkMicroflowListFromBackend 测试冷缓存时 listMicroflowsWithContainerGen 的 backend 路径。
func BenchmarkMicroflowListFromBackend(b *testing.B) {
	mprPath := findCorpusMPR(b)
	e := openBenchExecutor(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.cache = nil
		ctx := e.newExecContext(context.Background())
		ctx.initRoles() // mirror the registry dispatch path that wires role interfaces
		_, _ = listMicroflowsWithContainerGen(ctx)
	}
}
