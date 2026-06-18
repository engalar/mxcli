## TicketStatusBadge

彩色状态徽章，用于在 Mendix DataGrid 中直观显示工单状态。

**Widget ID**: `com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge`

## 属性

| 属性 | 类型 | 说明 |
|------|------|------|
| `statusValue` | Enum attribute | 绑定到工单状态枚举属性 |

支持的枚举 key 及颜色：

| Key | 显示颜色 |
|-----|---------|
| `Open` | 蓝色 |
| `InProgress` | 黄色 |
| `Resolved` | 绿色 |
| `Closed` | 灰色 |
| 其他 | 灰色（默认） |

## 构建

```bash
# 首次构建（国内推荐加 registry 加速）
mxcli widget build --registry https://registry.npmmirror.com

# 构建并安装到 Mendix 项目
mxcli widget build --registry https://registry.npmmirror.com --install -p /path/to/MyProject.mpr
```

构建产物：`dist/1.0.0/com.helpdesk.widget.TicketStatusBadge.mpk`

## MDL 用法

```mdl
column colStatus (attribute: Status, caption: 'Status', ShowContentAs: customContent, ColumnWidth: manual, Size: 140) {
  PLUGGABLEWIDGET 'com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge' wdgStatus (statusValue: Status)
}
```

## 开发

```bash
# 修改源码后重新构建
npm install   # 首次
npm run build # 产出 dist/1.0.0/*.mpk
```

修改 `src/TicketStatusBadgeSample.jsx` 可调整徽章颜色逻辑；
修改 `src/ui/TicketStatusBadge.css` 可调整徽章样式。

**注意**：`statusValue` prop 是 Mendix attribute 对象，不是原始字符串。
使用 `statusValue.value` 获取枚举 key，`statusValue.displayValue` 获取本地化显示文本。
