# SEM-03 Expression Type Checking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement SEM-03 — detect expression type mismatches (wrong TypeKind for target slot) covering all 41 remaining CE0117 errors in macnica that SEM-07 cannot catch.

**Architecture:** New `internal/expr/typecheck/` package with SOLID interfaces (Inferrer, VarScope, AttrCatalog, FuncReg, SlotResolver). ExprRecord gains 3 new scan-time fields; ParseResult gains AST field; meta.Index gains param/return/varKind maps; daemon calls `typecheck.TypeChecker.Check()` after ValidateSemantic.

**Tech Stack:** Go, `mdl/exprcheck` (AST types), `modelsdk/gen/microflows` + `datatypes` (param types), `go.mongodb.org/mongo-driver/bson` (scan), `github.com/mendixlabs/mxcli/internal/expr/meta` (index).

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `mdl/exprcheck/func_checker.go` | Modify | Add 10 DateTime extraction functions + export `FuncReturnKind()` |
| `internal/expr/scan/scan.go` | Modify | Add `TargetAttrQN`, `CalleeQN`, `ParamName` to `ExprRecord`; populate in `scanObj` |
| `internal/expr/parse/parse.go` | Modify | Add `AST exprcheck.RobustExpr` to `ParseResult`; capture from parser |
| `internal/expr/meta/index.go` | Modify | Add `mfParamKinds`, `mfReturnKinds`, `mfVarKinds` maps; populate in `buildMicroflowVars` |
| `internal/expr/meta/catalog_reader.go` | Modify | Add `VarTypeKind`, `MicroflowParamKind`, `MicroflowReturnKind` |
| `internal/expr/meta/mock_index.go` | Modify | Implement 3 new `IndexReader` methods |
| `internal/expr/validate/validate_sem.go` | Modify | Add 3 new methods to `IndexReader` interface |
| `internal/expr/typecheck/interfaces.go` | **NEW** | All 5 interfaces + `ValidationResult` type alias |
| `internal/expr/typecheck/compat.go` | **NEW** | `compatible()` rules + `fixSuggestion()` + `kindName()` |
| `internal/expr/typecheck/funcreg.go` | **NEW** | `defaultFuncReg` wrapping `exprcheck.FuncReturnKind` |
| `internal/expr/typecheck/inferrer.go` | **NEW** | `defaultInferrer` AST walker |
| `internal/expr/typecheck/slot_resolver.go` | **NEW** | Static slots map + 4 dynamic resolver functions |
| `internal/expr/typecheck/checker.go` | **NEW** | `TypeChecker.Check()` + `NewChecker()` factory |
| `internal/expr/typecheck/typecheck_test.go` | **NEW** | Unit tests for all typecheck components |
| `internal/expr/daemon/daemon.go` | Modify | Call `checker.Check(pr)` in validate loop |
| `cmd/mxcli/cmd_expr.go` | Modify | Same in no-daemon path |

---

## Task 1: Add DateTime Extraction Functions + Export FuncReturnKind

**Files:**
- Modify: `mdl/exprcheck/func_checker.go`

The `funcTable` is package-private, so we need an exported function `FuncReturnKind(name string) (TypeKind, bool)` that lets the typecheck package look up return types. We also add the 10 missing DateTime extraction functions.

- [ ] **Step 1: Add DateTime extraction functions to funcTable**

In `mdl/exprcheck/func_checker.go`, find the line `// DateTime — trim-to (UTC calendar)` group and add a new group after all existing DateTime entries (before the closing `}`):

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

- [ ] **Step 2: Add exported FuncReturnKind function**

Add after the existing `PublicFuncTable()` function:

```go
// FuncReturnKind returns the TypeKind of the return value of a named built-in
// function. Returns (KindUnknown, false) for unknown functions.
func FuncReturnKind(name string) (TypeKind, bool) {
	sig, ok := funcTable[name]
	if !ok {
		return KindUnknown, false
	}
	return sig.ret, true
}
```

- [ ] **Step 3: Write tests for new functions**

In `mdl/exprcheck/func_checker_test.go`, add to the existing test file:

```go
func TestFuncReturnKind_DateTimeExtraction(t *testing.T) {
	intFuncs := []string{"year", "month", "dayOfYear", "dayOfMonth",
		"weekOfYear", "dayOfWeek", "hour", "minute", "second", "millisecond"}
	for _, name := range intFuncs {
		k, ok := FuncReturnKind(name)
		if !ok {
			t.Errorf("FuncReturnKind(%q): not found", name)
			continue
		}
		if k != KindInteger {
			t.Errorf("FuncReturnKind(%q) = %v, want KindInteger", name, k)
		}
	}
}

func TestFuncReturnKind_Unknown(t *testing.T) {
	_, ok := FuncReturnKind("nonExistentFunction")
	if ok {
		t.Error("expected false for unknown function")
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./mdl/exprcheck/... -run TestFuncReturnKind -v
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add mdl/exprcheck/func_checker.go mdl/exprcheck/func_checker_test.go
git commit -m "feat(exprcheck): add DateTime extraction functions + FuncReturnKind export"
```

---

## Task 2: Extend ExprRecord with Type-Checking Context Fields

**Files:**
- Modify: `internal/expr/scan/scan.go`
- Modify: `internal/expr/scan/scan_test.go`

`scanObj` reads BSON via `bson.M` map access. For ChangeActionItem nodes, we read the additional `"Attribute"` field (target attribute 3-part QN). For MicroflowCallParameterMapping nodes, we read `"Microflow"` (callee QN) and `"Parameter"` (param name).

- [ ] **Step 1: Add fields to ExprRecord**

In `internal/expr/scan/scan.go`, add after the existing `UnitPath string` field:

```go
	// Type-checking context — populated only for specific UnitTypes; empty otherwise.
	TargetAttrQN string // Microflows$ChangeActionItem: target attribute "Module.Entity.AttrName"
	CalleeQN     string // *MicroflowCallParameterMapping: called microflow "Module.MFName"
	ParamName    string // *MicroflowCallParameterMapping: parameter name
```

- [ ] **Step 2: Populate fields in scanObj**

Replace the inner loop in `scanObj` (the section that creates `ExprRecord`) to also capture the new fields. Find the block that appends to `*out` and replace it:

```go
				for _, field := range fields {
					raw, _ := val[field].(string)
					raw = strings.TrimSpace(raw)
					if raw == "" || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
						continue
					}
					rec := ExprRecord{
						UnitID:   uid,
						Project:  project,
						UnitType: unitType,
						Field:    field,
						Raw:      raw,
						Category: categoryOf(unitType),
						UnitPath: relPath,
					}
					// Populate type-checking context fields.
					switch unitType {
					case "Microflows$ChangeActionItem":
						rec.TargetAttrQN, _ = val["Attribute"].(string)
					case "Microflows$MicroflowCallParameterMapping",
						"Mappings$MicroflowCallParameterMappingImpl",
						"Workflows$MicroflowCallParameterMapping":
						rec.CalleeQN, _ = val["Microflow"].(string)
						rec.ParamName, _ = val["Parameter"].(string)
					}
					*out = append(*out, rec)
				}
```

- [ ] **Step 3: Write test**

In `internal/expr/scan/scan_test.go`, add a new test (the test file already exists):

```go
func TestScanMprcontents_TypeCheckFields(t *testing.T) {
	// Verify TargetAttrQN is populated for ChangeActionItem expressions.
	recs, err := ScanMprcontents(macnicaMprcontentsPath(), Options{FilterType: "ChangeActionItem"})
	if err != nil {
		t.Fatal(err)
	}
	// At least some ChangeActionItem records should have TargetAttrQN set.
	withAttr := 0
	for _, r := range recs {
		if r.TargetAttrQN != "" {
			withAttr++
		}
	}
	if withAttr == 0 {
		t.Error("expected at least one ChangeActionItem with TargetAttrQN populated")
	}
}
```

Note: `macnicaMprcontentsPath()` is a helper already used in existing scan tests — use whichever helper the test file already defines, or define:
```go
func macnicaMprcontentsPath() string {
	return "/mnt/data_sdd/macnica/mendix-app/mprcontents"
}
```

- [ ] **Step 4: Run tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/scan/... -short -v
```

Expected: All PASS (including new test if MPR path is accessible; skip if not).

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/scan/scan.go internal/expr/scan/scan_test.go
git commit -m "feat(scan): add TargetAttrQN, CalleeQN, ParamName to ExprRecord"
```

