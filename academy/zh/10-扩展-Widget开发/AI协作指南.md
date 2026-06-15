# 模块 10：AI 协作指南 — Widget 开发

## 工具要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Node.js / Bun | 18+ / 最新 | 运行 esbuild（`mxcli widget build` 自动调用） |
| mxcli | 最新 | 构建 widget、在页面中使用 widget |
| Mendix Studio Pro | 11.x | 导入 .mpk，运行时测试 widget |

## 两种学习路径

### 路径 A：从零构建 Widget（推荐，5 步完成）

```bash
# 步骤 1：进入 widget 源码目录
cd academy/zh/10-扩展-Widget开发/widget-source

# 步骤 2：构建 widget（esbuild 自动安装，产出 TicketStatusBadge.mpk）
mxcli widget build

# 步骤 3：把 .mpk 放到 Mendix 项目的 widgets/ 目录
cp TicketStatusBadge.mpk /path/to/MyProject/widgets/

# 步骤 4a：空应用独立验证（创建测试模块 + 页面）
mxcli exec 参考实现/test-widget-standalone.mdl -p /path/to/MyProject.mpr

# 步骤 4b：Helpdesk 集成（需要先运行模块 01-05）
mxcli exec 参考实现/use-widget.mdl -p /path/to/MyProject.mpr
```

> **无需 `.def.json`**：mxcli 自动扫描 `widgets/*.mpk`，从中读取 widget 定义，
> 不需要手动运行 `mxcli widget extract`。

### 路径 B：修改 Widget 源码（完整开发路径）

1. 修改 `widget-source/src/TicketStatusBadge.jsx`（主逻辑）或 `TicketStatusBadge.xml`（属性定义）
2. `cd widget-source && mxcli widget build`
3. 把新的 `.mpk` 覆盖到项目 `widgets/` 目录
4. 再次执行 MDL 脚本测试

## Widget 工作原理

```
widget-source/
├── src/
│   ├── TicketStatusBadge.jsx   ← 主逻辑（React 函数组件）
│   ├── TicketStatusBadge.xml   ← 属性定义（告诉 Mendix 有哪些属性）
│   └── ...（editorConfig, icons）
├── package.json                ← 只需 esbuild，无其他依赖
└── package.xml                 ← MPK manifest

mxcli widget build
  └─→ esbuild 编译 .jsx → .js bundle
  └─→ 打包 XML + bundle → TicketStatusBadge.mpk

MyProject/widgets/TicketStatusBadge.mpk  ← 放这里
  └─→ mxcli exec 时自动发现（无需 extract）
  └─→ PLUGGABLEWIDGET 'com.helpdesk.widget.TicketStatusBadge' name (statusValue: Status)
```

## Widget 使用的 MDL 语法

```mdl
-- 在 DataGrid column 中使用 TicketStatusBadge
column colStatus (attribute: Status, caption: 'Status', ShowContentAs: customContent, ColumnWidth: manual, Size: 140) {
  PLUGGABLEWIDGET 'com.helpdesk.widget.TicketStatusBadge' wdgStatus (statusValue: Status)
}
```

- `'com.helpdesk.widget.TicketStatusBadge'` — widget ID（与 XML 中 id 属性一致）
- `statusValue: Status` — 属性绑定：将 `Status` 枚举属性绑定到 `statusValue` 属性

## 与 Claude 协作

```
我有一个自定义 Widget TicketStatusBadge，
widget ID 是 com.helpdesk.widget.TicketStatusBadge，
MPK 已放在项目的 widgets/ 目录下。
属性：statusValue（枚举类型，XML key 为 statusValue）。

帮我用 MDL 把它加到 HD.Ticket_Overview 工单列表的 Status 列，
替换默认的文字显示。
```

## 查看项目中可用的 Widget

```bash
mxcli widget list -p MyProject.mpr
```

输出示例（MPK 已在 widgets/ 目录时）：
```
--- Discovered in widgets/*.mpk (not yet extracted) ---

MDL Name (auto)   Display Name          Widget ID                              Description
ticketstatusbadge Ticket Status Badge   com.helpdesk.widget.TicketStatusBadge  Colored status badge...
```

## 常见问题

| 问题 | 解决方案 |
|------|----------|
| `mxcli widget build` 找不到 esbuild | 执行 `bun install` 或 `npm install` 后再 build |
| widget 在 `mxcli widget list` 看不到 | 确认 `.mpk` 在项目的 `widgets/` 子目录中 |
| `statusValue: Status` 不生效 | 确认实体有 `Status` 枚举属性，且 enum 类型正确 |
| `mxcli widget build` 报 widget ID 格式错误 | Widget ID 必须有 4+ 段（如 `com.helpdesk.widget.TicketStatusBadge`） |
