# FT 能力缺口修复 + MDL 内容补充 — Design

**Date**: 2026-06-08  
**Status**: Design approved, pending implementation  
**Pair file**: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

## 背景

`helpdesk-app.mdl` 的 FT 扩展（FieldTech 模块）与原始设计文档存在 8 处偏差。本规格覆盖其中 4 项可修复的能力缺口，每项采用"能力修复 → TDD → MDL 内容补充 → mxcli check + mx check 双验证"的逐项闭环策略。

**不在本规格范围内的偏差：**
- Business Events Service：Studio Pro Build Extension 崩溃（外部约束，需 marketplace 模块）
- OQL avg() 含算术：Mendix OQL 硬性约束
- Nanoflow 页面带参（CE0115）：需 Studio Pro BSON 样本
- OfflineNative 导航：归入本规格第 2 项
- ALTER PAGE `$currentObject` DataGrid：Mendix 语义约束，暂不处理

## 成功标准

每项能力修复完成后：
- `./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` → **0 errors**
- `~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/helpdesk-golden-11.6.6/minimal.mpr` → **0 errors**

## 实施顺序（逐项闭环）

### 能力 1：SYNCHRONIZE 语法（Medium）

#### 能力修复

Mendix `Microflows$SynchronizeAction` 是离线 nanoflow 的核心活动（上传本地变更 + 拉取服务端数据）。当前 mxcli grammar 完全缺失此关键字。

**官方文档来源**: `mendix_docs/content/en/docs/refguide10/modeling/application-logic/microflows-and-nanoflows/activities/client-activities/synchronize.md`

**同步模式（三种）:**
1. **All Objects** - 同步整个本地数据库（含 commit 对象）
2. **Unsynchronized Objects** - 仅同步有本地变更的对象  
3. **Selected Object(s)** - 同步指定对象或列表

**MDL 语法设计:**
```mdl
synchronize $Order;      -- Selected Object(s)：同步单个变量
synchronize $OrderList;  -- Selected Object(s)：同步列表
synchronize;             -- All Objects 或 Unsynchronized Objects（无参）
```

**改动范围:**

| 层 | 文件 | 改动 |
|----|------|------|
| Lexer | `mdl/grammar/MDLLexer.g4` | 新增 `SYNCHRONIZE` token |
| Parser | `mdl/grammar/MDLParser.g4` | 在 `nanoflowActivity` 规则中加 `synchronizeStmt` |
| AST | `mdl/ast/` 新文件 | `SynchronizeStmt { Variable string; Mode string }` |
| Visitor | `mdl/visitor/` | ANTLR listener → `SynchronizeStmt` |
| Executor | `mdl/executor/` 新文件 | `execSynchronize` → 写 `Microflows$SynchronizeAction` BSON |
| Grammar 重生 | `make grammar` | 重新生成 parser 文件 |

**BSON 字段（`Microflows$SynchronizeAction`）:**

基于 Mendix metamodel reflection data 中的 SynchronizeAction 类型：
- `SyncType`: enum（`All` / `Unsynchronized` / `SelectedObjects`）
- `VariableNames`: PartList of strings（Selected 模式下的变量名列表）

**TDD 策略:**
1. 写最小失败 MDL：`synchronize $Order;` 在 nanoflow 中 → parse error
2. 实现 Lexer/Parser/AST/Visitor → parse error 消失
3. 写 executor 单元测试：验证生成的 BSON 含 `SyncType = "SelectedObjects"` 和正确 `VariableNames`
4. 实现 executor → BSON 断言通过
5. `make grammar` + `mxcli check` + `mx check`

#### MDL 内容补充

```mdl
-- NF_FT_CheckIn：签到记录写入后同步
$Record = create FT.CheckInRecord (...);
commit $Record;
synchronize $Record;   -- 推送签到记录到服务器

-- NF_FT_CompleteOffline：完工记录同步
change $Order (Status = FT.DispatchStatus.Completed, ...);
commit $Order;
synchronize $Order;    -- 有网时推送，无网时排队

-- NF_FT_LoadOrders：拉取最新派工单
synchronize;           -- All Objects：从服务器拉最新数据
retrieve $Orders from FT.DispatchOrder ...;
```

