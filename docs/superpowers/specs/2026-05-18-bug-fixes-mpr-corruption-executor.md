# Bug 修复设计规格：MPR 损坏 + Executor 健壮性

**日期：** 2026-05-18  
**分支：** feature/expression-checker  
**涵盖：** BUG-01 / BUG-03 / BUG-04 / BUG-05 / BUG-06

---

## 1. 背景

M-0022 POC 阶段发现 6 个 Bug（BUG-02 已在当前分支修复）。本规格覆盖剩余 5 个修复。

| Bug | 严重度 | 现状 |
|-----|--------|------|
| BUG-01: drop attribute → MPR 损坏 | Critical | ⚠️ 代码已修复，缺回归测试 |
| BUG-03: XPath 引号标识符无提示 | Low | ❌ AST 层无检测 |
| BUG-04: grant 角色不存在时致命错误 | Medium | ❌ 未修复 |
| BUG-05: 参数名含 $ 前缀无清晰错误 | Medium | ❌ 未修复 |
| BUG-06: 脚本无事务，失败留孤儿对象 | Medium | ❌ 未修复 |

全部修复遵循 TDD：先写失败测试，再写实现。

---

## 2. BUG-01：drop attribute 回归测试

**结论：** 当前分支 `cleanupDroppedAttributeReferencesGen`（`cmd_alter_entity_gen.go:440`）已正确实现。

**验证依据：**
- BSON 中 `MemberAccess.Attribute` 字段为 UTF-8 字符串，格式 `"Module.Entity.AttrName"`
- `attrQN = s.Name.String() + "." + s.AttributeName` 构造出完全相同的格式
- QN 字符串比较可正确匹配并删除对应 `MemberAccess` 条目

**唯一工作：** 添加集成回归测试防止未来退化。

**测试文件：** `mdl/executor/roundtrip_mxcheck_test.go`

```go
func TestDropAttribute_WithAccessRules_NoCorruption(t *testing.T) {
    // 1. 创建持久化实体 + 属性
    // 2. 为该实体创建访问规则（含 read *, write *）
    // 3. DROP ATTRIBUTE
    // 4. 运行 mx check → 期望 0 errors（无 KeyNotFoundException）
    // 5. 确认已输出 "Removed N access rule member reference(s)"
}
```

---

## 3. BUG-03：XPath/属性路径中带引号标识符——AST 层拒绝

**问题：** 用户将 XPath 约束或属性路径中的标识符加上引号（如 `$Obj/"Attr"` 或 `["Module.Assoc" = $x]`），导致 CE 错误但提示不清晰。

**修复层级：** visitor 层（AST 构建阶段），而非运行时或文档。

**涉及文件：**
- `mdl/visitor/visitor_microflow.go`：属性路径解析（`$Var/Attr` 形式）
- `mdl/visitor/` XPath 约束解析（若有独立函数）

**设计：**

在解析属性访问路径时，检测路径中的带引号标识符并报诊断错误：

```go
func buildAttributePath(varName, rawAttr string) (string, error) {
    if isQuotedIdentifier(rawAttr) {
        bare := strings.Trim(rawAttr, `"`)
        return "", mdlerrors.NewValidationf(
            "attribute name %s in path must be unquoted — "+
                "use $%s/%s instead of $%s/%s",
            rawAttr, varName, bare, varName, rawAttr,
        )
    }
    return rawAttr, nil
}
```

对 XPath 约束中的关联引用（`[Module.Assoc = $x]`），若关联名出现引号也同样拒绝：

```
validation error: association reference in XPath must be unquoted
→ use [Module.Assoc = $Other] instead of ["Module.Assoc" = $Other]
```

**错误消息格式：**
```
validation error: attribute name "AttributeName" in path must be unquoted
→ use $Obj/AttributeName instead of $Obj/"AttributeName"
```

---

## 4. BUG-04：grant 角色不存在改为警告

**问题：** `findModuleRoleGen` 找不到模块角色时返回 `mdlerrors.NewNotFound`（致命错误），导致脚本中止。

**文件：** `mdl/executor/cmd_security_helpers.go:40`

**修改前：**
```go
return mdlerrors.NewNotFound("module role", role.Module+"."+role.Name)
```

**修改后：**
```go
fmt.Fprintf(ctx.Output, "WARNING: module role '%s.%s' not found — grant skipped\n",
    role.Module, role.Name)
