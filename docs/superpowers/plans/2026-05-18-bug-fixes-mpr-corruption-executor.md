# Bug 修复实现计划：MPR 损坏 + Executor 健壮性

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 BUG-01/03/04/05/06：为 drop attribute 添加回归测试；在 AST 层拒绝 XPath 带引号标识符；grant 角色不存在时改为警告；参数名含 `$` 时给出清晰错误；EXECUTE SCRIPT 加自动事务防止孤儿对象。

**Architecture:** 5 个独立修复，均在 `mdl/executor`/`mdl/visitor`/`mdl/backend` 层。BUG-04/05 只改现有函数行为；BUG-03 在 visitor 层加扫描；BUG-06 在 `FullBackend` 接口新增三个事务方法并修改 `writeUnitContents`。全部遵循 TDD。

**Tech Stack:** Go，`mdl/executor`，`mdl/visitor`，`mdl/backend/mpr`，`mdl/backend/mock`，`modelsdk/gen`，`mmpr.WriteTransaction`。

---

## 文件索引

| 文件 | 任务 | 操作 |
|------|------|------|
| `mdl/executor/roundtrip_mxcheck_test.go` | Task 1 | 新增集成测试 |
| `mdl/executor/cmd_security_helpers.go` | Task 2 | 修改：`validateModuleRole` 改为警告 |
| `mdl/executor/cmd_security_write_gen.go` | Task 2 | 修改：两处 `validateModuleRole` 错误处理 |
| `mdl/executor/cmd_security_write_entity_gen.go` | Task 2 | 修改：两处 `validateModuleRole` 错误处理 |
| `mdl/executor/cmd_security_write_page_gen.go` | Task 2 | 修改：一处 `validateModuleRole` 错误处理 |
| `mdl/executor/cmd_security_write_extservice_gen.go` | Task 2 | 修改：两处 `validateModuleRole` 错误处理 |
| `mdl/executor/cmd_security_mock_test.go` | Task 2 | 新增测试 |
| `mdl/executor/validate_microflow.go` | Task 3 | 修改：参数名 `$` 前缀检测 |
| `mdl/executor/cmd_microflows_gen_test.go` 或新建 | Task 3 | 新增测试 |
| `mdl/visitor/visitor_microflow.go` | Task 4 | 修改：XPath 引号标识符检测 |
| `mdl/visitor/visitor_microflow_test.go` 或新建 | Task 4 | 新增测试 |
| `mdl/backend/connection.go` | Task 5 | 新增 `ScriptTransactionBackend` 接口 |
| `mdl/backend/backend.go` | Task 5 | `FullBackend` 嵌入新接口 |
| `mdl/backend/mpr/backend.go` | Task 5 | 新增 `activeScriptTx` 字段和三个方法 |
| `mdl/backend/mpr/write_helpers.go` | Task 5 | 修改：`writeUnitContents` 复用 scriptTx |
| `mdl/backend/mock/backend.go` | Task 5 | 新增三个 no-op 方法 |
| `mdl/executor/cmd_misc.go` | Task 5 | 修改：`execExecuteScript` 加事务包装 |
| `mdl/executor/cmd_misc_test.go` 或新建 | Task 5 | 新增测试 |

---

## Task 1：BUG-01 — drop attribute 回归测试

**Files:**
- Modify: `mdl/executor/roundtrip_mxcheck_test.go`

**背景：** `cleanupDroppedAttributeReferencesGen` 已在当前分支正确实现。只需添加集成测试防止退化。测试文件头部有 `//go:build integration` 标签，且有 `setupTestEnv`、`runMxCheck` 等现有辅助函数。

- [ ] **Step 1：写失败测试（在任何实现之前）**

在 `mdl/executor/roundtrip_mxcheck_test.go` 末尾追加：

```go
// TestDropAttribute_WithAccessRules_NoCorruption verifies that dropping an
// attribute with active access rules (read *, write *) does not corrupt the
// MPR. Regression test for BUG-01.
func TestDropAttribute_WithAccessRules_NoCorruption(t *testing.T) {
    if !mxCheckAvailable() {
        t.Skip("mx binary not available")
    }

    env := setupTestEnv(t)
    defer env.teardown()

    entityQN := testModule + ".BUG01Entity"
    env.registerCleanup("entity", entityQN)

    // 1. Create entity with an attribute and a module role, then grant access.
    setup := `
