# SEM-03 Expression Type Checking — Design Spec

**Date:** 2026-05-18  
**Status:** Approved  
**Rule ID:** SEM-03  
**Goal:** Detect all 41 remaining CE0117 type-mismatch errors in macnica that MEMV Phase 3 cannot catch via attribute-existence checks alone.

---

## 1. Problem Statement

mx check reports 44 CE0117 ("Error(s) in expression") errors in macnica. MEMV Phase 3 (SEM-07) detects 3 via attribute-not-found checks. The remaining 41 are **type mismatches**: an expression returns a TypeKind that is incompatible with what the target slot expects.

Examples:
- `$Year = year($Date)` — `year()` returns Integer, but `$Year` is declared as String
- Log message template with `$count` (Integer) concatenated without `toString()`
- Microflow call argument passing Integer to a String parameter

---

## 2. Architecture

```
ExprRecord (UnitType, Field, Raw, UnitPath,
            TargetAttrQN*, CalleeQN*, ParamName*)     ← scan extended
         │
         ▼
parse.ParseExpression() → ParseResult
 ├── Record  scan.ExprRecord
 ├── Hints   []hints.Hint
 └── AST     exprcheck.RobustExpr                     ← parse extended

         │
         ├──────────────────────┐
         ▼                      ▼
SlotResolver                 Inferrer
.Expect(rec, cat)            .Infer(AST, scope, cat, funcs)
→ TypeKind (expected)        → TypeKind (actual)
         │                      │
         └────────────┬─────────┘
                      ▼
          typecheck.Check() → []ValidationResult (SEM-03)
```

*Fields marked `*` are new additions to ExprRecord.

All dependencies are injected through interfaces — no concrete types cross package boundaries.

---

## 3. ExprRecord Extensions (`internal/expr/scan/scan.go`)

Three new optional fields, populated only when the BSON unit type warrants it. Empty string means "not applicable".

```go
type ExprRecord struct {
    // … existing fields unchanged …

    // NEW — type-checking context captured at scan time
    TargetAttrQN string // ChangeActionItem: "Module.Entity.AttrName" (3-part QN)
    CalleeQN     string // MicroflowCallParameterMapping: "Module.MFName"
    ParamName    string // MicroflowCallParameterMapping: parameter name
}
```

**Scan-time population:**

| BSON `$Type` | Existing field read | New fields read |
|---|---|---|
| `Microflows$ChangeActionItem` | `Value` | `Attribute` → `TargetAttrQN` |
| `Microflows$MicroflowCallParameterMapping` | `Argument` | `Microflow` → `CalleeQN`, `Parameter` → `ParamName` |
| `Mappings$MicroflowCallParameterMappingImpl` | `Argument` | same as above |
| `Workflows$MicroflowCallParameterMapping` | `Expression` | same as above |

---

## 4. ParseResult Extension (`internal/expr/parse/parse.go`)

```go
type ParseResult struct {
    Record scan.ExprRecord
    OK     bool
    Hints  []hints.Hint
    AST    exprcheck.RobustExpr // NEW: root AST node; nil for XPath expressions
}
```

Change in `parseExprWithCatalog`:
```go
// Before:
_, hs := exprParser.Parse(rec.Raw, ctx)
// After:
ast, hs := exprParser.Parse(rec.Raw, ctx)
// Populate ParseResult.AST = ast
```

XPath expressions (routed through the ANTLR visitor) set `AST = nil`; the type checker skips them.

---

## 5. New Package: `internal/expr/typecheck/`

### 5.1 Interfaces (`interfaces.go`)

All interfaces are small and single-purpose (ISP). All implementations are injected (DIP).

```go
// Inferrer infers the TypeKind of a parsed AST node.
// S: only infers, does not report errors.
// O: new type rules added in Inferrer implementation, not in callers.
type Inferrer interface {
    Infer(node exprcheck.RobustExpr, scope VarScope,
          cat AttrCatalog, funcs FuncReg) exprcheck.TypeKind
}

// VarScope resolves $variable → TypeKind for the current microflow.
type VarScope interface {
    TypeOf(varName string) exprcheck.TypeKind
}

// AttrCatalog resolves entity attribute types and variable entity QNs.
type AttrCatalog interface {
    AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool)
    EntityQNOf(varName string) string
}

// FuncReg provides the return type of a named built-in function.
// O: new functions added to the registry, not to Inferrer.
type FuncReg interface {
    ReturnType(funcName string) (exprcheck.TypeKind, bool)
}

// SlotResolver resolves the expected TypeKind for an expression slot.
// S: expected-type resolution is separate from actual-type inference.
type SlotResolver interface {
    Expect(rec scan.ExprRecord, cat AttrCatalog) (exprcheck.TypeKind, bool)
}

// TypeChecker orchestrates all four interfaces.
type TypeChecker struct {
    infer  Inferrer
    slots  SlotResolver
    funcs  FuncReg
    scope  VarScope
    cat    AttrCatalog
}
```