---

## Task 3: Expose AST in ParseResult

**Files:**
- Modify: `internal/expr/parse/parse.go`

The parser `exprParser.Parse(raw, ctx)` already returns `(exprcheck.RobustExpr, []hints.Hint)` but the first return value is discarded with `_`. We store it in `ParseResult.AST`. XPath expressions use the ANTLR visitor and do not produce an exprcheck AST — their `AST` field stays nil.

- [ ] **Step 1: Add AST field to ParseResult**

In `internal/expr/parse/parse.go`, modify the `ParseResult` struct:

```go
type ParseResult struct {
	Record scan.ExprRecord
	OK     bool
	Hints  []hints.Hint
	AST    exprcheck.RobustExpr // nil for XPath expressions; non-nil for all others
}
```

- [ ] **Step 2: Capture AST in parseExprWithCatalog**

Find the two call sites that use `_, hs := exprParser.Parse(rec.Raw, ctx)` and change them to capture the AST. There are two functions: `parseExpr` (internal, short version) and `parseExprWithCatalog`. Update both:

```go
// In parseExpr (the non-catalog version used by ParseExpression):
ast, hs := exprParser.Parse(rec.Raw, ctx)
// ...
return ParseResult{Record: rec, OK: ok, Hints: hs, AST: ast}

// In parseExprWithCatalog:
ast, hs := exprParser.Parse(rec.Raw, ctx)
// ...
return ParseResult{Record: rec, OK: ok, Hints: hs, AST: ast}
```

XPath path (the `isXPathExpression` branch) already returns early with `ParseResult{Record: rec, OK: true}` — `AST` will be nil (zero value), which is correct.

- [ ] **Step 3: Verify existing tests still pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/parse/... -short -v
```

Expected: All PASS (no behavior change, just storing what was already computed).

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/parse/parse.go
git commit -m "feat(parse): expose AST in ParseResult for type-checking"
```

---

## Task 4: Extend meta.Index with Kind Maps

**Files:**
- Modify: `internal/expr/meta/index.go`
- Modify: `internal/expr/meta/catalog_reader.go`
- Modify: `internal/expr/meta/mock_index.go`
- Modify: `internal/expr/validate/validate_sem.go`

Add three new maps to `Index` and populate them in `buildMicroflowVars`. Add new `IndexReader` interface methods.

**Background on existing code:**
- `microflowVars map[string]map[string]string` stores `unitPath → varName → entityQN` (entity-only, for SEM-07)
- `mfReturnType map[string]string` is a local variable in `buildMicroflowVars` storing bare name → entityQN
- `MicroflowParameter.ParameterType()` returns `element.Element` — cast to `*genDt.BooleanType`, `*genDt.StringType`, etc.
- `CreateVariableAction.VariableDataType()` returns a string like `"String"`, `"Integer"`, `"Boolean"`, `"Decimal"`, `"Long"`, `"DateTime"`

- [ ] **Step 1: Add 3 new fields to Index struct**

In `internal/expr/meta/index.go`, add to the `Index` struct:

```go
type Index struct {
	entityAttrs        map[string]map[string]exprcheck.TypeKind
	entityAttrEnumQN   map[string]string
	enumValues         map[string][]string
	constants          map[string]exprcheck.TypeKind
	assocEndpoints     map[string]AssocMeta
	entityByID         map[string]string
	microflowVars      map[string]map[string]string
	incompleteEntities map[string]bool
	// NEW — for SEM-03 type checking
	mfVarKinds    map[string]map[string]exprcheck.TypeKind // unitPath → varName → TypeKind
	mfParamKinds  map[string]map[string]exprcheck.TypeKind // bare MF name → paramName → TypeKind
	mfReturnKinds map[string]exprcheck.TypeKind            // bare MF name → return TypeKind
}
```

- [ ] **Step 2: Initialize new maps in BuildFromBackend**

In `BuildFromBackend`, add the new map initializations alongside existing ones:

```go
	idx := &Index{
		// ... existing initializations ...
		mfVarKinds:    make(map[string]map[string]exprcheck.TypeKind),
		mfParamKinds:  make(map[string]map[string]exprcheck.TypeKind),
		mfReturnKinds: make(map[string]exprcheck.TypeKind),
	}
```

- [ ] **Step 3: Add helper to convert datatypes.Element → TypeKind**

Add this helper function in `internal/expr/meta/index.go` (near `entityQNFromElement`):

```go
// elementToKind converts a DataTypes element to its TypeKind.
// Handles both primitive types and ObjectType (entity).
func elementToKind(e element.Element) exprcheck.TypeKind {
	switch e.(type) {
	case *genDt.BooleanType:
		return exprcheck.KindBoolean
	case *genDt.StringType:
		return exprcheck.KindString
	case *genDt.IntegerType:
		return exprcheck.KindInteger
	case *genDt.LongType:
		return exprcheck.KindLong
	case *genDt.DecimalType:
		return exprcheck.KindDecimal
	case *genDt.DateTimeType:
		return exprcheck.KindDateTime
	case *genDt.BinaryType:
		return exprcheck.KindBinary
	case *genDt.ObjectType:
		return exprcheck.KindObject
	case *genDt.ListType:
		return exprcheck.KindList
	case *genDt.EnumerationType:
		return exprcheck.KindEnumeration
	}
	return exprcheck.KindUnknown
}

// dataTypeStringToKind converts CreateVariableAction.VariableDataType() string → TypeKind.
func dataTypeStringToKind(s string) exprcheck.TypeKind {
	switch s {
	case "String":
		return exprcheck.KindString
	case "Integer":
		return exprcheck.KindInteger
	case "Long":
		return exprcheck.KindLong
	case "Decimal":
		return exprcheck.KindDecimal
	case "Boolean":
		return exprcheck.KindBoolean
	case "DateTime":
		return exprcheck.KindDateTime
	case "Binary":
		return exprcheck.KindBinary
	}
	return exprcheck.KindUnknown
}
```

- [ ] **Step 4: Populate mfReturnKinds in Pass 1 of buildMicroflowVars**

The existing Pass 1 in `buildMicroflowVars` collects entity QN return types. Extend it to also collect TypeKind return types (including for non-entity returns like Boolean, String):

```go
	// Pass 1: build microflow return-type index (bare name → entityQN and TypeKind).
	for _, mf := range mfs {
		rt := mf.MicroflowReturnType()
		if rt == nil {
			continue
		}
		kind := elementToKind(rt)
		if kind != exprcheck.KindUnknown {
			idx.mfReturnKinds[mf.Name()] = kind
		}
		// Existing entityQN collection (for SEM-07 MicroflowCall var type)
		if qn := entityQNFromElement(rt); qn != "" {
			mfReturnType[mf.Name()] = qn
		}
	}
```

- [ ] **Step 5: Populate mfParamKinds and mfVarKinds in walkOC**

Extend `walkOC` to also populate the kind maps. Add to the `MicroflowParameter` case in `walkOC`:

```go
		case *genMf.MicroflowParameter:
			// Existing: entity QN for SEM-07
			if qn := entityQNFromElement(o.ParameterType()); qn != "" {
				varMap[o.Name()] = qn
			}
			// NEW: TypeKind for SEM-03
			kind := elementToKind(o.ParameterType())
			if kind != exprcheck.KindUnknown {
				if idx.mfVarKinds[unitPath] == nil {
					idx.mfVarKinds[unitPath] = make(map[string]exprcheck.TypeKind)
				}
				idx.mfVarKinds[unitPath][o.Name()] = kind
			}
```

`walkOC` needs to receive `unitPath` and `idx *Index` as additional parameters. Update its signature:

```go
func walkOC(oc *genMf.MicroflowObjectCollection, varMap map[string]string,
	mfReturnType map[string]string, unitPath string, idx *Index) {
```

Update all call sites accordingly.

Also populate `mfParamKinds` when processing a microflow's parameters (add in `buildMicroflowVars` Pass 2, before calling `walkOC`):

