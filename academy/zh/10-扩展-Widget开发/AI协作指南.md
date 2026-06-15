# 模块 10：AI 协作指南 — Widget 开发

## 工具要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Node.js | 18+ | 运行 Widget 工具链 |
| pluggable-widgets-tools | 最新 | 编译和打包 Widget |
| Mendix Studio Pro | 11.x | 导入 .mpk，测试 Widget |

## 两种学习路径

### 路径 A：只使用 Widget（5 分钟）

1. 把预编译的 `TicketStatusBadge.mpk`（从此模块提供的构建产物）拖入 Studio Pro
2. 运行 `参考实现/use-widget.mdl` 将 Widget 添加到页面
3. 启动应用，查看效果

### 路径 B：从零开发 Widget（完整路径）

```bash
# 1. 克隆 Widget 模板
npx @mendix/pluggable-widgets-tools@latest create-widget TicketStatusBadge

# 2. 替换源码（从 widget-source/src/ 复制）
cp widget-source/src/TicketStatusBadge.tsx src/TicketStatusBadge.tsx

# 3. 编译 + 打包
npm run build
# 产出：dist/TicketStatusBadge.mpk

# 4. 在 Studio Pro 中导入 .mpk
# App → Import module package → 选择 .mpk

# 5. 用 MDL 在页面中使用
mxcli exec 参考实现/use-widget.mdl -p MyProject.mpr
```

## 与 Claude 协作

```
我有一个 Mendix Pluggable Widget 叫 TicketStatusBadge，
def.json 已定义属性 statusValue（HD.TicketStatus 枚举）。
帮我用 MDL 的 PLUGGABLEWIDGET 语法把它添加到 HD.Ticket_Overview 工单列表的 Status 列中，
替换原来的文字显示。
```

## Widget 使用的 MDL 语法

```mdl
-- 在 DataGrid column 中使用自定义 Widget
column colStatus (caption: 'Status', ColumnWidth: manual, Size: 120, ShowContentAs: customContent) {
  PLUGGABLEWIDGET 'helpdesk.TicketStatusBadge' wdgStatus (
    statusValue: attribute Status
  )
}
```
