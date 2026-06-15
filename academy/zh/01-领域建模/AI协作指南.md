# 模块 01：AI 协作指南 — 领域建模

## 本模块产出

运行完本模块的参考实现后，你的项目将拥有：
- HD 模块
- 2 个枚举（工单状态、工单优先级）
- 3 个实体（Customer, Agent, Ticket）+ 1 个非持久化实体（TicketSearch）
- 2 个关联（Ticket→Customer, Ticket→Agent）
- 2 个常量（SLA 小时数）

## 与 Claude 协作的步骤

### Step 1：让 Claude 读取业务需求

在 Claude Code 中输入：

```
读取 academy/zh/01-领域建模/业务需求.md，帮我设计 Mendix 领域模型（实体、枚举、关联），用 MDL 实现
```

### Step 2：逐步确认设计

Claude 会提出一个设计方案。在确认前，检查：

1. 枚举值是否覆盖了需求中提到的所有状态和优先级？
2. 每个实体的属性是否完整？类型是否合适？
3. 关联方向是否正确？（工单→客户，工单→客服）

如果有问题，直接告诉 Claude：

```
工单还需要一个 "解决时间" 属性，类型是日期时间
```

### Step 3：生成 MDL 并验证

```bash
# 保存 Claude 生成的 MDL 为 my-domain.mdl，然后：
mxcli check my-domain.mdl
mxcli exec  my-domain.mdl -p MyProject.mpr
~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr 2>&1 | grep -c "StorageLoadException"
# 期望：0
```

### Step 4：常见坑

| 坑 | 症状 | 解决方法 |
|----|------|---------|
| 枚举引用错误 | `mxcli check` 报 "unknown type" | 确认枚举在实体之前定义 |
| 关联方向反了 | mx check 报 CE0XXX | from = 拥有外键的一方（Ticket），to = 被引用的一方（Customer） |
| 属性类型错误 | mx check 报 StorageLoadException | string 类型需要长度：`string(200)`，不能直接写 `string` |

## 参考实现

卡住了就看 `参考实现/domain-model.mdl`。注意：实体语法是 `create or modify persistent entity`，不要写成 `create entity`。
