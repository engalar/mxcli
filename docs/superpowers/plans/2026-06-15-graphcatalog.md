# GraphCatalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 `internal/mxgraph` 内存图引擎彻底替换 `mdl/catalog/` SQLite catalog，以 SOLID 原则设计接口，全程 TDD，性能基准 CI 守护。

**Architecture:** mxgraph 图引擎（通用）+ 5 个单域 IndexAdapter（Mendix 语义）+ `mdl/graphcatalog/` 类型化门面层（消费方只依赖接口）。linter 依赖 `LintReader`，executor search 依赖 `TraversalReader`，两者均不感知具体实现。持久化从 SQLite `catalog.db` 改为 gob snapshot `graph.gob`。

**Tech Stack:** Go 1.21+、`encoding/gob`（已有）、`internal/mxgraph`（已有）、`modelsdk`（已有）、`testing/benchmark`

**Spec:** `docs/superpowers/specs/2026-06-15-graphcatalog-design.md`

---

## 文件变更地图

### Phase 1 — mxgraph Bug 修复
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/engine_bench_test.go` |
| Modify | `internal/mxgraph/engine.go` |
| Create | `internal/mxgraph/query_bench_test.go` |
| Modify | `internal/mxgraph/query.go` |

### Phase 2 — 拆分 mpr 适配器
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/mpr/domainmodel.go` |
| Create | `internal/mxgraph/adapter/mpr/domainmodel_test.go` |
| Create | `internal/mxgraph/adapter/mpr/microflow.go` |
| Create | `internal/mxgraph/adapter/mpr/microflow_test.go` |
| Create | `internal/mxgraph/adapter/mpr/page.go` |
| Create | `internal/mxgraph/adapter/mpr/security.go` |
| Create | `internal/mxgraph/adapter/mpr/enumeration.go` |
| Modify | `internal/mxgraph/adapter/mpr/adapter.go` → 精简为包注释 + 公共辅助函数 |

### Phase 3 — graphcatalog 包
| 操作 | 文件 |
|------|------|
| Create | `mdl/graphcatalog/reader.go` |
| Create | `mdl/graphcatalog/nodes.go` |
| Create | `mdl/graphcatalog/graph.go` |
| Create | `mdl/graphcatalog/persist.go` |
| Create | `mdl/graphcatalog/graph_test.go` |
| Create | `mdl/graphcatalog/mock/mock.go` |

### Phase 4 — 迁移 linter
| 操作 | 文件 |
|------|------|
| Modify | `mdl/linter/context.go` |

### Phase 5 — 迁移 executor
| 操作 | 文件 |
|------|------|
| Modify | `mdl/executor/exec_context.go` |
| Modify | `mdl/executor/cmd_catalog.go` |
| Modify | `mdl/executor/cmd_search.go` |
| Modify | `mdl/executor/executor.go` |
| Modify | `mdl/executor/executor_dispatch.go` |

### Phase 6 — 迁移 serve.go
| 操作 | 文件 |
|------|------|
| Modify | `cmd/mxcli/serve.go` |

### Phase 7 — 删除 mdl/catalog/
| 操作 | 文件 |
|------|------|
| Delete | `mdl/catalog/`（整包） |

---

## Task 1：为 Edges() 写失败 benchmark

**Files:**
- Create: `internal/mxgraph/engine_bench_test.go`

- [ ] **Step 1: 写 benchmark 和辅助函数**

```go
// internal/mxgraph/engine_bench_test.go
package mxgraph

import (
	"fmt"
	"testing"
)

// buildLargeGraph 创建 nodeCount 个节点，每节点有 edgesPerNode 条出边
func buildLargeGraph(nodeCount, edgesPerNode int) *Graph {
	g := New()
	for i := 0; i < nodeCount; i++ {
		g.AddNode(NodeID(fmt.Sprintf("n%d", i)), "Entity", map[string]any{"idx": i})
	}
	edgeIdx := 0
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < edgesPerNode; j++ {
			to := (i + j + 1) % nodeCount
			eid := NodeID(fmt.Sprintf("e%d", edgeIdx))
			g.AddEdge(eid, NodeID(fmt.Sprintf("n%d", i)), NodeID(fmt.Sprintf("n%d", to)), "HAS_ATTRIBUTE", nil)
			edgeIdx++
		}
	}
	return g
}

// BenchmarkEdges_Outbound 当前是 O(E) 全边扫描，修复后应为 O(degree)
func BenchmarkEdges_Outbound(b *testing.B) {
	g := buildLargeGraph(1000, 5) // 1000 节点，每节点 5 条出边，共 5000 条边
	target := NodeID("n500")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Edges(target, Outbound)
	}
}

// BenchmarkEdges_Both 测试双向查询
func BenchmarkEdges_Both(b *testing.B) {
	g := buildLargeGraph(1000, 5)
	target := NodeID("n500")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Edges(target, Both)
	}
}

// BenchmarkRemoveNode_Large 当前 RemoveNode 有 O(E) 全边扫描
func BenchmarkRemoveNode_Large(b *testing.B) {
	const nodeCount = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g := buildLargeGraph(nodeCount, 5)
		b.StartTimer()
		g.RemoveNode(NodeID("n500"))
	}
}
```

- [ ] **Step 2: 运行 benchmark，记录基准（当前慢）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/mxgraph/ -run=^$ -bench=BenchmarkEdges -benchtime=3s -count=1
```

预期输出（当前 O(E)，应该很慢，记录 ns/op）：
```
BenchmarkEdges_Outbound-N     XXXXX ns/op
```

将输出保存到脑海中，修复后对比。

---

## Task 2：修复 Edges() — 新增 outEdgeIDs / inEdgeIDs 索引

**Files:**
- Modify: `internal/mxgraph/engine.go`

- [ ] **Step 1: 在 Graph struct 新增两个索引字段**

在 `engine.go` 的 `Graph` struct（第 8 行）中，增加两个字段：

```go
type Graph struct {
	mu       sync.RWMutex
	nodes    map[NodeID]*Node
	edges    map[NodeID]*Edge
	outEdges map[NodeID]map[RelType][]NodeID
	inEdges  map[NodeID]map[RelType][]NodeID
	byLabel  map[Label]map[NodeID]bool
	propIdx  map[Label]map[string]map[any]map[NodeID]bool
	// 新增：edge ID 索引，用于 O(degree) 的 Edges() 查询
	// key: node ID, value: relType → []edgeID（NodeID 类型复用，值是边的 ID）
	outEdgeIDs map[NodeID]map[RelType][]NodeID
	inEdgeIDs  map[NodeID]map[RelType][]NodeID
}
```

- [ ] **Step 2: 初始化新字段**

在 `New()` 函数（第 18 行）中：

```go
func New() *Graph {
	return &Graph{
		nodes:      map[NodeID]*Node{},
		edges:      map[NodeID]*Edge{},
		outEdges:   map[NodeID]map[RelType][]NodeID{},
		inEdges:    map[NodeID]map[RelType][]NodeID{},
		byLabel:    map[Label]map[NodeID]bool{},
		propIdx:    map[Label]map[string]map[any]map[NodeID]bool{},
		outEdgeIDs: map[NodeID]map[RelType][]NodeID{},
		inEdgeIDs:  map[NodeID]map[RelType][]NodeID{},
	}
}
```

- [ ] **Step 3: 在 AddEdge 中维护新索引**

在 `AddEdge`（第 64 行）末尾，在已有 inEdges 维护代码后追加：

```go
func (g *Graph) AddEdge(id, from, to NodeID, rel RelType, props map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e := &Edge{ID: id, From: from, To: to, Type: rel, Props: props}
	g.edges[id] = e
	if g.outEdges[from] == nil {
		g.outEdges[from] = map[RelType][]NodeID{}
	}
	g.outEdges[from][rel] = append(g.outEdges[from][rel], to)
	if g.inEdges[to] == nil {
		g.inEdges[to] = map[RelType][]NodeID{}
	}
	g.inEdges[to][rel] = append(g.inEdges[to][rel], from)
	// 新增：维护 edge ID 索引
	if g.outEdgeIDs[from] == nil {
		g.outEdgeIDs[from] = map[RelType][]NodeID{}
	}
	g.outEdgeIDs[from][rel] = append(g.outEdgeIDs[from][rel], id)
	if g.inEdgeIDs[to] == nil {
		g.inEdgeIDs[to] = map[RelType][]NodeID{}
	}
	g.inEdgeIDs[to][rel] = append(g.inEdgeIDs[to][rel], id)
}
```

- [ ] **Step 4: 在 RemoveNode 中用新索引替换 O(E) 全扫描**

将 `RemoveNode`（第 79 行）中的全边扫描（`for eid, e := range g.edges { if e.From == id || e.To == id {...} }`）替换为：

```go
func (g *Graph) RemoveNode(id NodeID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := g.nodes[id]
	if n == nil {
		return
	}
	delete(g.byLabel[n.Label], id)
	// 用 outEdgeIDs 删除所有出边（O(out-degree)）
	for rel, eids := range g.outEdgeIDs[id] {
		for _, eid := range eids {
			if e := g.edges[eid]; e != nil {
				g.removeInEdgeFromIndex(e.To, id, rel)
				g.removeFromInEdgeIDs(e.To, rel, eid)
			}
			delete(g.edges, eid)
		}
	}
	delete(g.outEdgeIDs, id)
	delete(g.outEdges, id)
	// 用 inEdgeIDs 删除所有入边（O(in-degree)）
	for rel, eids := range g.inEdgeIDs[id] {
		for _, eid := range eids {
			if e := g.edges[eid]; e != nil {
				g.removeEdgeFromIndex(e.From, id, rel)
				g.removeFromOutEdgeIDs(e.From, rel, eid)
			}
			delete(g.edges, eid)
		}
	}
	delete(g.inEdgeIDs, id)
	delete(g.inEdges, id)
	g.unindexProps(n)
	delete(g.nodes, id)
}
```

在文件末尾新增两个辅助方法：

```go
func (g *Graph) removeFromOutEdgeIDs(from NodeID, rel RelType, eid NodeID) {
	if m, ok := g.outEdgeIDs[from]; ok {
		eids := m[rel]
		for i, e := range eids {
			if e == eid {
				m[rel] = append(eids[:i], eids[i+1:]...)
				return
			}
		}
	}
}

