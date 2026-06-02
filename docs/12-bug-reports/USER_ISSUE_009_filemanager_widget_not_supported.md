# Issue 009: FILEMANAGER / FILEDOCUMENTUPLOADER widget 不支持 MDL

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 高 |
| 发现日期 | 2026-06-02 |

## 问题描述

`FILEMANAGER` 和 `FILEDOCUMENTUPLOADER` widget 在 MDL `CREATE PAGE` 语法中不受支持，导致文件上传功能（如 Excel 导入）必须手动在 Studio Pro 中配置。

## 代码确认

`mdl/grammar/domains/MDLPage.g4` 的 `widgetTypeV3` 规则列表中**完全不包含** `FileManager` 和 `FileDocumentUploader`：

```antlr
widgetTypeV3
    : LAYOUTGRID | ROW | COLUMN | DATAGRID | DATAVIEW | LISTVIEW | GALLERY
    | CONTAINER | NAVIGATIONLIST | ITEM | TEXTBOX | TEXTAREA | DATEPICKER
    | DROPDOWN | COMBOBOX | CHECKBOX | RADIOBUTTONS | REFERENCESELECTOR
    | ACTIONBUTTON | LINKBUTTON | TITLE | DYNAMICTEXT | STATICTEXT | SNIPPETCALL
    | CUSTOMWIDGET | TEXTFILTER | NUMBERFILTER | DROPDOWNFILTER | DATEFILTER
    | DROPDOWNSORT | FOOTER | HEADER | CONTROLBAR | FILTER | TEMPLATE
    | IMAGE | STATICIMAGE | DYNAMICIMAGE | CUSTOMCONTAINER | TABCONTAINER
    | TABPAGE | GROUPBOX
    -- ↑ FileManager 和 FileDocumentUploader 均缺席
```

后端类型存在：`generated/metamodel/types.go` 中有 `PagesFileManager`，说明 BSON 模型支持，仅 MDL 层未实现。

## 用户影响

Excel 导入等核心用例依赖文件上传控件，目前必须离开 mxcli 工作流手动操作 Studio Pro。

## 期望行为

```sql
create page MyModule.ImportPage layout MyFirstModule.Atlas_Default {
  dataview {
    entity: System.FileDocument
    content: {
      filemanager {
        allowed-extensions: 'xlsx,xls'
        max-file-size: 10
      }
    }
  }
}
```

## 实现步骤

1. 在 `MDLLexer.g4` 中添加 `FILEMANAGER` 和 `FILEDOCUMENTUPLOADER` token
2. 在 `MDLPage.g4` 的 `widgetTypeV3` 规则中加入这两个 token
3. 在 `mdl/visitor/` 中添加对应的 AST 构建逻辑
4. 在 `mdl/executor/cmd_pages_write_gen.go` 中添加 BSON 序列化（参考 `PagesFileManager` 类型结构）
5. 从 Studio Pro 创建的示例项目中提取 widget 模板（参考 `modelsdk/widgets/templates/README.md`）
6. 添加 MDL 测试用例到 `mdl-examples/doctype-tests/`

## 关联文件

- `mdl/grammar/domains/MDLPage.g4` — `widgetTypeV3` 规则（需添加）
- `mdl/grammar/MDLLexer.g4` — 需添加 token
- `generated/metamodel/types.go` — `PagesFileManager` 类型定义
- `modelsdk/widgets/templates/` — widget 模板目录