create or modify persistent entity "` + testModule + `"."BUG01Entity" (
    "KeepMe": String(100),
    "DropMe": String(50)
);
create module role "` + testModule + `"."User";
grant "` + testModule + `"."User" on "` + testModule + `"."BUG01Entity" (read *, write *);
`
    if err := env.executeMDL(setup); err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    // 2. Drop one attribute.
    var buf strings.Builder
    env.executor.Output = &buf
    dropMDL := `alter entity "` + testModule + `"."BUG01Entity" drop attribute "DropMe";`
    if err := env.executeMDL(dropMDL); err != nil {
        t.Fatalf("drop attribute failed: %v", err)
    }

    // 3. Confirm cleanup output is present.
    if !strings.Contains(buf.String(), "access rule member reference") {
        t.Errorf("expected cleanup message about access rule member references, got: %s", buf.String())
    }

    // 4. Disconnect and run mx check — must have 0 errors.
    env.executor.Execute(&ast.DisconnectStmt{})
    output, err := runMxCheck(t, env.projectPath)
    if err != nil {
        t.Errorf("mx check failed after drop attribute — MPR may be corrupted.\nOutput:\n%s", output)
    }
    if strings.Contains(output, "KeyNotFoundException") {
        t.Errorf("KeyNotFoundException in mx check output — UUID reference not cleaned up.\nOutput:\n%s", output)
    }
}
```

- [ ] **Step 2：确认测试失败（仅在当前分支 cleanup 函数不存在时预期失败）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ -tags integration \
  -run TestDropAttribute_WithAccessRules_NoCorruption -v -timeout 120s
```

预期：PASS（因为当前分支已有 cleanup 实现）。若 FAIL 则说明 cleanup 确实缺失，需先修复 `cleanupDroppedAttributeReferencesGen`。

- [ ] **Step 3：提交**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl/executor/roundtrip_mxcheck_test.go
git commit -m "test(executor): BUG-01 回归测试 — drop attribute 不损坏 MPR access rules"
```

---

## Task 2：BUG-04 — grant 角色不存在改为警告

**Files:**
- Modify: `mdl/executor/cmd_security_helpers.go`
- Modify: `mdl/executor/cmd_security_write_gen.go`
- Modify: `mdl/executor/cmd_security_write_entity_gen.go`
- Modify: `mdl/executor/cmd_security_write_page_gen.go`
- Modify: `mdl/executor/cmd_security_write_extservice_gen.go`
- Modify: `mdl/executor/cmd_security_mock_test.go`

**背景：** `validateModuleRole`（`cmd_security_helpers.go:21`）在角色不存在时返回 `mdlerrors.NewNotFound`（致命错误）。所有 grant 函数都调用它并直接 `return err`。修复方案：在各调用点判断是 NotFound 就输出 WARNING 并继续，其他错误仍然致命。

- [ ] **Step 1：写失败测试**

在 `mdl/executor/cmd_security_mock_test.go` 中添加：

```go
func TestGrantMicroflow_MissingRole_IsWarningNotError(t *testing.T) {
    ctx := newMockExecContext()
    var output strings.Builder
    ctx.Output = &output

    // Grant on a microflow where the module role does not exist.
    // Expect: no error returned, but WARNING written to output.
    stmt := &ast.GrantMicroflowAccessStmt{
        Microflow: ast.QualifiedName{Module: "TestModule", Name: "MyFlow"},
        Roles: []ast.QualifiedName{
            {Module: "TestModule", Name: "NonExistentRole"},
        },
    }
    err := execGrantMicroflowAccessGen(ctx, stmt)
    if err != nil {
        t.Errorf("expected nil error for missing module role, got: %v", err)
    }
    if !strings.Contains(output.String(), "WARNING") {
        t.Errorf("expected WARNING in output, got: %s", output.String())
    }
    if !strings.Contains(output.String(), "NonExistentRole") {
        t.Errorf("expected role name in warning output, got: %s", output.String())
    }
}
```

- [ ] **Step 2：运行确认测试失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ \
  -run TestGrantMicroflow_MissingRole_IsWarningNotError -v
