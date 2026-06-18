# 模块 10：AI 协作指南 — Widget 开发

## 工具要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Node.js | 18+ | 运行 `@mendix/pluggable-widgets-tools` 构建工具链 |
| mxcli | 最新 | 脚手架、构建 widget、安装到项目、在页面中使用 widget |
| Mendix Studio Pro | 11.x | 导入 .mpk，运行时测试 widget |

## 两种学习路径

### 路径 A：从零构建 Widget（推荐，5 步完成）

```bash
# 步骤 1：进入 widget 源码目录
cd academy/zh/10-扩展-Widget开发/widget-source

# 步骤 2：构建 widget（自动 npm install + pluggable-widgets-tools 编译，产出 .mpk）
mxcli widget build

# 国内环境加速（推荐）：
mxcli widget build --registry https://registry.npmmirror.com

# 步骤 3：安装到 Mendix 项目 widgets/ 目录
mxcli widget build --install -p /path/to/MyProject.mpr
# 或先构建再手动复制：
cp dist/1.0.0/com.helpdesk.widget.TicketStatusBadge.mpk /path/to/MyProject/widgets/

# 步骤 4a：空应用独立验证（创建测试模块 + 页面）
mxcli exec 参考实现/test-widget-standalone.mdl -p /path/to/MyProject.mpr

# 步骤 4b：Helpdesk 集成（需要先运行模块 01-09）
mxcli exec 参考实现/use-widget.mdl -p /path/to/MyProject.mpr
```

> **无需 `.def.json`**：mxcli 自动扫描 `widgets/*.mpk`，从中读取 widget 定义，
> 不需要手动运行 `mxcli widget extract`。

在网络受限环境使用代理：
```bash
# 通过 HTTPS 代理安装依赖
mxcli widget build --https-proxy http://192.168.2.35:29758

# 通过自定义 npm registry
mxcli widget build --registry http://npm-registry.internal:4873
```

### 路径 B：修改 Widget 源码（完整开发路径）

1. 修改 `widget-source/src/TicketStatusBadge.jsx`（主逻辑）、`components/TicketStatusBadgeSample.jsx`（渲染组件）或 `TicketStatusBadge.xml`（属性定义）
2. `cd widget-source && mxcli widget build --install -p /path/to/MyProject.mpr`
3. 再次执行 MDL 脚本测试

## Widget 工作原理

```
widget-source/
├── .eslintrc.js, prettier.config.js, .gitattributes, LICENSE
├── package.json                       ← @mendix/pluggable-widgets-tools + React 19
├── README.md
├── src/
│   ├── package.xml                    ← MPK manifest（含 <files> 段，Mendix 11 必需）
│   ├── TicketStatusBadge.xml          ← 属性定义（widget ID 含小写包名段）
│   ├── TicketStatusBadge.jsx          ← 主入口（React 函数组件）
│   ├── TicketStatusBadge.editorConfig.js    ← Studio Pro 设计时配置
│   ├── TicketStatusBadge.editorPreview.jsx  ← Studio Pro 预览（JSX 格式）
│   ├── TicketStatusBadge.{icon,tile}.png    ← 图标
│   ├── components/
│   │   └── TicketStatusBadgeSample.jsx      ← 渲染组件（读取 attribute 对象的 .displayValue）
│   └── ui/
│       └── TicketStatusBadge.css            ← 徽章样式

mxcli widget build
  └─→ npm install（检测到 @mendix/pluggable-widgets-tools 缺失时自动执行）
  └─→ pluggable-widgets-tools build:web
      ├─→ rollup 编译 .jsx → .js (AMD) + .mjs (ESM) 双格式
      └─→ 打包 XML + bundle → dist/1.0.0/*.mpk
  └─→ mxcli 后处理（Windows 兼容性修补）
  └─→ 产出完整 .mpk

MyProject/widgets/com.helpdesk.widget.TicketStatusBadge.mpk
  └─→ mxcli exec 时自动发现（无需 extract）
  └─→ PLUGGABLEWIDGET 'com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge' name (statusValue: Status)
```

## Widget ID 格式规则

Mendix 要求 widget ID 与 JS 文件路径严格对应：

| 字段 | 值 |
|------|----|
| `packagePath`（package.json） | `com.helpdesk.widget` |
| 构建输出路径 | `com/helpdesk/widget/ticketstatusbadge/TicketStatusBadge.mjs` |
| **widget ID（XML + MDL）** | `com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge` |

规律：**widget ID = `{packagePath}.{小写名称}.{WidgetName}`**，中间段必须与输出目录名一致（全小写）。

`mxcli widget new` 会自动按此规则生成正确的 widget ID。

## Mendix Attribute 属性对象

在 Mendix pluggable widget 中，`attribute` 类型的 prop 是一个**对象**，而不是原始值：

```js
// statusValue 不是字符串，而是 Mendix attribute 对象：
// {
//   value: "Open",            // 枚举 key（XML 中定义的 key）
//   displayValue: "进行中",   // 本地化显示文本
//   status: "available",      // "available" | "loading" | "unavailable"
//   readOnly: false,
//   ...
// }

// 正确写法：
const key   = statusValue?.value;          // 枚举 key，用于逻辑判断
const label = statusValue?.displayValue;   // 显示文本，用于渲染

// 错误写法（会触发 React error #31）：
return <span>{statusValue}</span>;         // ❌ 不能直接渲染对象
```

## Widget 使用的 MDL 语法

```mdl
-- 在 DataGrid column 中使用 TicketStatusBadge
column colStatus (attribute: Status, caption: 'Status', ShowContentAs: customContent, ColumnWidth: manual, Size: 140) {
  PLUGGABLEWIDGET 'com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge' wdgStatus (statusValue: Status)
}
```

- `'com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge'` — widget ID（与 XML 中 id 属性一致）
- `statusValue: Status` — 属性绑定：将 `Status` 枚举属性绑定到 `statusValue` 属性

## 与 Claude 协作

```
我有一个自定义 Widget TicketStatusBadge，
widget ID 是 com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge，
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

MDL Name (auto)   Display Name          Widget ID                                              Description
ticketstatusbadge Ticket Status Badge   com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge  Colored status badge...
```

## 常见问题

| 问题 | 解决方案 |
|------|----------|
| `mxcli widget build` 卡在 `npm install` | 设置 `--registry https://registry.npmmirror.com`（国内推荐） |
| `mxcli widget build` 构建慢 | `@mendix/pluggable-widgets-tools` 依赖 1000+ 包，首次安装约 5~30 秒（npmmirror） |
| widget 在 `mxcli widget list` 看不到 | 确认 `.mpk` 在项目的 `widgets/` 子目录中 |
| `statusValue: Status` 不生效 | 确认实体有 `Status` 枚举属性，且 enum 类型正确 |
| React error #31 / Objects are not valid as child | widget 代码直接渲染了 attribute 对象，改用 `statusValue.displayValue` |
| 部署报 "no definition for widget ..." | MDL 中 widget ID 拼写错误，确认使用含小写包名段的完整 ID |
| 部署报 "ES6 modules" 错误 | widget ID 与 JS 输出路径不匹配，检查 `src/TicketStatusBadge.xml` 中的 id 属性 |
| `exit handler never called` | npm 11 + HTTPS proxy 已知 bug，改用 `--registry` 方式替代 `--https-proxy` |
