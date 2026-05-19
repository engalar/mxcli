# System Token 全面更新 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以 mendix_docs 为权威来源，新建 token 注册表包，三层同步（类型推断 / 技能文档 / hints 文档），新增 SEM-08 UserRole 严格校验，并补充 MDL 示例脚本。

**Architecture:** 新建 `internal/expr/tokens/` 包作为单一权威来源。`inferrer.go` 查表替代 hardcode switch。SEM-08 在 `validate_sem.go` 单独实现，通过扩展 `IndexReader` 接口接入 `meta.Index` 的 UserRole 数据。

**Tech Stack:** Go 1.22+, `github.com/mendixlabs/mxcli` module, `github.com/mendixlabs/mxcli/modelsdk/gen/security` (UserRole), testify/assert

---

## File Map

| 文件 | 操作 | 职责 |
|---|---|---|
| `internal/expr/tokens/registry.go` | 新建 | 完整 token 列表、Lookup/LookupUserRole |
| `internal/expr/tokens/registry_test.go` | 新建 | 数量/精确查找/UserRole 前缀匹配 |
| `internal/expr/typecheck/inferrer.go` | 修改 | `inferToken()` 改为查表 |
| `internal/expr/typecheck/typecheck_test.go` | 修改 | token 类型推断测试补全 |
| `internal/expr/validate/validate_sem.go` | 修改 | `IndexReader` 增 HasUserRole + SEM-08 规则 |
| `internal/expr/validate/validate_test.go` | 修改 | SEM-08 测试用例 |
| `internal/expr/meta/index.go` | 修改 | `userRoles` 字段 + `buildUserRoles()` |
| `internal/expr/meta/catalog_reader.go` | 修改 | `HasUserRole()` 方法 |
| `internal/expr/meta/mock_index.go` | 修改 | `AddUserRole()`/`HasUserRole()` |
| `.claude/skills/mendix/xpath-constraints.md` | 修改 | 完整 token 表 + 3 条注意事项 |
| `docs/06-mdl-reference/expr-hints.md` | 修改 | SEM-08 条目 |
| `mdl-examples/tokens/time-tokens.mdl` | 新建 | 时间 token 示例 |
| `mdl-examples/tokens/user-tokens.mdl` | 新建 | CurrentUser/UserRole 示例 |
| `mdl-examples/tokens/datetime-arithmetic.mdl` | 新建 | 时间运算示例 |

---

### Task 1: 新建 Token Registry 包