func (g *Graph) removeFromInEdgeIDs(to NodeID, rel RelType, eid NodeID) {
	if m, ok := g.inEdgeIDs[to]; ok {
		eids := m[rel]
		for i, e := range eids {
			if e == eid {
				m[rel] = append(eids[:i], eids[i+1:]...)
				return
			}
		}
	}
}
```

- [ ] **Step 5: 修复 Edges() 使用新索引（替换整个方法）**

将 `Edges()`（第 209 行）替换为：

```go
func (g *Graph) Edges(id NodeID, dir Direction, relTypes ...RelType) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	relFilter := make(map[RelType]bool, len(relTypes))
	for _, rt := range relTypes {
		relFilter[rt] = true
	}
	useFilter := len(relTypes) > 0

	var result []*Edge
	seen := map[NodeID]bool{} // 防止 Both 模式下重复（自环情况）

	collect := func(idx map[NodeID]map[RelType][]NodeID) {
		for rel, eids := range idx[id] {
			if useFilter && !relFilter[rel] {
				continue
			}
			for _, eid := range eids {
				if seen[eid] {
					continue
				}
				seen[eid] = true
				if e := g.edges[eid]; e != nil {
					result = append(result, e)
				}
			}
		}
	}

	if dir == Outbound || dir == Both {
		collect(g.outEdgeIDs)
	}
	if dir == Inbound || dir == Both {
		collect(g.inEdgeIDs)
	}
	return result
}
```

- [ ] **Step 6: EdgeDeleted 事件也要维护新索引**

在 `Apply()` 的 `EdgeDeleted` 分支（第 174 行）中，在 `removeEdgeFromAdj(e)` 后追加：

```go
case EdgeDeleted:
	g.mu.Lock()
	e := g.edges[ev.Edge.ID]
	if e != nil {
		g.removeEdgeFromAdj(e)
		g.removeFromOutEdgeIDs(e.From, e.Type, e.ID)
		g.removeFromInEdgeIDs(e.To, e.Type, e.ID)
	}
	delete(g.edges, ev.Edge.ID)
	g.mu.Unlock()
```

- [ ] **Step 7: 运行全量单元测试确认无回归**

```bash
go test ./internal/mxgraph/... -v -count=1
```

预期：所有已有测试 PASS（TestGraphRemoveNode、TestGraphRemoveNodeCleansAdjacency 等）。

- [ ] **Step 8: 运行 benchmark 对比**

```bash
go test ./internal/mxgraph/ -run=^$ -bench=BenchmarkEdges -benchtime=3s -count=1
```

预期：`BenchmarkEdges_Outbound` 速度比 Task 1 记录的基准快 **10x 以上**（O(5) vs O(5000)）。

- [ ] **Step 9: Commit**

```bash
git add internal/mxgraph/engine.go internal/mxgraph/engine_bench_test.go
git commit -m "perf(mxgraph): fix Edges() O(E)→O(degree) via outEdgeIDs/inEdgeIDs index"
```

---

## Task 3：修复 FindPathSchemas() — 回溯替代 map 复制

**Files:**
- Create: `internal/mxgraph/query_bench_test.go`
- Modify: `internal/mxgraph/query.go`

- [ ] **Step 1: 写 benchmark（先跑，记录当前慢值）**

```go
// internal/mxgraph/query_bench_test.go
package mxgraph

import "testing"

// buildChainGraph 创建线性链：n0→n1→n2→...→nN，全部同 label 同 relType
func buildChainGraph(length int) *Graph {
	g := New()
	for i := 0; i <= length; i++ {
		g.AddNode(NodeID(fmt.Sprintf("n%d", i)), "Node", nil)
	}
	for i := 0; i < length; i++ {
		from := NodeID(fmt.Sprintf("n%d", i))
		to := NodeID(fmt.Sprintf("n%d", i+1))
		g.AddEdge(NodeID(fmt.Sprintf("e%d", i)), from, to, "NEXT", nil)
	}
	return g
}

func BenchmarkFindPathSchemas_Chain20(b *testing.B) {
	g := buildChainGraph(20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.FindPathSchemas("n0", "n20", 25)
	}
}
```

```bash
go test ./internal/mxgraph/ -run=^$ -bench=BenchmarkFindPathSchemas -benchtime=3s -count=1
```

记录当前 ns/op（当前 O(N²) map 复制，链长 20 时应明显较慢）。

- [ ] **Step 2: 修复 FindPathSchemas() 使用回溯法**

将 `query.go` 中的 `FindPathSchemas()` 函数（第 9-66 行）整体替换：

```go
func (g *Graph) FindPathSchemas(from, to NodeID, depthLimit int) []PathSchema {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.nodes[from] == nil || g.nodes[to] == nil {
		return nil
	}

	var schemas []PathSchema
	seen := map[string]bool{}

	// 共享回溯状态，不再每步复制 map
	visited := map[NodeID]bool{from: true}
	steps := make([]PathStep, 0, depthLimit)

	var dfs func(current NodeID, depth int)
	dfs = func(current NodeID, depth int) {
		if depth > depthLimit {
			return
		}
		if current == to && len(steps) > 0 {
			labelSeq := string(g.nodes[from].Label)
			for _, s := range steps {
				labelSeq += "→" + string(s.RelType) + "→" + string(s.NodeLabel)
			}
			if !seen[labelSeq] {
				seen[labelSeq] = true
				schemas = append(schemas, PathSchema{
					Steps: append([]PathStep{}, steps...), // 仅在找到路径时复制
					Label: labelSeq,
				})
			}
			return
		}

		for rel, targets := range g.outEdges[current] {
			for _, nextID := range targets {
				if visited[nextID] {
					continue
				}
				nextNode := g.nodes[nextID]
				if nextNode == nil {
					continue
				}
				// 回溯：加入 → 递归 → 移除
				visited[nextID] = true
				steps = append(steps, PathStep{NodeLabel: nextNode.Label, RelType: rel})
				dfs(nextID, depth+1)
				steps = steps[:len(steps)-1]
				delete(visited, nextID)
			}
		}
	}

	dfs(from, 0)
	return schemas
}
```

同时删除已不需要的 `pathState` struct（第 3-7 行）。

- [ ] **Step 3: 运行测试确认无回归**

```bash
go test ./internal/mxgraph/... -v -count=1 -run TestFindPath
```

预期：`TestFindPathSchemas` 和 `TestExplorePath` 均 PASS。

- [ ] **Step 4: 运行 benchmark 对比**

```bash
go test ./internal/mxgraph/ -run=^$ -bench=BenchmarkFindPathSchemas -benchtime=3s -count=1
```

预期：比 Step 1 的基准至少快 **5x**（链长 20 时，O(N) 回溯 vs O(N²) map 复制）。

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/query.go internal/mxgraph/query_bench_test.go
git commit -m "perf(mxgraph): fix FindPathSchemas() O(N²) map copy → O(depth) backtracking"
```

---

## Task 4：DomainModelAdapter（重构现有适配器）

**Files:**
- Create: `internal/mxgraph/adapter/mpr/domainmodel.go`
- Create: `internal/mxgraph/adapter/mpr/domainmodel_test.go`
- Modify: `internal/mxgraph/adapter/mpr/adapter.go`（保留 `nodeForElement` 辅助函数，删除 `Build`/`Watch`）

- [ ] **Step 1: 写失败测试（不依赖真实 MPR，用 fake sink 验证事件形状）**

```go
// internal/mxgraph/adapter/mpr/domainmodel_test.go
package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// fakeElement 模拟 element.Element，仅用于单元测试
// 需要实现 element.Element 接口（TypeName, ID, Properties）
// 此处用已有的 recordingSink（见 adapter_test.go 同包）

func TestDomainModelAdapter_Schema(t *testing.T) {
	a := &DomainModelAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"DomainModel", "Entity", "Attribute", "Association"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
}

func TestDomainModelAdapter_Name(t *testing.T) {
	a := &DomainModelAdapter{}
	if a.Name() != "domainmodel" {
		t.Errorf("Name() = %q, want domainmodel", a.Name())
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/mxgraph/adapter/mpr/... -run TestDomainModel -v
```

预期：`FAIL — DomainModelAdapter undefined`

- [ ] **Step 3: 创建 domainmodel.go，将现有 Adapter 的 Build 逻辑迁入**

