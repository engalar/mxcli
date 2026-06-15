# 模块 06：AI 协作指南 — 知识库模块

## 前提

先运行模块 01–05 的参考实现（或运行 capstone 参考实现的 01–05 文件）。

## 本模块涉及的新 MDL 概念

| 概念 | 语法示例 |
|------|---------|
| 自引用关联（分类树） | `from KB.Category to KB.Category` |
| 多对多中间表 | 创建空实体 KB.ArticleTag，两个关联分别指向两端 |
| unique 约束 | `Name: string(100) not null unique` |
| integer 属性 | `ViewCount: integer default 0` |

## 与 Claude 协作的步骤

### Step 1：让 Claude 设计 KB 领域模型

```
读取 academy/zh/06-知识库模块/业务需求.md，帮我用 MDL 实现知识库的领域模型：
- KB 模块
- KB.ArticleStatus 枚举（Draft/Published/Archived）
- KB.Category 实体（Name, Description, 自引用父分类关联）
- KB.Tag 实体（Name, unique）
- KB.Article 实体（Title, Content, Status, PublishedAt, ViewCount）
  - 关联 KB.Article → KB.Category
- KB.ArticleTag 中间表（多对多：Article ↔ Tag）
```

### Step 2：发布和归档微流

```
帮我实现两个微流：
- KB.ACT_Article_Publish：验证 Content 非空，Status Draft → Published，记录 PublishedAt
- KB.ACT_Article_Archive：校验当前是 Published，Status → Archived
```

### Step 3：常见坑

| 坑 | 解决 |
|----|------|
| 自引用关联命名 | 用 KB.Category_Parent（而非 KB.Category_Category）避免歧义 |
| 多对多中间表 | KB.ArticleTag 实体本身没有属性，只有两个 association |
| XPath 字符串过滤 | `where '[Status = ''Published'']'`（单引号在 XPath 内要双写） |
| PublishedAt 在发布前为空 | 发布微流用 `PublishedAt = '[%CurrentDateTime%]'` 赋值 |
| module role 不支持 or modify | 用 `create module role KB.Reader;`（不能写 `create or modify module role`） |