```go
	for _, mf := range mfs {
		unitPath := unitPathFromID(string(mf.ID()))
		varMap := make(map[string]string)
		paramKinds := make(map[string]exprcheck.TypeKind)

		if oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection); ok {
			// Collect param kinds for mfParamKinds (keyed by bare name)
			for _, obj := range oc.ObjectsItems() {
				if param, ok := obj.(*genMf.MicroflowParameter); ok {
					kind := elementToKind(param.ParameterType())
					if kind != exprcheck.KindUnknown {
						paramKinds[param.Name()] = kind
					}
				}
			}
			walkOC(oc, varMap, mfReturnType, unitPath, idx)
			idx.applyImplicitAttrs(oc, varMap)
		}

		if len(paramKinds) > 0 {
			idx.mfParamKinds[mf.Name()] = paramKinds
		}
		if len(varMap) > 0 {
			idx.microflowVars[unitPath] = varMap
		}
	}
```

Also extend `addActionVar` to populate `mfVarKinds` for `CreateObjectAction`, `RetrieveAction`, `MicroflowCallAction`, and `CreateVariableAction`. Pass `unitPath` and `idx` to `addActionVar` as well:

```go
func addActionVar(action element.Element, varMap map[string]string,
	mfReturnType map[string]string, unitPath string, idx *Index) {
	switch a := action.(type) {
	case *genMf.CreateObjectAction:
		n, q := a.OutputVariableName(), a.EntityQualifiedName()
		if n != "" && q != "" {
			varMap[n] = q
			idx.setVarKind(unitPath, n, exprcheck.KindObject)
		}
	case *genMf.RetrieveAction:
		n := a.OutputVariableName()
		if n == "" { return }
		if src, ok := a.RetrieveSource().(*genMf.DatabaseRetrieveSource); ok {
			if q := src.EntityQualifiedName(); q != "" {
				varMap[n] = q
				idx.setVarKind(unitPath, n, exprcheck.KindObject)
			}
		}
	case *genMf.MicroflowCallAction:
		n := a.OutputVariableName()
		if n == "" || !a.UseReturnVariable() { return }
		mc, ok := a.MicroflowCall().(*genMf.MicroflowCall)
		if !ok { return }
		callee := mc.MicroflowQualifiedName()
		if i := strings.LastIndex(callee, "."); i >= 0 { callee = callee[i+1:] }
		if entityQN, ok := mfReturnType[callee]; ok {
			varMap[n] = entityQN
			idx.setVarKind(unitPath, n, exprcheck.KindObject)
		} else if kind, ok := idx.mfReturnKinds[callee]; ok {
			idx.setVarKind(unitPath, n, kind)
		}
	case *genMf.CreateVariableAction:
		n := a.VariableName()
		if n == "" { return }
		kind := dataTypeStringToKind(a.VariableDataType())
		if kind != exprcheck.KindUnknown {
			idx.setVarKind(unitPath, n, kind)
		}
	}
}
```

Add helper:

```go
func (idx *Index) setVarKind(unitPath, varName string, kind exprcheck.TypeKind) {
	if idx.mfVarKinds[unitPath] == nil {
		idx.mfVarKinds[unitPath] = make(map[string]exprcheck.TypeKind)
	}
	idx.mfVarKinds[unitPath][varName] = kind
}
```

- [ ] **Step 6: Add new methods to catalog_reader.go**

In `internal/expr/meta/catalog_reader.go`:

```go
// VarTypeKind returns the TypeKind of a microflow variable.
// unitPath is scan.ExprRecord.UnitPath; varName has no leading $.
// Returns KindUnknown when the type is not tracked.
func (idx *Index) VarTypeKind(unitPath, varName string) exprcheck.TypeKind {
	if m, ok := idx.mfVarKinds[unitPath]; ok {
		if k, ok := m[varName]; ok {
			return k
		}
	}
	return exprcheck.KindUnknown
}

// MicroflowParamKind returns the TypeKind of a named parameter of a microflow.
// calleeQN is the bare microflow name (without module prefix).
func (idx *Index) MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool) {
	// Strip module prefix if present
	name := calleeQN
	if i := strings.LastIndex(calleeQN, "."); i >= 0 {
		name = calleeQN[i+1:]
	}
	params, ok := idx.mfParamKinds[name]
	if !ok {
		return exprcheck.KindUnknown, false
	}
	k, ok := params[paramName]
	return k, ok
}

// MicroflowReturnKind returns the TypeKind of a microflow's return value.
// mfName is the bare microflow name (without module prefix).
func (idx *Index) MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool) {
	name := mfName
	if i := strings.LastIndex(mfName, "."); i >= 0 {
		name = mfName[i+1:]
	}
	k, ok := idx.mfReturnKinds[name]
	return k, ok
}
```

Add `"strings"` import if not already present.

- [ ] **Step 7: Add 3 new methods to IndexReader interface in validate_sem.go**

In `internal/expr/validate/validate_sem.go`, add to the `IndexReader` interface:

```go
type IndexReader interface {
	// ... existing methods ...
	VarTypeKind(unitPath, varName string) exprcheck.TypeKind
	MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool)
	MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool)
}
```

- [ ] **Step 8: Implement new methods in mock_index.go**

In `internal/expr/meta/mock_index.go`:

```go
func (m *MockIndex) VarTypeKind(_, _ string) exprcheck.TypeKind { return exprcheck.KindUnknown }
func (m *MockIndex) MicroflowParamKind(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}
func (m *MockIndex) MicroflowReturnKind(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}
```

- [ ] **Step 9: Build and run all existing tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./... 2>&1
GOFLAGS=-mod=mod go test ./internal/expr/... -short -timeout 120s 2>&1 | grep -E "^(--- FAIL|FAIL|ok )"
```

Expected: All PASS (we extended, not changed behavior).

- [ ] **Step 10: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/meta/index.go internal/expr/meta/catalog_reader.go \
        internal/expr/meta/mock_index.go internal/expr/validate/validate_sem.go
git commit -m "feat(meta): add mfVarKinds/mfParamKinds/mfReturnKinds + IndexReader interface"
```

---

## Task 5: New typecheck Package — Interfaces, Compat, FuncReg

**Files:**
- Create: `internal/expr/typecheck/interfaces.go`
- Create: `internal/expr/typecheck/compat.go`
- Create: `internal/expr/typecheck/funcreg.go`
- Create: `internal/expr/typecheck/typecheck_test.go`

- [ ] **Step 1: Create interfaces.go**

```go
// SPDX-License-Identifier: Apache-2.0

// Package typecheck implements SEM-03: expression type-mismatch detection.
// It checks that every expression's inferred TypeKind matches the TypeKind
// expected by the slot it fills (variable assignment, function argument, etc.).
package typecheck

import (
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// Inferrer infers the TypeKind of a parsed AST node.
// S: infers only; does not report errors.
// O: new AST node types handled inside Infer, not in callers.
type Inferrer interface {
	Infer(node exprcheck.RobustExpr, scope VarScope, cat AttrCatalog, funcs FuncReg) exprcheck.TypeKind
}

// VarScope resolves a microflow variable name to its TypeKind.
// I: single method, single purpose.
type VarScope interface {
	TypeOf(varName string) exprcheck.TypeKind
}

// AttrCatalog resolves entity attribute types and variable entity QNs.
// I: two related methods kept together; no unrelated methods.
type AttrCatalog interface {
	AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool)
	EntityQNOf(varName string) string
}

// FuncReg provides the return TypeKind of a named built-in function.
// O: new functions registered in the implementation; callers unchanged.
type FuncReg interface {
	ReturnType(funcName string) (exprcheck.TypeKind, bool)
}

// SlotResolver resolves the expected TypeKind for an expression slot.
// S: expected-type resolution is fully separate from actual-type inference.
type SlotResolver interface {
	Expect(rec scan.ExprRecord, cat AttrCatalog) (exprcheck.TypeKind, bool)
}

// IndexReader is the subset of meta.IndexReader required by typecheck.
// D: typecheck depends on this abstraction, not on meta.Index directly.
type IndexReader interface {
	AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool)
	VarTypeKind(unitPath, varName string) exprcheck.TypeKind
	VarEntityQN(unitPath, varName string) string
	MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool)
	MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool)
}

// Result is a type alias so callers use the shared ValidationResult type.
type Result = validate.ValidationResult
```

- [ ] **Step 2: Create compat.go**

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

// compatible reports whether an expression of type actual can fill a slot
// expecting expected. Returns true (skip) when either side is KindUnknown —
// we prefer false negatives over false positives.
func compatible(actual, expected exprcheck.TypeKind) bool {
	if actual == exprcheck.KindUnknown || expected == exprcheck.KindUnknown {
		return true // can't determine; skip
	}
	if actual == exprcheck.KindAny || expected == exprcheck.KindAny {
		return true
	}
	if actual == expected {
		return true
	}
	// KindEmpty is assignable to object/list slots (null check is caller's responsibility).
	if actual == exprcheck.KindEmpty &&
		(expected == exprcheck.KindObject || expected == exprcheck.KindList) {
		return true
	}
	// Numeric widening: Integer ≤ Long ≤ Decimal.
	return numericWiden(actual, expected)
}

