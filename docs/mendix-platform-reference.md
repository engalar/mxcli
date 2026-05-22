# Mendix 平台事实参考

本文档记录从官方文档（`@../mendix_docs`）实证验证的 Mendix 平台事实。
每条事实注明来源，供设计和实现时核对。

---

## 1. 命名约定（官方文档：naming-conventions.md）

### 微流前缀（官方规定）

| 前缀 | 触发场景 |
|------|---------|
| `ACT_` | 按钮或页面触发的用户操作微流 |
| `SUB_` | 被其他微流调用的子微流 |
| `DS_` | 页面 datasource 微流 |
| `VAL_` | 数据验证微流 |
| `CAL_` | 计算属性微流（Calculated attribute） |
| `WFA_` | 工作流用户分配微流（Workflow user/group assignment） |
| `WFS_` | 工作流系统操作微流（由工作流活动直接调用） |
| `WFC_` | 工作流创建事件微流（On created event） |
| `SCE_` | 计划任务微流（Scheduled event） |
| `BCO_` / `ACO_` | Before/After commit 实体事件处理器 |
| `BCR_` / `ACR_` | Before/After create 实体事件处理器 |
| `BDE_` / `ADE_` | Before/After delete 实体事件处理器 |
| `BRO_` / `ARO_` | Before/After rollback 实体事件处理器 |
| `OEN_` / `OCH_` / `OLE_` | On enter/change/leave 属性事件处理器 |
| `CWS_` / `CRS_` | Consumed web/REST service operation |
| `PWS_` / `PRS_` / `POS_` | Published web/REST/OData service operation |
| `TEST_` / `UT_` | 单元测试微流 |

### 纳流前缀

**官方文档未规定**纳流命名前缀。项目惯例使用 `NF_`（与现有 mdl-examples 一致）。

### 其他文档类型命名

| 类型 | 规范 |
|------|------|
| Pages | `{Entity}_Overview`、`{Entity}_NewEdit`、`{Entity}_Detail` |
| Snippets | 描述性名称，无强制前缀 |
| Layouts | 描述性名称，如 `Atlas_Default`、`PopupLayout` |
| Enumerations | PascalCase，值用 CamelCase |
| Workflows | PascalCase，描述业务流程 |

---

## 2. 纳流（Nanoflow）能力

来源：`refguide/modeling/application-logic/microflows-and-nanoflows/activities/`

### 纳流支持的完整活动列表

**对象活动（Object Activities）— 全部支持：**
- Create object（创建对象，含持久化实体）
- Change object
- Commit object(s)（在线：发网络请求到 Runtime；离线：写本地 DB）
- Delete object(s)
- Retrieve object(s)（数据库查询，每次独立网络请求）
- Rollback object

**列表活动（List Activities）— 全部支持：**
- Aggregate list、Change list、Create list、List operation

**调用活动（Call Activities）：**
- Call JavaScript action（**纳流专属**）
- Call microflow（✅ 支持，离线时参数有限制）
- Call nanoflow（**纳流专属**）

**客户端活动（Client Activities）：**
- Close page、Show page、Show message、Validation feedback
- Synchronize（**纳流专属**，用于离线同步）

**变量活动：** Change variable、Create variable

**日志活动：** Log message

### 纳流明确不支持的活动

- Call Java action
- Download file
- Show home page
- 所有 Integration activities（REST、Web Service、Mapping、External Database）
- **所有工作流活动**（Workflow activities）
- External Object activities
- ML Kit（Call ML model）
- Metrics activities

### 纳流 vs 微流关键差异

