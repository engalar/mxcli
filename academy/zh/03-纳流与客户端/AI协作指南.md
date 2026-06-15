# 模块 03：AI 协作指南 — 纳流与客户端

## 微流 vs 纳流

| 特性 | 微流（Microflow） | 纳流（Nanoflow） |
|------|-----------------|----------------|
| 运行环境 | 服务器 | 客户端（浏览器） |
| 数据库访问 | 支持（完整） | 支持（有限） |
| 适合场景 | 复杂业务逻辑 | 简单操作、快速响应 |
| MDL 语法 | `create or modify microflow` | `create or modify nanoflow` |

## 与 Claude 协作的步骤

```
帮我用 MDL 实现三个纳流：
1. HD.NF_Ticket_QuickCreate：参数 $Customer（HD.Customer）和 $Subject（string），
   创建一张草稿工单并 commit，返回 HD.Ticket
2. HD.NF_TicketSearch_Apply：参数 $Search（HD.TicketSearch），
   从数据库检索工单（limit 100），返回 HD.Ticket 列表
3. HD.NF_Priority_GetLabel：参数 $Priority（HD.TicketPriority），
   返回字符串（Critical→'🔴 紧急', High→'🟠 高', Normal→'🟡 普通', Low→'🟢 低'）
```

## 常见坑

- 纳流的 `retrieve` 不支持复杂 XPath 引用，用简单的 `where [Status = 'Open']` 即可
- 纳流不支持 `show message`，只能 `return`
- 纳流返回列表时类型写法：`returns list of HD.Ticket as $Tickets`
