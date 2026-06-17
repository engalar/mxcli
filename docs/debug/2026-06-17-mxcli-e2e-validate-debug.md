# mxcli E2E 验证调试记录 — Workflow CE0109 + bson dump 性能

## 问题

`validate-academy-capstone.sh` 执行 mx check 报错：

```
[error] [CE0109] "Undefined variable 'Result'." at End event
```

根因在于 `autoBindCallMicroflowGenActivity`（`cmd_workflows_write_v2.go:958`）为 `CallMicroflowTask` 添加 `VoidConditionOutcome`，但工作流结束事件仍引用了未定义的 `$Result`。

## BSON 结构分析

`mxcli bson dump -p HelpDeskE2E.mpr -t workflow -o HD.WF_TicketEscalation --format json` 得到：

```json
{
  "$Type": "Workflows$Workflow",
  "Flow": {
    "Activities": [
      3,
      { "$Type": "Workflows$StartWorkflowActivity" },
      {
        "$Type": "Workflows$ExclusiveSplitActivity",
        "Outcomes": [
          3,
          {
            "Value": true,
            "Flow": { "Activities": [3, {
              "$Type": "Workflows$CallMicroflowTask",
              "Microflow": "HD.WFS_Reject",
              "Outcomes": [3, {
                "$Type": "Workflows$VoidConditionOutcome",
                "Flow": { "$Type": "Workflows$Flow" }
              }]
            }] }
          },
          {
            "Value": false,
            "Flow": { "Activities": [3, {
              "$Type": "Workflows$CallWorkflowActivity",
              "Workflow": "HD.WF_SUB_ManagerReview"
            }] }
          }
        ]
      },
      { "$Type": "Workflows$EndWorkflowActivity" }
    ]
  }
}
```

### 关键发现

- `CallMicroflowTask` 的 outcome 使用 `VoidConditionOutcome` + 空 `Flow`（正确，因为微流返回 void）
- `EndWorkflowActivity` 没有 `ReturnValue`/`ResultVariable` 字段——问题不在 Workflow 层面
- 根因在**被调微流**：`WFS_Reject`、`WFS_Approve`、`WFS_SendReminder` 缺少 `MicroflowReturnType` 元素，导致 mx check 在 End event 无法确定返回值类型

### CE0109 历史

类似的修复在 commit `00205268d`：
> Bug 3: MicroflowReturnType element missing from new microflows
> Fix: call SetMicroflowReturnType(DataType element) alongside SetReturnType(string)
> Caused: CE0109 cascade — callers' result variables seen as undefined

## bson dump 性能

### 现状

| 命令 | 耗时 |
|------|------|
| `go run ./cmd/mxcli bson dump -t workflow --list` | ~23s |
| `go run ./cmd/mxcli bson dump -t workflow -o X --format json` | ~55s |
| 预编译 binary + `--list` | ~14s |
| 预编译 binary + `--object --format json` | ~14s |

瓶颈：
1. `go run` 编译整个 binary（~40s）
2. `GetRawUnitByName` 对每个匹配的 unit 做 `listUnitsByType` → `resolveContents` → `bson.Unmarshal` → 模块名解析，O(N) 扫描

### mxgraph 索引方案

`mxgraph` 提供：
- `FindNodes(label, {"QualifiedName": qn})` — O(1) 属性索引，直接定位 unit ID
- `Node.Props` — 存有 Name、Module、QualifiedName 等元数据
- `MarshalSnapshot`/`UnmarshalSnapshot` — gob 持久化到 `.mxcli/graph.gob`

**当前缺失**：
1. 没有 `WorkflowsAdapter`——mxgraph 不索引 Workflows$Workflow
2. `bson dump` 没有尝试加载 mxgraph snapshot——每次 O(N) 扫描

**优化方案**：
1. 添加 `WorkflowAdapter`（参考 MicroflowAdapter 模式）
2. 注册到所有 adapter 注册点（`serve.go`, `cmd_graph.go`）
3. `bson dump` 优先从 `.mxcli/graph.gob` 加载 snapshot，通过 `FindNodes` 定位 unit ID，再用 `reader.GetRawUnitBytes(id)` 单文件读取 BSON

### 预期效果

- 首次：构建 mxgraph（与当前 O(N) scan 相当）
- 后续查询：O(1) lookup + 单文件读取 ≈ 亚秒级