| 维度 | 微流 | 纳流 |
|------|------|------|
| 执行位置 | 服务端 Runtime | 客户端（浏览器/移动设备） |
| 数据库访问 | 完整事务支持 | 支持但每次操作为独立网络请求 |
| Java actions | ✅ | ❌ |
| JavaScript actions | ❌ | ✅ |
| 工作流活动 | ✅ | ❌ |
| REST/WebService | ✅ | ❌ |
| 离线支持 | ❌ | ✅ |
| 事务（Rollback） | ✅ | ❌（无自动回滚） |
| 错误处理 | `ON ERROR` + `RAISE ERROR` | `ON ERROR` per action；`RAISE ERROR` **禁用** |
| 客户端操作执行时机 | 纳流结束时 | **立即执行**（open page 即刻生效） |

### 离线（Native mobile）调用微流的限制

从纳流调用微流时（离线场景），参数只允许：
- 基本类型（string、integer、boolean、decimal、datetime）
- 没有关联持久化实体的非持久化实体

---

## 3. 工作流（Workflow）

来源：`refguide/modeling/application-logic/workflows/`

### 3.1 工作流活动完整列表

| 活动类型 | 说明 |
|---------|------|
| Start event | 起点，每个工作流一个 |
| End event | 终点，支持**多个**（不同路径各自结束） |
| **User task** | 单用户任务，支持 XPath/Microflow targeting |
| **Multi-user task** | 多用户任务，多种决策方法 |
| **AI agent task** | AI 代理任务（Mendix 11.9+） |
| **Decision** | 条件分支（Boolean 或 Enumeration 类型） |
| **Parallel split** | 并行路径，全部完成后汇合 |
| **Call microflow** | 调用微流，返回值可驱动 outcomes |
| **Call workflow** | 调用子工作流（父子工作流） |
| **Jump** | 跳转到工作流中的指定活动 |
| **Wait for notification** | 挂起，等待 `Notify workflow` 微流活动唤醒 |
| **Timer** | 定时器（独立活动 or 边界事件） |
| Annotation | 注释，文档化用途 |
| **Boundary event** | 附加在其他活动上，仅支持 Timer 类型 |

### 3.2 Boundary Event 约束

- **唯一支持的边界事件类型：Timer**
- 可附加边界事件的活动：User task、Multi-user task、Call microflow、Call workflow、Wait for notification、AI agent task（11.9+）
- 两种子类型：
  - **Interrupting（中断）**：触发时终止父活动，沿边界路径继续
  - **Non-interrupting（非中断）**：触发时父活动继续运行，同时启动边界路径；支持 Recurrence（重复）

### 3.3 Multi-user task 决策方法

| 方法 | 说明 |
|------|------|
| Consensus | 所有参与者必须选同一结果 |
| Veto | 任何人选某结果即触发（否决权） |
| Majority | 多数人选的结果获胜 |
| Threshold | 达到指定数量/比例即触发 |
| Microflow | 自定义微流决定最终结果 |

### 3.4 父子工作流（Call workflow）约束

- 子工作流上下文实体**不必须与父相同**，可以是与父上下文有关联的对象（用表达式映射）
- 父工作流**等待**子工作流完成才继续
- **状态级联**：
  - 子失败/被中止 → 父失败
  - 父被中止/重启 → 子被中止
  - 父跳转到不同活动 → 子被中止
  - 父重试失败的活动 → 新子工作流被触发

### 3.5 工作流上下文实体约束

- 每个工作流**只有 1 个**上下文实体（`parameter $ContextVar: Module.Entity`）
- 必须是**持久化实体**（因为工作流实例需要持久化状态）

### 3.6 工作流页面参数要求

| 页面类型 | 必须声明的参数 |
|---------|--------------|
| 用户任务页面（Task page） | `$WorkflowUserTask: System.WorkflowUserTask` |
| 工作流概览页面（Overview page） | `$Workflow: System.Workflow` |

---

## 4. 微流中的工作流活动（完整 13 个）

来源：`refguide/modeling/application-logic/microflows-and-nanoflows/activities/workflow-activities/`

