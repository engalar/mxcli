# 模块 05：AI 协作指南 — 安全与权限

## 前提

先运行模块 01–04 的参考实现。

## 安全配置的四个层次

```
1. 模块角色（Module Role）     ← 定义角色名称
2. 实体访问规则（Entity Grant） ← 决定谁能读/写/删哪些实体
3. 微流/页面权限（Grants）      ← 决定谁能运行哪些逻辑和页面
4. 用户角色（User Role）        ← 把模块角色组合成系统用户类型
```

## 与 Claude 协作的步骤

```
帮我用 MDL 为 Helpdesk 配置三角色安全体系：
- CustomerRole：只读/写自己的工单（XPath 行级过滤），看不到 Internal 评论
- AgentRole：所有工单的完整访问
- ManagerRole：在 AgentRole 基础上增加删除权限

同时需要：
- 用户角色：Customer（含 HD.CustomerRole）, Agent（含 HD.AgentRole）, Manager（含 HD.ManagerRole）
- 演示用户：demo_customer / demo_agent / demo_manager，密码 Demo12345678
- 导航：Customer 首页→ My Tickets，Agent 首页→ All Tickets，Manager 首页→ All Tickets
```

## XPath 行级过滤说明

行级过滤用 `where '[xpath]'` 语法：

```mdl
-- 只看自己的工单：从 Ticket 找到关联的 Customer，再看 Customer 的 owner
grant HD.CustomerRole on HD.Ticket (create, read *, write *)
  where '[HD.Ticket_Customer/HD.Customer/System.owner=''[%CurrentUser%]'']';

-- 只看非内部评论
grant HD.CustomerRole on HD.TicketComment (create, read *)
  where '[IsInternal = false]';
```

注意：XPath 字符串内的单引号要**双写**（`''`）。

## 常见坑

| 坑 | 解决 |
|----|------|
| 演示用户密码不满足密码策略 | 先 `alter project security password policy (min_length: 8, require_digit: true, require_mixed_case: true)` |
| 演示用户登录时提示 "unknown user" | 检查是否先执行了 `alter project security demo users on` |
| 导航首页设置未生效 | `home page ... for Customer` 语法需要 User Role 名称完全匹配 |