```

预期：FAIL（当前行为是返回 error）。

- [ ] **Step 3：修改 validateModuleRole 返回值语义**

在 `mdl/executor/cmd_security_helpers.go` 中，将 `validateModuleRole` 改为输出警告并返回 `nil`：

```go
// validateModuleRole checks that a module role exists in the project.
// If the role is not found, it writes a WARNING to ctx.Output and returns nil
// so that grant scripts can continue even when a module has no roles yet.
// Returns a non-nil error only for backend failures (module not found, etc.).
func validateModuleRole(ctx *ExecContext, role ast.QualifiedName) error {
    module, err := findModule(ctx, role.Module)
    if err != nil {
        return mdlerrors.NewBackend(fmt.Sprintf("module not found for role %s.%s", role.Module, role.Name), err)
    }

    ms, err := ctx.Backend.GetModuleSecurityGen(module.ID)
    if err != nil {
        return mdlerrors.NewBackend(fmt.Sprintf("read module security for %s", role.Module), err)
    }

    if ms != nil {
        for _, item := range ms.ModuleRolesItems() {
            if mr, ok := item.(*genSec.ModuleRole); ok && mr.Name() == role.Name {
                return nil
            }
        }
    }

    // Role not found — warn and skip rather than aborting the script.
    fmt.Fprintf(ctx.Output, "WARNING: module role '%s.%s' not found — grant skipped\n",
        role.Module, role.Name)
    return nil
}
```

- [ ] **Step 4：运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ \
  -run TestGrantMicroflow_MissingRole_IsWarningNotError -v
```

预期：PASS。

- [ ] **Step 5：运行全部 security 测试确认无回归**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ -run TestGrant -v -short
```

预期：全部 PASS。

- [ ] **Step 6：提交**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl/executor/cmd_security_helpers.go mdl/executor/cmd_security_mock_test.go
git commit -m "fix(executor): BUG-04 grant 角色不存在改为 WARNING 并继续脚本"
```

---

## Task 3：BUG-05 — 参数名含 `$` 前缀清晰错误

**Files:**
- Modify: `mdl/executor/validate_microflow.go`
- Modify: `mdl/executor/cmd_microflows_gen_test.go`（或 `validate_microflow_test.go`，视已有文件而定）

**背景：** `ValidateMicroflow`（`validate_microflow.go:16`）在参数循环处校验 entity ref 是否有模块前缀，但不检测 `$` 前缀。带引号的 `"$Name"` 参数名（literal dollar sign）会导致 Mendix 拒绝模型但提示不清晰。

- [ ] **Step 1：找到现有测试文件位置**

```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/*microflow*test* 2>/dev/null || \
  ls /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/*validate*test* 2>/dev/null
```

使用已有文件（或新建 `mdl/executor/validate_microflow_test.go`）。

- [ ] **Step 2：写失败测试**

新增到测试文件：

```go
func TestValidateMicroflow_DollarPrefixParam_ReturnsError(t *testing.T) {
    stmt := &ast.CreateMicroflowStmt{
        Name: ast.QualifiedName{Module: "Mod", Name: "Flow"},
        Parameters: []ast.MicroflowParam{
            {Name: "$BadName", Type: ast.MicroflowDataType{Primitive: "String"}},
        },
    }
    violations := ValidateMicroflow(stmt)
    found := false
    for _, v := range violations {
        if strings.Contains(v.Message, "$BadName") &&
            strings.Contains(v.Message, "must not include '$'") {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("expected violation about '$' prefix in parameter name, got: %v", violations)
    }
}

func TestValidateMicroflow_NoDollarParam_NoError(t *testing.T) {
    stmt := &ast.CreateMicroflowStmt{
        Name: ast.QualifiedName{Module: "Mod", Name: "Flow"},
        Parameters: []ast.MicroflowParam{
            {Name: "GoodName", Type: ast.MicroflowDataType{Primitive: "String"}},
        },
    }
    violations := ValidateMicroflow(stmt)
    for _, v := range violations {
        if strings.Contains(v.Message, "must not include '$'") {
            t.Errorf("unexpected $ violation for param without dollar: %v", v)
        }
    }
}
```

- [ ] **Step 3：运行确认测试失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ \
  -run TestValidateMicroflow_DollarPrefixParam -v
