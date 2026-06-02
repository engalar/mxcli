# Issue 006: DECLARE List of Marketplace 模块实体类型失败（CE0053）

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 中 |
| 发现日期 | 2026-06-02 |

## 问题描述

使用 Marketplace 模块的实体类型声明列表变量，`mxcli check` 通过，但 `mx check` 报 CE0053（类型无效）。

## 复现步骤

```sql
create or modify microflow MyModule.ACT_Import () returns Nothing
begin
  declare $Columns List of ExcelImporter.Column = empty;
  -- ... 使用 $Columns
  return;
end;
```

- `mxcli check`：✅ 通过
- `mx check`：❌ CE0053: Type 'ExcelImporter.Column' is not valid

## 根因

`mxcli check` 不验证第三方（Marketplace）模块的类型是否实际存在于项目中。类型解析仅对当前项目的用户模块有效，对 marketplace 模块的类型引用缺乏查找能力。

## 期望行为

选项 A：`mxcli check --references` 能验证 `ExcelImporter.Column` 在当前 MPR 中确实存在
选项 B：当检测到跨模块类型引用时，给出警告（而非静默通过）

## 实际行为

`mxcli check` 对 `ExcelImporter.Column` 类型引用不做验证，导致假通过。

## 与 Issue 005 的关系

属于 Issue 005（mxcli check 与 mx check 语义鸿沟）的一个具体表现。修复方向是在 `--references` 模式下加载项目的完整类型注册表（包含 marketplace 模块类型）。

## 关联文件

- `mdl/executor/` — 类型解析逻辑
- `mdl/catalog/` — catalog 中的类型查询能力
- Issue 005（上层问题）
