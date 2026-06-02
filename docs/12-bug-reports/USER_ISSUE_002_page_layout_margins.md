# Issue 002: 页面 margin/padding 默认值不合理

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Nagano |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 低 |
| 发现日期 | 2026-06-02 |

## 问题描述

mxcli 创建页面时，margin 和 padding 未设置为合理的默认值，需要用户手动配置，与 Studio Pro 创建页面时的视觉效果差异明显。

## 复现步骤

```sql
create page MyModule.TestPage layout NativePhone.PopupLayout {
  dataview {
    entity: MyModule.Entity
    content: {
      textbox { attribute: Name }
    }
  }
}
```

创建后在 Studio Pro 中打开，发现内容区域无合理间距。

## 期望行为

页面内容区域应有与 Studio Pro 创建页面一致的默认 margin/padding（如 Atlas UI 风格的默认间距）。

## 实际行为

margin/padding 为零或未设置，页面视觉li 创建 widget 时使用的模板（`modelsdk/widgets/templates/` 和 `sdk/widgets/templates/`）不一定包含 Atlas UI 的默认 spacing 属性。

## 可能的解决方向

1. 对比 Studio Pro 实际创建的页面 BSON，提取默认 margin/padding 值
2. 在 widget 模板中加入合理的默认间距属性
3. 在 `CREATE PAGE` 的 CONTAINER 处理逻辑中注入默认 appearance 属性

## 关联文件

- `modelsdk/widgets/templates/` — widget 模板定义
- `mdl/executor/cmd_pages_write_gen.go` — 页面写入逻辑
- `mdl/backend/mpr/pages_write.go` — 页面 BSON 序列化
