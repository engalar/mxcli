# exprcheck inferKind 完备化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全 `inferKind()` 对全部 15 种 AST 节点的类型推断，并为 `not`/`and`/`or` 的操作数添加类型检查（E009），防止将来新增节点时静默遗漏。

**Architecture:** 所有改动在 `mdl/exprcheck/parser.go` 和 `parser_test.go` 内完成，不改动任何外部接口或 hints/registry。新增内部 helper `checkBoolOperand`，在 `parseNot`/`parseAnd`/`parseOr` 中调用。覆盖测试 `TestInferKind_Coverage` 枚举全部 15 种节点，作为防回归门禁。

**Tech Stack:** Go 1.26，`mdl/exprcheck` 包（递归下降解析器，`hints.SeverityError`，`hintsLocation` helper）

---

## 文件改动清单

```
mdl/exprcheck/parser.go       修改：inferKind() + 新增 checkBoolOperand() + parseNot/parseAnd/parseOr
mdl/exprcheck/parser_test.go  修改：新增 TestInferKind_Coverage + 6 个 E009 测试函数
```

不改动：`ast.go`、`interfaces.go`、`hints/registry.go`、adapters/、`slot_resolver.go`

---

### Task 1: 写覆盖测试 TestInferKind_Coverage（先失败）

**Files:**
- Modify: `mdl/exprcheck/parser_test.go`

- [ ] **Step 1: 在 parser_test.go 末尾追加覆盖测试**

在 `TestParser_E007_RecoveryAtPrimary` 之后添加：

```go
func TestInferKind_Coverage(t *testing.T) {
	ctx := Context{}
	cases := []struct {
		name string
		node RobustExpr
		want TypeKind
	}{
		{"StringLit", &StringLit{Value: "x"}, KindString},
		{"NumberLit int", &NumberLit{Value: "1", Kind: KindInteger}, KindInteger},
		{"NumberLit dec", &NumberLit{Value: "1.5", Kind: KindDecimal}, KindDecimal},
		{"BoolLit", &BoolLit{Value: true}, KindBoolean},
		{"EmptyExpr", &EmptyExpr{}, KindEmpty},
		{"VariableExpr no scope", &VariableExpr{Name: "x"}, KindUnknown},
		{"AttributePathExpr", &AttributePathExpr{Variable: "x", Path: []string{"Attr"}}, KindUnknown},
		{"QNameExpr", &QNameExpr{Module: "M", Name: "E", Sub: "V"}, KindUnknown},
		{"CallExpr known", &CallExpr{Name: "length", Args: []RobustExpr{&StringLit{Value: "x"}}}, KindInteger},
		{"CallExpr unknown func", &CallExpr{Name: "myCustomFunc"}, KindUnknown},
		{"BinExpr AND", &BinExpr{Op: "AND", L: &BoolLit{Value: true}, R: &BoolLit{Value: false}}, KindBoolean},
		{"BinExpr OR", &BinExpr{Op: "OR", L: &BoolLit{Value: true}, R: &BoolLit{Value: false}}, KindBoolean},
		{"BinExpr eq", &BinExpr{Op: "=", L: &StringLit{Value: "a"}, R: &StringLit{Value: "b"}}, KindBoolean},
		{"BinExpr + strings", &BinExpr{Op: "+", L: &StringLit{Value: "a"}, R: &StringLit{Value: "b"}}, KindString},
		{"UnaryExpr NOT", &UnaryExpr{Op: "NOT", Operand: &BoolLit{Value: true}}, KindBoolean},
		{"UnaryExpr minus int", &UnaryExpr{Op: "-", Operand: &NumberLit{Value: "1", Kind: KindInteger}}, KindInteger},
		{"ParenExpr bool", &ParenExpr{Inner: &BoolLit{Value: true}}, KindBoolean},
		{"IfThenElseExpr string branches", &IfThenElseExpr{
			Cond: &BoolLit{Value: true},
			Then: &StringLit{Value: "yes"},
			Else: &StringLit{Value: "no"},
		}, KindString},
		{"TokenExpr", &TokenExpr{Token: "Translation.text"}, KindString},
		{"ConstantRef", &ConstantRef{QName: "M.Const"}, KindUnknown},
		{"RecoveredExpr", &RecoveredExpr{SourceFragment: "???"}, KindUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferKind(tc.node, ctx)
			if got != tc.want {
				t.Errorf("inferKind(%T) = %v, want %v", tc.node, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/exprcheck/... -run TestInferKind_Coverage -v 2>&1 | head -40
```