```

预期：FAIL（当前无此校验）。

- [ ] **Step 4：实现校验**

在 `mdl/executor/validate_microflow.go` 的 `ValidateMicroflow` 函数中，在现有参数循环 `for _, p := range stmt.Parameters {` 处添加 `$` 检测（放在现有 entity ref 检查之前）：

```go
func ValidateMicroflow(stmt *ast.CreateMicroflowStmt) []linter.Violation {
    v := &microflowValidator{
        mfName:     stmt.Name.String(),
        returnType: stmt.ReturnType,
    }
    for _, p := range stmt.Parameters {
        // NEW: reject parameter names that start with '$'
        if strings.HasPrefix(p.Name, "$") {
            bare := p.Name[1:]
            v.addViolation("MDL009", linter.SeverityError,
                fmt.Sprintf("parameter name %q must not include '$' prefix in declaration", p.Name),
                fmt.Sprintf("declare as %q, reference in microflow body as $%s", bare, bare),
            )
        }
        // existing: reject entity refs without module prefix
        if p.Type.EntityRef != nil && p.Type.EntityRef.Module == "" {
            // ... existing code unchanged ...
        }
    }
    v.validate(stmt.Body)
    // ... rest unchanged ...
}
```

- [ ] **Step 5：运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ \
  -run TestValidateMicroflow_DollarPrefixParam -v
```

预期：PASS。

- [ ] **Step 6：运行全部 executor 测试确认无回归**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ -short -timeout 60s 2>&1 | tail -5
```

预期：所有测试通过。

- [ ] **Step 7：提交**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl/executor/validate_microflow.go mdl/executor/validate_microflow_test.go
git commit -m "fix(executor): BUG-05 参数名含 \$ 前缀时给出清晰诊断错误 (MDL009)"
```

---

## Task 4：BUG-03 — XPath/属性路径中带引号标识符的 AST 层拒绝

**Files:**
- Modify: `mdl/visitor/visitor_microflow.go`
- Create/Modify: `mdl/visitor/visitor_microflow_test.go`（或现有测试文件）

**背景：** 当用户在 MDL retrieve WHERE 子句写 `["Module.Assoc" = $x]` 或在微流体表达式访问 `$Obj/"Attr"`，引号标识符进入 Mendix 模型后报 CE 错误但提示不清晰。在 visitor 层（AST 构建阶段）扫描这些模式并给出引导错误。

XPath 约束字符串在 `visitor_microflow.go` 的 `buildMicroflowRetrieve`（或类似函数）处赋给 BSON 节点。在赋值前做扫描。

- [ ] **Step 1：找到 XPath 约束赋值点**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "XpathConstraint\|XPathConstraint\|xpathConstraint\|retrieve.*where\|WHERE\|buildRetrieve\|buildDatabase" \
  mdl/visitor/visitor_microflow.go | head -20
```

记录行号，在下一步使用。

- [ ] **Step 2：写失败测试**

新建或修改 `mdl/visitor/visitor_microflow_test.go`：

```go
package visitor_test

import (
    "strings"
    "testing"

    "github.com/mendixlabs/mxcli/mdl/visitor"
)

func TestBuild_XPathWithQuotedAssoc_ReturnsError(t *testing.T) {
    // A retrieve statement with a quoted association name in XPath.
    mdl := `
create microflow Mod.TestFlow ()
begin
    retrieve $Obj from Mod.Entity where ["Mod.Assoc" = $Other];
    return empty;
end;
`
    _, errs := visitor.Build(mdl)
    if len(errs) == 0 {
        t.Fatal("expected parse/validation error for quoted identifier in XPath, got none")
    }
    found := false
    for _, e := range errs {
        if strings.Contains(e.Error(), "unquoted") || strings.Contains(e.Error(), "quoted") {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("expected error mentioning 'unquoted', got: %v", errs)
    }
}

func TestBuild_XPathWithUnquotedAssoc_NoError(t *testing.T) {
    // Same but correct — unquoted association name.
    mdl := `
create microflow Mod.TestFlow ()
begin
    retrieve $Obj from Mod.Entity where [Mod.Assoc = $Other];
    return empty;
end;
`
    _, errs := visitor.Build(mdl)
    for _, e := range errs {
        if strings.Contains(e.Error(), "unquoted") {
            t.Errorf("unexpected 'unquoted' error for correct syntax: %v", e)
        }
    }
}
```

- [ ] **Step 3：运行确认测试失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/visitor/ \
  -run TestBuild_XPathWithQuoted -v
```

预期：FAIL（当前无此校验）。

- [ ] **Step 4：找到 XPath 字符串赋值点并加校验**

在 `mdl/visitor/visitor_microflow.go` 中找到将 WHERE 子句内容赋给 `XpathConstraint` 的位置（Step 1 找到的行号附近），添加扫描函数：

```go
// validateXPathConstraint checks for common authoring mistakes in XPath strings.
// Returns an error if a quoted identifier (e.g. "Module.Assoc") is found where
// an unquoted reference is required.
func validateXPathConstraint(xpath string) error {
    // Detect pattern: a double-quoted token that looks like Module.Name or an identifier
    // inside an XPath constraint — e.g. ["Module.Assoc" = $x] or [Code = "Value"]
    // Only flag when inside the brackets (the xpath already has brackets stripped or not).
    re := regexp.MustCompile(`"[A-Z][A-Za-z0-9_]*\.[A-Za-z][A-Za-z0-9_]*"`)
    if loc := re.FindStringIndex(xpath); loc != nil {
        quoted := xpath[loc[0]:loc[1]]
        bare := strings.Trim(quoted, `"`)
        return fmt.Errorf(
            "association or entity reference %s in XPath constraint must be unquoted — "+
                "use %s instead of %s",
            quoted, bare, quoted,
        )
    }
    return nil
}
```

Add `"regexp"` to imports. Call this function immediately before the XPath string is set on the BSON node:

```go
// wherever xpath string is constructed from the AST node:
xpathStr := buildXPathString(whereCtx) // existing logic
if err := validateXPathConstraint(xpathStr); err != nil {
    return nil, err // or add to errors slice following existing pattern
}
```

The exact insertion point depends on Step 1 findings — follow the existing error-handling pattern in the visitor.

- [ ] **Step 5：运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/visitor/ \
  -run TestBuild_XPathWithQuoted -v
```