**Files:**
- Create: `internal/expr/tokens/registry.go`
- Create: `internal/expr/tokens/registry_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/expr/tokens/registry_test.go
package tokens_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/tokens"
)

func TestAll_Count(t *testing.T) {
	// 2 对象 + 1 CurrentDateTime + 18 基础时间点(HasUTC) + 18 UTC变体 + 7 长度 + 2 布尔 + 1 Null = 49
	if got := len(tokens.All); got != 49 {
		t.Errorf("All has %d tokens, want 49", got)
	}
}

func TestAll_NoUTCDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, tok := range tokens.All {
		if seen[tok.Name] {
			t.Errorf("duplicate token name: %s", tok.Name)
		}
		seen[tok.Name] = true
	}
}

func TestLookup_StaticToken(t *testing.T) {
	cases := []struct {
		name string
		kind tokens.Kind
	}{
		{"CurrentDateTime", tokens.KindDateTime},
		{"BeginOfCurrentMinute", tokens.KindDateTime},
		{"BeginOfCurrentMinuteUTC", tokens.KindDateTime},
		{"BeginOfYesterday", tokens.KindDateTime},
		{"BeginOfTomorrow", tokens.KindDateTime},
		{"DayLength", tokens.KindDuration},
		{"SecondLength", tokens.KindDuration},
		{"HourLength", tokens.KindDuration},
		{"WeekLength", tokens.KindDuration},
		{"MonthLength", tokens.KindDuration},
		{"YearLength", tokens.KindDuration},
		{"CurrentUser", tokens.KindObjectRef},
		{"CurrentObject", tokens.KindObjectRef},
		{"True", tokens.KindBoolean},
		{"False", tokens.KindBoolean},
		{"Null", tokens.KindEmpty},
	}
	for _, tc := range cases {
		tok, ok := tokens.Lookup(tc.name)
		if !ok {
			t.Errorf("Lookup(%q): not found", tc.name)
			continue
		}
		if tok.Kind != tc.kind {
			t.Errorf("Lookup(%q).Kind = %v, want %v", tc.name, tok.Kind, tc.kind)
		}
	}
}

func TestLookup_Unknown(t *testing.T) {
	_, ok := tokens.Lookup("NotAToken")
	if ok {
		t.Error("Lookup(NotAToken) should return false")
	}
}

func TestLookupUserRole_Match(t *testing.T) {
	name, ok := tokens.LookupUserRole("UserRole_Administrator")
	if !ok {
		t.Error("LookupUserRole(UserRole_Administrator): expected ok")
	}
	if name != "Administrator" {
		t.Errorf("LookupUserRole role name = %q, want %q", name, "Administrator")
	}
}

func TestLookupUserRole_NoMatch(t *testing.T) {
	_, ok := tokens.LookupUserRole("CurrentUser")
	if ok {
		t.Error("LookupUserRole(CurrentUser): should not match")
	}
	_, ok = tokens.LookupUserRole("UserRole_") // empty role name
	if ok {
		t.Error("LookupUserRole(UserRole_) empty role: should not match")
	}
}

func TestAll_UTCPairing(t *testing.T) {
	// Each HasUTC=true token must have a corresponding UTC variant in All.
	names := map[string]bool{}
	for _, tok := range tokens.All {
		names[tok.Name] = true
	}
	for _, tok := range tokens.All {
		if tok.HasUTC && !tok.IsUTC {
			utcName := tok.Name + "UTC"
			if !names[utcName] {
				t.Errorf("token %q has HasUTC=true but %q not found in All", tok.Name, utcName)
			}
		}
	}
}

func TestAll_DurationKinds(t *testing.T) {
	want := []string{"SecondLength", "MinuteLength", "HourLength", "DayLength", "WeekLength", "MonthLength", "YearLength"}
	for _, name := range want {
		tok, ok := tokens.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%q): not found", name)
			continue
		}
		if tok.Kind != tokens.KindDuration {
			t.Errorf("Lookup(%q).Kind = %v, want KindDuration", name, tok.Kind)
		}
		if strings.HasSuffix(name, "UTC") {
			t.Errorf("duration token %q should not be UTC variant", name)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/tokens/... 2>&1 | head -20
```

期望输出：`cannot find package` 或 `no Go files`

- [ ] **Step 3: 实现 registry.go**

```go
// internal/expr/tokens/registry.go
package tokens

import "strings"

// Kind categorises a Mendix system token by its semantic type.
type Kind int

const (
	KindDateTime  Kind = iota // time-point tokens (BeginOfCurrentDay, etc.)
	KindDuration              // time-length tokens (DayLength, etc.) — integer milliseconds
	KindObjectRef             // object GUID tokens (CurrentUser, CurrentObject)
	KindBoolean               // True, False
	KindEmpty                 // Null
)

// Token is a single Mendix system token descriptor.
type Token struct {
	Name   string
	Kind   Kind
	HasUTC bool // a UTC variant of this token exists (only set on the non-UTC variant)
	IsUTC  bool // this IS the UTC variant
}

// All is the complete static token list derived from the Mendix documentation:
// https://docs.mendix.com/refguide/xpath-keywords-and-system-variables/
//
// Total: 2 object + 1 CurrentDateTime + 18 base time-point (HasUTC) +
//        18 UTC variants + 7 duration + 2 boolean + 1 null = 49
var All = []Token{
	// Object-related
	{Name: "CurrentUser",   Kind: KindObjectRef},
	{Name: "CurrentObject", Kind: KindObjectRef},

	// Time-point: no UTC variant
	{Name: "CurrentDateTime", Kind: KindDateTime},

	// Time-point: base (HasUTC=true) + UTC variants
	{Name: "BeginOfCurrentMinute",    Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentMinute",      Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentHour",      Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentHour",        Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentDay",       Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentDay",         Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfYesterday",        Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfYesterday",          Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfTomorrow",         Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfTomorrow",           Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentWeek",      Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentWeek",        Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentMonth",     Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentMonth",       Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentYear",      Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentYear",        Kind: KindDateTime, HasUTC: true},

	// UTC variants (18)
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

	// Duration (time-length, value in milliseconds — integer semantics)
	{Name: "SecondLength", Kind: KindDuration},
	{Name: "MinuteLength", Kind: KindDuration},
	{Name: "HourLength",   Kind: KindDuration},
	{Name: "DayLength",    Kind: KindDuration},
	{Name: "WeekLength",   Kind: KindDuration},
	{Name: "MonthLength",  Kind: KindDuration},
	{Name: "YearLength",   Kind: KindDuration},

	// Boolean / Null
	{Name: "True",  Kind: KindBoolean},
	{Name: "False", Kind: KindBoolean},
	{Name: "Null",  Kind: KindEmpty},
}

// index is built once from All for O(1) lookups.
var index map[string]Token

func init() {
	index = make(map[string]Token, len(All))
	for _, t := range All {
		index[t.Name] = t
	}
}

// Lookup returns the Token for an exact static token name.
// Returns (Token{}, false) for unknown names or UserRole_* patterns.
func Lookup(name string) (Token, bool) {
	t, ok := index[name]
	return t, ok
}

// LookupUserRole matches a "UserRole_<RoleName>" prefix pattern.
// Returns the role name portion and true when the pattern matches and
// the role name part is non-empty.
func LookupUserRole(name string) (roleName string, ok bool) {
	const prefix = "UserRole_"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	role := name[len(prefix):]
	if role == "" {
		return "", false
	}
	return role, true
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/tokens/... -v 2>&1 | tail -20
```

