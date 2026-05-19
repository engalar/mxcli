# System Token 全面更新设计

**日期**: 2026-05-19  
**状态**: 已批准，待实现  
**权威来源**: `mendix_docs/content/en/docs/refguide/modeling/xpath/xpath-constraints/xpath-keywords-and-system-variables.md`

---

## 背景与目标

当前 system token 知识散落在三个层，且均不完整：

| 层 | 文件 | 现状缺口 |
|---|---|---|
| 类型推断 | `internal/expr/typecheck/inferrer.go` | 仅 13 个 token；无 Minute/Hour 粒度、UTC 变体、Yesterday/Tomorrow；`UserRole_*` 无验证 |
| 技能文档 | `.claude/skills/mendix/xpath-constraints.md` | 缺 UTC 变体、SecondLength/HourLength/WeekLength/MonthLength/YearLength |
| 提示文档 | `docs/06-mdl-reference/expr-hints.md` | 无 SEM-08 规则 |

**目标**：以 mendix_docs 为权威来源，三层同步，新增 `internal/expr/tokens/` 注册表包作为单一事实来源；同时添加覆盖 XPath 和微流两个上下文的 MDL 示例脚本。不涉及 LSP。

---

## 第 1 节：Central Token Registry

### 新建包：`internal/expr/tokens/`

```go
// internal/expr/tokens/registry.go

type Kind int

const (
    KindDateTime  Kind = iota // 时间点 token
    KindDuration              // 时间长度 token（DayLength 等）
    KindObjectRef             // CurrentUser / CurrentObject
    KindBoolean               // True / False
    KindEmpty                 // Null
)

type Token struct {
    Name   string
    Kind   Kind
    HasUTC bool // 是否存在对应的 UTC 变体
    IsUTC  bool // 本身是否是 UTC 变体
}

// All 是静态 token 完整列表，共 43 个（19 时间点 + 18 UTC + 7 长度 + 2 对象 + 2 布尔 + 1 Null）
var All = []Token{
    // 对象相关
    {Name: "CurrentUser",   Kind: KindObjectRef},
    {Name: "CurrentObject", Kind: KindObjectRef},

    // 时间点（基础）
    {Name: "CurrentDateTime",        Kind: KindDateTime},
    {Name: "BeginOfCurrentMinute",   Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentMinute",     Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfCurrentHour",     Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentHour",       Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfCurrentDay",      Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentDay",        Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfYesterday",       Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfYesterday",         Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfTomorrow",        Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfTomorrow",          Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfCurrentWeek",     Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentWeek",       Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfCurrentMonth",    Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentMonth",      Kind: KindDateTime, HasUTC: true},
    {Name: "BeginOfCurrentYear",     Kind: KindDateTime, HasUTC: true},
    {Name: "EndOfCurrentYear",       Kind: KindDateTime, HasUTC: true},

    // 时间点（UTC 变体，18 个）
    {Name: "BeginOfCurrentMinuteUTC",  Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentMinuteUTC",    Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfCurrentHourUTC",    Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentHourUTC",      Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfCurrentDayUTC",     Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentDayUTC",       Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfYesterdayUTC",      Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfYesterdayUTC",        Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfTomorrowUTC",       Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfTomorrowUTC",         Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfCurrentWeekUTC",    Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentWeekUTC",      Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfCurrentMonthUTC",   Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentMonthUTC",     Kind: KindDateTime, IsUTC: true},
    {Name: "BeginOfCurrentYearUTC",    Kind: KindDateTime, IsUTC: true},
    {Name: "EndOfCurrentYearUTC",      Kind: KindDateTime, IsUTC: true},

    // 时间长度（7 个）
    {Name: "SecondLength", Kind: KindDuration},
    {Name: "MinuteLength", Kind: KindDuration},
    {Name: "HourLength",   Kind: KindDuration},
    {Name: "DayLength",    Kind: KindDuration},
    {Name: "WeekLength",   Kind: KindDuration},
    {Name: "MonthLength",  Kind: KindDuration},
    {Name: "YearLength",   Kind: KindDuration},

    // 布尔 / Null
    {Name: "True",  Kind: KindBoolean},
    {Name: "False", Kind: KindBoolean},
    {Name: "Null",  Kind: KindEmpty},
}

// Lookup 精确查找静态 token（O(1)，内部用 map 实现）
func Lookup(name string) (Token, bool)

// LookupUserRole 匹配 "UserRole_<RoleName>" 前缀，返回角色名部分
func LookupUserRole(name string) (roleName string, ok bool)
```

`UserRole_*` 不进 `All`（动态，无限多），由 `LookupUserRole()` 按前缀 `"UserRole_"` 匹配处理。

