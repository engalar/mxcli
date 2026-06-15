# MDL Run Graph Cache Warmup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 `mxgraph` 内存图预热 executor 冷启动缓存，将 entity/microflow 名称查找从 O(N²) SQL 降至 O(1) propIdx 查询，同时在 `execConnect` 时自动加载 graph 快照省去手动 `refresh graph`。

**Architecture:** 新增 `warmCacheFromGraph(ctx)` 函数，在 `newExecContext()` 中图可用时调用，将 `executorCache.entityNames` / `microflowNames` 从图节点预填。`execConnect()` 连接新项目时自动尝试加载 `.mxcli/graph.gob` 快照。已有 cache 检查逻辑（`if cache.entityNames != nil { return }`）保持不变，图只是更快的填充路径，退化到 backend 路径零风险。

**Tech Stack:** Go 1.21+、`internal/mxgraph`（已有）、`mdl/graphcatalog`（已有）、`testing/benchmark`

**Spec:** 前一轮 "mdl run howto speed up with mxgraph" 分析

---

## 文件变更地图

| 操作 | 文件 |
|------|------|
| Create | `mdl/executor/graph_warmup.go` |
| Create | `mdl/executor/graph_warmup_bench_test.go` |
| Modify | `mdl/executor/executor_dispatch.go:69`（newExecContext） |
| Modify | `mdl/executor/executor_connect.go:13`（execConnect） |

---

## Task 1：建立性能基准（先跑，记录当前慢值）

**Files:**
- Create: `mdl/executor/graph_warmup_bench_test.go`

- [ ] **Step 1: 写 benchmark 辅助函数和基准测试**

```go
// mdl/executor/graph_warmup_bench_test.go
package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mpr"
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
	be := mpr.NewBackend()
	if err := be.Connect(mprPath); err != nil {
		b.Fatalf("backend.Connect: %v", err)
	}
	b.Cleanup(func() { be.Disconnect() })
	e := New(be, os.Stdout)
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
		_, _ = listMicroflowsWithContainerGen(ctx)
	}
}
```

- [ ] **Step 2: 运行 benchmark，记录基准（这是当前慢值）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run=^$ -bench=BenchmarkEntityNames -benchtime=3s -count=1
go test ./mdl/executor/ -run=^$ -bench=BenchmarkMicroflowList -benchtime=3s -count=1
```

记录 ns/op 数值——修复后对比。预期当前很慢（corpus-b 有 700+ entity，需遍历全部域模型）。

- [ ] **Step 3: Commit（只含 bench 文件，不含任何实现）**

```bash
git add mdl/executor/graph_warmup_bench_test.go
git commit -m "test(executor): add cold-start benchmarks for entity/microflow name lookup"
```

---

## Task 2：实现 warmCacheFromGraph（SOLID S：单一职责）

**Files:**
- Create: `mdl/executor/graph_warmup.go`

- [ ] **Step 1: 写单元测试（先失败）**

在新建文件里写测试，验证图节点能正确填充 executorCache：

```go
// mdl/executor/graph_warmup_bench_test.go — 追加到文件末尾
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
```

需要在文件顶部加 import：

```go
import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/executor/ -run TestWarmCache -v
```

预期：`FAIL — warmCacheFromGraph undefined`

- [ ] **Step 3: 实现 graph_warmup.go**

```go
// mdl/executor/graph_warmup.go
package executor

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
)