期望输出：所有 `PASS`，无 `FAIL`

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/tokens/
git commit -m "feat(expr/tokens): add central system token registry

49 static tokens from mendix_docs authority source.
Lookup() for O(1) static lookup; LookupUserRole() for UserRole_* pattern.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 重写 inferToken() 使用 Registry

**Files:**
- Modify: `internal/expr/typecheck/inferrer.go:142-158`
- Modify: `internal/expr/typecheck/typecheck_test.go`

- [ ] **Step 1: 在 typecheck_test.go 末尾添加失败测试**

```go
// 在 typecheck_test.go 末尾追加（保留文件现有内容）

func TestInferToken_NewTokens(t *testing.T) {
	inf := typecheck.NewInferrer()
	scope := mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	// helper: build TokenExpr AST node
	tokenNode := func(name string) exprcheck.RobustExpr {
		return &exprcheck.TokenExpr{Token: name}
	}

	cases := []struct {
		token string
		want  exprcheck.TypeKind
	}{
		// Time-point tokens that were missing before
		{"BeginOfCurrentMinute",    exprcheck.KindDateTime},
		{"EndOfCurrentMinute",      exprcheck.KindDateTime},
		{"BeginOfCurrentHour",      exprcheck.KindDateTime},
		{"EndOfCurrentHour",        exprcheck.KindDateTime},
		{"BeginOfYesterday",        exprcheck.KindDateTime},
		{"EndOfYesterday",          exprcheck.KindDateTime},
		{"BeginOfTomorrow",         exprcheck.KindDateTime},
		{"EndOfTomorrow",           exprcheck.KindDateTime},
		// UTC variants
		{"BeginOfCurrentDayUTC",    exprcheck.KindDateTime},
		{"BeginOfCurrentWeekUTC",   exprcheck.KindDateTime},
		{"BeginOfCurrentMinuteUTC", exprcheck.KindDateTime},
		// Duration tokens (value = integer milliseconds)
		{"SecondLength", exprcheck.KindInteger},
		{"MinuteLength", exprcheck.KindInteger},
		{"HourLength",   exprcheck.KindInteger},
		{"DayLength",    exprcheck.KindInteger},
		{"WeekLength",   exprcheck.KindInteger},
		{"MonthLength",  exprcheck.KindInteger},
		{"YearLength",   exprcheck.KindInteger},
		// UserRole_* → KindObject (type always correct; name validated by SEM-08)
		{"UserRole_Administrator", exprcheck.KindObject},
		{"UserRole_AnyName",       exprcheck.KindObject},
	}

	for _, tc := range cases {
		got := inf.Infer(tokenNode(tc.token), scope, cat, funcs)
		if got != tc.want {
			t.Errorf("Infer(TokenExpr{%q}) = %v, want %v", tc.token, got, tc.want)
		}
	}
}

// mockScope is already defined earlier in the file; if not, add:
type mockScope struct{ kinds map[string]exprcheck.TypeKind }
func (m mockScope) TypeOf(name string) exprcheck.TypeKind {
	if m.kinds == nil { return exprcheck.KindUnknown }
	return m.kinds[name]
}
```

**注意**: 检查文件中是否已有 `mockScope` 类型定义；若已有则跳过 mockScope 定义部分。

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/typecheck/... -run TestInferToken_NewTokens -v 2>&1
```

期望输出：`FAIL — BeginOfCurrentMinute = KindUnknown, want KindDateTime`

- [ ] **Step 3: 修改 inferrer.go**

将 `inferrer.go` 中 `inferToken()` 函数（第 142-158 行）及 import 块改为：

```go
// internal/expr/typecheck/inferrer.go
// 在文件顶部 import 块中添加：
//   "github.com/mendixlabs/mxcli/internal/expr/tokens"