| 活动 | 用途 |
|------|------|
| **Call workflow** | 触发一个工作流实例，传入 context 对象 |
| **Change workflow state** | 控制工作流状态：Abort / Continue / Pause / Unpause / Restart / Retry |
| **Complete user task** | 以指定 outcome 完成用户任务（系统侧） |
| **Apply jump-to option** | 应用跳转选项（与 Generate 配合使用） |
| **Generate jump-to options** | 生成当前路径可跳转活动列表（返回 `System.WorkflowJumpToDetails` 列表） |
| **Retrieve workflow activity records** | 检索活动历史时间线（`System.WorkflowActivityRecord`） |
| **Retrieve workflow context** | 检索工作流上下文实体对象 |
| **Retrieve workflows** | 检索与某 context 对象关联的工作流实例列表 |
| **Show user task page** | 打开用户任务页面（支持 Auto-assign、Who Can Open 配置） |
| **Show workflow admin page** | 打开工作流管理/概览页面 |
| **Lock workflow** | 锁定工作流定义，防止启动新实例；可暂停现有实例 |
| **Unlock workflow** | 解锁工作流定义；可恢复暂停的实例 |
| **Notify workflow** | 唤醒 Wait for notification 或触发事件子流程 |

`Notify workflow` 返回值（Boolean）：
- `true`：成功通知，或通知被接受并排队
- `false`：无匹配接收者，或事件子流程已在运行

---

## 5. 安全模型

来源：`refguide/modeling/security/`

### 5.1 模块角色 vs 用户角色

| 维度 | 用户角色（User Role） | 模块角色（Module Role） |
|------|---------------------|----------------------|
| 定义范围 | 应用级别（App-level） | 模块级别（Module-level） |
| 用户可见 | ✅（用户管理界面显示） | ❌ |
| 用途 | 聚合多个模块角色，分配给用户 | 控制具体的实体/页面/微流权限 |
| 复用 | 模块特定 | 可随模块发布到 Marketplace 复用 |

### 5.2 Entity Access（实体访问规则）

- XPath 约束只能应用于**持久化实体**
- XPath 约束语法示例：
  ```
  [HD.Ticket_Customer/HD.Customer/System.owner = '[%CurrentUser%]']
  [IsInternal = false]
  [Status = 'Published']
  [System.owner = '[%CurrentUser%]']
  ```

### 5.3 导航菜单可见性

- **菜单项没有直接的 `roles` 属性**
- 可见性**自动派生**自其指向的页面/微流的访问权限
- MDL 导航语法中无 `roles (...)` 属性

---

## 6. MDL 表达式语法

来源：项目源码（`mdl/visitor/visitor_microflow_expression.go`）

### 常量引用

在 MDL 表达式中，常量用 `@` 前缀引用：
```
@Module.ConstantName
```

示例：
```mdl
declare $Hours integer = @HD.SLA_HIGH_HOURS;
SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
```

### 时间函数

```
addHours('[%CurrentDateTime%]', N)
addDays('[%CurrentDateTime%]', N)
[%CurrentDateTime%]        -- 当前时间
[%BeginOfCurrentDay%]      -- 今天零点
[%WeekLength%]             -- 一周时长（用于算术）
[%DayLength%]              -- 一天时长（用于算术）
```

### 工作流 Timer 表达式

```
'addHours([%CurrentDateTime%], 24)'
'addDays([%CurrentDateTime%], 3)'
```

---

## 7. 其他重要约束

### 非持久化实体

- 非持久化实体上的 XPath 约束无效（数据库不应用约束）
- Retrieve object(s) 的 from database 选项对非持久化实体无意义

### 关联命名悖论（BSON 反直觉）

**见 CLAUDE.md**：`ParentPointer` 指向 FROM 实体（外键持有方），`ChildPointer` 指向 TO 实体。与命名直觉相反。

### mx check 验证

- `mprcontents/mprname` 必须与 `.mpr` 文件名一致，否则 `StorageMprNameDiscrepancyException`
- 测试目录：`testdata/corpus-b/app.mpr`（11.6.4）、`testdata/expr-checker/minimal.mpr`（11.6.6）