// warmCacheFromGraph seeds executorCache name-lookup maps from the in-memory
// graph, avoiding cold-start backend scans.
//
// Only fills maps that are nil — existing cache entries (from a previous
// warm-up or manual population) are never overwritten. This function is
// intentionally O(nodes) over the graph, not O(N²) over backend calls.
//
// SOLID:
//   - S: single job — graph → cache translation, no backend I/O
//   - O: adds a fast path; existing backend fallback untouched
//   - D: depends on *graphcatalog.ProjectGraph abstraction, not concrete mxgraph types
func warmCacheFromGraph(cache *executorCache, pg *graphcatalog.ProjectGraph) {
	if cache == nil || pg == nil {
		return
	}

	g := pg.Graph() // *mxgraph.Graph — package-internal accessor

	// ── Entity names ──────────────────────────────────────────────────
	if cache.entityNames == nil {
		nodes := g.FindNodes("Entity", nil)
		if len(nodes) > 0 {
			m := make(map[model.ID]string, len(nodes))
			for _, n := range nodes {
				qn, _ := n.Props["QualifiedName"].(string)
				if qn != "" {
					m[model.ID(n.ID)] = qn
				}
			}
			if len(m) > 0 {
				cache.entityNames = m
			}
		}
	}

	// ── Microflow + nanoflow names ─────────────────────────────────────
	if cache.microflowNames == nil {
		mfNodes := g.FindNodes("Microflow", nil)
		nfNodes := g.FindNodes("Nanoflow", nil)
		total := len(mfNodes) + len(nfNodes)
		if total > 0 {
			m := make(map[model.ID]string, total)
			for _, n := range mfNodes {
				qn, _ := n.Props["QualifiedName"].(string)
				if qn != "" {
					m[model.ID(n.ID)] = qn
				}
			}
			for _, n := range nfNodes {
				qn, _ := n.Props["QualifiedName"].(string)
				if qn != "" {
					m[model.ID(n.ID)] = qn
				}
			}
			if len(m) > 0 {
				cache.microflowNames = m
			}
		}
	}

	// ── Page names ─────────────────────────────────────────────────────
	if cache.pageNames == nil {
		nodes := g.FindNodes("Page", nil)
		if len(nodes) > 0 {
			m := make(map[model.ID]string, len(nodes))
			for _, n := range nodes {
				qn, _ := n.Props["QualifiedName"].(string)
				if qn != "" {
					m[model.ID(n.ID)] = qn
				}
			}
			if len(m) > 0 {
				cache.pageNames = m
			}
		}
	}
}
```

**注意：** `pg.Graph()` 方法需要在 `mdl/graphcatalog/graph.go` 新增（当前没有暴露内部 graph）：

```go
// Graph returns the underlying mxgraph.Graph for low-level access.
// Callers must not mutate the returned graph.
func (pg *ProjectGraph) Graph() *mxgraph.Graph {
	return pg.mgr.Query()
}
```

在 `mdl/graphcatalog/graph.go` 中 `NewProjectGraph` 定义下方追加此方法。

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./mdl/executor/ -run TestWarmCache -v
go test ./mdl/graphcatalog/... -v -count=1
```