期望：4 个子测试失败——`UnaryExpr NOT`（got KindUnknown, want KindBoolean）、`UnaryExpr minus int`（got KindUnknown, want KindInteger）、`IfThenElseExpr string branches`（got KindUnknown, want KindString）、`TokenExpr`（got KindUnknown, want KindString）。其余 17 个应 PASS。

---

### Task 2: 修复 inferKind — 补全 7 个缺失 case

**Files:**
- Modify: `mdl/exprcheck/parser.go`

- [ ] **Step 1: 在 inferKind() 的 switch 中添加缺失 case**

找到 `parser.go` 中 `inferKind` 函数的 `switch n := e.(type) {` 块（约第 403 行），在 `case *BinExpr:` 之后、`}` 之前插入：

```go
	case *UnaryExpr:
		if n.Op == "NOT" {
			return KindBoolean
		}
		return inferKind(n.Operand, ctx)
	case *IfThenElseExpr:
		if n.Then != nil {
			if k := inferKind(n.Then, ctx); k != KindUnknown {
				return k
			}
		}
		if n.Else != nil {
			return inferKind(n.Else, ctx)
		}
		return KindUnknown
	case *TokenExpr:
		return KindString
	case *AttributePathExpr, *QNameExpr, *ConstantRef, *RecoveredExpr:
		return KindUnknown
```

完整的 `inferKind` 函数 switch 应如下（仅展示 switch 体，其余不变）：

```go
func inferKind(e RobustExpr, ctx Context) TypeKind {
	switch n := e.(type) {
	case *StringLit:
		return KindString
	case *NumberLit:
		return n.Kind
	case *BoolLit:
		return KindBoolean
	case *EmptyExpr:
		return KindEmpty
	case *VariableExpr:
		if ctx.Scope != nil {
			if k, ok := ctx.Scope.Lookup(n.Name); ok {
				return k
			}
		}
	case *CallExpr:
		if sig, ok := funcTable[n.Name]; ok {
			return sig.ret
		}
	case *ParenExpr:
		return inferKind(n.Inner, ctx)
	case *BinExpr:
		if n.Op == "+" {
			l := inferKind(n.L, ctx)
			r := inferKind(n.R, ctx)
			if l == KindString && r == KindString {
				return KindString
			}
			return l
		}
		if n.Op == "AND" || n.Op == "OR" || n.Op == "=" || n.Op == "!=" ||
			n.Op == "<" || n.Op == "<=" || n.Op == ">" || n.Op == ">=" {
			return KindBoolean
		}
	case *UnaryExpr:
		if n.Op == "NOT" {
			return KindBoolean
		}
		return inferKind(n.Operand, ctx)
	case *IfThenElseExpr:
		if n.Then != nil {
			if k := inferKind(n.Then, ctx); k != KindUnknown {
				return k
			}
		}
		if n.Else != nil {
			return inferKind(n.Else, ctx)
		}
		return KindUnknown
	case *TokenExpr:
		return KindString
	case *AttributePathExpr, *QNameExpr, *ConstantRef, *RecoveredExpr:
		return KindUnknown
	}
	return KindUnknown
}
```

- [ ] **Step 2: 运行覆盖测试确认通过**

```bash
go test ./mdl/exprcheck/... -run TestInferKind_Coverage -v 2>&1 | tail -10
```

期望：`--- PASS: TestInferKind_Coverage`，21 个子测试全绿。

- [ ] **Step 3: 运行全量测试确认无回归**

```bash
go test ./mdl/exprcheck/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```

期望：全部 PASS，无 FAIL。

- [ ] **Step 4: Commit**

```bash
git add mdl/exprcheck/parser.go mdl/exprcheck/parser_test.go
git commit -m "feat(exprcheck): complete inferKind for all 15 AST node types

Add cases for UnaryExpr (NOT→Boolean, minus→pass-through), IfThenElseExpr
(infer from Then/Else branches), TokenExpr (always String), and explicit
KindUnknown for AttributePathExpr, QNameExpr, ConstantRef, RecoveredExpr.
TestInferKind_Coverage gate ensures future nodes are not silently missed."
```

