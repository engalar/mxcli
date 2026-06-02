# Issue 010: CONTAINER Visible 条件表达式设置导致 CE0117

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 中 |
| 发现日期 | 2026-06-02 |

## 问题描述

在 MDL 中为 CONTAINER widget 设置 `visible: [Attribute != '']` 条件表达式后，`mxcli exec` 写入 MPR，Studio Pro 打开报 CE0117 错误。

## 复现步骤

```sql
alter page MyModule.MyPage {
  replace container MyContainer {
    container MyContainer (
      visible: [$CurrentObject/Status != 'Draft']
    ) {
      textbox { attribute: Name }
    }
  }
}
```

执行后在 Studio Pro 中打开：CE0117

## 代码分析

读路径（DESCRIBE）在 `cmd_pages_describe_output.go` 的 `appendAppearanceProps` 中支持读取并输出 `VisibleIf`，但写路径（exec）可能在 BSON 序列化时出现问题：

- 条件可见性的 BSON 表示需要特定的 `conditionSettings` 结构
- CONTAINER（`Forms$DivContainer` / `Pages$DivContainer`）的 conditionSettings 写入路径可能与其他 widget 不一致

## 期望行为

CONTAINER 的 `visible: [expr]` 应与 TEXTBOX、ACTIONBUTTON 等其他支持条件可见性的 widget 行为一致，写入有效的 BSON conditionSettings，不产生 CE0117。

## 调试建议

1. 在 Studio Pro 中手动设置 CONTAINER 的条件可见性，导出 BSON
2. 与 mxcli exec 生成的 BSON 对比，找出 conditionSettings 结构差异
3. 参考 `.claude/skills/debug-bson.md` 的 BSON 调试流程

## 关联文件

- `mdl/executor/cmd_pages_describe_output.go` — `appendAppearanceProps`（读路径，第 45–72 行）
- `mdl/backend/mpr/pages_write.go` — 写路径（BSON conditionSettings 序列化）
- `.claude/skills/debug-bson.md` — BSON 调试工作流
