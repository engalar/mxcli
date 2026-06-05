# Access Gap Analysis — Design Spec

**Date:** 2026-06-05  
**Feature:** `mxcli check -p app.mpr` 自动追加权限缺口分析，输出可执行 MDL 修复片段  
**Motivation:** Account Management 章节引入的 CE2729 错误（实体权限缺口）是 AI 写 MDL 时的高频失误，需要在 `mxcli check` 层面提前拦截并给出精确修复建议。

---

## Mendix 权限模型（正确版本）

```
入口 1：导航 → Page
              └─ Widget
                   ├─ 直接引用 Entity    → 检查 ModuleRole entity grant      ← CE2729
                   └─ 直接调用 MF（按钮/datasource）
                        ├─ 检查 ModuleRole execute grant on MF               ← CE2731 类
                        └─ 若 MF.ApplyEntityAccess = ON
                             └─ 检查 MF 内用到的 Entity 的 grant（不递归）

入口 2：导航 → MF（无参微流入口）
              ├─ 检查 ModuleRole execute grant on MF
              └─ 若 MF.ApplyEntityAccess = ON
                   └─ 检查 MF 内用到的 Entity 的 grant（不递归）
```

**关键约束：**
- 只检查离 Page/Nav 最近的**直接调用层 MF** 的 execute grant
- Sub-MF（被直接层 MF 再调用的）**不检查** execute grant
- MF `ApplyEntityAccess = OFF`（Mendix 默认）时，MF 内部实体访问**绕过**权限系统，不检查
- 只有 `ApplyEntityAccess = ON` 时才向下查实体 grant，且**不递归**进 sub-MF

---

## 集成方式

**触发条件：**
- `mxcli check script.mdl`（无 MPR）→ 跳过（无 grant 数据）
- `mxcli check script.mdl -p app.mpr` → 自动追加 access analysis（在 reference check 之后）
- `mxcli check script.mdl -p app.mpr --no-access` → opt-out 开关

**输出位置：** 追加在现有 check 输出末尾，格式与 lint HINT 一致：

```
Access analysis:
[ACCESS-001] CustomerRole → page HD.ManageMyAccount → widget dvProfile → entity HD.UserProfile: no read access
  → Fix: grant HD.CustomerRole on HD.UserProfile (read *);

[ACCESS-002] AgentRole → page HD.ManageMyAccount → widget dvProfile → entity HD.UserProfile: no read access
  → Fix: grant HD.AgentRole on HD.UserProfile (read *);

[ACCESS-003] CustomerRole → page HD.ChangeMyPassword → widget dvPwdFull → entity HD.PasswordForm: no access
  → Fix: grant HD.CustomerRole on HD.PasswordForm (create, read *, write *);

3 access gaps found across 2 pages. Run `mxcli check` again after applying fixes.
```

---

## 核心算法

```
AccessAnalyzer(mpr):
  1. 读取权限数据
     - UserRole → []ModuleRole 映射
     - entity grants: {ModuleRole → {EntityQN → AccessRule}}
     - page grants:   {ModuleRole → {PageQN → bool}}
     - mf grants:     {ModuleRole → {MFQN → bool}}
     - MF.ApplyEntityAccess: {MFQN → bool}

  2. 读取结构数据
     - 每个 Page 的 Widget 树
       - Widget 直接引用的 Entity（datasource entity、attribute binding）
       - Widget 直接调用的 MF（button action、microflow datasource）
     - 每个 MF 内部操作的 Entity（retrieve/change/create/delete target）
     - Navigation profile 的入口（page 入口 / MF 入口）

  3. 对每个 UserRole:
     a. 收集该 UserRole 可访问的 ModuleRole 集合 R
     b. 遍历 Nav 入口:
        - 若入口是 Page P: 检查 R 中是否有 page grant on P
          → 若有: 收集 P 的 Widget 直接引用的 Entity 集合 E_page
                  收集 P 的 Widget 直接调用的 MF 集合 MF_direct
        - 若入口是 MF M: 检查 R 中是否有 execute grant on M
          → MF_direct = {M}
     c. 对 MF_direct 中每个 MF M:
        - 检查 execute grant（入口 MF 已在 b 中检查，按钮触发的 MF 在此检查）
        - 若 M.ApplyEntityAccess = ON: 收集 M 内操作的 Entity 加入 E_mf
     d. 对 E_page ∪ E_mf 中每个 Entity:
        - 检查 R 中是否有足够的 entity grant（read / create / write / delete）
        - 若缺失: 记录 AccessGap{role, path, entity, missingGrant}

  4. 对每个 AccessGap 生成:
     - 人类可读的诊断路径（role → page → widget → entity）
     - 可执行的 MDL 修复片段
```

---

## 实现层拆分

| 包 / 文件 | 职责 |
|-----------|------|
| `mdl/executor/access/graph.go` | `AccessGraph` 类型：从 MPR 读取并构建权限图 |
| `mdl/executor/access/analyzer.go` | `Analyzer.Run()` 遍历图，返回 `[]AccessGap` |
| `mdl/executor/access/hint.go` | `GapToMDL()` 将 `AccessGap` 转为 MDL grant 片段 + 诊断消息 |
| `mdl/executor/access/analyzer_test.go` | 单元测试（MockBackend 驱动） |
| `mdl/executor/cmd_check.go` | 集成点：`--references` 通过后调用 `Analyzer.Run()`，输出 hints |
| `mdl/backend/` | 新接口方法：`ListPageAccessGrants()`, `ListEntityAccessGrants()`, `ListMFAccessGrants()`, `GetMFApplyEntityAccess()`, `GetPageWidgetTree()` |

---

## AccessGap 数据结构

```go
type AccessGap struct {
    UserRole    string   // e.g. "Customer"
    ModuleRole  string   // e.g. "HD.CustomerRole"
    Path        []string // ["page HD.ManageMyAccount", "widget dvProfile", "entity HD.UserProfile"]
    EntityQN    string   // "HD.UserProfile"
    MissingOps  []string // ["read"] or ["create", "read *", "write *"]
    SuggestedMDL string  // "grant HD.CustomerRole on HD.UserProfile (read *);"
}
```

---

## MDL 修复片段生成规则

| 缺失访问 | 推断修复 |
|---------|---------|
| entity 在 DataView（readonly）上 | `grant Role on Entity (read *)` |
| entity 在 DataView（editable）上 | `grant Role on Entity (create, read *, write *)` |
| entity 在 DataGrid 上 | `grant Role on Entity (read *)` |
| entity 在 MF datasource（非持久化） | `grant Role on Entity (create, read *, write *)` |
| MF execute 缺失 | `grant execute on microflow Module.MF to Role;` |

---

## 测试策略

| 层 | 测试 |
|----|------|
| L1 单元 | `AccessGraph` 从 mock grant 数据正确构建图 |
| L1 单元 | `Analyzer.Run()` 发现 CE2729 类缺口、正确跳过 ApplyEntityAccess=OFF 的 MF |
| L2 集成 | `mxcli check helpdesk.mdl -p app.mpr` 输出包含 CE2729 对应的 ACCESS-001 HINT |
| L3 回归 | 修复后重跑，HINT 消失 |

---

## 范围之外（后续迭代）

- XPath 行级过滤器分析（`where '[System.owner=''[%CurrentUser%]'']'` 等）
- CE0463 类 widget 定义不一致分析
- MF 内部 sub-MF 的递归分析
- `mxcli lint` 静态规则（无 MPR 时的浅层检查）
