# 模块 08：AI 协作指南 — Java Action 扩展

## 工具要求

| 工具 | 用途 | 最低版本 |
|------|------|---------|
| Mendix Studio Pro | 创建 Java Action 框架（`.java` 文件生成）| 11.x |
| JDK | 编译 Java 代码 | 11+ |
| mxcli | 调用 Java Action 的微流（MDL） | 最新版 |

> **快速路径（无 Studio Pro）：**
> 使用 `java-source/JA_HashPassword.java` 中提供的源码，在 Studio Pro 中创建同名 Java Action 后，
> 将源码粘贴进去。或直接使用 Marketplace 中的 `CommunityCommons` 模块，它已内置 `BCrypt` 工具类。

## 两种路径

### 路径 A：使用 CommunityCommons（最快，推荐演示用）

1. 在 Studio Pro 中安装 CommunityCommons Marketplace 模块
2. 在微流中调用 `CommunityCommons.BCryptHash` 和 `CommunityCommons.BCryptCheck`
3. 对应 MDL（调用方式）见 `参考实现/call-java-action.mdl`

### 路径 B：自己实现 Java Action（完整学习路径）

1. Studio Pro → App Explorer → 右键模块 HD → Add Java Action
2. 命名：`JA_HashPassword`，添加参数 `Password`（String），返回类型 `String`
3. 打开 `javasource/helpdesk/actions/JA_HashPassword.java`
4. 把 `java-source/JA_HashPassword.java` 的内容粘贴进去
5. 在 Studio Pro 中添加 `bcrypt.jar` 依赖（或用 Maven）
6. 用 mxcli exec 运行 `参考实现/call-java-action.mdl` 创建调用微流

## 与 Claude 协作

```
帮我用 MDL 实现一个微流 HD.ACT_ChangePassword，
它调用 Java Action JA_HashPassword（模块内，参数：Password: string，返回 string），
然后把哈希结果存储到当前用户的 PasswordHash 属性。

另外实现 HD.ACT_VerifyPassword，
调用 JA_VerifyPassword（参数：Password: string，HashedPassword: string，返回 boolean）
```

## 关键 MDL 语法

```mdl
-- 调用 Java Action 的语法
declare $Hash: string = call java action HD.JA_HashPassword (
  Password = $Password
);
```

## 常见坑

| 坑 | 解决 |
|----|------|
| `call java action` 找不到 Action | Java Action 必须在 Studio Pro 中创建后，mxcli 才能引用 |
| 返回类型不匹配 | 确认 Java Action 在 Studio Pro 中的返回类型与 MDL 声明一致 |
| BCrypt jar 缺失 | 在 Studio Pro 的 `userlib/` 目录放入 bcrypt jar，或用 Maven |