### 5.2 FuncRegistry (`funcreg.go`)

Thin adapter over the existing `exprcheck.AllFuncSigs()` map. **Zero duplication.**

```go
type defaultFuncReg struct {
    sigs map[string]exprcheck.PublicFuncSig
}

func NewFuncReg() FuncReg {
    return &defaultFuncReg{sigs: exprcheck.AllFuncSigs()}
}

func (r *defaultFuncReg) ReturnType(name string) (exprcheck.TypeKind, bool) {
    s, ok := r.sigs[name]
    if !ok { return exprcheck.KindUnknown, false }
    return s.ReturnType, true
}
```

**Missing functions to add to `funcTable` in `mdl/exprcheck/func_checker.go`** (DateTime extraction group):

```go
// DateTime — extraction → Integer
"year":        {args: []TypeKind{KindDateTime}, ret: KindInteger},
"month":       {args: []TypeKind{KindDateTime}, ret: KindInteger},
"dayOfYear":   {args: []TypeKind{KindDateTime}, ret: KindInteger},
"dayOfMonth":  {args: []TypeKind{KindDateTime}, ret: KindInteger},
"weekOfYear":  {args: []TypeKind{KindDateTime}, ret: KindInteger},
"dayOfWeek":   {args: []TypeKind{KindDateTime}, ret: KindInteger},
"hour":        {args: []TypeKind{KindDateTime}, ret: KindInteger},
"minute":      {args: []TypeKind{KindDateTime}, ret: KindInteger},
"second":      {args: []TypeKind{KindDateTime}, ret: KindInteger},
"millisecond": {args: []TypeKind{KindDateTime}, ret: KindInteger},
```

### 5.3 TypeInferrer (`inferrer.go`)

AST type switch — one case per node type. Returns `KindUnknown` on any ambiguity (fail-safe: no false positives).

```go
func (d *defaultInferrer) Infer(node exprcheck.RobustExpr,
    scope VarScope, cat AttrCatalog, funcs FuncReg) exprcheck.TypeKind {
    switch n := node.(type) {
    case *exprcheck.StringLit:       return KindString
    case *exprcheck.NumberLit:       return n.Kind        // already KindInteger or KindDecimal
    case *exprcheck.BoolLit:         return KindBoolean
    case *exprcheck.EmptyExpr:       return KindEmpty
    case *exprcheck.ConstantRef:     return KindUnknown   // handled by SEM-05
    case *exprcheck.VariableExpr:    return scope.TypeOf(n.Name)
    case *exprcheck.AttributePathExpr: return inferAttrPath(n, scope, cat)
    case *exprcheck.CallExpr:
        if k, ok := funcs.ReturnType(n.Name); ok { return k }
        return KindUnknown
    case *exprcheck.BinExpr:         return inferBinExpr(n, scope, cat, funcs, d)
    case *exprcheck.UnaryExpr:
        if n.Op == "not" { return KindBoolean }
        return d.Infer(n.Operand, scope, cat, funcs)
    case *exprcheck.IfThenElseExpr:
        if t := d.Infer(n.Then, scope, cat, funcs); t != KindUnknown { return t }
        return d.Infer(n.Else, scope, cat, funcs)
    case *exprcheck.TokenExpr:       return inferToken(n.Token)
    case *exprcheck.ParenExpr:       return d.Infer(n.Inner, scope, cat, funcs)
    case *exprcheck.RecoveredExpr:   return KindUnknown   // parse failure; skip
    }
    return KindUnknown
}
```

**`inferAttrPath`**: resolves entity QN from `scope`, then `cat.AttributeKind()` for the last path segment. Intermediate segments (associations) return `KindUnknown` — multi-hop attribute type inference is out of scope for SEM-03.