func inferToken(token string) exprcheck.TypeKind {
	if t, ok := tokens.Lookup(token); ok {
		return kindToTypeKind(t.Kind)
	}
	if _, ok := tokens.LookupUserRole(token); ok {
		return exprcheck.KindObject
	}
	return exprcheck.KindUnknown
}

func kindToTypeKind(k tokens.Kind) exprcheck.TypeKind {
	switch k {
	case tokens.KindDateTime:
		return exprcheck.KindDateTime
	case tokens.KindDuration:
		// Duration tokens are millisecond integer values at runtime.
		return exprcheck.KindInteger
	case tokens.KindObjectRef:
		return exprcheck.KindObject
	case tokens.KindBoolean:
		return exprcheck.KindBoolean
	case tokens.KindEmpty:
		return exprcheck.KindEmpty
	}
	return exprcheck.KindUnknown
}
```

完整的修改后 `inferrer.go` import 块：

```go
import (
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/tokens"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)
```

- [ ] **Step 4: 运行全部 typecheck 测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/typecheck/... -v 2>&1 | tail -30
```

期望输出：所有 `PASS`，无 `FAIL`

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/inferrer.go internal/expr/typecheck/typecheck_test.go
git commit -m "feat(expr/typecheck): rewrite inferToken() using token registry

Replaces hardcoded 13-token switch with tokens.Lookup() + LookupUserRole().
Now covers 49 static tokens (Minute/Hour granularity, UTC variants,
Yesterday/Tomorrow, all 7 duration lengths) plus dynamic UserRole_* pattern.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: HasUserRole + meta.Index + SEM-08

**Files:**
- Modify: `internal/expr/validate/validate_sem.go` (IndexReader interface + checkUserRoleTokens)
- Modify: `internal/expr/validate/validate_test.go`
- Modify: `internal/expr/meta/index.go` (userRoles field + buildUserRoles)
- Modify: `internal/expr/meta/catalog_reader.go` (HasUserRole method)
- Modify: `internal/expr/meta/mock_index.go` (AddUserRole + HasUserRole)

- [ ] **Step 1: 写失败测试 — validate_test.go**

在 `validate_test.go` 末尾追加：

```go
func TestSEM08_UnknownUserRole(t *testing.T) {
	idx := meta.NewMockIndex(nil)
	idx.AddUserRole("Administrator")
	// expression with valid role
	prValid := parse.ParseExpression(makeRec("[%UserRole_Administrator%]", "Microflows$ExpressionSplitCondition", ""))
	issues := validate.ValidateSemantic(prValid, idx)
	for _, i := range issues {
		if i.RuleID == "SEM-08" {
			t.Errorf("valid role should not trigger SEM-08, got: %s", i.Message)
		}
	}

	// expression with unknown role
	prBad := parse.ParseExpression(makeRec("[%UserRole_NonExistent%]", "Microflows$ExpressionSplitCondition", ""))
	issues = validate.ValidateSemantic(prBad, idx)
	found := false
	for _, i := range issues {
		if i.RuleID == "SEM-08" {
			assert.Equal(t, "ERROR", i.Severity)
			assert.Contains(t, i.Message, "NonExistent")
			found = true
		}
	}
	assert.True(t, found, "unknown user role must trigger SEM-08")
}

func TestSEM08_NilIdx_NoError(t *testing.T) {
	pr := parse.ParseExpression(makeRec("[%UserRole_X%]", "Microflows$ExpressionSplitCondition", ""))
	issues := validate.ValidateSemantic(pr, nil)
	assert.Empty(t, issues, "nil idx must not trigger SEM-08")
}
```

- [ ] **Step 2: 运行测试确认编译失败（AddUserRole 未定义）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/validate/... 2>&1 | head -15
```

期望输出：编译错误 `idx.AddUserRole undefined`

- [ ] **Step 3: 扩展 IndexReader 接口（validate_sem.go）**

在 `validate_sem.go` 的 `IndexReader` 接口（第 18-41 行）末尾加一个方法：

```go
// HasUserRole reports whether a user role with the given name exists in the project.
// Used by SEM-08 to validate [%UserRole_Name%] token references.
HasUserRole(name string) bool
```

在 `ValidateSemantic()` 函数末尾（第 56 行前）追加一行：

```go
out = append(out, checkUserRoleTokens(rec.Raw, rec, idx)...)
```

在文件末尾追加 SEM-08 实现：

```go
// ── SEM-08: UserRole token validation ────────────────────────────────────────

// userRoleTokenRe matches [%UserRole_Name%] patterns and captures the role name.
var userRoleTokenRe = regexp.MustCompile(`\[%UserRole_(\w+)%\]`)

func checkUserRoleTokens(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range userRoleTokenRe.FindAllStringSubmatch(raw, -1) {
		roleName := m[1]
		if !idx.HasUserRole(roleName) {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-08",
				Severity: "ERROR",
				Message:  fmt.Sprintf("User role '%s' does not exist in this project.", roleName),
				Fix:      "Check the role name — it may have been renamed or removed in project security settings.",
			})
		}
	}
	return out
}
```

- [ ] **Step 4: 扩展 mock_index.go**

在 `mock_index.go` 的 `MockIndex` struct 中追加字段：

```go
userRoles map[string]bool
```

在 `NewMockIndex()` 中初始化：

```go
userRoles: map[string]bool{},
```

追加两个方法：

```go
// AddUserRole registers a user role name for testing.
func (m *MockIndex) AddUserRole(name string) { m.userRoles[name] = true }