```go
// internal/mxgraph/adapter/mpr/domainmodel.go
package mpr

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// DomainModelAdapter 发射 DomainModel / Entity / Attribute / Association 节点和边。
type DomainModelAdapter struct {
	Model *modelsdk.Model
}

func (a *DomainModelAdapter) Name() string { return "domainmodel" }

func (a *DomainModelAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"DomainModel", "Entity", "Attribute", "Association"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_ENTITY", "DomainModel", "Entity"},
			{"HAS_ATTRIBUTE", "Entity", "Attribute"},
			{"HAS_ASSOCIATION", "DomainModel", "Association"},
			{"GENERALIZES", "Entity", "Entity"},
		},
	}
}

func (a *DomainModelAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}

		if elem.TypeName() != "DomainModels$DomainModel" {
			continue
		}

		dmNode := nodeForElement(elem, "DomainModel")
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: dmNode})

		for _, prop := range elem.Properties() {
			switch prop.Name() {
			case "Entities":
				cl, ok := prop.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, child := range cl.ChildElements() {
					if child == nil {
						continue
					}
					ct := child.TypeName()
					if ct != "DomainModels$Entity" && ct != "DomainModels$EntityImpl" {
						continue
					}
					entityNode := nodeForElement(child, "Entity")
					events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: entityNode})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", dmNode.ID, entityNode.ID)),
							From: dmNode.ID,
							To:   entityNode.ID,
							Type: "HAS_ENTITY",
						},
					})
					// Attributes
					for _, ap := range child.Properties() {
						if ap.Name() != "Attributes" {
							continue
						}
						cl2, ok := ap.(element.ChildListProperty)
						if !ok {
							continue
						}
						for _, attr := range cl2.ChildElements() {
							if attr == nil {
								continue
							}
							attrNode := nodeForElement(attr, "Attribute")
							events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: attrNode})
							events = append(events, mxgraph.Event{
								Type: mxgraph.EdgeCreated,
								Edge: &mxgraph.Edge{
									ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", entityNode.ID, attrNode.ID)),
									From: entityNode.ID,
									To:   attrNode.ID,
									Type: "HAS_ATTRIBUTE",
								},
							})
						}
					}
				}
			case "Associations":
				cl, ok := prop.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, child := range cl.ChildElements() {
					if child == nil {
						continue
					}
					assocNode := nodeForElement(child, "Association")
					events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: assocNode})
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->%s", dmNode.ID, assocNode.ID)),
							From: dmNode.ID,
							To:   assocNode.ID,
							Type: "HAS_ASSOCIATION",
						},
					})
				}
			}
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *DomainModelAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 4: 清理 adapter.go，保留公共辅助函数**

将 `internal/mxgraph/adapter/mpr/adapter.go` 精简为：

```go
// Package mpr provides mxgraph IndexAdapter implementations for Mendix .mpr projects.
// Each adapter covers one domain and can be registered independently with IndexManager.
//
// Available adapters:
//   - DomainModelAdapter — entities, attributes, associations
//   - MicroflowAdapter   — microflows, nanoflows, call/create/retrieve refs
//   - PageAdapter        — pages, layouts, snippets, datasource/action refs
//   - SecurityAdapter    — permissions, role mappings
//   - EnumerationAdapter — enumerations, enum values
package mpr

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// nodeForElement 从 element.Element 提取 Props 创建 mxgraph.Node。
// 注意：仅提取 WritableProperty（有 BSONValue），避免把所有子元素列表塞入 Props。
func nodeForElement(elem element.Element, label mxgraph.Label) *mxgraph.Node {
	props := map[string]any{"$Type": elem.TypeName()}
	for _, p := range elem.Properties() {
		if wp, ok := p.(element.WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				props[p.Name()] = v
			}
		}
	}
	return &mxgraph.Node{ID: mxgraph.NodeID(elem.ID()), Label: label, Props: props}
}
```

删除原有的 `Adapter` struct 和其 `Build`/`Watch`/`Schema`/`Name` 方法（这些已迁入 domainmodel.go）。

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/mxgraph/adapter/mpr/... -v -count=1
```

预期：所有测试 PASS（包括原有 `TestMprAdapterFindPath` 如果 MPR 可用，否则 skip）。

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/adapter/mpr/
git commit -m "refactor(mxgraph/adapter): extract DomainModelAdapter from monolithic Adapter"
```

---

## Task 5：MicroflowAdapter（CALLS / CREATES / RETRIEVES 边）

**Files:**
- Create: `internal/mxgraph/adapter/mpr/microflow.go`
- Create: `internal/mxgraph/adapter/mpr/microflow_test.go`

**前置：** 在实现前，先查阅 `reference/mendixmodellib/reflection-data/Microflows/` 目录，确认以下类型名：
- 微流单元 TypeName（通常是 `Microflows$Microflow`）
- 调用微流活动 TypeName（`Microflows$MicroflowCallAction` 或类似）
- 创建对象活动 TypeName（`Microflows$CreateChangeAction`，见 CLAUDE.md）
- 检索活动 TypeName（`Microflows$RetrieveAction`）

- [ ] **Step 1: 确认 BSON 类型名**

```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/reference/mendixmodellib/reflection-data/ | grep -i micro
# 找到 Microflows 目录后：
ls /mnt/data_sdd/gh/mxcli-wt-02/reference/mendixmodellib/reflection-data/Microflows/ | head -20
```

在输出中找到 `Microflow.json`、`MicroflowCallAction.json`、`CreateChangeAction.json`、`RetrieveAction.json` 等，读取其中的 `"qualifiedName"` 字段即为 TypeName。

- [ ] **Step 2: 写测试（形状断言）**

```go
// internal/mxgraph/adapter/mpr/microflow_test.go
package mpr

import (
	"testing"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func TestMicroflowAdapter_Schema(t *testing.T) {
	a := &MicroflowAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Microflow", "Nanoflow"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
	relTypes := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		relTypes[et.Type] = true
	}
	for _, want := range []mxgraph.RelType{"CALLS", "CREATES", "RETRIEVES", "SHOWS_PAGE"} {
		if !relTypes[want] {
			t.Errorf("Schema missing edge type %q", want)
		}
	}
}

func TestMicroflowAdapter_Name(t *testing.T) {
	a := &MicroflowAdapter{}
	if a.Name() != "microflow" {
		t.Errorf("Name() = %q, want microflow", a.Name())
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./internal/mxgraph/adapter/mpr/... -run TestMicroflow -v
```

预期：`FAIL — MicroflowAdapter undefined`

- [ ] **Step 4: 实现 microflow.go**

```go
// internal/mxgraph/adapter/mpr/microflow.go
package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// MicroflowAdapter 发射 Microflow/Nanoflow 节点及 CALLS/CREATES/RETRIEVES/SHOWS_PAGE 边。
// BSON 类型名来自 reference/mendixmodellib/reflection-data/Microflows/
type MicroflowAdapter struct {
	Model *modelsdk.Model
}

func (a *MicroflowAdapter) Name() string { return "microflow" }

func (a *MicroflowAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Microflow", "Nanoflow"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"CALLS", "Microflow", "Microflow"},
			{"CALLS", "Microflow", "Nanoflow"},
			{"CALLS", "Nanoflow", "Nanoflow"},
			{"CREATES", "Microflow", "Entity"},
			{"RETRIEVES", "Microflow", "Entity"},
			{"SHOWS_PAGE", "Microflow", "Page"},
		},
	}
}

func (a *MicroflowAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}

		typeName := elem.TypeName()
		var label mxgraph.Label
		switch typeName {
		case "Microflows$Microflow":
			label = "Microflow"
		case "Microflows$Nanoflow":
			label = "Nanoflow"
		default:
			continue
		}

		mfNode := nodeForElement(elem, label)
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: mfNode})

		// 遍历 ObjectCollection → MicroflowObjects 提取活动引用
		for _, prop := range elem.Properties() {
			if prop.Name() != "ObjectCollection" {
				continue
			}
			cp, ok := prop.(element.ChildProperty)
			if !ok {
				continue
			}
			collection := cp.ChildElement()
			if collection == nil {
				continue
			}
			for _, objProp := range collection.Properties() {
				if objProp.Name() != "Objects" {
					continue
				}
				cl, ok := objProp.(element.ChildListProperty)
				if !ok {
					continue
				}
				for _, obj := range cl.ChildElements() {
					if obj == nil {
						continue
					}
					edges := extractActivityEdges(mfNode.ID, obj)
					events = append(events, edges...)
				}
			}
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// extractActivityEdges 从一个微流活动对象提取引用边。
// 已知 BSON 类型名（来自 CLAUDE.md 和 reflection-data）：
//   - MicroflowCallAction → CALLS 微流
//   - CreateChangeAction  → CREATES 实体
//   - RetrieveAction      → RETRIEVES 实体
//   - ShowFormAction      → SHOWS_PAGE 页面（"Form" 是旧称）
func extractActivityEdges(fromID mxgraph.NodeID, obj element.Element) []mxgraph.Event {
	var events []mxgraph.Event
	typeName := obj.TypeName()

	edgeID := func(suffix string) mxgraph.NodeID {
		return mxgraph.NodeID(fmt.Sprintf("%s--%s--%s", fromID, typeName, suffix))
	}

	switch {
	case strings.HasSuffix(typeName, "$MicroflowCallAction"):
		// MicroflowCall.MicroflowRef 是被调用的微流 ID
		for _, p := range obj.Properties() {
			if p.Name() != "MicroflowRef" {
				continue
			}
			if rp, ok := p.(element.WritableProperty); ok {
				if v := rp.BSONValue(); v != nil {
					targetID := fmt.Sprintf("%v", v)
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   edgeID(targetID),
							From: fromID,
							To:   mxgraph.NodeID(targetID),
							Type: "CALLS",
						},
					})
				}
			}
		}
	case strings.HasSuffix(typeName, "$CreateChangeAction"):
		// CreateChangeAction.Entity 是被创建的实体
		for _, p := range obj.Properties() {
			if p.Name() != "Entity" {
				continue
			}
			if rp, ok := p.(element.WritableProperty); ok {
				if v := rp.BSONValue(); v != nil {
					targetID := fmt.Sprintf("%v", v)
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   edgeID(targetID),
							From: fromID,
							To:   mxgraph.NodeID(targetID),
							Type: "CREATES",
						},
					})
				}
			}
		}
	case strings.HasSuffix(typeName, "$RetrieveAction"):
		// RetrieveAction 的实体引用在 Range.Entity 或直接 Entity 属性
		for _, p := range obj.Properties() {
			if p.Name() != "EntityRef" && p.Name() != "Entity" {
				continue
			}
			if rp, ok := p.(element.WritableProperty); ok {
				if v := rp.BSONValue(); v != nil {
					targetID := fmt.Sprintf("%v", v)
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   edgeID(targetID),
							From: fromID,
							To:   mxgraph.NodeID(targetID),
							Type: "RETRIEVES",
						},
					})
				}
			}
		}
	case strings.HasSuffix(typeName, "$ShowFormAction"):
		// ShowFormAction.FormToOpen 是被打开的页面
		for _, p := range obj.Properties() {
			if p.Name() != "FormToOpen" {
				continue
			}
			if rp, ok := p.(element.WritableProperty); ok {
				if v := rp.BSONValue(); v != nil {
					targetID := fmt.Sprintf("%v", v)
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   edgeID(targetID),
							From: fromID,
							To:   mxgraph.NodeID(targetID),
							Type: "SHOWS_PAGE",
						},
					})
				}
			}
		}
	}
	return events
}

