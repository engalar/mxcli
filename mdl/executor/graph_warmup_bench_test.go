// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	mpr "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

// heavyBench gates real-MPR benchmarks behind MXCLI_BENCH_HEAVY=1.
//
// Default behaviour is to skip, printing the exact command needed to run.
// This prevents accidental resource exhaustion on developer machines.
//
// Resource-safe invocation (copy-paste ready):
//
//	MXCLI_BENCH_HEAVY=1 GOMEMLIMIT=512MiB GOMAXPROCS=2 \
//	  nice -n 19 \
//	  go test ./mdl/executor/ -bench=<BenchmarkName> -benchtime=1x -count=3 -parallel 1
func heavyBench(b *testing.B) bool {
	b.Helper()
	if os.Getenv("MXCLI_BENCH_HEAVY") != "1" {
		b.Skipf("lightweight mode — skipping real-MPR benchmark to avoid resource exhaustion.\n"+
			"\tTo run: MXCLI_BENCH_HEAVY=1 GOMEMLIMIT=512MiB GOMAXPROCS=2 nice -n 19 "+
			"go test ./mdl/executor/ -bench=%s -benchtime=1x -count=3 -parallel 1", b.Name())
		return false
	}
	// Hard cap: even with -benchtime=60s, never run more than 3 iterations.
	// Each iteration opens real SQL connections; more than 3 rarely adds signal.
	if b.N > 3 {
		b.N = 3
	}
	return true
}

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
	b.Skip("no test MPR found")
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
	if !heavyBench(b) {
		return
	}
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
	if !heavyBench(b) {
		return
	}
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

func TestWarmCacheFromGraph_EntityNames(t *testing.T) {
	g := mxgraph.New()
	g.AddNode("uuid-entity-1", "Entity", map[string]any{
		"Name":          "Ticket",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.Ticket",
	})
	g.AddNode("uuid-entity-2", "Entity", map[string]any{
		"Name":          "Agent",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.Agent",
	})
	mgr := mxgraph.NewIndexManagerFromGraph(g)
	pg := graphcatalog.NewProjectGraph(mgr)

	cache := &executorCache{}
	warmCacheFromGraph(cache, pg)

	if len(cache.entityNames) != 2 {
		t.Fatalf("entityNames len = %d, want 2", len(cache.entityNames))
	}
	got := cache.entityNames[model.ID("uuid-entity-1")]
	if got != "Helpdesk.Ticket" {
		t.Errorf("entityNames[uuid-entity-1] = %q, want Helpdesk.Ticket", got)
	}
}

func TestWarmCacheFromGraph_MicroflowNames(t *testing.T) {
	g := mxgraph.New()
	g.AddNode("uuid-mf-1", "Microflow", map[string]any{
		"Name":          "ACT_Process",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.ACT_Process",
	})
	g.AddNode("uuid-nf-1", "Nanoflow", map[string]any{
		"Name":          "NF_Validate",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.NF_Validate",
	})
	mgr := mxgraph.NewIndexManagerFromGraph(g)
	pg := graphcatalog.NewProjectGraph(mgr)

	cache := &executorCache{}
	warmCacheFromGraph(cache, pg)

	if len(cache.microflowNames) != 2 {
		t.Fatalf("microflowNames len = %d, want 2", len(cache.microflowNames))
	}
	if cache.microflowNames[model.ID("uuid-mf-1")] != "Helpdesk.ACT_Process" {
		t.Error("microflow QN mismatch")
	}
	if cache.microflowNames[model.ID("uuid-nf-1")] != "Helpdesk.NF_Validate" {
		t.Error("nanoflow QN mismatch")
	}
}

func TestWarmCacheFromGraph_NilSafe(t *testing.T) {
	// graph 为 nil 时不应 panic
	warmCacheFromGraph(nil, nil)
	warmCacheFromGraph(&executorCache{}, nil)
}

