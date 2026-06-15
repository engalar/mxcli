# 模块 02：AI 协作指南 — 微流业务逻辑

## 前提

先运行领域模型：

```bash
mxcli exec academy/zh/01-领域建模/参考实现/domain-model.mdl -p MyProject.mpr
```

## 与 Claude 协作的步骤

### Step 1：让 Claude 实现状态机

```
读取 academy/zh/02-微流业务逻辑/业务需求.md，
帮我用 MDL 实现工单的业务逻辑微流：
- HD.ACT_Ticket_Submit（提交）
- HD.ACT_Ticket_Assign（指派）
- HD.ACT_Ticket_Resolve（解决）
- HD.ACT_Ticket_Reopen（重开）
- HD.ACT_Ticket_Close（关闭）
```

### Step 2：验证 SLA 计算

SLA 计算用 `addHours('[%CurrentDateTime%]', N)` 函数，N 可以是常量引用 `@HD.SLA_HIGH_HOURS`。

要求 Claude 确认生成的微流中包含：

```mdl
change $Ticket (
  Status   = HD.TicketStatus.Open,
  SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
);
```

### Step 3：验证状态前置检查

每个微流都应该先检查当前状态是否正确，例如：

```mdl
if $Ticket/Status != HD.TicketStatus.Open {
  show message 'Only Open tickets can be assigned.' type warning;
  return false;
}
```

### Step 4：常见坑

| 坑 | 原因 | 解决 |
|----|------|------|
| 常量引用报错 | 写成 `HD.SLA_HIGH_HOURS` 而非 `@HD.SLA_HIGH_HOURS` | 加 `@` 前缀 |
| SLA 计算只比较日期不比较时间 | 逾期判断漏了 `and $Ticket/SLADueAt != empty` | 加空值保护 |
| 重开时没清除 ResolvedAt | 只改了 Status | 同时 `change $Ticket (ResolvedAt = empty)` |

## 参考实现

看 `参考实现/microflows.mdl`。
