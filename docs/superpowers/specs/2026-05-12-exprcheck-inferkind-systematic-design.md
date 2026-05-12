# exprcheck: inferKind 完备化 + 操作数类型检查

**日期**: 2026-05-12  
**状态**: 待实现  
**范围**: `mdl/exprcheck/parser.go` + `mdl/exprcheck/parser_test.go`

## 背景

`exprcheck` 包的表达式解析器使用 `inferKind()` 推断 AST 节点的类型。该函数目前只处理 15 种 `RobustExpr` 节点中的 8 种，7 种节点缺失——包括 `UnaryExpr{NOT}`。

根本原因（来自 git 考古）：`inferKind` 在 commit `74a966d4` 中引入，目的仅为检测 E004（`+` 运算符混类型），不是对所有操作符的系统覆盖。`parseNot` 从第一个 commit 就存在，但两条代码路径从未对齐。

直接后果：`not $Validation/IsValid` 解析正确，但类型推断返回 `KindUnknown`，导致：

1. E009（slot-type-mismatch）对 `not` 操作数永远不触发
2. `not 'hello'`、`not 5` 等明显错误静默通过
3. AND/OR 操作数同样无类型检查

## 目标

- `inferKind()` 对全部 15 种 AST 节点返回正确或合理的 `TypeKind`
- `not`/`and`/`or` 操作数若类型已知且非 Boolean，发 E009
- 覆盖测试防止未来新增 AST 节点时静默漏掉

## 不在范围内

- catalog 接入（`AttributePathExpr` 类型推断需要 catalog，本次保持 `KindUnknown`）
- 新增 hint code（复用已有 E009）
- AND/OR 操作数的跨 catalog 推断
- 改动 adapters、hints/registry、接口层

## 方案 A：inferKind 完备化 + 操作数检查 + 覆盖测试

### 1. inferKind — 补全 7 个缺失 case

```
节点                  返回值                    说明
─────────────────────────────────────────────────────────────────
UnaryExpr{Op:"NOT"}  KindBoolean               not 总产生 Boolean
UnaryExpr{Op:"-"}    inferKind(n.Operand, ctx) 负号穿透数值类型
IfThenElseExpr       inferKind(Then)，          从 then 分支推断，
                     若 Unknown → inferKind(Else) then/else 类型不一致时保守
TokenExpr            KindString                翻译 token 永远是字符串
AttributePathExpr    KindUnknown               无 catalog 无法推断（显式）
QNameExpr            KindUnknown               同上
ConstantRef          KindUnknown               同上
RecoveredExpr        KindUnknown               同上（已是默认值，显式写出）
```

### 2. 操作数类型检查 helper

新增内部函数：

```go
// checkBoolOperand emits E009 when expr's inferred kind is known and non-Boolean.
// op is the operator name used in the hint message ("not", "and", "or").
func checkBoolOperand(expr RobustExpr, ctx Context, op string) []Hint
```

逻辑：
- 调用 `inferKind(expr, ctx)`
- 若 `kind == KindUnknown` 或 `kind == KindBoolean` → 不发 hint（保守，不误报）
- 否则 → 发 E009，`YouWrote` = `op + " <operand>"`, `Fix` = 说明 operand 应为 Boolean

触发点：
- `parseNot`：对 `parseCmp` 结果调用一次
- `parseAnd`：对 left（首次进入循环时）和每个 right 各调用一次
- `parseOr`：同 `parseAnd`

### 3. E009 触发矩阵

| 表达式 | 操作数 kind | 触发 E009 |
|--------|------------|----------|
| `not 'hello'` | KindString | ✓ |
| `not 5` | KindInteger | ✓ |
| `'hello' and true` | KindString（left） | ✓ |
| `true and 5` | KindInteger（right） | ✓ |
| `not true` | KindBoolean | ✗ |
| `not (1 = 1)` | KindBoolean | ✗ |
| `not $Validation/IsValid` | KindUnknown | ✗（无法静态判断） |
| `true and $x/Attr` | KindUnknown（right） | ✗ |

### 4. 覆盖测试 TestInferKind_Coverage

表驱动测试，枚举全部 15 种 `RobustExpr` 子类型，每条验证：
- `inferKind` 不 panic
- 返回值符合预期（Unknown 的节点只验证不 panic，不断言具体值）

| 节点 | 期望值 |
|------|--------|
| `StringLit{"x"}` | KindString |
| `NumberLit{"1", KindInteger}` | KindInteger |
| `NumberLit{"1.5", KindDecimal}` | KindDecimal |
| `BoolLit{true}` | KindBoolean |
| `EmptyExpr{}` | KindEmpty |
| `VariableExpr{"x"}` | KindUnknown（无 scope） |
| `AttributePathExpr{x, [Attr]}` | KindUnknown |
| `QNameExpr{M, E, V}` | KindUnknown |
| `CallExpr{"length", [StringLit]}` | KindInteger |
| `CallExpr{"unknownFunc"}` | KindUnknown |
| `BinExpr{AND, BoolLit, BoolLit}` | KindBoolean |
| `BinExpr{OR, BoolLit, BoolLit}` | KindBoolean |
| `BinExpr{=, StringLit, StringLit}` | KindBoolean |
| `BinExpr{+, StringLit, StringLit}` | KindString |
| `UnaryExpr{NOT, BoolLit}` | KindBoolean |
| `UnaryExpr{-, NumberLit(int)}` | KindInteger |
| `ParenExpr{BoolLit}` | KindBoolean |
| `IfThenElseExpr{_, StringLit, StringLit}` | KindString |
| `TokenExpr{"Translation.text"}` | KindString |
| `ConstantRef{"M.Const"}` | KindUnknown |
| `RecoveredExpr{"???"}` | KindUnknown |

新增 parser 测试函数：

- `TestParser_E009_NotNonBoolean` — `not 'hello'`、`not 5` 触发 E009
- `TestParser_E009_NotBoolLit_NoHint` — `not true`、`not (1 = 1)` 无 hint
- `TestParser_E009_NotUnknownOperand_NoHint` — `not $x/Attr` 无 hint
- `TestParser_E009_AndNonBoolean` — `'hello' and true`、`true and 5` 触发 E009
- `TestParser_E009_OrNonBoolean` — `'hello' or true` 触发 E009
- `TestParser_E009_AndOrUnknown_NoHint` — `true and $x/Attr` 无 hint

## 文件改动清单

```
mdl/exprcheck/parser.go
  inferKind()          +7 case（UnaryExpr×2, IfThenElseExpr, TokenExpr, 4×KindUnknown 显式）
  checkBoolOperand()   新增内部 helper
  parseNot()           调用 checkBoolOperand
  parseAnd()           调用 checkBoolOperand（left 首次 + 每个 right）
  parseOr()            调用 checkBoolOperand（left 首次 + 每个 right）

mdl/exprcheck/parser_test.go
  TestInferKind_Coverage          新增（21 条表驱动）
  TestParser_E009_Not*            新增 3 个函数
  TestParser_E009_AndOr*          新增 3 个函数
```

不改动文件：`hints/registry.go`、`ast.go`、`interfaces.go`、adapters/、`slot_resolver.go`

## 验收标准

1. `go test ./mdl/exprcheck/...` 全绿
2. `not 'hello'` → E009 触发，`YouWrote` 和 `Fix` 字段非空
3. `not $Validation/IsValid` → 零 hint
4. `TestInferKind_Coverage` 覆盖 21 个节点，全部通过
5. `make vet && make fmt` 无报错