func (a *MicroflowAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

**注意：** `extractActivityEdges` 中的属性名（`MicroflowRef`、`Entity`、`FormToOpen`）需要用真实 MPR 验证。如果有测试 MPR（`testdata/corpus-b/app.mpr`），在 Step 5 中验证。

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/mxgraph/adapter/mpr/... -v -count=1
```

预期：`TestMicroflowAdapter_Schema` 和 `TestMicroflowAdapter_Name` PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/adapter/mpr/microflow.go internal/mxgraph/adapter/mpr/microflow_test.go
git commit -m "feat(mxgraph/adapter): add MicroflowAdapter with CALLS/CREATES/RETRIEVES/SHOWS_PAGE edges"
```

---

## Task 6：PageAdapter / SecurityAdapter / EnumerationAdapter

**Files:**
- Create: `internal/mxgraph/adapter/mpr/page.go`
- Create: `internal/mxgraph/adapter/mpr/security.go`
- Create: `internal/mxgraph/adapter/mpr/enumeration.go`

三个适配器结构与 Task 5 完全一致，仅 BSON 类型名和 Props 不同。每个适配器：
1. 实现 `mxgraph.IndexAdapter` 接口（Name / Schema / Build / Watch）
2. 写 Name + Schema 测试（同 Task 5 模式）
3. 用 `strings.HasSuffix(typeName, "$XxxType")` 匹配单元

- [ ] **Step 1: 写三个适配器的 Schema 测试**

```go
// 追加到 domainmodel_test.go 或新建 adapters_test.go
func TestPageAdapter_Schema(t *testing.T) {
	a := &PageAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Page", "Layout", "Snippet"} {
		if !labels[want] {
			t.Errorf("PageAdapter.Schema missing label %q", want)
		}
	}
}

func TestSecurityAdapter_Schema(t *testing.T) {
	a := &SecurityAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Permission", "RoleMapping"} {
		if !labels[want] {
			t.Errorf("SecurityAdapter.Schema missing label %q", want)
		}
	}
}

func TestEnumerationAdapter_Schema(t *testing.T) {
	a := &EnumerationAdapter{}
	s := a.Schema()
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"Enumeration", "EnumValue"} {
		if !labels[want] {
			t.Errorf("EnumerationAdapter.Schema missing label %q", want)
		}
	}
}
```

- [ ] **Step 2: 实现 page.go**

```go
// internal/mxgraph/adapter/mpr/page.go
package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

type PageAdapter struct {
	Model *modelsdk.Model
}

func (a *PageAdapter) Name() string { return "page" }

func (a *PageAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Page", "Layout", "Snippet"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_LAYOUT", "Page", "Layout"},
			{"HAS_ACTION", "Page", "Microflow"},
			{"HAS_ACTION", "Page", "Nanoflow"},
			{"HAS_DATASOURCE", "Page", "Microflow"},
			{"HAS_DATASOURCE", "Page", "Entity"},
		},
	}
}

func (a *PageAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event
	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}
		typeName := elem.TypeName()
		var label mxgraph.Label
		switch {
		case strings.HasSuffix(typeName, "$Page") || strings.HasSuffix(typeName, "$Form"):
			label = "Page"
		case strings.HasSuffix(typeName, "$Layout"):
			label = "Layout"
		case strings.HasSuffix(typeName, "$Snippet"):
			label = "Snippet"
		default:
			continue
		}
		node := nodeForElement(elem, label)
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})

		// Layout 引用
		for _, p := range elem.Properties() {
			if p.Name() != "Layout" && p.Name() != "MasterLayout" {
				continue
			}
			if rp, ok := p.(element.WritableProperty); ok {
				if v := rp.BSONValue(); v != nil {
					targetID := fmt.Sprintf("%v", v)
					events = append(events, mxgraph.Event{
						Type: mxgraph.EdgeCreated,
						Edge: &mxgraph.Edge{
							ID:   mxgraph.NodeID(fmt.Sprintf("%s->layout->%s", node.ID, targetID)),
							From: node.ID,
							To:   mxgraph.NodeID(targetID),
							Type: "HAS_LAYOUT",
						},
					})
				}
			}
		}
	}
	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *PageAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 3: 实现 security.go**

```go
// internal/mxgraph/adapter/mpr/security.go
package mpr

import (
	"context"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
)

type SecurityAdapter struct {
	Model *modelsdk.Model
}

func (a *SecurityAdapter) Name() string { return "security" }

func (a *SecurityAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Permission", "RoleMapping"},
	}
}

func (a *SecurityAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event
	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}
		typeName := elem.TypeName()
		var label mxgraph.Label
		switch {
		case strings.Contains(typeName, "Security$EntityAccessRuleSet") ||
			strings.Contains(typeName, "Security$MemberAccessRule"):
			label = "Permission"
		case strings.Contains(typeName, "Security$UserRole"):
			label = "RoleMapping"
		default:
			continue
		}
		node := nodeForElement(elem, label)
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
	}
	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *SecurityAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 4: 实现 enumeration.go**

```go
// internal/mxgraph/adapter/mpr/enumeration.go
package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

type EnumerationAdapter struct {
	Model *modelsdk.Model
}

func (a *EnumerationAdapter) Name() string { return "enumeration" }

func (a *EnumerationAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Enumeration", "EnumValue"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_VALUE", "Enumeration", "EnumValue"},
		},
	}
}

func (a *EnumerationAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event
	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}
		if !strings.HasSuffix(elem.TypeName(), "$Enumeration") {
			continue
		}
		enumNode := nodeForElement(elem, "Enumeration")
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: enumNode})

		for _, prop := range elem.Properties() {
			if prop.Name() != "Values" {
				continue
			}
			cl, ok := prop.(element.ChildListProperty)
			if !ok {
				continue
			}
			for _, child := range cl.ChildElements() {
				if child == nil {
					continue
				}
				valNode := nodeForElement(child, "EnumValue")
				events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: valNode})
				events = append(events, mxgraph.Event{
					Type: mxgraph.EdgeCreated,
					Edge: &mxgraph.Edge{
						ID:   mxgraph.NodeID(fmt.Sprintf("%s->val->%s", enumNode.ID, valNode.ID)),
						From: enumNode.ID,
						To:   valNode.ID,
						Type: "HAS_VALUE",
					},
				})
			}
		}
	}
	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func (a *EnumerationAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 5: 运行所有适配器测试**

```bash
go test ./internal/mxgraph/adapter/mpr/... -v -count=1
```

预期：所有 Schema/Name 测试 PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/adapter/mpr/page.go internal/mxgraph/adapter/mpr/security.go internal/mxgraph/adapter/mpr/enumeration.go
git commit -m "feat(mxgraph/adapter): add PageAdapter, SecurityAdapter, EnumerationAdapter"
```

---

## Task 7：graphcatalog 接口定义和类型化节点

**Files:**
- Create: `mdl/graphcatalog/reader.go`
- Create: `mdl/graphcatalog/nodes.go`

- [ ] **Step 1: 写测试——接口可被 mock 实现（编译即测试）**

```go
// mdl/graphcatalog/reader_test.go
package graphcatalog_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
)

// 编译期检查：MockProjectGraph 实现 LintReader 和 TraversalReader
func TestInterfaceCompliance(t *testing.T) {
	var _ graphcatalog.LintReader = (*mock.MockProjectGraph)(nil)
	var _ graphcatalog.TraversalReader = (*mock.MockProjectGraph)(nil)
}
```

- [ ] **Step 2: 运行，确认失败**

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

预期：`FAIL — graphcatalog package not found`

- [ ] **Step 3: 创建 nodes.go**

```go
// mdl/graphcatalog/nodes.go
package graphcatalog

// EntityNode 对应 graph 中 label="Entity" 的节点。
type EntityNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	IsExternal    bool
}

// AttributeNode 对应 label="Attribute"。
type AttributeNode struct {
	ID       string
	Name     string
	DataType string
	Entity   string // 所属实体 QualifiedName
}

// AssociationNode 对应 label="Association"。
type AssociationNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	Owner         string // "Default" | "Both" | "Neither"
}

// MicroflowNode 对应 label="Microflow" 或 "Nanoflow"。
type MicroflowNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	IsNanoflow    bool
	ReturnType    string
}

// PageNode 对应 label="Page"。
type PageNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// SnippetNode 对应 label="Snippet"。
type SnippetNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// EnumerationNode 对应 label="Enumeration"。
type EnumerationNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// PermissionNode 对应 label="Permission"。
type PermissionNode struct {
	ID         string
	EntityName string
	ModuleRole string
	AccessRights string
}

// RoleMappingNode 对应 label="RoleMapping"。
type RoleMappingNode struct {
	ID         string
	UserRole   string
	ModuleRole string
}

// WidgetNode 对应 label="Widget"。
type WidgetNode struct {
	ID         string
	WidgetType string
	Name       string
	PageID     string
}

// DatabaseConnectionNode 对应 label="DatabaseConnection"。
type DatabaseConnectionNode struct {
	ID           string
	Name         string
	DatabaseType string
}

// CallEdge 表示一条调用关系（CALLS 边）。
type CallEdge struct {
	Caller string // QualifiedName
	Callee string // QualifiedName
	Depth  int    // transitive 遍历时的层级，直接调用为 1
}

// RefEdge 表示任意引用关系（CREATES / RETRIEVES / SHOWS_PAGE / HAS_ACTION 等）。
type RefEdge struct {
	Source  string // QualifiedName
	Target  string // QualifiedName
	RefKind string // "CALLS" | "CREATES" | "RETRIEVES" | "SHOWS_PAGE" | ...
}
```