// HasUserRole implements IndexReader.
func (m *MockIndex) HasUserRole(name string) bool { return m.userRoles[name] }
```

- [ ] **Step 5: 扩展 meta.Index — index.go**

在 `Index` struct（第 28 行）追加字段：

```go
userRoles map[string]bool
```

在 `BuildFromBackend()` 的 `idx := &Index{...}` 初始化块中追加：

```go
userRoles: make(map[string]bool),
```

在 `BuildFromBackend()` 函数末尾（`return idx, nil` 前）追加调用：

```go
if err := idx.buildUserRoles(b); err != nil {
    return nil, err
}
```

在文件末尾追加 `buildUserRoles()`：

```go
func (idx *Index) buildUserRoles(b backend.FullBackend) error {
	ps, err := b.GetProjectSecurityGen()
	if err != nil || ps == nil {
		return nil // project security not accessible — skip without error
	}
	for _, elem := range ps.UserRolesItems() {
		ur, ok := elem.(*genSec.UserRole)
		if !ok {
			continue
		}
		if name := ur.Name(); name != "" {
			idx.userRoles[name] = true
		}
	}
	return nil
}
```

在文件顶部 import 中加入（如尚未导入）：

```go
genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
```

- [ ] **Step 6: 添加 HasUserRole 到 catalog_reader.go**

在 `catalog_reader.go` 末尾追加：

```go
// HasUserRole reports whether a user role with the given name exists in the project.
func (idx *Index) HasUserRole(name string) bool {
	return idx.userRoles[name]
}
```

- [ ] **Step 7: 运行全部相关测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/... -v 2>&1 | grep -E "PASS|FAIL|SEM-08" | head -30
```

期望输出：所有 `PASS`，含 `TestSEM08_UnknownUserRole` 和 `TestSEM08_NilIdx_NoError`

- [ ] **Step 8: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/validate/validate_sem.go internal/expr/validate/validate_test.go \
        internal/expr/meta/index.go internal/expr/meta/catalog_reader.go \
        internal/expr/meta/mock_index.go
git commit -m "feat(expr): add SEM-08 UserRole token validation

IndexReader.HasUserRole() + meta.Index.buildUserRoles() reads from
GetProjectSecurityGen(). checkUserRoleTokens() matches [%UserRole_*%]
and emits SEM-08 error when role name not found in project.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 文档层更新

**Files:**
- Modify: `.claude/skills/mendix/xpath-constraints.md`
- Modify: `docs/06-mdl-reference/expr-hints.md`

- [ ] **Step 1: 更新 xpath-constraints.md — System Variables 小节**

找到文件中现有的 System Variables 相关内容，将其替换为以下完整小节（在 "## XPath vs Mendix Expressions" 表格之后添加，或替换已有的 token 部分）：