---

## 第 2 节：类型推断层 + SEM-08

### `internal/expr/typecheck/inferrer.go`

`inferToken()` 签名扩展，增加 `meta.Index` 参数：

```go
func inferToken(token string, meta meta.Index) exprcheck.TypeKind {
    // 1. 静态 token 查表
    if t, ok := tokens.Lookup(token); ok {
        return kindToTypeKind(t.Kind)
    }
    // 2. 动态 UserRole_* 匹配
    if roleName, ok := tokens.LookupUserRole(token); ok {
        if meta != nil && !meta.HasUserRole(roleName) {
            return exprcheck.KindUnknown // 驱动 SEM-08
        }
        return exprcheck.KindObjectRef
    }
    return exprcheck.KindUnknown
}

func kindToTypeKind(k tokens.Kind) exprcheck.TypeKind {
    switch k {
    case tokens.KindDateTime:  return exprcheck.KindDateTime
    case tokens.KindDuration:  return exprcheck.KindInteger // 毫秒数，整型语义
    case tokens.KindObjectRef: return exprcheck.KindObject
    case tokens.KindBoolean:   return exprcheck.KindBoolean
    case tokens.KindEmpty:     return exprcheck.KindEmpty
    }
    return exprcheck.KindUnknown
}
```

### `internal/expr/meta/index.go`

追加接口方法：

```go
type Index interface {
    // ... 现有方法不变
    HasUserRole(name string) bool
}
```

`CatalogReader` 实现通过查询 `System.UserRole` 的 `Name` 列实现。

### `internal/expr/validate/validate_sem.go`

新增规则 SEM-08：

```
ID:       SEM-08
Code:     unknown-user-role
Severity: error
Message:  "user role '%s' does not exist in this project"
Trigger:  inferToken 返回 KindUnknown 且 token 前缀为 UserRole_
```

**Duration token 上下文检查**：`[%DayLength%]` 单独使用无意义，但因其出现在 XPath 字符串内部，checker 无法感知上下文结构。本期记为 TODO，不实现。

---

## 第 3 节：文档层更新

### `.claude/skills/mendix/xpath-constraints.md`

在 System Variables 小节替换为完整表格，并补充三条注意事项：

1. **UTC 变体警告**：客户端表达式中，若属性 `Localize=false`，不要用 UTC 变体（时区转换会执行两次）
2. **不支持括号分组**：system variable 是字符串形式，不能用括号组合子表达式
3. **长度 token 必须在同一字符串内**：`'[%BeginOfCurrentDay%] - 3 * [%YearLength%]'`

同文件同步更新命令 `.claude/commands/mendix/` 对应的副本（若存在）。

### `docs/06-mdl-reference/expr-hints.md`

- 更新 E007（unknown-token）描述，提及完整 token 列表的参考位置
- 新增 SEM-08 条目（unknown-user-role）

### `.claude/skills/mendix/system-module.md`

无需改动（已完整）。

---

## 第 4 节：MDL 示例脚本

新建 `mdl-examples/tokens/` 目录，三个文件：

### `time-tokens.mdl`

- XPath：本周注册的客户、最近三年的订单
- 微流：写入 `[%CurrentDateTime%]` 到字段

### `user-tokens.mdl`

- XPath：`[%CurrentUser%]` 过滤当前用户数据
- 安全规则：`[%UserRole_Administrator%]` 实体访问控制

### `datetime-arithmetic.mdl`

- 综合时间运算：过去一小时、本月区间、昨天全天
- 同时覆盖 XPath 和微流两个上下文

所有示例文件可直接作为 `mxcli check` 的回归输入。

---

## 实现范围总结

| 工作项 | 文件 | 类型 |
|---|---|---|
| Token registry 包 | `internal/expr/tokens/registry.go` + 测试 | 新建 |
| inferToken 重写 | `internal/expr/typecheck/inferrer.go` | 修改 |
| meta.Index 扩展 | `internal/expr/meta/index.go` + CatalogReader | 修改 |
| SEM-08 规则 | `internal/expr/validate/validate_sem.go` | 修改 |
| 技能文档 | `.claude/skills/mendix/xpath-constraints.md` | 修改 |
| expr-hints 文档 | `docs/06-mdl-reference/expr-hints.md` | 修改 |
| MDL 示例 × 3 | `mdl-examples/tokens/*.mdl` | 新建 |

**不在范围内**：LSP 补全、Duration token 上下文合法性检查、`cmd/mxcli/skills/` 副本同步（由 `make sync-skills` 处理）。