func TestNewExecContext_WarmsCacheFromGraph(t *testing.T) {
	// 构造内存图，不需要真实 MPR
	g := mxgraph.New()
	g.AddNode("uuid-e1", "Entity", map[string]any{
		"QualifiedName": "MyModule.MyEntity",
		"Module":        "MyModule",
		"Name":          "MyEntity",
	})
	mgr := mxgraph.NewIndexManagerFromGraph(g)
	pg := graphcatalog.NewProjectGraph(mgr)

	// Executor 无 backend（nil），graph 已有
	e := &Executor{
		graphCatalog: pg,
		cache:        nil, // 冷缓存
	}

	ctx := e.newExecContext(context.Background())

	// newExecContext 应已预热缓存
	if ctx.Cache == nil {
		t.Fatal("cache should not be nil after newExecContext")
	}
	if len(ctx.Cache.entityNames) == 0 {
		t.Error("expected entity names pre-warmed from graph")
	}
	got := ctx.Cache.entityNames[model.ID("uuid-e1")]
	if got != "MyModule.MyEntity" {
		t.Errorf("entityNames[uuid-e1] = %q, want MyModule.MyEntity", got)
	}
}

func TestExecConnect_AutoLoadsGraphSnapshot(t *testing.T) {
	// 准备一个临时目录模拟项目目录结构
	dir := t.TempDir()
	mxcliDir := filepath.Join(dir, ".mxcli")
	if err := os.MkdirAll(mxcliDir, 0700); err != nil {
		t.Fatal(err)
	}

	// 构造一个小图并序列化为 graph.gob
	g := mxgraph.New()
	g.AddNode("uuid-e1", "Entity", map[string]any{
		"QualifiedName": "Test.Entity1",
		"Module":        "Test",
		"Name":          "Entity1",
	})
	mgr := mxgraph.NewIndexManagerFromGraph(g)
	pg := graphcatalog.NewProjectGraph(mgr)
	data, err := pg.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	snapPath := graphcatalog.SnapshotPath(dir)
	if err := os.WriteFile(snapPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	// 不经过 execConnect，直接测试 tryLoadGraphSnapshot
	cache := &executorCache{}
	var loadedGraph *graphcatalog.ProjectGraph
	tryLoadGraphSnapshot(dir, cache, &loadedGraph)

	if loadedGraph == nil {
		t.Fatal("expected graph to be loaded from snapshot")
	}
	if len(cache.entityNames) == 0 {
		t.Error("expected entity names pre-warmed after snapshot load")
	}
	if cache.entityNames[model.ID("uuid-e1")] != "Test.Entity1" {
		t.Errorf("entity name = %q, want Test.Entity1", cache.entityNames[model.ID("uuid-e1")])
	}
}

// openBenchExecutorWithGraph 创建带预建 graph 的 Executor。
// 用于测量 graph 加速路径的性能。
func openBenchExecutorWithGraph(b *testing.B, mprPath string) *Executor {
	b.Helper()
	be := mpr.New()
	if err := be.Connect(mprPath); err != nil {
		b.Fatalf("backend.Connect: %v", err)
	}
	b.Cleanup(func() { _ = be.Disconnect() })
	e := New(os.Stdout)
	e.SetBackend(be)
	e.mprPath = mprPath // buildGraph opens the project via modelsdk.Open(ctx.MprPath)

	// 首次构建 graph（这是一次性开销，不计入 benchmark）
	b.StopTimer()
	if _, err := e.BuildGraph(); err != nil {
		b.Fatalf("BuildGraph: %v", err)
	}
	b.StartTimer()
	return e
}

// BenchmarkEntityNamesFromGraph 是性能红线：必须比 BenchmarkEntityNamesFromBackend 快 5x+。
// 如果比值低于 5x，说明 warmCacheFromGraph 或 graph propIdx 出了问题。
func BenchmarkEntityNamesFromGraph(b *testing.B) {
	if !heavyBench(b) {
		return
	}
	mprPath := findCorpusMPR(b)
	e := openBenchExecutorWithGraph(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.cache = nil // 重置 executorCache（但 graphCatalog 保留）
		ctx := e.newExecContext(context.Background())
		ctx.initRoles() // mirror the registry dispatch path that wires role interfaces
		h := &ContainerHierarchy{}
		_ = getEntityNames(ctx, h)
	}
}

// BenchmarkMicroflowListFromGraph 测试图预热路径的微流列举。
func BenchmarkMicroflowListFromGraph(b *testing.B) {
	if !heavyBench(b) {
		return
	}
	mprPath := findCorpusMPR(b)
	e := openBenchExecutorWithGraph(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.cache = nil
		ctx := e.newExecContext(context.Background())
		ctx.initRoles() // mirror the registry dispatch path that wires role interfaces
		_, _ = listMicroflowsWithContainerGen(ctx)
	}
}