```markdown
## System Variables（[% ... %]）

在 XPath 约束中使用时必须加单引号：`'[%CurrentUser%]'`  
在微流表达式中使用时**不加**引号：`[%CurrentDateTime%]`

> ⚠️ **不支持括号分组**：system variable 是字符串形式，不能用括号组合子表达式。  
> ⚠️ **时间长度 token 必须在同一字符串内**：`'[%BeginOfCurrentDay%] - 3 * [%YearLength%]'`  
> ⚠️ **UTC 变体警告**：客户端表达式中，若属性 `Localize=false`，不要用 UTC 变体（时区转换会执行两次）。

### 对象相关

| Token | 描述 |
|---|---|
| `[%CurrentUser%]` | 当前登录用户的 GUID（System.User） |
| `[%CurrentObject%]` | 当前上下文对象的 GUID |

### 用户角色

每个 UserRole 对应一个动态 token，格式为 `[%UserRole_<RoleName>%]`：

```xpath
[System.UserRoles = '[%UserRole_Administrator%]']
```

### 时间点（Time-Point）

| Token | UTC 变体 | 描述 |
|---|---|---|
| `[%CurrentDateTime%]` | — | 当前日期时间 |
| `[%BeginOfCurrentMinute%]` | `[%BeginOfCurrentMinuteUTC%]` | 当前分钟开始 |
| `[%EndOfCurrentMinute%]` | `[%EndOfCurrentMinuteUTC%]` | 当前分钟结束 |
| `[%BeginOfCurrentHour%]` | `[%BeginOfCurrentHourUTC%]` | 当前小时开始 |
| `[%EndOfCurrentHour%]` | `[%EndOfCurrentHourUTC%]` | 当前小时结束 |
| `[%BeginOfCurrentDay%]` | `[%BeginOfCurrentDayUTC%]` | 今天开始 |
| `[%EndOfCurrentDay%]` | `[%EndOfCurrentDayUTC%]` | 今天结束 |
| `[%BeginOfYesterday%]` | `[%BeginOfYesterdayUTC%]` | 昨天开始 |
| `[%EndOfYesterday%]` | `[%EndOfYesterdayUTC%]` | 昨天结束 |
| `[%BeginOfTomorrow%]` | `[%BeginOfTomorrowUTC%]` | 明天开始 |
| `[%EndOfTomorrow%]` | `[%EndOfTomorrowUTC%]` | 明天结束 |
| `[%BeginOfCurrentWeek%]` | `[%BeginOfCurrentWeekUTC%]` | 本周开始 |
| `[%EndOfCurrentWeek%]` | `[%EndOfCurrentWeekUTC%]` | 本周结束 |
| `[%BeginOfCurrentMonth%]` | `[%BeginOfCurrentMonthUTC%]` | 本月开始 |
| `[%EndOfCurrentMonth%]` | `[%EndOfCurrentMonthUTC%]` | 本月结束 |
| `[%BeginOfCurrentYear%]` | `[%BeginOfCurrentYearUTC%]` | 本年开始 |
| `[%EndOfCurrentYear%]` | `[%EndOfCurrentYearUTC%]` | 本年结束 |

### 时间长度（Time-Length，用于加减运算）

| Token | 描述 |
|---|---|
| `[%SecondLength%]` | 一秒（毫秒数） |
| `[%MinuteLength%]` | 一分钟（毫秒数） |
| `[%HourLength%]` | 一小时（毫秒数） |
| `[%DayLength%]` | 一天 24 小时（毫秒数） |
| `[%WeekLength%]` | 一周（毫秒数） |
| `[%MonthLength%]` | 一个月（毫秒数） |
| `[%YearLength%]` | 一年（毫秒数） |

```xpath
-- 过去三年内注册的客户
[DateRegistered > '[%BeginOfCurrentDay%] - 3 * [%YearLength%]']

-- 过去一小时内的事件
[Timestamp >= '[%CurrentDateTime%] - 1 * [%HourLength%]']
```
```

- [ ] **Step 2: 更新 expr-hints.md — 新增 SEM-08 条目**

找到 `docs/06-mdl-reference/expr-hints.md` 文件中 E007 部分之后（约第 117 行），在其后追加：

```markdown
## SEM-08 — unknown-user-role (error)

**触发条件**：表达式中包含 `[%UserRole_Name%]` token，但该角色名在项目安全设置中不存在。

**示例**：
```
[%UserRole_Manger%]  -- 拼写错误，应为 Manager
```

**修复**：检查角色名拼写，或在 Studio Pro 项目安全设置中添加该用户角色。

**注意**：此规则仅在连接了 MPR 文件的情况下生效（需要项目元数据）。语法检查模式（`--no-daemon`）不会触发此规则。
```

同时在 E007 的描述中追加一句说明（找到 `## E007 — unknown-token` 段落，在其内容末尾追加）：