func numericWiden(actual, expected exprcheck.TypeKind) bool {
	rank := map[exprcheck.TypeKind]int{
		exprcheck.KindInteger: 1,
		exprcheck.KindLong:    2,
		exprcheck.KindDecimal: 3,
	}
	ra, oka := rank[actual]
	re, oke := rank[expected]
	return oka && oke && ra <= re
}

var kindDisplayName = map[exprcheck.TypeKind]string{
	exprcheck.KindString:      "String",
	exprcheck.KindInteger:     "Integer",
	exprcheck.KindLong:        "Long",
	exprcheck.KindDecimal:     "Decimal",
	exprcheck.KindBoolean:     "Boolean",
	exprcheck.KindDateTime:    "DateTime",
	exprcheck.KindObject:      "Object",
	exprcheck.KindList:        "List",
	exprcheck.KindBinary:      "Binary",
	exprcheck.KindEnumeration: "Enumeration",
	exprcheck.KindEmpty:       "Empty",
	exprcheck.KindAny:         "Any",
	exprcheck.KindUnknown:     "Unknown",
}

func kindName(k exprcheck.TypeKind) string {
	if s, ok := kindDisplayName[k]; ok {
		return s
	}
	return "Unknown"
}

// fixSuggestion returns a human-readable fix hint for the actual→expected mismatch.
func fixSuggestion(actual, expected exprcheck.TypeKind, rawExpr string) string {
	// Truncate raw expr for display.
	disp := rawExpr
	if len(disp) > 40 {
		disp = disp[:40] + "..."
	}

	isNumeric := func(k exprcheck.TypeKind) bool {
		return k == exprcheck.KindInteger || k == exprcheck.KindLong || k == exprcheck.KindDecimal
	}

	switch {
	case isNumeric(actual) && expected == exprcheck.KindString:
		return "Wrap with toString(): toString(" + disp + ")"
	case actual == exprcheck.KindBoolean && expected == exprcheck.KindString:
		return "Wrap with toString(): toString(" + disp + ")"
	case actual == exprcheck.KindDateTime && expected == exprcheck.KindString:
		return "Wrap with formatDateTime(): formatDateTime(" + disp + ", 'format')"
	case actual == exprcheck.KindString && expected == exprcheck.KindInteger:
		return "Wrap with parseInteger(): parseInteger(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindDecimal:
		return "Wrap with parseDecimal(): parseDecimal(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindBoolean:
		return "Wrap with parseBoolean(): parseBoolean(" + disp + ")"
	case actual == exprcheck.KindString && expected == exprcheck.KindDateTime:
		return "Wrap with parseDateTime(): parseDateTime(" + disp + ", 'format')"
	case actual == exprcheck.KindDateTime && isNumeric(expected):
		return "Extract the needed component: year(" + disp + "), month(" + disp + "), etc."
	case actual == exprcheck.KindEmpty:
		return "Expression may be empty; add a null-check: if " + disp + " = empty then ... else " + disp
	case expected == exprcheck.KindBoolean:
		return "Slot requires a Boolean expression (e.g. a comparison or boolean function call)"
	default:
		return "Expression type (" + kindName(actual) + ") is not compatible with expected type (" + kindName(expected) + ")"
	}
}
```

- [ ] **Step 3: Create funcreg.go**

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

type defaultFuncReg struct{}

// NewFuncReg returns a FuncReg backed by the exprcheck built-in function table.
func NewFuncReg() FuncReg { return &defaultFuncReg{} }

func (r *defaultFuncReg) ReturnType(name string) (exprcheck.TypeKind, bool) {
	return exprcheck.FuncReturnKind(name)
}
```

- [ ] **Step 4: Write unit tests for compat and funcreg**

Create `internal/expr/typecheck/typecheck_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// ── compatible() tests ────────────────────────────────────────────────────────

func TestCompatible_SameKind(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindString) {
		t.Error("same kind should be compatible")
	}
}

func TestCompatible_UnknownSkips(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindUnknown, exprcheck.KindString) {
		t.Error("KindUnknown actual should skip (return true)")
	}
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindUnknown) {
		t.Error("KindUnknown expected should skip (return true)")
	}
}

func TestCompatible_NumericWidening(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindLong) {
		t.Error("Integer should widen to Long")
	}
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindDecimal) {
		t.Error("Integer should widen to Decimal")
	}
	if typecheck.Compatible(exprcheck.KindDecimal, exprcheck.KindInteger) {
		t.Error("Decimal should NOT narrow to Integer")
	}
}

func TestCompatible_EmptyToObject(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindObject) {
		t.Error("Empty should be compatible with Object slot")
	}
	if typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindString) {
		t.Error("Empty should NOT be compatible with String slot")
	}
}

func TestCompatible_StringVsInteger(t *testing.T) {
	if typecheck.Compatible(exprcheck.KindString, exprcheck.KindInteger) {
		t.Error("String should NOT be compatible with Integer slot")
	}
}

// ── FuncReg tests ─────────────────────────────────────────────────────────────

func TestFuncReg_DateTimeExtractionFunctions(t *testing.T) {
	reg := typecheck.NewFuncReg()
	fns := []string{"year", "month", "dayOfYear", "dayOfMonth",
		"weekOfYear", "dayOfWeek", "hour", "minute", "second", "millisecond"}
	for _, name := range fns {
		k, ok := reg.ReturnType(name)
		if !ok {
			t.Errorf("ReturnType(%q): not found", name)
			continue
		}
		if k != exprcheck.KindInteger {
			t.Errorf("ReturnType(%q) = %v, want KindInteger", name, k)
		}
	}
}

func TestFuncReg_KnownStringFunctions(t *testing.T) {
	reg := typecheck.NewFuncReg()
	cases := map[string]exprcheck.TypeKind{
		"toString":    exprcheck.KindString,
		"toLowerCase": exprcheck.KindString,
		"length":      exprcheck.KindInteger,
		"contains":    exprcheck.KindBoolean,
		"round":       exprcheck.KindDecimal,
		"addDays":     exprcheck.KindDateTime,
	}
	for name, want := range cases {
		got, ok := reg.ReturnType(name)
		if !ok {
			t.Errorf("ReturnType(%q): not found", name)
			continue
		}
		if got != want {
			t.Errorf("ReturnType(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestFuncReg_Unknown(t *testing.T) {
	reg := typecheck.NewFuncReg()
	_, ok := reg.ReturnType("notAFunction")
	if ok {
		t.Error("expected false for unknown function")
	}
}
```

Note: `typecheck.Compatible` and `typecheck.FixSuggestion` need to be exported (capitalize). Update `compat.go` to export them:
- `compatible` → `Compatible`
- `fixSuggestion` → `FixSuggestion`
- `kindName` → `KindName` (or keep unexported, used only internally)

- [ ] **Step 5: Run tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/typecheck/... -v
```

Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/
git commit -m "feat(typecheck): interfaces, compat rules, FuncReg"
```

---

## Task 6: TypeInferrer — AST Walker

**Files:**
- Create: `internal/expr/typecheck/inferrer.go`
- Modify: `internal/expr/typecheck/typecheck_test.go`

- [ ] **Step 1: Create inferrer.go**

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

type defaultInferrer struct{}

// NewInferrer returns the default AST-walking TypeKind inferrer.
func NewInferrer() Inferrer { return &defaultInferrer{} }

