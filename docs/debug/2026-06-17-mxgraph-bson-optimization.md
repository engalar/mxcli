# mxgraph + bson dump 性能优化记录

## 问题

`mxcli bson dump` 每次执行都做 O(N) 扫描 + BSON 全反序列化，55s 才能输出一个 workflow 的 JSON。

## 优化方案：mxgraph snapshot 缓存

### 架构

```
bson dump 命令
  ├─ tryLoadMxGraph()     → 读取 .mxcli/graph.gob (O(1))
  ├─ buildMxGraph()       → 构建图 + 保存 snapshot (首次慢)
  └─ fallback (reader)    → 原 O(N) scan (无 snapshot 时)
```

### 改动文件

| 文件 | 改动 |
|------|------|
| `cmd/mxcli/cmd_bson_dump.go` | 新增 `tryLoadMxGraph`、`buildMxGraph`、`outputBson`；接入 mxgraph 快路径 |
| `internal/mxgraph/adapter/mpr/workflow_adapter.go` | 新增 WorkflowAdapter：索引 Workflows$Workflow |
| `cmd/mxcli/serve.go` | 注册 WorkflowAdapter |
| `mdl/executor/cmd_graph.go` | 注册 WorkflowAdapter |
| `internal/mxgraph/persist.go` | gob 注册 `element.ID` 和 `NodeID` |

### 关键踩坑

1. **gen codec 的 Properties() 为空**
   - `initMicroflow()` 末尾有 `SetProperties(...)`，但 `initWorkflow()` 没有
   - 导致 `nodeForElement` 的 `elem.Properties()` 返回 0，WorkflowAdapter 拿不到 Name
   - **解决**: WorkflowAdapter 手动调用 `elem.(interface{ Name() string }).Name()` 提取属性

2. **gob 序列化失败: type not registered for interface: element.ID**
   - 其他 adapter（Microflow/Page 等）通过 `nodeForElement` 存储的 Props 值可能隐含 `element.ID` 类型
   - `gob.Register` 需要注册所有可能出现在 `map[string]any` 中的具体类型
   - **解决**: 在 `persist.go` 的 init() 中注册 `element.ID("")` 和 `NodeID("")`

3. **删除 .mxcli 目录不影响功能**
   - 无 snapshot 时 `buildMxGraph` 自动构建并保存
   - 构建失败时静默回退到 O(N) scan

### 性能数据

| 场景 | 优化前 | 优化后 |
|------|--------|--------|
| 首次运行 (go run + 构建) | 55s | ~14s |
| 二次运行 (snapshot 加载) | 55s | **0.4s** |
| `--object --format json` | 55s | **0.4s** |

### 后续方向

- 添加 Workflow 相关的边（CALLS、TRIGGERS 等）增强图遍历能力

## 第二次迭代：snapshot 下沉 + Workflow 边

### snapshot 下沉到 mmpr.Reader

`modelsdk/mpr/reader.go` 的 `OpenWithOptions` 自动加载 `.mxcli/graph.gob` 到 `r.mxGraph`。新增：
- `GetMxGraph() *mxgraph.Graph` — 所有 reader 消费者共用
- `SetMxGraph(g *mxgraph.Graph)` — 构建后注入

`cmd_bson_dump.go` 改为调用 `reader.GetMxGraph()`，再回退到 `buildMxGraph()`。去掉了独立的 `tryLoadMxGraph()`。

### WorkflowAdapter 边

遍历 workflow 的 Flow → Activities 树，递归进入 outcome flows 和 boundary events：
- `CallMicroflowTask` → `CALLS` 到对应的 microflow QN
- `CallWorkflowActivity` → `CALLS` 到对应的 workflow QN

### 关键踩坑

4. **gen_imports.go 缺少 workflows**
   - `internal/mxgraph/adapter/mpr/gen_imports.go` 空白导入 `gen/workflows` 以触发 codec registry 初始化
   - 缺少时 `LoadUnit` 返回 `*element.Base` 而非 `*genWf.Workflow`，导致类型断言失败
   - 同样影响边界事件解析——`BoundaryEvents` 子元素也需要 registry 才能正确解码

5. **CE6686: VoidConditionOutcome 不匹配**
   - 微流没有 `returns` 子句时，`s.ReturnType` 为 nil，executor 不设置 `ReturnType`/`MicroflowReturnType`
   - mx check 认为返回类型为"未知"，而 workflow 的 `CallMicroflowTask` 有 `VoidConditionOutcome`
   - 两者不匹配 → CE6686
   - **修复**: 在 `cmd_microflows_create_v2.go` 的 `else` 分支中设置 `ReturnType="Nothing"` + `MicroflowReturnType=VoidType`