```markdown
完整的有效 token 列表参见 `.claude/skills/mendix/xpath-constraints.md` 的 System Variables 小节。
```

- [ ] **Step 3: 验证文档文件无语法错误**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
# 确认文件存在且大小合理
wc -l .claude/skills/mendix/xpath-constraints.md docs/06-mdl-reference/expr-hints.md
```

期望输出：两个文件行数均增加（xpath-constraints.md 应 > 100 行）

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add .claude/skills/mendix/xpath-constraints.md docs/06-mdl-reference/expr-hints.md
git commit -m "docs: update system token documentation to full mendix_docs coverage

xpath-constraints.md: complete token table with UTC variants, Yesterday/Tomorrow,
Minute/Hour granularity, all 7 duration tokens, 3 usage warnings.
expr-hints.md: add SEM-08 unknown-user-role entry.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: MDL 示例脚本

**Files:**
- Create: `mdl-examples/tokens/time-tokens.mdl`
- Create: `mdl-examples/tokens/user-tokens.mdl`
- Create: `mdl-examples/tokens/datetime-arithmetic.mdl`

- [ ] **Step 1: 创建 time-tokens.mdl**

```
-- time-tokens.mdl
-- 演示时间点和时间长度 token 在 XPath 约束和微流表达式中的用法。
-- 语法检查：mxcli check time-tokens.mdl

-- ── XPath 约束示例 ──────────────────────────────────────────────────────────

-- 本周注册的客户
-- retrieve $ThisWeekCustomers from Sales.Customer
--   where [DateRegistered >= '[%BeginOfCurrentWeek%]'
--      and DateRegistered <  '[%EndOfCurrentWeek%]'];

-- 本月的订单
-- retrieve $MonthOrders from Sales.Order
--   where [OrderDate >= '[%BeginOfCurrentMonth%]'
--      and OrderDate <= '[%EndOfCurrentMonth%]'];

-- 今年至今的订单
-- retrieve $YearOrders from Sales.Order
--   where [OrderDate >= '[%BeginOfCurrentYear%]'];

-- 最近三年注册的客户（时间运算：时间点 + 长度计算，必须在同一字符串内）
-- retrieve $RecentCustomers from Sales.Customer
--   where [DateRegistered > '[%BeginOfCurrentDay%] - 3 * [%YearLength%]'];

-- 过去一小时内的日志事件
-- retrieve $RecentLogs from Logging.Event
--   where [Timestamp >= '[%CurrentDateTime%] - 1 * [%HourLength%]'];

-- 过去30分钟的活动（MinuteLength）
-- retrieve $Recent30Min from Logging.Event
--   where [Timestamp >= '[%CurrentDateTime%] - 30 * [%MinuteLength%]'];

-- ── 微流表达式示例 ───────────────────────────────────────────────────────────

-- 将当前时间戳写入字段
-- microflow UpdateTimestamp($Order: Sales.Order): void
-- change $Order set (
--   LastModified = [%CurrentDateTime%]
-- );

-- 在条件表达式中使用当前时间
-- if $Order/DueDate < [%CurrentDateTime%] then 'Overdue' else 'OnTime'

-- ── 注意事项 ─────────────────────────────────────────────────────────────────
-- 1. XPath 约束中 token 必须加单引号：'[%BeginOfCurrentWeek%]'
-- 2. 微流表达式中不加引号：[%CurrentDateTime%]
-- 3. 时间长度 token 必须和时间点 token 在同一字符串内
-- 4. UTC 变体（BeginOfCurrentDayUTC 等）：若属性 Localize=false，
--    客户端表达式不要用 UTC 变体（时区转换会执行两次）
```

- [ ] **Step 2: 创建 user-tokens.mdl**

```
-- user-tokens.mdl
-- 演示 CurrentUser 和 UserRole_* token 的用法。
-- 语法检查：mxcli check user-tokens.mdl

-- ── CurrentUser：过滤当前用户的数据 ─────────────────────────────────────────

-- 检索当前用户的所有订单（通过关联过滤）
-- retrieve $MyOrders from Sales.Order
--   where [Sales.Order_Customer = '[%CurrentUser%]'];

-- 检索当前用户创建的记录（CreatedBy 关联）
-- retrieve $MyRecords from MyModule.Record
--   where [MyModule.Record_CreatedBy = '[%CurrentUser%]'];

-- 检索当前用户拥有的对象（System.owner）
-- retrieve $OwnedDocs from MyModule.Document
--   where [System.owner = '[%CurrentUser%]'];