预期：两组测试全绿。

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/graph_warmup.go mdl/executor/graph_warmup_bench_test.go mdl/graphcatalog/graph.go
git commit -m "feat(executor): add warmCacheFromGraph — seed entity/microflow name caches from mxgraph"
```

---

## Task 3：接入 newExecContext（SOLID O：扩展不修改）

**Files:**
- Modify: `mdl/executor/executor_dispatch.go:69`（newExecContext）

- [ ] **Step 1: 写集成测试（先失败）**

追加到 `graph_warmup_bench_test.go`：

```go
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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/executor/ -run TestNewExecContext_WarmsCacheFromGraph -v
```

预期：`FAIL — entityNames 为空`（因为 warmCacheFromGraph 还没被调用）

- [ ] **Step 3: 在 newExecContext 末尾调用 warmCacheFromGraph**

在 `executor_dispatch.go` 的 `newExecContext()` 函数末尾，在 `return &ExecContext{...}` 之后、return 之前（或改为先赋值再调用）：

找到当前的 return 语句（约第 74-135 行），改写为：

```go
func (e *Executor) newExecContext(ctx context.Context) *ExecContext {
	e.catalogMu.RLock()
	cat := e.catalog
	gen := e.catalogGen
	e.catalogMu.RUnlock()

	// 确保 cache 已初始化（Connect 时设置，但直接构造的 Executor 可能为 nil）
	if e.cache == nil {
		e.cache = &executorCache{}
	}

	// 如果 graph 可用且 cache 尚冷，从图预热名称索引（O(nodes) 而非 O(N²) SQL）
	if e.graphCatalog != nil {
		warmCacheFromGraph(e.cache, e.graphCatalog)
	}

	return &ExecContext{
		// ... 原有字段保持不变 ...
```

**只需在现有 return 前插入两行（nil guard + warmCacheFromGraph 调用），其余不动。**

具体：在 `executor_dispatch.go` 中找到 `func (e *Executor) newExecContext` 的开头，在 `e.catalogMu.RLock()` 之后、`return &ExecContext{` 之前，插入：

```go
	// Ensure cache exists (Connect sets it; direct Executor construction may leave it nil).
	if e.cache == nil {
		e.cache = &executorCache{}
	}
	// Warm name caches from graph if available — O(nodes) vs O(N²) backend scan.
	if e.graphCatalog != nil {
		warmCacheFromGraph(e.cache, e.graphCatalog)
	}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./mdl/executor/ -run TestNewExecContext_WarmsCacheFromGraph -v
go test ./mdl/executor/ -count=1 -run TestCatalogRefs -tags integration 2>&1 | tail -5
```

预期：TestNewExecContext_WarmsCacheFromGraph PASS，集成测试无回归。

- [ ] **Step 5: 运行全量单元测试**

```bash
go test ./mdl/executor/... -count=1 2>&1 | grep -E "FAIL|ok"
```

预期：无 FAIL（已知的 pre-existing panic TestCreateAgentEditorModel_Mock 等除外）。

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/executor_dispatch.go mdl/executor/graph_warmup_bench_test.go
git commit -m "feat(executor): warm entity/microflow name caches from graph in newExecContext"
```

---

## Task 4：execConnect 自动加载 graph 快照

**Files:**
- Modify: `mdl/executor/executor_connect.go:13`（execConnect）

- [ ] **Step 1: 写测试（先失败）**

追加到 `graph_warmup_bench_test.go`：

```go
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

	// execConnect を通さず直接 tryLoadGraphSnapshot をテスト
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
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/executor/ -run TestExecConnect_AutoLoadsGraphSnapshot -v
```

预期：`FAIL — tryLoadGraphSnapshot undefined`

- [ ] **Step 3: 在 graph_warmup.go 中新增 tryLoadGraphSnapshot**

追加到 `mdl/executor/graph_warmup.go`：

```go
import (
	"os"
	"path/filepath"
	// ... 其余 import 保持不变
)

// tryLoadGraphSnapshot 尝试从项目目录的 .mxcli/graph.gob 加载图快照。
// 成功时预热 cache 并设置 *out；失败时静默返回（graph 是可选加速器）。
//
// 设计为独立函数（而非 execConnect 内联）方便单独测试。
func tryLoadGraphSnapshot(projectDir string, cache *executorCache, out **graphcatalog.ProjectGraph) {
	if projectDir == "" || cache == nil || out == nil {
		return
	}
	snapPath := graphcatalog.SnapshotPath(projectDir)
	data, err := os.ReadFile(snapPath)
	if err != nil {
		return // 文件不存在或无权限——静默跳过
	}
	pg, err := graphcatalog.UnmarshalSnapshot(data)
	if err != nil {
		return // 快照损坏——静默跳过，不影响连接
	}
	*out = pg
	warmCacheFromGraph(cache, pg)
}
```

- [ ] **Step 4: 在 execConnect 中调用 tryLoadGraphSnapshot**

在 `executor_connect.go` 的 `execConnect()` 中，在 `ctx.Cache = &executorCache{}` 之后追加：

```go
ctx.Cache = &executorCache{} // Initialize fresh cache

// Auto-load graph snapshot if available — lets subsequent commands use the
// pre-built index without an explicit "refresh graph" command.
if s.Path != "" {
    tryLoadGraphSnapshot(filepath.Dir(s.Path), ctx.Cache, &ctx.Graph)
}
```

需要在 `executor_connect.go` 顶部 import 中加入 `"path/filepath"`：

```go
import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)
```

- [ ] **Step 5: 运行测试**

```bash
go test ./mdl/executor/ -run TestExecConnect_AutoLoadsGraphSnapshot -v
go build ./... 2>&1 | head -10
```

预期：测试 PASS，build 绿。

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/graph_warmup.go mdl/executor/executor_connect.go mdl/executor/graph_warmup_bench_test.go
git commit -m "feat(executor): auto-load graph snapshot on execConnect for zero-config cache warmup"
```

---

## Task 5：验证性能提升（perf ratchet）

**Files:**
- Modify: `mdl/executor/graph_warmup_bench_test.go`（新增 with-graph benchmarks）

- [ ] **Step 1: 新增"有图"版 benchmark 用于对比**

追加到 `graph_warmup_bench_test.go`：

```go
// openBenchExecutorWithGraph 创建带预建 graph 的 Executor。
// 用于测量 graph 加速路径的性能。
func openBenchExecutorWithGraph(b *testing.B, mprPath string) *Executor {
	b.Helper()
	be := mpr.NewBackend()
	if err := be.Connect(mprPath); err != nil {
		b.Fatalf("backend.Connect: %v", err)
	}
	b.Cleanup(func() { be.Disconnect() })
	e := New(be, os.Stdout)

	// 首次构建 graph（这是一次性开销，不计入 benchmark）
	b.StopTimer()
	if _, err := e.BuildGraph(); err != nil {
		b.Fatalf("BuildGraph: %v", err)
	}
	b.StartTimer()
	return e
}

// BenchmarkEntityNamesFromGraph 测试图预热路径的 entity 名称查找。
// 与 BenchmarkEntityNamesFromBackend 对比。
func BenchmarkEntityNamesFromGraph(b *testing.B) {
	mprPath := findCorpusMPR(b)
	e := openBenchExecutorWithGraph(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.cache = nil // 重置 executorCache（但 graphCatalog 保留）
		ctx := e.newExecContext(context.Background())
		h := &ContainerHierarchy{}
		_ = getEntityNames(ctx, h)
	}
}

// BenchmarkMicroflowListFromGraph 测试图预热路径的微流列举。
func BenchmarkMicroflowListFromGraph(b *testing.B) {
	mprPath := findCorpusMPR(b)
	e := openBenchExecutorWithGraph(b, mprPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.cache = nil
		ctx := e.newExecContext(context.Background())
		_, _ = listMicroflowsWithContainerGen(ctx)
	}
}
```

- [ ] **Step 2: 运行对比 benchmark**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run=^$ \
  -bench="BenchmarkEntityNames|BenchmarkMicroflowList" \
  -benchtime=3s -count=1
```

预期输出格式（具体数字会不同）：
```
BenchmarkEntityNamesFromBackend    X ns/op   # 当前（慢）
BenchmarkEntityNamesFromGraph      Y ns/op   # 修复后（快）
BenchmarkMicroflowListFromBackend  A ns/op   # 当前（慢）
BenchmarkMicroflowListFromGraph    B ns/op   # 修复后（快）
```

**验收红线：** `BenchmarkEntityNamesFromGraph` 比 `BenchmarkEntityNamesFromBackend` 快 **5x 以上**（entity names 从 O(N·M) SQL 扫描到 O(nodes) 图遍历）。

- [ ] **Step 3: 运行全量测试确认零回归**

```bash
go test ./internal/mxgraph/... ./mdl/graphcatalog/... ./mdl/executor/... -count=1 2>&1 | grep -E "FAIL|ok"
```

预期：所有包 `ok`，无 `FAIL`（pre-existing panic 不算新回归）。

- [ ] **Step 4: 最终 commit（含 benchmark 红线注释）**

在 `graph_warmup_bench_test.go` 中在 `BenchmarkEntityNamesFromGraph` 上方加注释：

```go
// BenchmarkEntityNamesFromGraph 是性能红线：必须比 BenchmarkEntityNamesFromBackend 快 5x+。
// 如果比值低于 5x，说明 warmCacheFromGraph 或 graph propIdx 出了问题。
```

```bash
git add mdl/executor/graph_warmup_bench_test.go
git commit -m "test(executor): add graph-vs-backend benchmark ratchet for entity/microflow warmup"
```

---

## 完整性自查

| 需求 | 对应 Task |
|------|----------|
| 性能基准（先建） | Task 1 |
| warmCacheFromGraph（SRP） | Task 2 |
| newExecContext 接入（OCP） | Task 3 |
| auto-load snapshot（零配置） | Task 4 |
| benchmark 验收红线 | Task 5 |
| TDD（先写测试） | 每个 Task 都先写失败测试 |
| SOLID S | warmCacheFromGraph 单一职责 |
| SOLID O | 已有 cache 检查逻辑不变，只加快速路径 |
| SOLID D | 依赖 *graphcatalog.ProjectGraph，不依赖 mxgraph 内部 |
| 零回归 | Task 5 Step 3 全量测试 |
