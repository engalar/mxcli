# 模块 09：AI 协作指南 — JS Action 扩展

## JS Action vs Java Action

| 特性 | JS Action | Java Action |
|------|-----------|-------------|
| 运行环境 | 客户端（浏览器） | 服务器 |
| MDL 语法 | `call javascript action` | `call java action` |
| 适合场景 | 浏览器 API、UI 交互 | 服务器计算、数据库操作 |
| 创建方式 | 必须在 Studio Pro 中创建（生成 `.js` 文件），MDL 按名称调用 | 只能在 Studio Pro 中创建 |

## 与 Claude 协作

> **工作流**：JS Action 的实际 JavaScript 代码必须在 Studio Pro 中编写（Studio Pro 会在
> `javascriptsource/<module>/actions/` 下生成对应的 `.js` 文件）。先在 Studio Pro 中创建好
> JS Action，再让 Claude 生成调用它们的纳流 MDL。

**第一步：让 Claude 描述每个 JS Action 应实现的逻辑（供在 Studio Pro 中编写）**

```
我需要三个 JavaScript Action，请给出每个 Action 的 JS 实现代码，
以便我在 Mendix Studio Pro 中手动创建：

1. HD.JSA_CopyToClipboard（参数：Text: string，返回 void）
   用 navigator.clipboard.writeText() 写入剪贴板，成功后显示 mx.ui.info('已复制')

2. HD.JSA_NotifyHighPriority（参数：Subject: string，返回 void）
   检查 Notification.permission，如果是 granted 则创建桌面通知
   标题：'高优先级工单'，正文：Subject

3. HD.JSA_FormatRelativeTime（参数：DateTime: DateTime，返回 string）
   计算 now - DateTime 的差值（毫秒），返回"刚刚/N分钟前/N小时前/N天前"
```

**第二步：在 Studio Pro 中创建好 JS Action 后，让 Claude 生成调用纳流的 MDL**

```
HD 模块中已有以下 JavaScript Action：
- HD.JSA_CopyToClipboard（参数：Text: string，返回 void）
- HD.JSA_NotifyHighPriority（参数：Subject: string，返回 void）
- HD.JSA_FormatRelativeTime（参数：DateTime: DateTime，返回 string）

请生成调用这三个 JS Action 的纳流 MDL（call javascript action 语法）。
```

## JS Action 的 MDL 写法

> **注意**：MDL 不支持内联 JS 代码。JavaScript Action 必须先在 Studio Pro 中创建
> （Studio Pro 会在 `javascriptsource/<module>/actions/` 下生成 `.js` 文件），然后在
> mxcli 中通过 `call javascript action Module.ActionName(params)` 按名称调用。

```mdl
-- 前提：Studio Pro 中已存在 HD.JSA_CopyToClipboard（参数 Text: string）
create or modify nanoflow HD.NF_CopyToClipboard
  ($Text: string)
  returns void
  folder 'UI'
{
  call javascript action HD.JSA_CopyToClipboard(Text = $Text);
}
/

-- 带返回值的示例：HD.JSA_FormatRelativeTime（参数 DateTime: DateTime，返回 string）
create or modify nanoflow HD.NF_FormatRelativeTime
  ($DateTime: datetime)
  returns string as $Result
  folder 'UI'
{
  $Result = call javascript action HD.JSA_FormatRelativeTime(DateTime = $DateTime);
  return $Result;
}
/
```

## 常见坑

| 坑 | 解决 |
|----|------|
| 剪贴板 API 需要 HTTPS | 本地开发用 localhost 会自动授权；生产环境必须 HTTPS |
| 通知权限未授权 | 先调用 `Notification.requestPermission()`，再检查 permission |
| DateTime 传入 JS | Mendix 的 DateTime 是 JS Date 对象，直接 `.getTime()` 获取毫秒 |
| `call javascript action (code = '...')` 语法不存在 | MDL 只支持 `call javascript action Module.ActionName(params)` 按名称调用；内联 JS 代码必须在 Studio Pro 中写入对应的 `.js` 文件 |