---

### 能力 2：OfflineNative Navigation 创建（Medium）

#### 能力修复

当前 `cmd_navigation.go` 中 `execAlterNavigation` 强制验证 profile 必须已存在。需在 `create or replace navigation` 语义中支持创建新 profile。

**官方文档来源**: `mendix_docs/content/en/docs/refguide10/modeling/app-explorer/app/navigation/_index.md`

**必填字段（Native Navigation Profile）:**
- Default Home Page（page 或 nanoflow）
- Role-based Home Pages（可选）
- Login page（可选）
- Navigation menu（可选）

**Profile 类型映射:**
```
Responsive    → Navigation$WebNavigationProfile
OfflineNative → Navigation$NativeMobileNavigationProfile
```

**改动范围:**

| 层 | 文件 | 改动 |
|----|------|------|
| Backend interface | `mdl/backend/` | 新增 `CreateNavigationProfile(profile *model.NavigationProfile) error` |
| Backend MPR impl | `mdl/backend/mpr/` | 实现：创建新 navigation document，设置 profile type，写入 home/menu |
| Executor | `mdl/executor/cmd_navigation.go` | `create or replace navigation`：若 profile 不存在则调用 `CreateNavigationProfile`；若已存在则走现有 patch 路径 |

**BSON 结构（初始化模板）：**  
从 clean MPR 中现有 NativeMobile 配置（如有）或从 `Navigation$NativeMobileNavigationProfile` reflection data 提取必填字段。

**TDD 策略:**
1. 写 executor 集成测试：对不含 OfflineNative 的 clean MPR，执行 `create or replace navigation OfflineNative ...` → 验证 profile 被创建
2. 实现 CreateNavigationProfile → 测试通过
3. 写 `mx check` 集成验证：创建后无 StorageLoadException

#### MDL 内容补充

```mdl
create or replace navigation OfflineNative
  home page FT.DispatchQueue for FieldTech
  home page HD.Ticket_Overview
  login page Administration.login
  menu (
    menu item 'My Orders'    page FT.DispatchQueue;
    menu item 'Order Detail' page FT.DispatchOrder_Detail;
  );
```

---

### 能力 3：reversed() XPath（Hard）

#### 能力修复

**官方文档关键约束**（来源：`mendix_docs/content/en/docs/refguide10/modeling/domain-model/associations/query-over.md`）：

> `[reversed()]` 仅适用于**自引用关联（self-referencing associations）**，用于消除方向歧义。对于不同类型实体间的关联，平台自动确定方向。

**原设计的 MDL 内容存在错误：**  
`[KB.ArticleTag_Article[reversed()]/KB.ArticleTag/...]` 中 ArticleTag → Article 是不同类型关联，不应使用 `reversed()`。CE0161 可能是 Mendix 正确拒绝了非法路径。

**调查策略（先于实现）：**

1. 用 `KB.Category_Parent`（Category 自引用父关联）写最小单跳 reversed() XPath：
   ```mdl
   retrieve $Children from KB.Category
     where [KB.Category_Parent[reversed()] = $ParentCategory]
     limit 10;
   ```
2. 执行 `mxcli check` + `mx check`，判断 CE0161 是否出现：
   - **若 0 errors**：单跳自引用 reversed() 正常，原 CE0161 是多跳不同类型路径的错误，无需修复 mxcli（原设计 MDL 语义错误）
   - **若 CE0161 仍出现**：mxcli 对 reversed() BSON 格式有问题，需对比 reflection data 中 `XPathConstraint` 的正确格式后修复 executor

**改动范围（若需修复）：**

| 层 | 文件 | 改动 |
|----|------|------|
| Executor/visitor | `mdl/visitor/visitor_xpath.go` 或 `entity_from_ast.go` | 修正 reversed() 路径的 BSON 编码 |
| 测试 | `mdl/visitor/visitor_xpath_test.go` | 新增 reversed() 单跳 + 自引用场景测试 |

#### MDL 内容补充

使用正确的自引用关联场景（与 `KB.Category_Parent` 自引用关联匹配）：