- [ ] **Step 4: 创建 reader.go（接口定义）**

```go
// mdl/graphcatalog/reader.go
package graphcatalog

// DomainReader 读取域模型对象。linter 域模型规则使用。
type DomainReader interface {
	Entities(module string) []EntityNode
	Entity(qualifiedName string) *EntityNode
	Attributes(entityQualifiedName string) []AttributeNode
	Associations(module string) []AssociationNode
}

// BehaviorReader 读取行为对象（微流、页面、代码片段）。linter 行为规则使用。
type BehaviorReader interface {
	Microflows(module string) []MicroflowNode
	Microflow(qualifiedName string) *MicroflowNode
	Pages(module string) []PageNode
	Snippets(module string) []SnippetNode
	Enumerations(module string) []EnumerationNode
}

// SecurityReader 读取安全规则。linter 安全规则使用。
type SecurityReader interface {
	Permissions() []PermissionNode
	RoleMappings() []RoleMappingNode
}

// ExtensionReader 读取扩展对象（小部件、数据库连接）。linter 扩展规则使用。
type ExtensionReader interface {
	Widgets(pageQualifiedName string) []WidgetNode
	DatabaseConnections() []DatabaseConnectionNode
}

// TraversalReader 执行图遍历。executor search 命令使用。
type TraversalReader interface {
	Callers(qualifiedName string, transitive bool) []CallEdge
	Callees(qualifiedName string, transitive bool) []CallEdge
	Impact(qualifiedName string) []RefEdge
	References(qualifiedName string) []RefEdge
}

// LintReader 是 linter 所需的完整接口（聚合 4 个子接口）。
type LintReader interface {
	DomainReader
	BehaviorReader
	SecurityReader
	ExtensionReader
}
```

- [ ] **Step 5: 创建 mock.go（让 Task 7 Step 1 的测试通过）**

```go
// mdl/graphcatalog/mock/mock.go
package mock

import "github.com/mendixlabs/mxcli/mdl/graphcatalog"

// MockProjectGraph 实现 graphcatalog.LintReader 和 graphcatalog.TraversalReader。
// 每个方法对应一个 Func 字段；未配置时 panic 给出明确错误。
type MockProjectGraph struct {
	EntitiesFunc            func(module string) []graphcatalog.EntityNode
	EntityFunc              func(qualifiedName string) *graphcatalog.EntityNode
	AttributesFunc          func(entityQN string) []graphcatalog.AttributeNode
	AssociationsFunc        func(module string) []graphcatalog.AssociationNode
	MicroflowsFunc          func(module string) []graphcatalog.MicroflowNode
	MicroflowFunc           func(qualifiedName string) *graphcatalog.MicroflowNode
	PagesFunc               func(module string) []graphcatalog.PageNode
	SnippetsFunc            func(module string) []graphcatalog.SnippetNode
	EnumerationsFunc        func(module string) []graphcatalog.EnumerationNode
	PermissionsFunc         func() []graphcatalog.PermissionNode
	RoleMappingsFunc        func() []graphcatalog.RoleMappingNode
	WidgetsFunc             func(pageQN string) []graphcatalog.WidgetNode
	DatabaseConnectionsFunc func() []graphcatalog.DatabaseConnectionNode
	CallersFunc             func(qualifiedName string, transitive bool) []graphcatalog.CallEdge
	CalleesFunc             func(qualifiedName string, transitive bool) []graphcatalog.CallEdge
	ImpactFunc              func(qualifiedName string) []graphcatalog.RefEdge
	ReferencesFunc          func(qualifiedName string) []graphcatalog.RefEdge
}

// 编译期接口检查
var _ graphcatalog.LintReader = (*MockProjectGraph)(nil)
var _ graphcatalog.TraversalReader = (*MockProjectGraph)(nil)

func (m *MockProjectGraph) Entities(module string) []graphcatalog.EntityNode {
	if m.EntitiesFunc != nil {
		return m.EntitiesFunc(module)
	}
	panic("MockProjectGraph.Entities not configured")
}
func (m *MockProjectGraph) Entity(qn string) *graphcatalog.EntityNode {
	if m.EntityFunc != nil {
		return m.EntityFunc(qn)
	}
	panic("MockProjectGraph.Entity not configured")
}
func (m *MockProjectGraph) Attributes(entityQN string) []graphcatalog.AttributeNode {
	if m.AttributesFunc != nil {
		return m.AttributesFunc(entityQN)
	}
	panic("MockProjectGraph.Attributes not configured")
}
func (m *MockProjectGraph) Associations(module string) []graphcatalog.AssociationNode {
	if m.AssociationsFunc != nil {
		return m.AssociationsFunc(module)
	}
	panic("MockProjectGraph.Associations not configured")
}
func (m *MockProjectGraph) Microflows(module string) []graphcatalog.MicroflowNode {
	if m.MicroflowsFunc != nil {
		return m.MicroflowsFunc(module)
	}
	panic("MockProjectGraph.Microflows not configured")
}
func (m *MockProjectGraph) Microflow(qn string) *graphcatalog.MicroflowNode {
	if m.MicroflowFunc != nil {
		return m.MicroflowFunc(qn)
	}
	panic("MockProjectGraph.Microflow not configured")
}
func (m *MockProjectGraph) Pages(module string) []graphcatalog.PageNode {
	if m.PagesFunc != nil {
		return m.PagesFunc(module)
	}
	panic("MockProjectGraph.Pages not configured")
}
func (m *MockProjectGraph) Snippets(module string) []graphcatalog.SnippetNode {
	if m.SnippetsFunc != nil {
		return m.SnippetsFunc(module)
	}
	panic("MockProjectGraph.Snippets not configured")
}
func (m *MockProjectGraph) Enumerations(module string) []graphcatalog.EnumerationNode {
	if m.EnumerationsFunc != nil {
		return m.EnumerationsFunc(module)
	}
	panic("MockProjectGraph.Enumerations not configured")
}
func (m *MockProjectGraph) Permissions() []graphcatalog.PermissionNode {
	if m.PermissionsFunc != nil {
		return m.PermissionsFunc()
	}
	panic("MockProjectGraph.Permissions not configured")
}
func (m *MockProjectGraph) RoleMappings() []graphcatalog.RoleMappingNode {
	if m.RoleMappingsFunc != nil {
		return m.RoleMappingsFunc()
	}
	panic("MockProjectGraph.RoleMappings not configured")
}
func (m *MockProjectGraph) Widgets(pageQN string) []graphcatalog.WidgetNode {
	if m.WidgetsFunc != nil {
		return m.WidgetsFunc(pageQN)
	}
	panic("MockProjectGraph.Widgets not configured")
}
func (m *MockProjectGraph) DatabaseConnections() []graphcatalog.DatabaseConnectionNode {
	if m.DatabaseConnectionsFunc != nil {
		return m.DatabaseConnectionsFunc()
	}
	panic("MockProjectGraph.DatabaseConnections not configured")
}
func (m *MockProjectGraph) Callers(qn string, transitive bool) []graphcatalog.CallEdge {
	if m.CallersFunc != nil {
		return m.CallersFunc(qn, transitive)
	}
	panic("MockProjectGraph.Callers not configured")
}
func (m *MockProjectGraph) Callees(qn string, transitive bool) []graphcatalog.CallEdge {
	if m.CalleesFunc != nil {
		return m.CalleesFunc(qn, transitive)
	}
	panic("MockProjectGraph.Callees not configured")
}
func (m *MockProjectGraph) Impact(qn string) []graphcatalog.RefEdge {
	if m.ImpactFunc != nil {
		return m.ImpactFunc(qn)
	}
	panic("MockProjectGraph.Impact not configured")
}
func (m *MockProjectGraph) References(qn string) []graphcatalog.RefEdge {
	if m.ReferencesFunc != nil {
		return m.ReferencesFunc(qn)
	}
	panic("MockProjectGraph.References not configured")
}
```

- [ ] **Step 6: 运行测试**

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

预期：`TestInterfaceCompliance` PASS（编译期接口检查通过）。

- [ ] **Step 7: Commit**

```bash
git add mdl/graphcatalog/
git commit -m "feat(graphcatalog): add reader interfaces, typed nodes, and mock implementation"
```

---

## Task 8：ProjectGraph 实现

**Files:**
- Create: `mdl/graphcatalog/graph.go`
- Create: `mdl/graphcatalog/graph_test.go`

- [ ] **Step 1: 写失败测试（用内存图，不依赖 MPR）**

