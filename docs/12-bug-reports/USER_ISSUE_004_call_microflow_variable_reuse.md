# Issue 004: CALL MICROFLOW 不复用已有变量，导致 CE0111

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 高 |
| 发现日期 | 2026-06-02 |

## 问题描述

在同一微流中多次使用相同的输出变量名调用 `CALL MICROFLOW`，每次都创建新变量而非复用已有变量，导致 Mendix 报 CE0111（重复变量名）。

## 复现步骤

```sql
create or modify microflow MyModule.ACT_Process () returns Nothing
begin
  $R = call microflow MyModule.Step1 ();
  $R = call microflow MyModule.Step2 ();  -- CE0111: duplicate variable $R
  $R = call microflow MyModule.Step3 ();  -- CE0111: duplicate variable $R
  return;
end;
```

## 期望行为

第二次及以后对 `$R` 的赋值应复用已存在的变量（即不创建新的 OutputVariableName，而是将结果写入已有变量）。

## 实际行为

每次 `CALL MICROFLOW` 都通过 `SetOutputVariableName(s.OutputVariable)` 无条件创建新变量，Mendix runtime 检测到重名报 CE0111。

## 状态说明（v0.11.0）

`mxcli check --references` 已能检测并报告重复变量（CE0111），但根本问题（变量复用）尚未修复。当前 workaround 是使用唯一变量名（`$R1`、`$R2`...）。

## Bug 代码位置

**`mdl/executor/flowbuilder_calls_flow_gen.go`**（约第 46–82 行）：

```go
action.SetOutputVariableName(s.OutputVariable)
action.SetUseReturnVariable(s.OutputVariable != "")
```

缺少对当前作用域已有变量的检查逻辑。

## 修复建议

在 flowBuilder 中维护已声明变量的集合。若 `s.OutputVariable` 已在集合中，则不设置 `OutputVariableName`（或设置为空），仅做赋值操作；若不在集合中，则正常声明并加入集合。

## 关联文件

- `mdl/executor/flowbuilder_calls_flow_gen.go` — CALL MICROFLOW 执行逻辑（第 46–82 行）
- `mdl/executor/flowbuilder.go` — 变量作用域管理
