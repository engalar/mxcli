# 模块 09：AI 协作指南 — JS Action 扩展

## JS Action vs Java Action

| 特性 | JS Action | Java Action |
|------|-----------|-------------|
| 运行环境 | 客户端（浏览器） | 服务器 |
| MDL 语法 | `call javascript action` | `call java action` |
| 适合场景 | 浏览器 API、UI 交互 | 服务器计算、数据库操作 |
| 创建方式 | Studio Pro 或内联 MDL | 只能在 Studio Pro 中创建 |

## 与 Claude 协作

```
帮我用 MDL 实现三个 JavaScript Action 纳流：

1. HD.NF_CopyToClipboard($Text: string)：
   用 navigator.clipboard.writeText() 把 $Text 写入剪贴板
   成功后调用 mx.ui.info('已复制') 显示提示

2. HD.NF_NotifyHighPriority($Subject: string)：
   检查 Notification.permission，如果是 granted 则创建通知
   标题：'高优先级工单'，正文：$Subject

3. HD.NF_FormatRelativeTime($DateTime: DateTime) returns string：
   计算 now - datetime 的差值（毫秒），转换为"刚刚/N分钟前/N小时前/N天前"
```

## JS Action 的 MDL 写法

```mdl
-- 内联 JavaScript Action（Mendix 10.6+ 支持在纳流中内联 JS）
create or modify nanoflow HD.NF_CopyToClipboard
  ($Text: string)
  returns void
  folder 'UI'
{
  call javascript action (
    code = 'navigator.clipboard.writeText($Text).then(() => mx.ui.info("已复制"));',
    parameters = { $Text: string }
  );
}
/
```

## 常见坑

| 坑 | 解决 |
|----|------|
| 剪贴板 API 需要 HTTPS | 本地开发用 localhost 会自动授权；生产环境必须 HTTPS |
| 通知权限未授权 | 先调用 `Notification.requestPermission()`，再检查 permission |
| DateTime 传入 JS | Mendix 的 DateTime 是 JS Date 对象，直接 `.getTime()` 获取毫秒 |
