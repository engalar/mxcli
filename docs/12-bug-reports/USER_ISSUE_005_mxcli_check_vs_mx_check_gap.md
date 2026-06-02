# Issue 005: mxcli check 与 mx check 语义验证鸿沟

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open（已知设计限制） |
| 优先级 | 中 |
| 发现日期 | 2026-06-02 |

## 问题描述

通过 `mxcli check` 的脚本在 `mx check`（Mendix 完整验证器）中仍可产生大量语义错误，包括：

- **CE0111**：重复变量名（见 Issue 004）
- **CE0109**：未定义变量
- **CE0053**：类型无效（见 Issue 006）
- **CE0066**：Entity access 过期（见 Issue 007）

## 根因

`mxcli check` 是语法级 + 部分结构检查；`mx check` 是 Mendix Studio Pro 内置的完整语义验证器，覆盖跨文档引用、类型系统、安全一致性等深层约束。两者的检查层次本质不同。

## 用户影响

用户需要：执行脚本 → 打开 Studio Pro → mx check 发现错误 → 关闭 Studio Pro → 修改脚本 → 循环。`mxcli check` 给用户"通过了"的错误预期，实际执行后才暴露问题。

## 期望行为

用户请求 `mxcli check --full` 选项，对 mx check 覆盖的语义规则进行等效验证。

## 可行路径

| 错误码 | 检测可行性 | 说明 |
|--------|-----------|------|
| CE0111 | ✅ 已实现（v0.11.0） | `mxcli check --references` |
| CE0109 | 中 | 需要跨活动变量流分析 |
| CE0053 | 中 | 需要解析第三方模块类型（见 Issue 006） |
| CE0066 | 高 | 可检测 access rule 与关联方向不一致 |

## 关联文件

- `mdl/executor/` — 执行器（含已有语义检查逻辑）
- `mdl/linter/rules/` — 现有 lint 规则
- Issue 004、006、007（关联的具体错误码）