预期：PASS。

- [ ] **Step 6：运行全部 visitor 测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/visitor/ -short -timeout 60s 2>&1 | tail -5
```

预期：全部 PASS。

- [ ] **Step 7：提交**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl/visitor/visitor_microflow.go mdl/visitor/visitor_microflow_test.go
git commit -m "fix(visitor): BUG-03 XPath 带引号标识符在 AST 层报清晰错误"
```

---

## Task 5：BUG-06 — EXECUTE SCRIPT 自动事务

**Files:**
- Modify: `mdl/backend/connection.go`（新增接口）
- Modify: `mdl/backend/backend.go`（`FullBackend` 嵌入新接口）
- Modify: `mdl/backend/mpr/backend.go`（新增字段和三个方法）
- Modify: `mdl/backend/mpr/write_helpers.go`（`writeUnitContents` 复用 scriptTx）
- Modify: `mdl/backend/mock/backend.go`（新增 no-op 方法）
- Modify: `mdl/executor/cmd_misc.go`（`execExecuteScript` 加事务包装）

**背景：**
- `execExecuteScript` 逐语句调用 `ctx.ExecuteFn(stmt)`
- 每个写操作（UpdateEntityGen 等）最终调用 `writeUnitContents`，后者自己开一个 sqlite write transaction 并立刻 commit
- 要实现脚本级原子性，需要让 `writeUnitContents` 在 scriptTx 活跃时复用同一个 transaction 而不是自己开新的
- 通过在 `MprBackend` 上存储 `activeScriptTx`，并在 `FullBackend` 接口新增三个方法暴露控制

### Step 1：写失败测试（mock 测试）

- [ ] **写测试文件**