-- ── UserRole_*：基于角色的实体访问控制 ──────────────────────────────────────

-- 只有 Administrator 角色可以读写所有字段
-- grant Administrator on Sales.Product (read *, write *)
--   where '[System.UserRoles = ''[%UserRole_Administrator%]'']';

-- 只有 Manager 和 Administrator 角色可以访问薪资数据
-- grant Manager on HR.Employee (read (Salary))
--   where '[System.UserRoles = ''[%UserRole_Manager%]'']';

-- ── 微流表达式中使用 CurrentUser ─────────────────────────────────────────────

-- 在创建对象时关联当前用户
-- create Sales.Order $NewOrder set (
--   OrderDate = [%CurrentDateTime%]
-- );
-- 注：关联当前用户需通过 Association，不直接写在 SET 表达式中

-- ── 注意事项 ─────────────────────────────────────────────────────────────────
-- 1. XPath 中 [%CurrentUser%] 必须加引号：'[%CurrentUser%]'
-- 2. [%UserRole_Name%] 中的 Name 必须与项目安全设置中的用户角色名完全一致
-- 3. 实体访问规则中的 XPath 嵌套在字符串内，单引号需双写转义：
--    where '[System.UserRoles = ''[%UserRole_Administrator%]'']'
```

- [ ] **Step 3: 创建 datetime-arithmetic.mdl**

```
-- datetime-arithmetic.mdl
-- 演示时间运算的综合用法：时间点 + 时间长度 token 组合。
-- 语法检查：mxcli check datetime-arithmetic.mdl

-- ── 常见时间区间查询 ─────────────────────────────────────────────────────────

-- 昨天全天（BeginOfYesterday / BeginOfCurrentDay）
-- retrieve $YesterdayEvents from Logging.Event
--   where [Timestamp >= '[%BeginOfYesterday%]'
--      and Timestamp <  '[%BeginOfCurrentDay%]'];

-- 明天全天
-- retrieve $TomorrowTasks from HR.Task
--   where [DueDate >= '[%BeginOfTomorrow%]'
--      and DueDate <  '[%EndOfTomorrow%]'];

-- 本月开始到现在
-- retrieve $MonthOrders from Sales.Order
--   where [OrderDate >= '[%BeginOfCurrentMonth%]'
--      and OrderDate <= '[%CurrentDateTime%]'];

-- 过去 7 天（WeekLength）
-- retrieve $Last7Days from Logging.Event
--   where [Timestamp >= '[%CurrentDateTime%] - 1 * [%WeekLength%]'];

-- 过去 90 天（DayLength × 90）
-- retrieve $Last90Days from Sales.Order
--   where [OrderDate >= '[%CurrentDateTime%] - 90 * [%DayLength%]'];

-- ── 工作流 Timer 示例 ────────────────────────────────────────────────────────

-- 3 天后到期的提醒（工作流 WAIT FOR TIMER）
-- WAIT FOR TIMER 'addDays([%CurrentDateTime%], 3)';

-- ── 微流中的时间判断 ─────────────────────────────────────────────────────────

-- 判断一个日期是否在过去一周内
-- if $Record/CreatedDate >= '[%BeginOfCurrentWeek%]'
--    and $Record/CreatedDate <= '[%CurrentDateTime%]'
-- then true else false

-- ── 规则汇总 ─────────────────────────────────────────────────────────────────
-- ✓ 时间点 + 时间长度必须在同一字符串内
-- ✓ XPath 约束中整个表达式加引号
-- ✓ 算术运算用 * 乘以 token：N * [%DayLength%]
-- ✓ 支持加减：[%CurrentDateTime%] - 7 * [%DayLength%]
-- ✓ 括号分组不支持（system variable 是字符串，不能用括号）
```

- [ ] **Step 4: 验证 mdl-examples/tokens/ 目录结构**

```bash
ls -la /mnt/data_sdd/gh/mxcli-wt-02/mdl-examples/tokens/
```

期望输出：三个 `.mdl` 文件均存在

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl-examples/tokens/
git commit -m "docs(examples): add system token MDL example scripts

Three files covering time-point/duration tokens, CurrentUser/UserRole,
and time arithmetic patterns. Both XPath constraint and microflow
expression contexts demonstrated.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## 完成验证

所有任务完成后运行：

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/expr/... -v 2>&1 | grep -E "^(PASS|FAIL|---)" | head -40
```

期望输出：全部 `PASS`，无 `FAIL`。