```go
// mdl/graphcatalog/graph_test.go
package graphcatalog_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

func buildTestProjectGraph() *graphcatalog.ProjectGraph {
	mgr := mxgraph.NewIndexManager()
	// 手动填充图节点
	g := mgr.Query()
	g.AddNode("e1", "Entity", map[string]any{
		"Name":   "Ticket",
		"Module": "Helpdesk",
		"QualifiedName": "Helpdesk.Ticket",
	})
	g.AddNode("e2", "Entity", map[string]any{
		"Name":   "Agent",
		"Module": "Helpdesk",
		"QualifiedName": "Helpdesk.Agent",
	})
	g.AddNode("mf1", "Microflow", map[string]any{
		"Name":          "ACT_CreateTicket",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.ACT_CreateTicket",
	})
	g.AddNode("mf2", "Microflow", map[string]any{
		"Name":          "ACT_AssignAgent",
		"Module":        "Helpdesk",
		"QualifiedName": "Helpdesk.ACT_AssignAgent",
	})
	g.AddEdge("call1", "mf2", "mf1", "CALLS", nil)
	g.AddEdge("creates1", "mf1", "e1", "CREATES", nil)
	return graphcatalog.NewProjectGraph(mgr)
}

func TestProjectGraph_Entities(t *testing.T) {
	pg := buildTestProjectGraph()
	entities := pg.Entities("Helpdesk")
	if len(entities) != 2 {
		t.Fatalf("Entities(Helpdesk) = %d, want 2", len(entities))
	}
}

func TestProjectGraph_Entity(t *testing.T) {
	pg := buildTestProjectGraph()
	e := pg.Entity("Helpdesk.Ticket")
	if e == nil {
		t.Fatal("Entity(Helpdesk.Ticket) returned nil")
	}
	if e.Name != "Ticket" {
		t.Errorf("Name = %q, want Ticket", e.Name)
	}
}

func TestProjectGraph_Callers_direct(t *testing.T) {
	pg := buildTestProjectGraph()
	callers := pg.Callers("Helpdesk.ACT_CreateTicket", false)
	if len(callers) != 1 {
		t.Fatalf("Callers(ACT_CreateTicket) = %d, want 1", len(callers))
	}
	if callers[0].Caller != "Helpdesk.ACT_AssignAgent" {
		t.Errorf("Caller = %q, want Helpdesk.ACT_AssignAgent", callers[0].Caller)
	}
}

func TestProjectGraph_References(t *testing.T) {
	pg := buildTestProjectGraph()
	refs := pg.References("Helpdesk.Ticket")
	if len(refs) == 0 {
		t.Fatal("References(Ticket) returned empty")
	}
	found := false
	for _, r := range refs {
		if r.RefKind == "CREATES" {
			found = true
		}
	}
	if !found {
		t.Error("expected CREATES ref for Ticket")
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/graphcatalog/... -run TestProjectGraph -v
```

预期：`FAIL — NewProjectGraph undefined`

- [ ] **Step 3: 实现 graph.go**

```go
// mdl/graphcatalog/graph.go
package graphcatalog

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// ProjectGraph 实现 LintReader 和 TraversalReader，封装 mxgraph.IndexManager。
type ProjectGraph struct {
	mgr *mxgraph.IndexManager
}

// 编译期接口检查
var _ LintReader = (*ProjectGraph)(nil)
var _ TraversalReader = (*ProjectGraph)(nil)

// NewProjectGraph 创建 ProjectGraph，接管已构建的 IndexManager。
func NewProjectGraph(mgr *mxgraph.IndexManager) *ProjectGraph {
	return &ProjectGraph{mgr: mgr}
}

// g 是内部便捷访问器。
func (pg *ProjectGraph) g() *mxgraph.Graph {
	return pg.mgr.Query()
}

// nodeToQN 从节点 Props 提取 QualifiedName，回退到 Name。
func nodeToQN(n *mxgraph.Node) string {
	if qn, ok := n.Props["QualifiedName"].(string); ok && qn != "" {
		return qn
	}
	if name, ok := n.Props["Name"].(string); ok {
		return name
	}
	return string(n.ID)
}

// ── DomainReader ──────────────────────────────────────────────

func (pg *ProjectGraph) Entities(module string) []EntityNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes("Entity", filter)
	result := make([]EntityNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, entityFromNode(n))
	}
	return result
}

func (pg *ProjectGraph) Entity(qualifiedName string) *EntityNode {
	nodes := pg.g().FindNodes("Entity", map[string]any{"QualifiedName": qualifiedName})
	if len(nodes) == 0 {
		return nil
	}
	e := entityFromNode(nodes[0])
	return &e
}

func (pg *ProjectGraph) Attributes(entityQN string) []AttributeNode {
	entityNodes := pg.g().FindNodes("Entity", map[string]any{"QualifiedName": entityQN})
	if len(entityNodes) == 0 {
		return nil
	}
	attrNodes := pg.g().Neighbors(entityNodes[0].ID, "HAS_ATTRIBUTE")
	result := make([]AttributeNode, 0, len(attrNodes))
	for _, n := range attrNodes {
		result = append(result, AttributeNode{
			ID:       string(n.ID),
			Name:     strProp(n, "Name"),
			DataType: strProp(n, "DataType"),
			Entity:   entityQN,
		})
	}
	return result
}

func (pg *ProjectGraph) Associations(module string) []AssociationNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes("Association", filter)
	result := make([]AssociationNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, AssociationNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
			Owner:         strProp(n, "Owner"),
		})
	}
	return result
}

// ── BehaviorReader ────────────────────────────────────────────

func (pg *ProjectGraph) Microflows(module string) []MicroflowNode {
	result := pg.microflowsByLabel("Microflow", module)
	result = append(result, pg.microflowsByLabel("Nanoflow", module)...)
	return result
}

func (pg *ProjectGraph) microflowsByLabel(label mxgraph.Label, module string) []MicroflowNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes(label, filter)
	result := make([]MicroflowNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, MicroflowNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
			IsNanoflow:    label == "Nanoflow",
			ReturnType:    strProp(n, "ReturnType"),
		})
	}
	return result
}

func (pg *ProjectGraph) Microflow(qn string) *MicroflowNode {
	for _, label := range []mxgraph.Label{"Microflow", "Nanoflow"} {
		nodes := pg.g().FindNodes(label, map[string]any{"QualifiedName": qn})
		if len(nodes) > 0 {
			mf := MicroflowNode{
				ID:            string(nodes[0].ID),
				Name:          strProp(nodes[0], "Name"),
				QualifiedName: nodeToQN(nodes[0]),
				Module:        strProp(nodes[0], "Module"),
				IsNanoflow:    label == "Nanoflow",
			}
			return &mf
		}
	}
	return nil
}

func (pg *ProjectGraph) Pages(module string) []PageNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes("Page", filter)
	result := make([]PageNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, PageNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

func (pg *ProjectGraph) Snippets(module string) []SnippetNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes("Snippet", filter)
	result := make([]SnippetNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, SnippetNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

func (pg *ProjectGraph) Enumerations(module string) []EnumerationNode {
	var filter map[string]any
	if module != "" {
		filter = map[string]any{"Module": module}
	}
	nodes := pg.g().FindNodes("Enumeration", filter)
	result := make([]EnumerationNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, EnumerationNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

// ── SecurityReader ────────────────────────────────────────────

func (pg *ProjectGraph) Permissions() []PermissionNode {
	nodes := pg.g().FindNodes("Permission", nil)
	result := make([]PermissionNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, PermissionNode{
			ID:           string(n.ID),
			EntityName:   strProp(n, "EntityName"),
			ModuleRole:   strProp(n, "ModuleRole"),
			AccessRights: strProp(n, "AccessRights"),
		})
	}
	return result
}

func (pg *ProjectGraph) RoleMappings() []RoleMappingNode {
	nodes := pg.g().FindNodes("RoleMapping", nil)
	result := make([]RoleMappingNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, RoleMappingNode{
			ID:         string(n.ID),
			UserRole:   strProp(n, "UserRole"),
			ModuleRole: strProp(n, "ModuleRole"),
		})
	}
	return result
}

// ── ExtensionReader ───────────────────────────────────────────

func (pg *ProjectGraph) Widgets(pageQN string) []WidgetNode {
	pageNodes := pg.g().FindNodes("Page", map[string]any{"QualifiedName": pageQN})
	if len(pageNodes) == 0 {
		return nil
	}
	widgetNodes := pg.g().Neighbors(pageNodes[0].ID, "HAS_WIDGET")
	result := make([]WidgetNode, 0, len(widgetNodes))
	for _, n := range widgetNodes {
		result = append(result, WidgetNode{
			ID:         string(n.ID),
			WidgetType: strProp(n, "WidgetType"),
			Name:       strProp(n, "Name"),
			PageID:     string(pageNodes[0].ID),
		})
	}
	return result
}

func (pg *ProjectGraph) DatabaseConnections() []DatabaseConnectionNode {
	nodes := pg.g().FindNodes("DatabaseConnection", nil)
	result := make([]DatabaseConnectionNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, DatabaseConnectionNode{
			ID:           string(n.ID),
			Name:         strProp(n, "Name"),
			DatabaseType: strProp(n, "DatabaseType"),
		})
	}
	return result
}

// ── TraversalReader ───────────────────────────────────────────

func (pg *ProjectGraph) Callers(qualifiedName string, transitive bool) []CallEdge {
	target := pg.findNodeByQN(qualifiedName)
	if target == nil {
		return nil
	}
	if !transitive {
		edges := pg.g().Edges(target.ID, mxgraph.Inbound, "CALLS")
		result := make([]CallEdge, 0, len(edges))
		for _, e := range edges {
			caller := pg.g().GetNode(e.From)
			if caller == nil {
				continue
			}
			result = append(result, CallEdge{
				Caller: nodeToQN(caller),
				Callee: qualifiedName,
				Depth:  1,
			})
		}
		return result
	}
	// Transitive: BFS 反向遍历 CALLS 边
	return pg.bfsCallers(target.ID, qualifiedName)
}

func (pg *ProjectGraph) bfsCallers(targetID mxgraph.NodeID, targetQN string) []CallEdge {
	visited := map[mxgraph.NodeID]bool{targetID: true}
	var result []CallEdge
	type item struct {
		id    mxgraph.NodeID
		depth int
	}
	queue := []item{{targetID, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		edges := pg.g().Edges(cur.id, mxgraph.Inbound, "CALLS")
		for _, e := range edges {
			if visited[e.From] {
				continue
			}
			visited[e.From] = true
			caller := pg.g().GetNode(e.From)
			if caller == nil {
				continue
			}
			result = append(result, CallEdge{
				Caller: nodeToQN(caller),
				Callee: targetQN,
				Depth:  cur.depth + 1,
			})
			queue = append(queue, item{e.From, cur.depth + 1})
		}
	}
	return result
}

func (pg *ProjectGraph) Callees(qualifiedName string, transitive bool) []CallEdge {
	source := pg.findNodeByQN(qualifiedName)
	if source == nil {
		return nil
	}
	if !transitive {
		edges := pg.g().Edges(source.ID, mxgraph.Outbound, "CALLS")
		result := make([]CallEdge, 0, len(edges))
		for _, e := range edges {
			callee := pg.g().GetNode(e.To)
			if callee == nil {
				continue
			}
			result = append(result, CallEdge{
				Caller: qualifiedName,
				Callee: nodeToQN(callee),
				Depth:  1,
			})
		}
		return result
	}
	return pg.bfsCallees(source.ID, qualifiedName)
}

func (pg *ProjectGraph) bfsCallees(sourceID mxgraph.NodeID, sourceQN string) []CallEdge {
	visited := map[mxgraph.NodeID]bool{sourceID: true}
	var result []CallEdge
	type item struct {
		id    mxgraph.NodeID
		depth int
	}
	queue := []item{{sourceID, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		edges := pg.g().Edges(cur.id, mxgraph.Outbound, "CALLS")
		for _, e := range edges {
			if visited[e.To] {
				continue
			}
			visited[e.To] = true
			callee := pg.g().GetNode(e.To)
			if callee == nil {
				continue
			}
			result = append(result, CallEdge{
				Caller: sourceQN,
				Callee: nodeToQN(callee),
				Depth:  cur.depth + 1,
			})
			queue = append(queue, item{e.To, cur.depth + 1})
		}
	}
	return result
}

func (pg *ProjectGraph) Impact(qualifiedName string) []RefEdge {
	// Impact = 所有指向 target 的边（任意 RelType）
	target := pg.findNodeByQN(qualifiedName)
	if target == nil {
		return nil
	}
	edges := pg.g().Edges(target.ID, mxgraph.Inbound)
	result := make([]RefEdge, 0, len(edges))
	for _, e := range edges {
		src := pg.g().GetNode(e.From)
		if src == nil {
			continue
		}
		result = append(result, RefEdge{
			Source:  nodeToQN(src),
			Target:  qualifiedName,
			RefKind: string(e.Type),
		})
	}
	return result
}

func (pg *ProjectGraph) References(qualifiedName string) []RefEdge {
	// References = target 发出的所有边
	source := pg.findNodeByQN(qualifiedName)
	if source == nil {
		return nil
	}
	edges := pg.g().Edges(source.ID, mxgraph.Outbound)
	result := make([]RefEdge, 0, len(edges))
	for _, e := range edges {
		tgt := pg.g().GetNode(e.To)
		if tgt == nil {
			continue
		}
		result = append(result, RefEdge{
			Source:  qualifiedName,
			Target:  nodeToQN(tgt),
			RefKind: string(e.Type),
		})
	}
	return result
}

// ── 内部辅助 ─────────────────────────────────────────────────

// findNodeByQN 按 QualifiedName 属性查找节点（线性搜索所有 label）。
// 性能可接受：查询频率低，且 propIdx 已覆盖 "QualifiedName" 属性。
func (pg *ProjectGraph) findNodeByQN(qn string) *mxgraph.Node {
	filter := map[string]any{"QualifiedName": qn}
	for _, label := range []mxgraph.Label{"Microflow", "Nanoflow", "Entity", "Page", "Snippet", "Enumeration", "Layout"} {
		nodes := pg.g().FindNodes(label, filter)
		if len(nodes) > 0 {
			return nodes[0]
		}
	}
	return nil
}

func strProp(n *mxgraph.Node, key string) string {
	if v, ok := n.Props[key].(string); ok {
		return v
	}
	return ""
}

func entityFromNode(n *mxgraph.Node) EntityNode {
	isExt, _ := n.Props["IsExternal"].(bool)
	return EntityNode{
		ID:            string(n.ID),
		Name:          strProp(n, "Name"),
		QualifiedName: nodeToQN(n),
		Module:        strProp(n, "Module"),
		IsExternal:    isExt,
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

预期：`TestProjectGraph_Entities`、`TestProjectGraph_Entity`、`TestProjectGraph_Callers_direct`、`TestProjectGraph_References` 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add mdl/graphcatalog/graph.go mdl/graphcatalog/graph_test.go
git commit -m "feat(graphcatalog): implement ProjectGraph with all reader interface methods"
```