**`inferBinExpr` rules:**

| Operator | Rule |
|---|---|
| `=` `!=` `<` `>` `<=` `>=` `and` `or` | → `KindBoolean` |
| `+` (both String) | → `KindString` |
| `+` `-` `*` `/` `div` `mod` (numeric) | → widest of Integer < Long < Decimal |
| Mixed String + non-String | → `KindUnknown` (E004 already reports this) |

**`inferToken` mappings:**

| Token | TypeKind |
|---|---|
| `CurrentDateTime`, `CurrentBeginOfDay/Week/Month/Year` | `KindDateTime` |
| `CurrentUser`, `CurrentObject` | `KindObject` |
| `True`, `False` | `KindBoolean` |
| `Null` | `KindEmpty` |

### 5.4 SlotResolver (`slot_resolver.go`)

```go
var staticSlots = map[string]exprcheck.TypeKind{
    "Microflows$ExpressionSplitCondition/Expression": KindBoolean,
    "Microflows$WhileLoopCondition/WhileExpression":  KindBoolean,
    "DomainModels$AccessRule/XPathConstraint":        KindBoolean,
    "Microflows$CustomRange/LimitExpression":         KindInteger,
    "Microflows$CustomRange/OffsetExpression":        KindInteger,
    "Microflows$TemplateParameter/Expression":        KindString,
}

var dynamicSlots = map[string]slotResolverFn{
    "Microflows$ChangeActionItem/Value":                       resolveAttrTarget,
    "Microflows$ChangeVariableAction/Value":                   resolveVarTarget,
    "Microflows$CreateVariableAction/InitialValue":            resolveVarTarget,
    "Microflows$MicroflowCallParameterMapping/Argument":       resolveCallArgTarget,
    "Mappings$MicroflowCallParameterMappingImpl/Argument":     resolveCallArgTarget,
    "Workflows$MicroflowCallParameterMapping/Expression":      resolveCallArgTarget,
    "Microflows$EndEvent/ReturnValue":                         resolveMicroflowReturn,
}
```

**Dynamic resolver functions:**

| Function | Input source | Logic |
|---|---|---|
| `resolveAttrTarget` | `rec.TargetAttrQN` | Split `"Module.Entity.Attr"` → `cat.AttributeKind(entityQN, attr)` |
| `resolveVarTarget` | `rec.UnitPath` + varName from slot | `idx.VarTypeKind(unitPath, varName)` |
| `resolveCallArgTarget` | `rec.CalleeQN` + `rec.ParamName` | `idx.MicroflowParamKind(calleeQN, paramName)` |
| `resolveMicroflowReturn` | `rec.UnitPath` | `idx.MicroflowReturnKind(unitPath)` |

---

## 6. meta.Index Extensions (`internal/expr/meta/`)

### 6.1 New index fields

```go
type Index struct {
    // … existing fields …
    mfParamKinds  map[string]map[string]exprcheck.TypeKind // "Module.MFName" → paramName → TypeKind
    mfReturnKinds map[string]exprcheck.TypeKind            // mfName (bare) → return TypeKind
}
```

### 6.2 VarMap upgraded to TypeKind

Current `microflowVars map[string]map[string]string` stores `varName → entityQN` (entity-only). Upgrade to `map[string]map[string]exprcheck.TypeKind` stores the TypeKind directly (entity variables → `KindObject` with entityQN stored separately, or keep dual maps).

**Option chosen**: Keep `microflowVars` for entityQN (used by SEM-07 path validation), add parallel `mfVarKinds map[string]map[string]TypeKind` for raw TypeKind (used by SEM-03 slot resolution).

### 6.3 New IndexReader interface methods

```go
type IndexReader interface {
    // … existing methods …
    VarTypeKind(unitPath, varName string) exprcheck.TypeKind
    MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool)
    MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool) // bare name key
}
```

---

## 7. Integration (`internal/expr/daemon/daemon.go`)

```go
// In daemon.validate() — one new line per expression
checker := typecheck.NewChecker(d.index)
for _, pr := range parseResults {
    emit(validate.ValidateSyntax(pr))
    emit(validate.ValidateSemantic(pr, d.index))
    emit(checker.Check(pr))                          // NEW
}
```

`typecheck.NewChecker(idx IndexReader)` wires all components from the index.