新建 `mdl/executor/script_tx_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
    "strings"
    "testing"

    "github.com/mendixlabs/mxcli/mdl/backend/mock"
    "github.com/mendixlabs/mxcli/mdl/executor"
)

// TestExecuteScript_FailureMidScript_RollbackCalled verifies that when
// a statement inside EXECUTE SCRIPT fails, Rollback is called on the
// script transaction and the error is propagated.
func TestExecuteScript_FailureMidScript_RollbackCalled(t *testing.T) {
    rollbackCalled := false
    commitCalled := false

    mb := &mock.MockBackend{}
    mb.BeginScriptTransactionFunc = func() (executor.ScriptTransaction, error) {
        return &mockScriptTx{
            commitFn:   func() error { commitCalled = true; return nil },
            rollbackFn: func() error { rollbackCalled = true; return nil },
        }, nil
    }

    // ... test that on script error, rollbackCalled == true, commitCalled == false
    _ = rollbackCalled
    _ = commitCalled
    // Full test implementation after interfaces are defined in Step 3.
    t.Skip("implementation pending Step 3")
}

type mockScriptTx struct {
    commitFn   func() error
    rollbackFn func() error
}
func (m *mockScriptTx) Commit() error   { return m.commitFn() }
func (m *mockScriptTx) Rollback() error { return m.rollbackFn() }
```

- [ ] **运行确认编译失败（接口未定义）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./mdl/executor/ 2>&1 | head -10
```

预期：编译错误（`ScriptTransaction` 未定义，`BeginScriptTransactionFunc` 未知）。

### Step 2：定义 ScriptTransaction 接口

- [ ] **在 `mdl/backend/connection.go` 新增**

找到文件末尾或合适位置，添加：

```go
// ScriptTransaction represents an open write transaction held for the
// duration of an EXECUTE SCRIPT block. Commit persists all writes;
// Rollback discards them.
type ScriptTransaction interface {
    Commit() error
    Rollback() error
}

// ScriptTransactionBackend is implemented by backends that support
// atomic multi-statement script execution.
type ScriptTransactionBackend interface {
    // BeginScriptTransaction opens a write transaction that spans multiple
    // write operations. The caller must call Commit or Rollback.
    BeginScriptTransaction() (ScriptTransaction, error)
}
```

- [ ] **在 `mdl/backend/backend.go` 的 `FullBackend` 嵌入新接口**

在 `FullBackend` 的接口列表末尾添加：

```go
type FullBackend interface {
    ConnectionBackend
    // ... all existing sub-interfaces unchanged ...
    WidgetBuilderBackend
    ScriptTransactionBackend // NEW
}
```

- [ ] **编译确认接口定义正确**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./mdl/backend/... 2>&1 | head -20
```

预期：`MprBackend` 和 `MockBackend` 缺少 `BeginScriptTransaction` 方法报错 — 这是预期的（接口未实现）。

### Step 3：实现 MprBackend

- [ ] **在 `mdl/backend/mpr/backend.go` 新增字段和方法**

在 `MprBackend` 结构体新增字段（紧邻 `msdkWriter` 字段）：

```go
type MprBackend struct {
    // ... existing fields ...
    msdkWriter    modelsdkmpr.UnitWriter
    activeScriptTx modelsdkmpr.WriteTransaction // non-nil when script tx is active; NEW
}
```

在文件末尾新增三个方法：

```go
// BeginScriptTransaction opens a write transaction for the duration of an
// EXECUTE SCRIPT block. While active, writeUnitContents reuses this tx.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
    if b.msdkWriter == nil {
        return nil, fmt.Errorf("modelsdk writer not initialized")
    }
    wtx, err := b.msdkWriter.BeginWriteTransaction()
    if err != nil {
        return nil, fmt.Errorf("begin script transaction: %w", err)
    }
    b.activeScriptTx = wtx
    return &mprScriptTx{b: b, wtx: wtx}, nil
}

// mprScriptTx implements backend.ScriptTransaction for *MprBackend.
type mprScriptTx struct {
    b   *MprBackend
    wtx modelsdkmpr.WriteTransaction
}

func (tx *mprScriptTx) Commit() error {
    err := tx.wtx.Commit()
    tx.b.activeScriptTx = nil
    if err == nil {
        tx.b.msdkReader.InvalidateCache()
    }
    return err
}

func (tx *mprScriptTx) Rollback() error {
    err := tx.wtx.Rollback()
    tx.b.activeScriptTx = nil
    return err
}
```

You will need to find the correct import for `modelsdkmpr.WriteTransaction`. Check existing imports in `backend.go` — it likely uses `mmpr "github.com/mendixlabs/modelsdkmpr"` or similar. The `wtx` from `msdkWriter.BeginWriteTransaction()` has `WriteUnit`, `Commit`, and `Rollback` methods.

- [ ] **修改 `mdl/backend/mpr/write_helpers.go` 复用 scriptTx**