---

## Task 9：持久化（gob snapshot 替换 catalog.db）

**Files:**
- Create: `mdl/graphcatalog/persist.go`

- [ ] **Step 1: 写 roundtrip 测试**

在 `mdl/graphcatalog/graph_test.go` 末尾追加：

```go
func TestProjectGraph_Persist_Roundtrip(t *testing.T) {
	pg := buildTestProjectGraph()

	// 序列化
	data, err := pg.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("MarshalSnapshot returned empty data")
	}

	// 反序列化
	pg2, err := UnmarshalSnapshot(data)
	if err != nil {
		t.Fatalf("UnmarshalSnapshot: %v", err)
	}

	entities := pg2.Entities("Helpdesk")
	if len(entities) != 2 {
		t.Fatalf("after roundtrip, Entities(Helpdesk) = %d, want 2", len(entities))
	}
	callers := pg2.Callers("Helpdesk.ACT_CreateTicket", false)
	if len(callers) != 1 {
		t.Fatalf("after roundtrip, Callers = %d, want 1", len(callers))
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/graphcatalog/... -run TestProjectGraph_Persist -v
```

预期：`FAIL — MarshalSnapshot undefined`

- [ ] **Step 3: 实现 persist.go**

```go
// mdl/graphcatalog/persist.go
package graphcatalog

import (
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// MarshalSnapshot 将图序列化为 gob 二进制（委托给 mxgraph.MarshalSnapshot）。
func (pg *ProjectGraph) MarshalSnapshot() ([]byte, error) {
	return mxgraph.MarshalSnapshot(pg.mgr.Query())
}

// UnmarshalSnapshot 从 gob 二进制恢复 ProjectGraph。
func UnmarshalSnapshot(data []byte) (*ProjectGraph, error) {
	g, err := mxgraph.UnmarshalSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("graphcatalog: unmarshal snapshot: %w", err)
	}
	// 用已恢复的图创建一个只读 IndexManager
	mgr := mxgraph.NewIndexManagerFromGraph(g)
	return &ProjectGraph{mgr: mgr}, nil
}

// SnapshotPath 返回给定项目目录下 graph.gob 的标准路径。
func SnapshotPath(projectDir string) string {
	return projectDir + "/.mxcli/graph.gob"
}
```

**注意：** `mxgraph.NewIndexManagerFromGraph(g)` 需要在 `mxgraph/adapter.go` 中新增：

```go
// NewIndexManagerFromGraph 从已有图创建只读 IndexManager（用于从 snapshot 恢复）。
func NewIndexManagerFromGraph(g *Graph) *IndexManager {
	return &IndexManager{
		graph:    g,
		adapters: map[string]IndexAdapter{},
	}
}
```

在 `internal/mxgraph/adapter.go` 末尾追加这个函数。

- [ ] **Step 4: 运行测试**

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

预期：所有测试 PASS，包括 `TestProjectGraph_Persist_Roundtrip`。

- [ ] **Step 5: Commit**

```bash
git add mdl/graphcatalog/persist.go internal/mxgraph/adapter.go
git commit -m "feat(graphcatalog): add gob snapshot persist, replacing catalog.db"
```

---

## Task 10：迁移 linter/context.go

**Files:**
- Modify: `mdl/linter/context.go`

- [ ] **Step 1: 在 context.go 顶部添加新依赖字段**

读取 `mdl/linter/context.go` 当前内容，然后：

1. 将 import 中的 `"github.com/mendixlabs/mxcli/mdl/catalog"` 替换为 `"github.com/mendixlabs/mxcli/mdl/graphcatalog"`
2. 将 struct 字段从 `catalog *catalog.Catalog` + `db catalog.CatalogDB` 替换为 `reader graphcatalog.LintReader`
3. 将 `NewLintContext`、`NewLintContextFromDB`、`NewLintContextFromDBAndReader` 替换为：

```go
// NewLintContext 创建 LintContext，接受 graphcatalog.LintReader。
func NewLintContext(reader graphcatalog.LintReader) *LintContext {
	return &LintContext{reader: reader}
}
```

4. 将所有 `ctx.db.Query("SELECT ... FROM entities ...")` 类方法替换为调用 `ctx.reader.Entities(...)` 等：

```go
// Entities 返回指定模块的所有实体。
func (ctx *LintContext) Entities(module string) []graphcatalog.EntityNode {
	return ctx.reader.Entities(module)
}

// Entity 按 qualified name 返回实体。
func (ctx *LintContext) Entity(qualifiedName string) *graphcatalog.EntityNode {
	return ctx.reader.Entity(qualifiedName)
}

// Attributes 返回实体的所有属性。
func (ctx *LintContext) Attributes(entityQN string) []graphcatalog.AttributeNode {
	return ctx.reader.Attributes(entityQN)
}

// Microflows 返回指定模块的所有微流。
func (ctx *LintContext) Microflows(module string) []graphcatalog.MicroflowNode {
	return ctx.reader.Microflows(module)
}

// Pages 返回指定模块的所有页面。
func (ctx *LintContext) Pages(module string) []graphcatalog.PageNode {
	return ctx.reader.Pages(module)
}

// Permissions 返回所有权限规则。
func (ctx *LintContext) Permissions() []graphcatalog.PermissionNode {
	return ctx.reader.Permissions()
}

// RoleMappings 返回所有角色映射。
func (ctx *LintContext) RoleMappings() []graphcatalog.RoleMappingNode {
	return ctx.reader.RoleMappings()
}

// Enumerations 返回指定模块的所有枚举。
func (ctx *LintContext) Enumerations(module string) []graphcatalog.EnumerationNode {
	return ctx.reader.Enumerations(module)
}

// Snippets 返回指定模块的所有代码片段。
func (ctx *LintContext) Snippets(module string) []graphcatalog.SnippetNode {
	return ctx.reader.Snippets(module)
}

// DatabaseConnections 返回所有数据库连接。
func (ctx *LintContext) DatabaseConnections() []graphcatalog.DatabaseConnectionNode {
	return ctx.reader.DatabaseConnections()
}
```