---

### Task 3: 写 E009 失败测试 — not 操作数检查

**Files:**
- Modify: `mdl/exprcheck/parser_test.go`

- [ ] **Step 1: 在 parser_test.go 末尾追加 3 个 not 相关测试**

```go
func TestParser_E009_NotNonBoolean(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"string operand", "not 'hello'"},
		{"integer operand", "not 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if !hasCode(hs, "E009") {
				t.Fatalf("expected E009 for %q, got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_NotBoolLit_NoHint(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"bool literal", "not true"},
		{"bool expression", "not (1 = 1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if hasCode(hs, "E009") {
				t.Fatalf("E009 must not fire for bool operand %q; got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_NotUnknownOperand_NoHint(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("not $Validation/IsValid", Context{Microflow: "M.F"})
	if hasCode(hs, "E009") {
		t.Fatalf("E009 must not fire for unknown-kind operand; got %+v", hs)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/exprcheck/... -run TestParser_E009_Not -v 2>&1 | head -20
```

期望：`TestParser_E009_NotNonBoolean` 的两个子测试 FAIL（"expected E009"），`NoHint` 两个测试 PASS（因为当前不会发 E009）。

---

### Task 4: 实现 checkBoolOperand 并接入 parseNot

**Files:**
- Modify: `mdl/exprcheck/parser.go`

- [ ] **Step 1: 在 parser.go 末尾（hintsLocation 之前）添加 checkBoolOperand helper**

在 `func hintsLocation` 之前插入：

```go
// checkBoolOperand emits E009 when expr's inferred kind is known and non-Boolean.
// op is the operator keyword ("not", "and", "or") used in the hint message.
func checkBoolOperand(expr RobustExpr, ctx Context, op string) []Hint {
	k := inferKind(expr, ctx)
	if k == KindUnknown || k == KindBoolean {
		return nil
	}
	return []Hint{{
		Code:     "E009",
		Slug:     "slot-type-mismatch",
		Severity: hints.SeverityError,
		Where:    hintsLocation(ctx, expr.Pos()),
		YouWrote: op + " <" + typeKindName(k) + ">",
		Problem: "'" + op + "' requires a Boolean operand, but this expression has kind " +
			typeKindName(k) + ".",
		Fix: "Replace the operand with a Boolean expression " +
			"(e.g. a comparison, a Boolean attribute path, or true/false).",
	}}
}
```

- [ ] **Step 2: 修改 parseNot 调用 checkBoolOperand**

找到 `parseNot`（约第 43 行），将其修改为：

```go
func parseNot(s *Stream, ctx Context) (RobustExpr, []Hint) {
	if matchKeyword(s, "not") {
		inner, h := parseCmp(s, ctx)
		h = append(h, checkBoolOperand(inner, ctx, "not")...)
		return &UnaryExpr{Op: "NOT", Operand: inner}, h
	}
	return parseCmp(s, ctx)
}
```

- [ ] **Step 3: 运行 not 相关测试确认全绿**

```bash
go test ./mdl/exprcheck/... -run TestParser_E009_Not -v 2>&1
```

期望：3 个测试函数全部 PASS。

- [ ] **Step 4: 运行全量测试确认无回归**

```bash
go test ./mdl/exprcheck/... 2>&1 | tail -5
```

期望：`ok  	github.com/mendixlabs/mxcli/mdl/exprcheck`，无 FAIL。

- [ ] **Step 5: Commit**

```bash
git add mdl/exprcheck/parser.go mdl/exprcheck/parser_test.go
git commit -m "feat(exprcheck): E009 for non-Boolean operand of 'not'

Add checkBoolOperand helper that emits E009 when inferred kind is
known and non-Boolean. Wire into parseNot. not 'hello' / not 5
now produce E009; not \$x/Attr (unknown kind) stays silent."
```

---

### Task 5: 写 E009 失败测试 — and/or 操作数检查

**Files:**
- Modify: `mdl/exprcheck/parser_test.go`

- [ ] **Step 1: 在 parser_test.go 末尾追加 3 个 and/or 相关测试**