func (d *defaultInferrer) Infer(
	node exprcheck.RobustExpr,
	scope VarScope,
	cat AttrCatalog,
	funcs FuncReg,
) exprcheck.TypeKind {
	if node == nil {
		return exprcheck.KindUnknown
	}
	switch n := node.(type) {
	case *exprcheck.StringLit:
		return exprcheck.KindString
	case *exprcheck.NumberLit:
		return n.Kind // already KindInteger or KindDecimal from parser
	case *exprcheck.BoolLit:
		return exprcheck.KindBoolean
	case *exprcheck.EmptyExpr:
		return exprcheck.KindEmpty
	case *exprcheck.ConstantRef:
		return exprcheck.KindUnknown // handled by SEM-05
	case *exprcheck.VariableExpr:
		return scope.TypeOf(n.Name)
	case *exprcheck.AttributePathExpr:
		return inferAttrPath(n, scope, cat)
	case *exprcheck.CallExpr:
		if k, ok := funcs.ReturnType(n.Name); ok {
			return k
		}
		return exprcheck.KindUnknown
	case *exprcheck.BinExpr:
		return d.inferBinExpr(n, scope, cat, funcs)
	case *exprcheck.UnaryExpr:
		if n.Op == "not" {
			return exprcheck.KindBoolean
		}
		return d.Infer(n.Operand, scope, cat, funcs)
	case *exprcheck.IfThenElseExpr:
		if t := d.Infer(n.Then, scope, cat, funcs); t != exprcheck.KindUnknown {
			return t
		}
		return d.Infer(n.Else, scope, cat, funcs)
	case *exprcheck.TokenExpr:
		return inferToken(n.Token)
	case *exprcheck.ParenExpr:
		return d.Infer(n.Inner, scope, cat, funcs)
	case *exprcheck.RecoveredExpr:
		return exprcheck.KindUnknown // parse failure; skip to avoid false positives
	case *exprcheck.QNameExpr:
		return exprcheck.KindUnknown // enum reference; handled by SEM-04
	}
	return exprcheck.KindUnknown
}

func inferAttrPath(n *exprcheck.AttributePathExpr, scope VarScope, cat AttrCatalog) exprcheck.TypeKind {
	if len(n.Path) == 0 {
		// Bare $var reference — return entity object kind
		k := scope.TypeOf(n.Variable)
		if k != exprcheck.KindUnknown {
			return k
		}
		if cat.EntityQNOf(n.Variable) != "" {
			return exprcheck.KindObject
		}
		return exprcheck.KindUnknown
	}
	// Last segment is the attribute; intermediate segments are associations (unknown target).
	// For single-hop ($var/attr): resolve from entity.
	if len(n.Path) == 1 {
		entityQN := cat.EntityQNOf(n.Variable)
		if entityQN == "" {
			return exprcheck.KindUnknown
		}
		k, ok := cat.AttributeKind(entityQN, n.Path[0])
		if !ok {
			return exprcheck.KindUnknown
		}
		return k
	}
	// Multi-hop ($var/assoc/attr or deeper): can't reliably infer without full
	// association-target tracking — return Unknown to avoid false positives.
	return exprcheck.KindUnknown
}

func (d *defaultInferrer) inferBinExpr(
	n *exprcheck.BinExpr,
	scope VarScope, cat AttrCatalog, funcs FuncReg,
) exprcheck.TypeKind {
	op := strings.ToLower(n.Op)

	// Comparison and logical operators always return Boolean.
	switch op {
	case "=", "!=", "<", ">", "<=", ">=", "and", "or":
		return exprcheck.KindBoolean
	}

	lk := d.Infer(n.L, scope, cat, funcs)
	rk := d.Infer(n.R, scope, cat, funcs)

	switch op {
	case "+":
		if lk == exprcheck.KindString || rk == exprcheck.KindString {
			// String concatenation — mixed types are E004 (already flagged); return Unknown.
			if lk == exprcheck.KindString && rk == exprcheck.KindString {
				return exprcheck.KindString
			}
			return exprcheck.KindUnknown
		}
		return widenNumeric(lk, rk)
	case "-", "*", "/", "div", "mod":
		return widenNumeric(lk, rk)
	}
	return exprcheck.KindUnknown
}

// widenNumeric returns the wider of two numeric TypeKinds.
// Returns KindUnknown if either side is not numeric.
func widenNumeric(a, b exprcheck.TypeKind) exprcheck.TypeKind {
	rank := map[exprcheck.TypeKind]int{
		exprcheck.KindInteger: 1,
		exprcheck.KindLong:    2,
		exprcheck.KindDecimal: 3,
	}
	ra, oka := rank[a]
	rb, okb := rank[b]
	if !oka || !okb {
		return exprcheck.KindUnknown
	}
	if ra >= rb {
		return a
	}
	return b
}

func inferToken(token string) exprcheck.TypeKind {
	switch token {
	case "CurrentDateTime",
		"CurrentBeginOfDay", "CurrentBeginOfWeek",
		"CurrentBeginOfMonth", "CurrentBeginOfYear",
		"CurrentEndOfDay", "CurrentEndOfWeek",
		"CurrentEndOfMonth", "CurrentEndOfYear":
		return exprcheck.KindDateTime
	case "CurrentUser", "CurrentObject":
		return exprcheck.KindObject
	case "True", "False":
		return exprcheck.KindBoolean
	case "Null":
		return exprcheck.KindEmpty
	}
	return exprcheck.KindUnknown
}
```

- [ ] **Step 2: Write inferrer unit tests**

Add to `internal/expr/typecheck/typecheck_test.go`:

```go
// ── Inferrer tests ────────────────────────────────────────────────────────────

type mockScope struct{ kinds map[string]exprcheck.TypeKind }
func (m *mockScope) TypeOf(name string) exprcheck.TypeKind {
	if k, ok := m.kinds[name]; ok { return k }
	return exprcheck.KindUnknown
}

type mockCat struct {
	attrs     map[string]exprcheck.TypeKind // "Entity.Attr" → kind
	entityQNs map[string]string              // varName → entityQN
}
func (m *mockCat) AttributeKind(entityQN, attr string) (exprcheck.TypeKind, bool) {
	k, ok := m.attrs[entityQN+"."+attr]
	return k, ok
}
func (m *mockCat) EntityQNOf(varName string) string { return m.entityQNs[varName] }

func newInferrer() typecheck.Inferrer { return typecheck.NewInferrer() }