现有 `writeUnitContents` 函数（第 22 行起）修改为：

```go
func (b *MprBackend) writeUnitContents(unitID model.ID, contents []byte) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }

    // If a script-level transaction is active, reuse it instead of starting a new one.
    if b.activeScriptTx != nil {
        if err := b.activeScriptTx.WriteUnit(string(unitID), contents); err != nil {
            return fmt.Errorf("write unit (in script tx): %w", err)
        }
        // Do NOT commit here — the script transaction owner commits at the end.
        return nil
    }

    // Normal path: own transaction per write.
    wtx, err := b.msdkWriter.BeginWriteTransaction()
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    if err := wtx.WriteUnit(string(unitID), contents); err != nil {
        _ = wtx.Rollback()
        return fmt.Errorf("write unit: %w", err)
    }
    if err := wtx.Commit(); err != nil {
        return err
    }
    b.msdkReader.InvalidateCache()
    return nil
}
```

### Step 4：实现 MockBackend

- [ ] **在 `mdl/backend/mock/backend.go` 新增函数字段和方法**

找到 `MockBackend` 结构体，新增字段（可放在 `// ConnectionBackend` 分组内）：

```go
// ScriptTransactionBackend
BeginScriptTransactionFunc func() (backend.ScriptTransaction, error)
```

在文件末尾新增方法：

```go
func (m *MockBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
    if m.BeginScriptTransactionFunc != nil {
        return m.BeginScriptTransactionFunc()
    }
    // Default: no-op transaction (commits and rollbacks silently succeed)
    return &noopScriptTx{}, nil
}

type noopScriptTx struct{}
func (t *noopScriptTx) Commit() error   { return nil }
func (t *noopScriptTx) Rollback() error { return nil }
```

- [ ] **编译确认接口全部实现**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./mdl/... 2>&1 | head -20
```

预期：编译通过（无接口未实现错误）。

### Step 5：修改 execExecuteScript 加事务包装

- [ ] **修改 `mdl/executor/cmd_misc.go` 的 `execExecuteScript`**

找到函数 `execExecuteScript`（第 408 行），在 `fmt.Fprintf(ctx.Output, "Executing script: %s\n", ...)` 之前加入事务逻辑：

```go
func execExecuteScript(ctx *ExecContext, s *ast.ExecuteScriptStmt) error {
    if ctx.ScriptDepth >= maxScriptDepth {
        return mdlerrors.NewValidationf("maximum script nesting depth (%d) exceeded — possible recursive EXECUTE SCRIPT", maxScriptDepth)
    }

    // ... existing: path resolution, ReadFile, stripSlashSeparators, Build ...

    // Open a script-level transaction only for the outermost script.
    // Nested EXECUTE SCRIPT calls reuse the parent transaction (activeScriptTx
    // is already set on the backend).
    isRoot := ctx.ScriptDepth == 0
    var scriptTx backend.ScriptTransaction
    if isRoot && ctx.Backend != nil && ctx.Backend.IsConnected() {
        var err error
        scriptTx, err = ctx.Backend.BeginScriptTransaction()
        if err != nil {
            return fmt.Errorf("begin transaction for script '%s': %w", s.Path, err)
        }
        defer func() {
            if scriptTx != nil {
                _ = scriptTx.Rollback() // safety net: Rollback after Commit is a no-op
            }
        }()
    }

    fmt.Fprintf(ctx.Output, "Executing script: %s\n", s.Path)
    if ctx.ExecuteFn == nil {
        return mdlerrors.NewBackend("execute script", errors.New("ExecuteFn not set"))
    }
    ctx.ScriptDepth++
    defer func() { ctx.ScriptDepth-- }()

    for _, stmt := range prog.Statements {
        if err := ctx.ExecuteFn(stmt); err != nil {
            if errors.Is(err, ErrExit) {
                fmt.Fprintf(ctx.Output, "Script exited: %s\n", s.Path)
                if scriptTx != nil {
                    _ = scriptTx.Rollback()
                    scriptTx = nil
                }
                return nil
            }
            if scriptTx != nil {
                _ = scriptTx.Rollback()
                scriptTx = nil
                fmt.Fprintf(ctx.Output, "Script '%s' rolled back due to error\n", s.Path)
            }
            return fmt.Errorf("error in script '%s': %w", s.Path, err)
        }
    }

    if scriptTx != nil {
        if err := scriptTx.Commit(); err != nil {
            return fmt.Errorf("commit script '%s': %w", s.Path, err)
        }
        scriptTx = nil
    }
    fmt.Fprintf(ctx.Output, "Script completed: %s\n", s.Path)
    return nil
}
```

Add `"github.com/mendixlabs/mxcli/mdl/backend"` to imports if not present.

### Step 6：完成并运行测试

- [ ] **完善测试文件 `mdl/executor/script_tx_test.go`**

将 `t.Skip(...)` 替换为完整测试：

```go
func TestExecuteScript_FailureMidScript_RollbackCalled(t *testing.T) {
    rollbackCalled := false
    commitCalled := false

    mb := &mock.MockBackend{
        IsConnectedFunc: func() bool { return true },
        BeginScriptTransactionFunc: func() (backend.ScriptTransaction, error) {
            return &mockScriptTx{
                commitFn:   func() error { commitCalled = true; return nil },
                rollbackFn: func() error { rollbackCalled = true; return nil },
            }, nil
        },
    }

    ctx := &executor.ExecContext{
        Backend: mb,
        Output:  &strings.Builder{},
    }
    // ExecuteFn that fails on the second call
    callCount := 0
    ctx.ExecuteFn = func(stmt ast.Statement) error {
        callCount++
        if callCount == 2 {
            return fmt.Errorf("simulated failure")
        }
        return nil
    }

    // Write a temp script with 3 statements
    f, _ := os.CreateTemp("", "test-*.mdl")
    fmt.Fprintln(f, "show modules;")
    fmt.Fprintln(f, "show modules;")
    fmt.Fprintln(f, "show modules;")
    f.Close()
    defer os.Remove(f.Name())

    err := executor.ExecExecuteScriptForTest(ctx, f.Name())
    if err == nil {
        t.Error("expected error from failing statement, got nil")
    }
    if !rollbackCalled {
        t.Error("expected Rollback to be called on failure")
    }
    if commitCalled {
        t.Error("expected Commit NOT to be called on failure")
    }
}

