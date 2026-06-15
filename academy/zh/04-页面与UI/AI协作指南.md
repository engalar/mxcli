# 模块 04：AI 协作指南 — 页面与 UI

## 前提

先运行模块 01–03 的参考实现。

## Atlas UI 组件速查

| 需求 | Atlas 组件 | MDL 关键字 |
|------|-----------|-----------|
| 列表页 | DataGrid | `datagrid` |
| 详情/表单 | DataView | `dataview` |
| 弹窗页面 | PopupLayout | `layout: Atlas_Core.PopupLayout` |
| 2 列布局 | LayoutGrid | `layoutgrid ... row ... column (desktopwidth: 8)` |
| 按钮 | ActionButton | `actionbutton ... buttonstyle: primary` |
| 文本展示 | DynamicText | `dynamictext` |
| 文本输入 | TextBox | `textbox` |
| 多行输入 | TextArea | `textarea` |

## 与 Claude 协作的步骤

```
帮我用 MDL 创建 Helpdesk 的主要页面：
1. HD.Ticket_Overview：工单列表（所有工单），用 Atlas_Default 布局，DataGrid 展示，
   列包括 Subject/Status/Priority/SLADueAt，有文字过滤器和下拉过滤器，
   有 New Ticket 和 Open（打开详情）按钮
2. HD.Ticket_Detail：工单详情，用 2 列布局（左 8 右 4），左边 DataView 展示字段，
   右边放操作按钮（Submit/Assign/Resolve/Reopen/Close），底部有评论 DataGrid
3. HD.Ticket_NewEdit：新建/编辑表单，Subject textbox + Description textarea + Save/Cancel
4. HD.MyTickets_Overview：我的工单列表，datasource: microflow HD.DS_MyTickets，
   有 New Ticket 按钮和 Open 按钮
5. HD.AddComment_Form：弹窗，输入 Content（textarea）和 IsInternal（checkbox），Save 按钮
```

## 常见坑

| 坑 | 解决 |
|----|------|
| `action: show_page` 需要 page params | 写法：`show_page HD.Ticket_Detail (Ticket: $currentObject)` |
| 弹窗页面需要 PopupLayout | `layout: Atlas_Core.PopupLayout` |
| 页面 params 格式 | `params: { $Ticket: HD.Ticket }` |
| DataView 绑定 page param | `datasource: $Ticket` |
| 按钮在 footer 里 | `footer ftrActions { actionbutton ... }` |