return nil
```

**适用范围：** 所有 grant executor 函数（`execGrantMicroflowAccessGen`、`execGrantEntityAccessGen`、`execGrantPageAccessGen`、`execGrantNanoflowAccessGen` 等）中调用 `findModuleRoleGen` 并得到角色不存在的路径，均改为 WARNING + continue。

**不改变的行为：**
- 模块本身不存在（非角色不存在）仍报错
- `CREATE MODULE ROLE` 失败仍报错
- 其他类型的 `NewNotFound` 不受影响

---

## 5. BUG-05：参数名含 `$` 前缀的清晰诊断

**问题：** 用户写 `("$Name": String)`（带引号的 `$Name`），参数名字面包含 `$`，Mendix 拒绝该模型但错误信息不清晰。

注：不带引号的 `$Name` 在 grammar 中被正确识别为 `VARIABLE` token 并自动剥离 `$`（`visitor_microflow.go:259-261`），实际不会出错。

**文件：** `mdl/executor/validate_microflow.go`（参数校验循环处）

```go
for _, p := range stmt.Parameters {
    if strings.HasPrefix(p.Name, "$") {
        return mdlerrors.NewValidationf(
            "parameter name %q must not include '$' prefix in declaration — "+
                "declare as %q, reference in microflow body as $%s",
            p.Name, p.Name[1:], p.Name[1:],
        )
    }
}
```

**同样适用于：** nanoflow 参数校验（若存在独立校验函数）。

**错误示例（修复后）：**
```
validation error: parameter name "$Name" must not include '$' prefix in declaration
— declare as "Name", reference in microflow body as $Name
```

---

## 6. BUG-06：EXECUTE SCRIPT 自动事务

**问题：** `execExecuteScript` 逐语句执行，失败时已执行的语句已提交，留下孤儿对象。

**文件：** `mdl/executor/cmd_misc.go`（`execExecuteScript` 函数）

**设计约束：**
- 只有最外层脚本（`ScriptDepth == 0`）开事务；嵌套 `EXECUTE SCRIPT` 复用父事务
- 使用 `ctx.Backend.BeginWriteTransaction()` ——若 `FullBackend` 接口未暴露此方法，在 `mdl/backend/connection.go` 新增
- 失败时 Rollback，成功时 Commit
- `defer wtx.Rollback()` 作为异常保险（Commit 后 Rollback 为空操作）

**实现：**

```go
func execExecuteScript(ctx *ExecContext, s *ast.ExecuteScriptStmt) error {
    if ctx.ScriptDepth >= maxScriptDepth {
        return mdlerrors.NewValidationf("maximum script nesting depth exceeded")
    }

    // ... 现有：路径校验、读文件、stripSlashSeparators、解析 ...

    isRoot := ctx.ScriptDepth == 0
    var wtx WriteTransaction
    if isRoot {
        var err error
        wtx, err = ctx.Backend.BeginWriteTransaction()
        if err != nil {
            return fmt.Errorf("begin transaction for script '%s': %w", s.Path, err)
        }
        defer func() {
            if wtx != nil {
                _ = wtx.Rollback() // 安全保险：Commit 后为空操作
            }
        }()
    }

    fmt.Fprintf(ctx.Output, "Executing script: %s\n", s.Path)
    ctx.ScriptDepth++
    defer func() { ctx.ScriptDepth-- }()

    for _, stmt := range prog.Statements {
        if err := ctx.ExecuteFn(stmt); err != nil {
            if isRoot && wtx != nil {
                _ = wtx.Rollback()
                wtx = nil
                fmt.Fprintf(ctx.Output, "Script '%s' rolled back due to error\n", s.Path)
            }
            return fmt.Errorf("error in script '%s': %w", s.Path, err)
        }
    }

    if isRoot && wtx != nil {
        if err := wtx.Commit(); err != nil {
            return fmt.Errorf("commit script '%s': %w", s.Path, err)
        }
        wtx = nil
    }
    fmt.Fprintf(ctx.Output, "Script completed: %s\n", s.Path)
    return nil
}
```

**Backend 接口扩展（若需要）：**

在 `mdl/backend/connection.go` 的 `Writer` 或 `FullBackend` 接口新增：
```go
BeginWriteTransaction() (WriteTransaction, error)
```

其中 `WriteTransaction` 接口：
```go
type WriteTransaction interface {
    Commit() error
    Rollback() error
}
```

若 `msdkWriter.BeginWriteTransaction()` 已实现（`mdl/backend/mpr/backend.go` 中可见），则只需在接口层暴露。

---

## 7. 修改文件汇总

| 文件 | Bug | 类型 |
|------|-----|------|
| `mdl/executor/roundtrip_mxcheck_test.go` | BUG-01 | 新增测试 |
| `mdl/visitor/visitor_microflow.go` | BUG-03 | 修改：属性路径引号检测 |
| `mdl/executor/cmd_security_helpers.go` | BUG-04 | 修改：NewNotFound → WARNING |
| `mdl/executor/cmd_security_write_*.go`（多个） | BUG-04 | 修改：相关 grant 函数 |
| `mdl/executor/validate_microflow.go` | BUG-05 | 修改：参数名 $ 检测 |
| `mdl/executor/cmd_misc.go` | BUG-06 | 修改：加事务包装 |
| `mdl/backend/connection.go`（可能） | BUG-06 | 修改：暴露 BeginWriteTransaction |

---

## 8. 测试策略

每个修复均遵循 TDD 顺序：
1. 写失败测试
2. 运行确认失败
3. 写最小实现
4. 运行确认通过
5. 提交

| Bug | 测试类型 |
|-----|---------|
| BUG-01 | 集成测试（需 MPR + mx check） |
| BUG-03 | visitor 单元测试：输入带引号标识符，期望 error |
| BUG-04 | executor mock 测试：角色不存在时期望 WARNING 输出，无 error |
| BUG-05 | validate_microflow 单元测试：`$Name` 参数期望清晰 error |
| BUG-06 | executor mock 测试：脚本失败时期望回滚，无孤儿对象 |
