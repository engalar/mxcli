# Issue 008: 页面 DESCRIBE 输出不能 roundtrip（mxcli check 失败）

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open（微流已修复，页面待处理） |
| 优先级 | 中 |
| 发现日期 | 2026-06-02 |

## 问题描述

`DESCRIBE PAGE` 输出的 MDL 语法中存在 `mxcli check` 无法识别的语法（如 `call_microflow`、`show_page` 等），导致将 DESCRIBE 结果直接粘贴执行时报语法错误。

## 状态说明

- **微流**：v0.11.0 已修复，DESCRIBE MICROFLOW 可 roundtrip（含 IF/ELSE、LOOP、多个 CALL MICROFLOW）
- **页面**：❌ 仍存在问题，`call_microflow` / `show_page` 等 action 语法未被 parser 识别

## 复现步骤

```sql
-- 1. 获取页面描述
DESCRIBE PAGE MyModule.MyPage;

-- 2. 复制输出，执行 mxcli check
-- 结果：报语法错误（call_microflow 等）
```

## 期望行为

DESCRIBE 输出的 MDL 应能直接通过 `mxcli check` 并可重新执行以重建相同页面。

## 受影响的语法

从 `mdl/executor/cmd_pages_describe_output.go` 可见以下 action 输出格式与 parser 期望不一致：
- `call_microflow` — parser grammar 中可能使用不同关键词
- `show_page` — 同上
- 其他 button action 语法

## 修复建议

1. 梳理 `MDLPage.g4` 中 action 相关规则的实际 token 序列
2. 对比 `cmd_pages_describe_output.go` 中的字符串拼接输出
3. 对齐输出格式与 grammar 规则，或在 grammar 中添加对当前输出的兼容解析

## 关联文件

- `mdl/executor/cmd_pages_describe_output.go` — 页面 DESCRIBE 输出逻辑
- `mdl/grammar/domains/MDLPage.g4` — 页面 grammar（action 规则）
- `mdl/visitor/` — ANTLR visitor（解析 DESCRIBE 输出时的实际解析路径）