func TestExecuteScript_Success_CommitCalled(t *testing.T) {
    rollbackCalled := false
    commitCalled := false

    mb := &mock.MockBackend{
        IsConnectedFunc: func() bool { return true },
        BeginScriptTransactionFunc: func() (backend.ScriptTransaction, error) {
            return &mockScriptTx{
                commitFn:   func() error { commitCalled = true; return nil },
                rollbackFn: func() error { rollbackCalled = true; return nil },
            }, nil
        },
    }

    ctx := &executor.ExecContext{
        Backend:    mb,
        Output:     &strings.Builder{},
        ExecuteFn:  func(stmt ast.Statement) error { return nil },
    }

    f, _ := os.CreateTemp("", "test-*.mdl")
    fmt.Fprintln(f, "show modules;")
    f.Close()
    defer os.Remove(f.Name())

    err := executor.ExecExecuteScriptForTest(ctx, f.Name())
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if !commitCalled {
        t.Error("expected Commit to be called on success")
    }
    if rollbackCalled {
        t.Error("expected Rollback NOT to be called on success")
    }
}
```

注：`executor.ExecExecuteScriptForTest` 是一个仅用于测试的导出包装，在 `cmd_misc.go` 或测试辅助文件中添加：

```go
// ExecExecuteScriptForTest exposes execExecuteScript for unit testing.
func ExecExecuteScriptForTest(ctx *ExecContext, path string) error {
    return execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path})
}
```

- [ ] **运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/executor/ \
  -run TestExecuteScript_ -v -timeout 30s
```

预期：PASS。

- [ ] **运行全部测试确认无回归**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/... -short -timeout 120s 2>&1 | grep -E "^(--- FAIL|FAIL|ok )"
```

预期：全部 PASS。

- [ ] **提交**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add \
  mdl/backend/connection.go \
  mdl/backend/backend.go \
  mdl/backend/mpr/backend.go \
  mdl/backend/mpr/write_helpers.go \
  mdl/backend/mock/backend.go \
  mdl/executor/cmd_misc.go \
  mdl/executor/script_tx_test.go
git commit -m "fix(executor): BUG-06 EXECUTE SCRIPT 加自动事务，失败时 Rollback"
```