```mdl
-- DS_CategoryChildren: 演示 reversed() 在自引用关联中的正确用法
-- KB.Category_Parent 从 KB.Category 指向其父 KB.Category（自引用）
-- [reversed()] 表示：找所有将 $ParentCategory 作为父节点的 Category（即子节点）
create or modify microflow FT.DS_CategoryChildren
  ($ParentCategory: KB.Category)
  returns list of KB.Category as $Children
  folder 'Dispatch'
{
  retrieve $Children from KB.Category
    where [KB.Category_Parent[reversed()] = $ParentCategory]
    limit 50;
  return $Children;
}
```

---

### 能力 4：Import/Export Mapping source 链接（Hard）

#### 能力修复

CE0271 "The selected source is not valid" 表明 mapping 文档中的 JSON Structure 引用未被 Mendix 识别为有效 source。

**官方文档来源**: `mendix_docs/content/en/docs/refguide/modeling/integration/mapping-documents/import-mappings.md`

关键发现：mapping 与 source 的绑定涉及元素树选择（"Select elements..."），不仅是 QN 字符串引用。

**调查策略（先于实现）：**

1. 在 clean MPR 中找现有 import/export mapping 样本，dump BSON
2. 对比 mxcli 生成的 mapping BSON 与 Studio Pro 生成的差异：
   - `JsonStructure` 字段格式：string QN vs binary ID
   - 是否缺少 `Elements` 树（选中的 JSON 路径节点）
   - 是否缺少 `RootElementName` 或其他必填字段
3. 根据差异确定修复点

**可能根因与修复位置：**

| 可能根因 | 修复文件 |
|---------|---------|
| `JsonStructure` 需要 binary ID 而非 QN 字符串 | `mdl/executor/cmd_import_mappings.go` + `mdl/backend/mpr/` |
| 缺少 `Elements` 子树（mapping elements 选择） | executor/backend 补写 mapping elements |
| `RootElementName` 字段未正确设置 | executor 补充字段 |

**TDD 策略:**
1. Dump clean MPR 中已有 mapping 的 BSON → 与 mxcli 生成对比
2. 写最小失败测试：执行 `create import mapping` 后 mx check → CE0271 出现
3. 修复 BSON 格式 → CE0271 消失 → 测试通过

#### MDL 内容补充

从注释恢复为可执行语句：

```mdl
create import mapping FT.WorkOrderImport_Mapping
  with json structure FT.WorkOrderPayload
{
  create FT.WorkOrderImport {
    TicketRef   = ticketRef,
    TechName    = techName,
    ScheduledAt = scheduledAt
  }
};

create export mapping FT.DispatchOrder_Export
  with json structure FT.WorkOrderPayload
{
  FT.DispatchOrder {
    ticketRef   = SiteNotes,
    scheduledAt = DispatchedAt
  }
};
```

恢复 `FT.ACT_WorkOrder_Import` 中的调用：

```mdl
declare $Imported: FT.WorkOrderImport = empty;
$Imported = import from mapping FT.WorkOrderImport_Mapping($JsonPayload);
if $Imported != empty {
  $Order = create FT.DispatchOrder (
    Status    = FT.DispatchStatus.Pending,
    SiteNotes = 'Imported: ' + $Imported/TicketRef
  );
  commit $Order;
}
```

---

## 依赖关系与实施顺序

```
能力1 (SYNCHRONIZE)   → make grammar → MDL content 1
      ↓
能力2 (OfflineNative) → MDL content 2
      ↓
能力3 (reversed())    → 先调查 → 可能仅 MDL 改动 → MDL content 3
      ↓
能力4 (Import/Export) → 先调查 → BSON 修复 → MDL content 4
```

能力 1 必须先做（grammar 变更需 `make grammar`，后续步骤都依赖它）。能力 3 和 4 各有调查阶段，需先确认根因再实施。

## 不变内容

以下偏差不在本规格范围，保持现状（注释形式）：
- Business Events Service（`FT.FieldWorkEvents`）
- ALTER PAGE dispatch history DataGrid（`$currentObject` 上下文约束）
- Nanoflow 页面带参（CE0115，需 Studio Pro BSON 样本）