No-daemon path receives the same addition in `runExprValidateNoDaemon` — but since the index is nil in no-daemon mode, `checker.Check()` returns nil immediately (consistent with existing SEM-07 behaviour).

---

## 8. Error Reporting

**Rule ID:** `SEM-03`  
**Severity:** `ERROR`

```
[ERROR] SEM-03: Expression returns Integer but slot expects String.
  Fix: Wrap with toString(): toString(year($Date))
```

**Fix suggestion table** (`fixSuggestion(actual, expected TypeKind) string`):

| actual | expected | Suggestion |
|---|---|---|
| Integer/Long/Decimal | String | `"Wrap with toString(): toString(<expr>)"` |
| String | Integer | `"Wrap with parseInteger(): parseInteger(<expr>)"` |
| String | Decimal | `"Wrap with parseDecimal(): parseDecimal(<expr>)"` |
| String | Boolean | `"Wrap with parseBoolean(): parseBoolean(<expr>)"` |
| String | DateTime | `"Wrap with parseDateTime(): parseDateTime(<expr>, 'format')"` |
| Boolean | String | `"Wrap with toString(): toString(<expr>)"` |
| DateTime | Integer | `"Extract the needed component: year(<expr>), month(<expr>), etc."` |
| Empty | non-empty | `"Expression may be empty; add a null-check before use"` |
| any | Boolean | `"Slot requires a Boolean expression (comparison or boolean function)"` |
| default | — | `"Expression type (<actual>) is not compatible with <expected>"` |

**Type compatibility** (`compatible(actual, expected TypeKind) bool`):
- `actual == expected` → ✅
- `KindUnknown` (either side) → ✅ skip (fail-safe; prefer false negatives)
- `KindEmpty` assigned to `KindObject` or `KindList` → ✅
- `KindInteger` ≤ `KindLong` ≤ `KindDecimal` (widening) → ✅
- `KindAny` (either side) → ✅

---

## 9. Testing Strategy

- **Unit tests** for `Inferrer`: table-driven, cover all AST node types and edge cases
- **Unit tests** for `SlotResolver`: one test per static slot, one per dynamic resolver function
- **Unit tests** for `FuncReg`: verify all 10 new DateTime extraction functions return Integer
- **Integration test** `TestSEM03_Macnica`: run full pipeline on macnica MPR, assert `len(SEM-03 results) >= 35` (accounting for ~6 undetectable due to `KindUnknown` paths)
- **No-false-positive test** `TestSEM03_Mx2026AIDay`: assert 0 SEM-03 results on Mx2026AIDay (mx check says 0 errors)

---

## 10. Files Changed / Created

| File | Change |
|---|---|
| `internal/expr/scan/scan.go` | Add `TargetAttrQN`, `CalleeQN`, `ParamName` to `ExprRecord`; populate in `scanObj` |
| `internal/expr/parse/parse.go` | Add `AST exprcheck.RobustExpr` to `ParseResult`; store from parser |
| `mdl/exprcheck/func_checker.go` | Add 10 DateTime extraction functions to `funcTable` |
| `internal/expr/meta/index.go` | Add `mfParamKinds`, `mfReturnKinds`, `mfVarKinds`; populate in `buildMicroflowVars` |
| `internal/expr/meta/catalog_reader.go` | Add `VarTypeKind`, `MicroflowParamKind`, `MicroflowReturnKind` |
| `internal/expr/meta/mock_index.go` | Implement new `IndexReader` methods |
| `internal/expr/validate/validate_sem.go` | Add new methods to `IndexReader` interface |
| `internal/expr/typecheck/interfaces.go` | **NEW** — all interfaces |
| `internal/expr/typecheck/inferrer.go` | **NEW** — `defaultInferrer` |
| `internal/expr/typecheck/funcreg.go` | **NEW** — `defaultFuncReg` |
| `internal/expr/typecheck/slot_resolver.go` | **NEW** — static + dynamic slot resolution |
| `internal/expr/typecheck/checker.go` | **NEW** — `TypeChecker.Check()` entry point, factory |
| `internal/expr/typecheck/compat.go` | **NEW** — `compatible()` and `fixSuggestion()` |
| `internal/expr/daemon/daemon.go` | Add `checker.Check(pr)` call in validate loop |
| `cmd/mxcli/cmd_expr.go` | Same addition in no-daemon path |