func TestInferrer_Literals(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	// Parse a few literal expressions to get AST nodes.
	// We test via the parser to get real AST nodes.
	tests := []struct {
		raw  string
		want exprcheck.TypeKind
	}{
		{"'hello'", exprcheck.KindString},
		{"42", exprcheck.KindInteger},
		{"3.14", exprcheck.KindDecimal},
		{"true", exprcheck.KindBoolean},
		{"false", exprcheck.KindBoolean},
		{"empty", exprcheck.KindEmpty},
	}

	parser := exprcheck.NewParser()
	for _, tt := range tests {
		ast, _ := parser.Parse(tt.raw, exprcheck.Context{})
		got := inf.Infer(ast, scope, cat, funcs)
		if got != tt.want {
			t.Errorf("Infer(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestInferrer_VariableExpr(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{kinds: map[string]exprcheck.TypeKind{"MyVar": exprcheck.KindString}}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("$MyVar", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindString {
		t.Errorf("Infer($MyVar) = %v, want KindString", got)
	}
}

func TestInferrer_FunctionCall(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	tests := []struct {
		raw  string
		want exprcheck.TypeKind
	}{
		{"year($d)", exprcheck.KindInteger},
		{"toString(42)", exprcheck.KindString},
		{"length('abc')", exprcheck.KindInteger},
		{"contains('abc', 'a')", exprcheck.KindBoolean},
	}
	for _, tt := range tests {
		ast, _ := parser.Parse(tt.raw, exprcheck.Context{})
		got := inf.Infer(ast, scope, cat, funcs)
		if got != tt.want {
			t.Errorf("Infer(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestInferrer_AttrPath_SingleHop(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{}
	cat := &mockCat{
		entityQNs: map[string]string{"Entity": "Mod.Entity"},
		attrs:     map[string]exprcheck.TypeKind{"Mod.Entity.Name": exprcheck.KindString},
	}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("$Entity/Name", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindString {
		t.Errorf("Infer($Entity/Name) = %v, want KindString", got)
	}
}

func TestInferrer_BooleanComparison(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("1 = 2", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindBoolean {
		t.Errorf("Infer(1 = 2) = %v, want KindBoolean", got)
	}
}

func TestInferrer_RecoveredExprIsUnknown(t *testing.T) {
	inf := newInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	// A recovered (failed-to-parse) node should return KindUnknown.
	recovered := &exprcheck.RecoveredExpr{}
	got := inf.Infer(recovered, scope, cat, funcs)
	if got != exprcheck.KindUnknown {
		t.Errorf("Infer(RecoveredExpr) = %v, want KindUnknown", got)
	}
}
```

Note: `exprcheck.RecoveredExpr` needs to be accessible from outside the package. Check if it's exported — if the struct is defined as `RecoveredExpr` (capital R), it is.

- [ ] **Step 3: Run inferrer tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/typecheck/... -run TestInferrer -v
```

Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/inferrer.go internal/expr/typecheck/typecheck_test.go
git commit -m "feat(typecheck): defaultInferrer AST walker"
```

---

## Task 7: SlotResolver — Static and Dynamic Slot Expected Types

**Files:**
- Create: `internal/expr/typecheck/slot_resolver.go`
- Modify: `internal/expr/typecheck/typecheck_test.go`

- [ ] **Step 1: Create slot_resolver.go**

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// slotKey builds the lookup key from a record's UnitType and Field.
func slotKey(rec scan.ExprRecord) string {
	return rec.UnitType + "/" + rec.Field
}

// staticSlots maps UnitType/Field → expected TypeKind for slots with fixed expectations.
var staticSlots = map[string]exprcheck.TypeKind{
	"Microflows$ExpressionSplitCondition/Expression": exprcheck.KindBoolean,
	"Microflows$WhileLoopCondition/WhileExpression":  exprcheck.KindBoolean,
	"DomainModels$AccessRule/XPathConstraint":        exprcheck.KindBoolean,
	"Microflows$CustomRange/LimitExpression":         exprcheck.KindInteger,
	"Microflows$CustomRange/OffsetExpression":        exprcheck.KindInteger,
	"Microflows$TemplateParameter/Expression":        exprcheck.KindString,
}

type slotResolverFn func(rec scan.ExprRecord, cat AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool)

// dynamicSlots maps UnitType/Field → resolver function for slots whose expected
// type must be looked up from the index at runtime.
var dynamicSlotKeys = map[string]slotResolverFn{
	"Microflows$ChangeActionItem/Value":                   resolveAttrTarget,
	"Microflows$ChangeVariableAction/Value":               resolveVarTarget,
	"Microflows$CreateVariableAction/InitialValue":        resolveVarTarget,
	"Microflows$MicroflowCallParameterMapping/Argument":       resolveCallArgTarget,
	"Mappings$MicroflowCallParameterMappingImpl/Argument":     resolveCallArgTarget,
	"Workflows$MicroflowCallParameterMapping/Expression":      resolveCallArgTarget,
	"Microflows$EndEvent/ReturnValue":                         resolveMicroflowReturn,
}

type defaultSlotResolver struct{ idx IndexReader }

// NewSlotResolver returns a SlotResolver backed by the given index.
func NewSlotResolver(idx IndexReader) SlotResolver {
	return &defaultSlotResolver{idx: idx}
}

func (r *defaultSlotResolver) Expect(rec scan.ExprRecord, cat AttrCatalog) (exprcheck.TypeKind, bool) {
	key := slotKey(rec)

	// 1. Static slots.
	if k, ok := staticSlots[key]; ok {
		return k, true
	}

	// 2. Dynamic slots.
	if fn, ok := dynamicSlotKeys[key]; ok {
		return fn(rec, cat, r.idx)
	}

	return exprcheck.KindUnknown, false
}

// resolveAttrTarget resolves the expected type from rec.TargetAttrQN.
// TargetAttrQN has the form "Module.Entity.AttrName" (3-part dot-separated).
func resolveAttrTarget(rec scan.ExprRecord, cat AttrCatalog, _ IndexReader) (exprcheck.TypeKind, bool) {
	qn := rec.TargetAttrQN
	if qn == "" {
		return exprcheck.KindUnknown, false
	}
	// Split "Module.Entity.AttrName" → entityQN="Module.Entity", attr="AttrName".
	last := strings.LastIndex(qn, ".")
	if last <= 0 {
		return exprcheck.KindUnknown, false
	}
	entityQN := qn[:last]
	attrName := qn[last+1:]
	return cat.AttributeKind(entityQN, attrName)
}

// resolveVarTarget resolves the expected type from the target variable's declared kind.
// For ChangeVariableAction and CreateVariableAction, the variable being set is the
// ChangeVariableName / VariableName stored in the same BSON node — but at scan time
// we only have the expression, not the variable name. We look up by unitPath and
// check what the expression's producing variable's target kind is.
//
// Limitation: ChangeVariableAction.ChangeVariableName is not captured in ExprRecord.
// We fall back to KindUnknown for now — this slot type requires a future extension
// to also capture the target variable name in ExprRecord.
func resolveVarTarget(_ scan.ExprRecord, _ AttrCatalog, _ IndexReader) (exprcheck.TypeKind, bool) {
	// TODO(SEM-03.1): capture ChangeVariableName/VariableName in ExprRecord and look up
	// from idx.VarTypeKind(unitPath, varName). Currently returns unknown to avoid false positives.
	return exprcheck.KindUnknown, false
}

// resolveCallArgTarget resolves the expected parameter type using rec.CalleeQN and rec.ParamName.
func resolveCallArgTarget(rec scan.ExprRecord, _ AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool) {
	if rec.CalleeQN == "" || rec.ParamName == "" {
		return exprcheck.KindUnknown, false
	}
	return idx.MicroflowParamKind(rec.CalleeQN, rec.ParamName)
}

// resolveMicroflowReturn resolves the expected type for an EndEvent ReturnValue
// using the microflow's declared return type.
func resolveMicroflowReturn(rec scan.ExprRecord, _ AttrCatalog, idx IndexReader) (exprcheck.TypeKind, bool) {
	// The microflow's bare name is in the unitPath: "xx/xx/uuid.mxunit" → look up by unitPath.
	// We stored mfReturnKinds by bare MF name, but at this point we only have unitPath.
	// Use MicroflowReturnKind with the empty string to signal "look up by unitPath".
	// Actually: mfReturnKinds is keyed by bare name; we need the MF name.
	// Since we don't store unitPath→mfName mapping, fall back to KindUnknown.
	// TODO(SEM-03.2): add unitPath → bare MF name lookup to Index.
	_ = rec
	return exprcheck.KindUnknown, false
}
```

**Note on `resolveVarTarget` and `resolveMicroflowReturn`:** Both have known limitations noted in TODO comments. They return `KindUnknown` which means those slots are skipped (no false positives). These can be addressed in a follow-up task (SEM-03.1 and SEM-03.2).

- [ ] **Step 2: Write slot resolver tests**

Add to `internal/expr/typecheck/typecheck_test.go`:

```go
// ── SlotResolver tests ────────────────────────────────────────────────────────

type mockIdx struct {
	paramKinds map[string]map[string]exprcheck.TypeKind
}

func (m *mockIdx) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	// Reuse mockCat logic
	return exprcheck.KindUnknown, false
}
func (m *mockIdx) VarTypeKind(_, _ string) exprcheck.TypeKind { return exprcheck.KindUnknown }
func (m *mockIdx) VarEntityQN(_, _ string) string             { return "" }
func (m *mockIdx) MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool) {
	// Strip module prefix
	name := calleeQN
	if i := strings.LastIndex(calleeQN, "."); i >= 0 {
		name = calleeQN[i+1:]
	}
	if params, ok := m.paramKinds[name]; ok {
		if k, ok2 := params[paramName]; ok2 {
			return k, true
		}
	}
	return exprcheck.KindUnknown, false
}
func (m *mockIdx) MicroflowReturnKind(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func TestSlotResolver_StaticSlots(t *testing.T) {
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	tests := []struct {
		unitType string
		field    string
		want     exprcheck.TypeKind
	}{
		{"Microflows$ExpressionSplitCondition", "Expression", exprcheck.KindBoolean},
		{"Microflows$WhileLoopCondition", "WhileExpression", exprcheck.KindBoolean},
		{"Microflows$TemplateParameter", "Expression", exprcheck.KindString},
		{"Microflows$CustomRange", "LimitExpression", exprcheck.KindInteger},
	}

	for _, tt := range tests {
		rec := scan.ExprRecord{UnitType: tt.unitType, Field: tt.field}
		got, ok := resolver.Expect(rec, cat)
		if !ok {
			t.Errorf("Expect(%s/%s): got !ok", tt.unitType, tt.field)
			continue
		}
		if got != tt.want {
			t.Errorf("Expect(%s/%s) = %v, want %v", tt.unitType, tt.field, got, tt.want)
		}
	}
}

func TestSlotResolver_AttrTarget(t *testing.T) {
	idx := &mockIdx{}
	cat := &mockCat{
		attrs: map[string]exprcheck.TypeKind{
			"WF_Engine.WFInstance.Wf_No": exprcheck.KindString,
		},
	}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{
		UnitType:     "Microflows$ChangeActionItem",
		Field:        "Value",
		TargetAttrQN: "WF_Engine.WFInstance.Wf_No",
	}
	got, ok := resolver.Expect(rec, cat)
	if !ok {
		t.Fatal("expected ok=true for ChangeActionItem with TargetAttrQN")
	}
	if got != exprcheck.KindString {
		t.Errorf("got %v, want KindString", got)
	}
}

func TestSlotResolver_CallArgTarget(t *testing.T) {
	idx := &mockIdx{
		paramKinds: map[string]map[string]exprcheck.TypeKind{
			"SUB_AdvanceStepStatus": {
				"WFInstanceId": exprcheck.KindLong,
			},
		},
	}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{
		UnitType:  "Microflows$MicroflowCallParameterMapping",
		Field:     "Argument",
		CalleeQN:  "WF_Engine.SUB_AdvanceStepStatus",
		ParamName: "WFInstanceId",
	}
	got, ok := resolver.Expect(rec, cat)
	if !ok {
		t.Fatal("expected ok=true for call arg target")
	}
	if got != exprcheck.KindLong {
		t.Errorf("got %v, want KindLong", got)
	}
}

func TestSlotResolver_UnknownSlot(t *testing.T) {
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{UnitType: "SomeUnknown$Type", Field: "Value"}
	_, ok := resolver.Expect(rec, cat)
	if ok {
		t.Error("expected ok=false for unknown slot")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/typecheck/... -v
```

Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/slot_resolver.go internal/expr/typecheck/typecheck_test.go
git commit -m "feat(typecheck): SlotResolver with static and dynamic slot resolution"
```

---

## Task 8: TypeChecker — Orchestrator and Integration Point

**Files:**
- Create: `internal/expr/typecheck/checker.go`

- [ ] **Step 1: Create checker.go**

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"fmt"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// TypeChecker orchestrates Inferrer, SlotResolver, FuncReg, VarScope, and AttrCatalog
// to produce SEM-03 ValidationResults for a single parsed expression.
type TypeChecker struct {
	infer Inferrer
	slots SlotResolver
	funcs FuncReg
	scope varScopeAdapter
	cat   attrCatalogAdapter
}

// NewChecker builds a TypeChecker wired to the provided index.
// The index must implement the IndexReader interface defined in this package.
func NewChecker(idx IndexReader) *TypeChecker {
	return &TypeChecker{
		infer: NewInferrer(),
		slots: NewSlotResolver(idx),
		funcs: NewFuncReg(),
		scope: varScopeAdapter{idx: idx},
		cat:   attrCatalogAdapter{idx: idx},
	}
}

// Check runs SEM-03 type checking on a single ParseResult.
// Returns nil if no type mismatch is found or if checking is skipped
// (AST is nil, expected type unknown, or inferred type unknown).
func (tc *TypeChecker) Check(pr parse.ParseResult) []validate.ValidationResult {
	// Skip XPath expressions — no AST.
	if pr.AST == nil {
		return nil
	}

	// Skip if the checker itself is nil (nil-safe, used in no-daemon mode).
	if tc == nil {
		return nil
	}

	rec := pr.Record

	// Resolve expected TypeKind for this slot.
	expected, ok := tc.slots.Expect(rec, &tc.cat)
	if !ok || expected == exprcheck.KindUnknown {
		return nil
	}

	// Infer actual TypeKind from the AST.
	actual := tc.infer.Infer(pr.AST, &tc.scope, &tc.cat, tc.funcs)
	if actual == exprcheck.KindUnknown {
		return nil // can't determine; skip to avoid false positives
	}

	// Compare.
	if Compatible(actual, expected) {
		return nil
	}

	raw := rec.Raw
	return []validate.ValidationResult{{
		UnitID:   rec.UnitID,
		Project:  rec.Project,
		UnitType: rec.UnitType,
		Field:    rec.Field,
		Raw:      raw,
		RuleID:   "SEM-03",
		Severity: "ERROR",
		Message:  fmt.Sprintf("Expression returns %s but slot expects %s.", KindName(actual), KindName(expected)),
		Fix:      FixSuggestion(actual, expected, raw),
	}}
}

// varScopeAdapter adapts IndexReader to the VarScope interface.
type varScopeAdapter struct{ idx IndexReader }

func (a *varScopeAdapter) TypeOf(varName string) exprcheck.TypeKind {
	// unitPath is carried by the TypeChecker per-check; we need it here.
	// Since TypeOf is called during Infer, we pass unitPath via a per-call closure.
	// Solution: store unitPath in the adapter during each Check() call.
	return a.idx.VarTypeKind(a.unitPath, varName)
}

// Note: varScopeAdapter needs unitPath per-call. Refactor to pass it in Check():
```

Wait — the `VarScope.TypeOf(varName)` doesn't accept `unitPath`. We need to thread `unitPath` through. The cleanest approach: make the adapter stateful, setting `unitPath` at the start of each `Check()` call:

```go
// varScopeAdapter adapts IndexReader to the VarScope interface for a single microflow unit.
// unitPath is set at the start of each Check() call.
type varScopeAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *varScopeAdapter) TypeOf(varName string) exprcheck.TypeKind {
	return a.idx.VarTypeKind(a.unitPath, varName)
}

// attrCatalogAdapter adapts IndexReader to the AttrCatalog interface.
type attrCatalogAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *attrCatalogAdapter) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	return a.idx.AttributeKind(entityQN, attrName)
}

func (a *attrCatalogAdapter) EntityQNOf(varName string) string {
	return a.idx.VarEntityQN(a.unitPath, varName)
}
```

Full `checker.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"fmt"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// TypeChecker orchestrates all typecheck components to produce SEM-03 results.
type TypeChecker struct {
	infer Inferrer
	slots SlotResolver
	funcs FuncReg
	idx   IndexReader
}

// NewChecker builds a TypeChecker wired to the provided index.
func NewChecker(idx IndexReader) *TypeChecker {
	if idx == nil {
		return nil
	}
	return &TypeChecker{
		infer: NewInferrer(),
		slots: NewSlotResolver(idx),
		funcs: NewFuncReg(),
		idx:   idx,
	}
}

// Check runs SEM-03 type checking on a single ParseResult.
func (tc *TypeChecker) Check(pr parse.ParseResult) []validate.ValidationResult {
	if tc == nil || pr.AST == nil {
		return nil
	}

	rec := pr.Record

	// Build per-call adapters with the current unitPath.
	scope := &varScopeAdapter{idx: tc.idx, unitPath: rec.UnitPath}
	cat := &attrCatalogAdapter{idx: tc.idx, unitPath: rec.UnitPath}

	// Resolve expected TypeKind for this slot.
	expected, ok := tc.slots.Expect(rec, cat)
	if !ok || expected == exprcheck.KindUnknown {
		return nil
	}

	// Infer actual TypeKind from the AST.
	actual := tc.infer.Infer(pr.AST, scope, cat, tc.funcs)
	if actual == exprcheck.KindUnknown {
		return nil
	}

	if Compatible(actual, expected) {
		return nil
	}

	return []validate.ValidationResult{{
		UnitID:   rec.UnitID,
		Project:  rec.Project,
		UnitType: rec.UnitType,
		Field:    rec.Field,
		Raw:      rec.Raw,
		RuleID:   "SEM-03",
		Severity: "ERROR",
		Message:  fmt.Sprintf("Expression returns %s but slot expects %s.", KindName(actual), KindName(expected)),
		Fix:      FixSuggestion(actual, expected, rec.Raw),
	}}
}

// varScopeAdapter adapts IndexReader to VarScope for a specific microflow unit.
type varScopeAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *varScopeAdapter) TypeOf(varName string) exprcheck.TypeKind {
	return a.idx.VarTypeKind(a.unitPath, varName)
}

// attrCatalogAdapter adapts IndexReader to AttrCatalog for a specific microflow unit.
type attrCatalogAdapter struct {
	idx      IndexReader
	unitPath string
}

func (a *attrCatalogAdapter) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	return a.idx.AttributeKind(entityQN, attrName)
}

func (a *attrCatalogAdapter) EntityQNOf(varName string) string {
	return a.idx.VarEntityQN(a.unitPath, varName)
}
```

- [ ] **Step 2: Build and run all typecheck tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./internal/expr/typecheck/...
GOFLAGS=-mod=mod go test ./internal/expr/typecheck/... -v
```

Expected: All PASS.

- [ ] **Step 3: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/checker.go
git commit -m "feat(typecheck): TypeChecker orchestrator + adapter wiring"
```

---

## Task 9: Integration — Wire into Daemon and No-Daemon Paths

**Files:**
- Modify: `internal/expr/daemon/daemon.go`
- Modify: `cmd/mxcli/cmd_expr.go`

- [ ] **Step 1: Add typecheck call in daemon.validate()**

In `internal/expr/daemon/daemon.go`, in the `validate()` function, add the import and modify the loop:

Add import:
```go
import (
    // ... existing imports ...
    "github.com/mendixlabs/mxcli/internal/expr/typecheck"
)
```

Modify the loop in `validate()`:
```go
	parseResults := parse.BatchParseWithCatalog(records, d.index)

	checker := typecheck.NewChecker(d.index) // NEW

	wantSev := strings.ToUpper(strings.TrimSpace(req.Severity))
	var items []ValidationItem
	emit := func(vrs []validate.ValidationResult) {
		for _, vr := range vrs {
			if wantSev != "" && vr.Severity != wantSev {
				continue
			}
			items = append(items, ValidationItem{
				UnitID:   vr.UnitID,
				UnitType: vr.UnitType,
				Field:    vr.Field,
				Raw:      vr.Raw,
				RuleID:   vr.RuleID,
				Severity: vr.Severity,
				Message:  vr.Message,
				Fix:      vr.Fix,
			})
		}
	}
	for _, pr := range parseResults {
		emit(validate.ValidateSyntax(pr))
		emit(validate.ValidateSemantic(pr, d.index))
		emit(checker.Check(pr))  // NEW
	}
```

- [ ] **Step 2: Add typecheck call in no-daemon path**

In `cmd/mxcli/cmd_expr.go`, find `runExprValidateNoDaemon` and add the checker:

```go
func runExprValidateNoDaemon(mprcontentsPath string) error {
	// ... existing scan + parse + validate code ...

	checker := typecheck.NewChecker(nil) // nil index → no-op (SEM-03 needs index)

	for _, pr := range parseResults {
		issues = append(issues, validate.ValidateSyntax(pr)...)
		issues = append(issues, validate.ValidateSemantic(pr, nil)...)
		if r := checker.Check(pr); r != nil {
			issues = append(issues, r...)
		}
	}
	// ... rest of function ...
}
```

Add import in `cmd_expr.go`:
```go
import (
    // ... existing ...
    "github.com/mendixlabs/mxcli/internal/expr/typecheck"
)
```

- [ ] **Step 3: Build**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build -o bin/mxcli ./cmd/mxcli/ 2>&1
```

Expected: clean build.

- [ ] **Step 4: Run all unit tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/... ./mdl/exprcheck/... -short -timeout 120s 2>&1 | grep -E "^(--- FAIL|FAIL|ok )"
```

Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/daemon/daemon.go cmd/mxcli/cmd_expr.go
git commit -m "feat(integration): wire typecheck.TypeChecker into validate pipeline"
```

---

## Task 10: Integration Test and Baseline Verification

**Files:**
- Modify: `internal/expr/meta/integration_sem_test.go` (create if not exists)

- [ ] **Step 1: Stop any running daemons**

```bash
/path/to/bin/mxcli expr daemon stop -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr 2>/dev/null || true
/path/to/bin/mxcli expr daemon stop -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" 2>/dev/null || true
```

(Use the actual binary path: `/mnt/data_sdd/gh/mxcli-wt-02/bin/mxcli`)

- [ ] **Step 2: Run macnica daemon validation and check SEM-03 count**

```bash
/mnt/data_sdd/gh/mxcli-wt-02/bin/mxcli expr validate \
  -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
  --format json 2>&1 | python3 -c "
import json,sys
from collections import Counter
data=json.load(sys.stdin)
rules=Counter(i['RuleID'] for i in data)
print('Total:', len(data))
for r,n in rules.most_common():
    print(f'  {n:4d}  {r}')
"
```

Expected (minimum): `SEM-03` count ≥ 10 (likely higher, exact number depends on how many slots are resolvable). `SEM-07` count = 3 (unchanged). No regressions.

- [ ] **Step 3: Run Mx2026AIDay — must be 0 errors**

```bash
/mnt/data_sdd/gh/mxcli-wt-02/bin/mxcli expr validate \
  -p "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr" \
  --format text 2>&1 | tail -3
```

Expected: `Total: 0 issues  ERROR:0  WARNING:0  INFO:0`

If any SEM-03 results appear here, they are false positives that must be fixed (tighten `Compatible()` rules or add `KindUnknown` guards in `Inferrer`).

- [ ] **Step 4: Write integration test**

Create `internal/expr/typecheck/integration_test.go`:

```go
//go:build integration

package typecheck_test

import (
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"
const mx2026MPR = "/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr"

func runSEM03(t *testing.T, mprPath string) []typecheck.Result {
	t.Helper()
	b, err := mprbackend.NewFromPath(mprPath)
	if err != nil { t.Skipf("MPR not accessible: %v", err) }
	idx, err := meta.BuildFromBackend(b)
	if err != nil { t.Fatalf("BuildFromBackend: %v", err) }

	mprcontents := scan.MprContentsPath(mprPath)
	records, err := scan.ScanMprcontents(mprcontents, scan.Options{})
	if err != nil { t.Fatalf("ScanMprcontents: %v", err) }

	parseResults := parse.BatchParseWithCatalog(records, idx)
	checker := typecheck.NewChecker(idx)

	var results []typecheck.Result
	for _, pr := range parseResults {
		results = append(results, checker.Check(pr)...)
	}
	return results
}

func TestSEM03_Macnica_HasDetections(t *testing.T) {
	results := runSEM03(t, macnicaMPR)
	sem03 := 0
	for _, r := range results {
		if r.RuleID == "SEM-03" { sem03++ }
	}
	t.Logf("SEM-03 detections: %d", sem03)
	if sem03 == 0 {
		t.Error("expected at least 1 SEM-03 detection in macnica")
	}
}

func TestSEM03_Mx2026AIDay_NoFalsePositives(t *testing.T) {
	results := runSEM03(t, mx2026MPR)
	for _, r := range results {
		if r.RuleID == "SEM-03" {
			t.Errorf("false positive SEM-03 in Mx2026AIDay: %s on %s [%s]",
				r.Message, r.UnitType, r.Raw)
		}
	}
}
```

- [ ] **Step 5: Run integration tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/typecheck/ -tags integration -v -timeout 300s
```

Expected: Both tests PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/... ./mdl/exprcheck/... -short -timeout 120s 2>&1 | grep -E "^(--- FAIL|FAIL|ok )"
```

Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git add internal/expr/typecheck/integration_test.go
git commit -m "test(typecheck): integration tests for SEM-03 macnica detections + Mx2026AIDay no-FP"
```

---

## Self-Review

**Spec coverage check:**

| Spec Section | Tasks |
|---|---|
| §3 ExprRecord extensions | Task 2 |
| §4 ParseResult AST field | Task 3 |
| §5.2 FuncReg + missing DateTime functions | Task 1 + Task 5 |
| §5.3 TypeInferrer | Task 6 |
| §5.4 SlotResolver | Task 7 |
| §5.1 Interfaces | Task 5 (interfaces.go) |
| §6 meta.Index extensions | Task 4 |
| §7 Integration | Task 9 |
| §8 Error reporting (SEM-03 format, fix suggestions) | Task 5 (compat.go), Task 8 |
| §9 Testing | Task 5–8 unit tests + Task 10 integration |

**Known limitations (documented in TODO comments):**
- `resolveVarTarget`: ChangeVariableAction target variable name not captured in ExprRecord → always KindUnknown. This covers 0 of the 19 "Change variable" CE0117s for now; follow-up: SEM-03.1.
- `resolveMicroflowReturn`: unitPath → bare MF name mapping not in Index → always KindUnknown. Follow-up: SEM-03.2.
- Multi-hop `$var/assoc/attr` paths return KindUnknown in Inferrer (single-hop only supported).

**The core detections that WILL work with this plan:**
- Log message templates (TemplateParameter/Expression → KindString)
- ExpressionSplitCondition (→ KindBoolean)
- ChangeActionItem values (→ attribute type, when TargetAttrQN is set)
- MicroflowCallParameterMapping arguments (→ param kind, when CalleeQN/ParamName set)
