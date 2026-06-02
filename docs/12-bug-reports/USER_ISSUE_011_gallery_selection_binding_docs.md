# Issue 011: GALLERY SELECTION 绑定限制未文档化

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open（文档缺口） |
| 优先级 | 低 |
| 发现日期 | 2026-06-02 |

## 问题描述

SELECTION 绑定与 DATAGRID2（插件式 widget）搭配时，`mxcli exec` 后 selection 可能不稳定或失效；改用内置 GALLERY widget 可以正常工作，但此限制和变通方案未在文档中说明。

## 背景

Mendix 有两类 widget：
- **内置 widget**（built-in）：如 GALLERY、LISTVIEW，由 mxcli 直接控制 BSON
- **插件式 widget**（pluggable）：如 DATAGRID2，通过 `.def.json` 模板和 `WidgetRegistry` 处理

DATAGRID2 的 selection 属性是插件式 widget 的内部属性，mxcli 对其写入的 BSON 结构可能与 Studio Pro 生成的有细微差异，导致 selection 行为不一致。

## 用户影响

用户在尝试 DataGrid2+Selection 组合时可能花大量时间调试，不知道应改用内置 GALLERY。

## 期望行为

文档（`.claude/skills/create-page.md` 或 `docs/01-project/MDL_QUICK_REFERENCE.md`）中应明确说明：

1. DataGrid2（插件式）的 SELECTION 绑定在 MDL 中的支持状态
2. 如需 selection 功能，推荐使用内置 GALLERY widget
3. 已知限制：插件式 widget 的某些内部属性可能不完整支持

## 文档修改建议

在 `docs/01-project/MDL_QUICK_REFERENCE.md` 的 widget 支持表格中加入一列"MDL 支持程度"，区分：
- ✅ 完整支持
- ⚠️ 部分支持（注明限制）
- ❌ 不支持

## 关联文件

- `docs/01-project/MDL_QUICK_REFERENCE.md` — 需补充支持状态说明
- `.claude/skills/create-page.md` — 需补充 pluggable vs built-in widget 限制说明
- `modelsdk/widgets/` — 插件式 widget 处理逻辑