```go
func TestParser_E009_AndNonBoolean(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"string left operand", "'hello' and true"},
		{"integer right operand", "true and 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if !hasCode(hs, "E009") {
				t.Fatalf("expected E009 for %q, got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_OrNonBoolean(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("'hello' or true", Context{Microflow: "M.F"})
	if !hasCode(hs, "E009") {
		t.Fatalf("expected E009, got %+v", hs)
	}
}

func TestParser_E009_AndOrUnknown_NoHint(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"and with unknown right", "true and $x/Attr"},
		{"or with unknown left", "$x/Attr or true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if hasCode(hs, "E009") {
				t.Fatalf("E009 must not fire for unknown-kind operand %q; got %+v", tc.src, hs)
			}
		})
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/exprcheck/... -run TestParser_E009_And -v 2>&1 | head -20
go test ./mdl/exprcheck/... -run TestParser_E009_Or -v 2>&1 | head -20
```

期望：`TestParser_E009_AndNonBoolean` 和 `TestParser_E009_OrNonBoolean` FAIL（"expected E009"），`NoHint` 测试 PASS。

---

### Task 6: 接入 parseAnd 和 parseOr

**Files:**
- Modify: `mdl/exprcheck/parser.go`

- [ ] **Step 1: 修改 parseAnd 调用 checkBoolOperand**

找到 `parseAnd`（约第 33 行），将其修改为：

```go
func parseAnd(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseNot(s, ctx)
	first := true
	for matchKeyword(s, "and") {
		if first {
			hints = append(hints, checkBoolOperand(left, ctx, "and")...)
			first = false
		}
		right, h := parseNot(s, ctx)
		hints = append(hints, h...)
		hints = append(hints, checkBoolOperand(right, ctx, "and")...)
		left = &BinExpr{Op: "AND", L: left, R: right}
	}
	return left, hints
}
```

- [ ] **Step 2: 修改 parseOr 调用 checkBoolOperand**

找到 `parseOr`（约第 23 行），将其修改为：

```go
func parseOr(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseAnd(s, ctx)
	first := true
	for matchKeyword(s, "or") {
		if first {
			hints = append(hints, checkBoolOperand(left, ctx, "or")...)
			first = false
		}
		right, h := parseAnd(s, ctx)
		hints = append(hints, h...)
		hints = append(hints, checkBoolOperand(right, ctx, "or")...)
		left = &BinExpr{Op: "OR", L: left, R: right}
	}
	return left, hints
}
```

- [ ] **Step 3: 运行 and/or 测试确认全绿**

```bash
go test ./mdl/exprcheck/... -run "TestParser_E009_And|TestParser_E009_Or" -v 2>&1
```

期望：全部 PASS。

- [ ] **Step 4: 运行全量测试确认无回归**

```bash
go test ./mdl/exprcheck/... 2>&1 | tail -5
```

期望：`ok  	github.com/mendixlabs/mxcli/mdl/exprcheck`，无 FAIL。

- [ ] **Step 5: make vet + fmt**

```bash
make fmt && make vet 2>&1 | tail -10
```

期望：无错误输出。

- [ ] **Step 6: Commit**

```bash
git add mdl/exprcheck/parser.go mdl/exprcheck/parser_test.go
git commit -m "feat(exprcheck): E009 for non-Boolean operands of 'and'/'or'

Wire checkBoolOperand into parseAnd and parseOr. Left operand is
checked on first entry to the loop; each right operand is checked
after parsing. Unknown-kind operands (e.g. attribute paths) are
silently skipped to avoid false positives."
```

---

## 验收标准

```bash
go test ./mdl/exprcheck/... -v 2>&1 | grep -E "PASS|FAIL|ok"
```

- `TestInferKind_Coverage`：21 个子测试全绿
- `TestParser_E009_NotNonBoolean`：2 个子测试绿
- `TestParser_E009_NotBoolLit_NoHint`：2 个子测试绿
- `TestParser_E009_NotUnknownOperand_NoHint`：绿
- `TestParser_E009_AndNonBoolean`：2 个子测试绿
- `TestParser_E009_OrNonBoolean`：绿
- `TestParser_E009_AndOrUnknown_NoHint`：2 个子测试绿
- 所有原有测试：无回归
