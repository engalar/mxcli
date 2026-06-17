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

- 将 snapshot 加载/保存下沉到 `mmpr.Reader` 层，让所有 reader 操作都受益
- 添加 Workflow 相关的边（CALLS、TRIGGERS 等）增强图遍历能力