5. 删除所有原有的 SQL 查询方法和 Entity/Attribute/Microflow/Page 等旧 struct 定义（这些现在由 graphcatalog.EntityNode 等替代）。

- [ ] **Step 2: 修复所有因 linter/context.go 接口变化导致的编译错误**

```bash
go build ./mdl/linter/... 2>&1 | head -30
```

逐一修复每个 linter rule 文件中对旧 `ctx.Entity`（返回旧 struct）的调用，改为使用 `graphcatalog.EntityNode` 的字段。

- [ ] **Step 3: 运行 linter 测试**

```bash
go test ./mdl/linter/... -v -count=1 2>&1 | tail -20
```

预期：所有 linter 测试 PASS（linter rule 测试可以使用 `mock.MockProjectGraph`）。

- [ ] **Step 4: Commit**

```bash
git add mdl/linter/
git commit -m "refactor(linter): replace catalog.Catalog with graphcatalog.LintReader interface"
```

---

## Task 11：迁移 executor（exec_context + cmd_search + cmd_catalog）

**Files:**
- Modify: `mdl/executor/exec_context.go`
- Modify: `mdl/executor/cmd_search.go`
- Modify: `mdl/executor/cmd_catalog.go`
- Modify: `mdl/executor/executor.go`
- Modify: `mdl/executor/executor_dispatch.go`

- [ ] **Step 1: 更新 exec_context.go**

在 `exec_context.go`（第 57 行）中：
1. 将 `Catalog *catalog.Catalog` 替换为 `GraphCatalog graphcatalog.TraversalReader`
2. 将 `SyncCatalog func(*catalog.Catalog)` 替换为 `SyncGraphCatalog func(*graphcatalog.ProjectGraph)`
3. 更新 import

```go
// mdl/executor/exec_context.go（相关字段）
import (
    "github.com/mendixlabs/mxcli/mdl/graphcatalog"
    // 删除 mdl/catalog import
)

type ExecContext struct {
    // ... 其他字段不变 ...
    GraphCatalog    graphcatalog.TraversalReader
    SyncGraphCatalog func(*graphcatalog.ProjectGraph)
}
```

- [ ] **Step 2: 更新 cmd_search.go**

将 `execShowCallers`、`execShowCallees`、`execShowReferences`、`execShowImpact` 从 SQL 查询改为调用 `ctx.GraphCatalog`：

```go
// 示例：execShowCallers
func execShowCallers(ctx *ExecContext, stmt *ast.ShowCallersStmt) error {
	if ctx.GraphCatalog == nil {
		return fmt.Errorf("graph catalog not built — run: refresh graph full")
	}
	callers := ctx.GraphCatalog.Callers(stmt.Name, stmt.Transitive)
	if len(callers) == 0 {
		ctx.Println("(no callers found)")
		return nil
	}
	w := tabwriter.NewWriter(ctx.Writer(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CALLER\tDEPTH")
	for _, c := range callers {
		fmt.Fprintf(w, "%s\t%d\n", c.Caller, c.Depth)
	}
	return w.Flush()
}
```

按相同模式实现其余三个函数（Callees/References/Impact），分别调用 `ctx.GraphCatalog.Callees`、`ctx.GraphCatalog.References`、`ctx.GraphCatalog.Impact`。

- [ ] **Step 3: 更新 cmd_catalog.go**

1. 删除 `execShowCatalogTables`、`execDescribeCatalogTable`、`execCatalogQuery`（`SELECT FROM CATALOG.xxx` 支持）
2. 将 `execRefreshCatalogStmt` 改为 `execRefreshGraphStmt`，构建 `ProjectGraph` 而非 `*catalog.Catalog`：

```go
func execRefreshGraphStmt(ctx *ExecContext, stmt *ast.RefreshCatalogStmt) error {
	mgr := mxgraph.NewIndexManager()
	// 注册所有 5 个适配器
	mgr.RegisterAdapter(&mpradapter.DomainModelAdapter{Model: ctx.Model})
	mgr.RegisterAdapter(&mpradapter.MicroflowAdapter{Model: ctx.Model})
	mgr.RegisterAdapter(&mpradapter.PageAdapter{Model: ctx.Model})
	mgr.RegisterAdapter(&mpradapter.SecurityAdapter{Model: ctx.Model})
	mgr.RegisterAdapter(&mpradapter.EnumerationAdapter{Model: ctx.Model})

	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := mgr.BuildAll(buildCtx); err != nil {
		return fmt.Errorf("refresh graph: %w", err)
	}

	pg := graphcatalog.NewProjectGraph(mgr)

	// 持久化到 .mxcli/graph.gob
	if ctx.ProjectDir != "" {
		data, err := pg.MarshalSnapshot()
		if err == nil {
			snapPath := graphcatalog.SnapshotPath(ctx.ProjectDir)
			_ = os.MkdirAll(filepath.Dir(snapPath), 0700)
			_ = os.WriteFile(snapPath, data, 0600)
		}
	}

	ctx.GraphCatalog = pg
	if ctx.SyncGraphCatalog != nil {
		ctx.SyncGraphCatalog(pg)
	}
	ctx.Printf("Graph built: %d nodes, %d edges\n",
		len(mgr.Query().AllNodes()), len(mgr.Query().AllEdges()))
	return nil
}
```

- [ ] **Step 4: 修复编译错误并运行测试**

```bash
go build ./mdl/executor/... 2>&1 | head -30
go test ./mdl/executor/... -count=1 2>&1 | tail -20
```

逐一修复编译错误，主要是移除 `catalog.*` 引用，替换为 `graphcatalog.*`。

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/
git commit -m "refactor(executor): replace catalog.Catalog with graphcatalog.ProjectGraph"
```

---

## Task 12：迁移 serve.go

**Files:**
- Modify: `cmd/mxcli/serve.go`

- [ ] **Step 1: 替换 buildCatalog 为 buildProjectGraph**

```go
// cmd/mxcli/serve.go
func buildProjectGraph(projectPath string) (*graphcatalog.ProjectGraph, error) {
	be, err := modelsdk.Open(projectPath)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	defer be.Close()

	mgr := mxgraph.NewIndexManager()
	mgr.RegisterAdapter(&mpradapter.DomainModelAdapter{Model: be})
	mgr.RegisterAdapter(&mpradapter.MicroflowAdapter{Model: be})
	mgr.RegisterAdapter(&mpradapter.PageAdapter{Model: be})
	mgr.RegisterAdapter(&mpradapter.EnumerationAdapter{Model: be})

	if err := mgr.BuildAll(context.Background()); err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}
	return graphcatalog.NewProjectGraph(mgr), nil
}
```

- [ ] **Step 2: 更新 getModuleStats 使用 ProjectGraph**

将 `getModuleStats(cat *catalog.Catalog)` 改为 `getModuleStats(pg *graphcatalog.ProjectGraph)`，用 `pg.Microflows("")` 和 `pg.Pages("")` 等替代 SQL 查询。

- [ ] **Step 3: 运行编译**

```bash
go build ./cmd/mxcli/... 2>&1
```

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli/serve.go
git commit -m "refactor(serve): replace buildCatalog with buildProjectGraph using mxgraph"
```

---

## Task 13：删除 mdl/catalog/（Phase 7）

**前置条件：** Task 10、11、12 全部完成且测试通过；`make test` 无 FAIL。

- [ ] **Step 1: 确认 catalog 无引用**

```bash
grep -r '"github.com/mendixlabs/mxcli/mdl/catalog"' . --include="*.go" | grep -v "_test.go"
```

预期：**无输出**（零引用）。

- [ ] **Step 2: 删除整包**

```bash
rm -rf /mnt/data_sdd/gh/mxcli-wt-02/mdl/catalog/
```

- [ ] **Step 3: 运行全量测试**

```bash
make test 2>&1 | tail -20
```

预期：`ok` 全部通过，无 `FAIL`。

- [ ] **Step 4: 运行全量 benchmark 对比**

```bash
go test ./internal/mxgraph/ -run=^$ -bench=. -benchtime=3s -count=1
```

确认 `BenchmarkEdges_Outbound` 和 `BenchmarkRemoveNode_Large` 性能指标均达到目标（Task 2/3 记录的基准对比）。

- [ ] **Step 5: 最终 commit**

```bash
git add -A
git commit -m "feat: replace mdl/catalog SQLite with mxgraph in-memory graph (graphcatalog)

- Drop SELECT FROM CATALOG.xxx and ad-hoc SQL support
- Edges() O(E)→O(degree) via outEdgeIDs/inEdgeIDs index
- FindPathSchemas() O(N²)→O(depth) via backtracking DFS
- 5 single-domain mpr adapters (SOLID SRP)
- graphcatalog.LintReader / TraversalReader interfaces (SOLID ISP/DIP)
- gob snapshot replaces .mxcli/catalog.db"
```

---

## 完整性自查

| Spec 要求 | 对应 Task |
|-----------|----------|
| Edges() O(degree) | Task 2 |
| RemoveNode() O(degree) | Task 2 |
| FindPathSchemas backtracking | Task 3 |
| SRP: 5 个单域适配器 | Task 4, 5, 6 |
| ISP: DomainReader / BehaviorReader / SecurityReader / ExtensionReader / TraversalReader | Task 7 |
| DIP: linter 依赖接口 | Task 10 |
| DIP: executor 依赖接口 | Task 11 |
| Mock with Func fields + panic | Task 7 |
| gob snapshot 替代 catalog.db | Task 9 |
| 丢弃 SELECT FROM CATALOG.xxx | Task 11 (cmd_catalog) |
| 删除 mdl/catalog/ | Task 13 |
| 性能 benchmark CI 守护 | Task 1, 3 |
| TDD 全程 | 每个 Task 都先写测试 |
