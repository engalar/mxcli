# Proposal: Robust Expression Parser & Checker (MPR-mined)

**Status:** Draft v3 (supersedes prior drafts in this file)
**Date:** 2026-05-11
**Branch:** `feature/expression-checker` (based on `feat/modelsdk-core` + merged
`feature/mpk-template-derivation`)
**Supersedes:** `PROPOSAL_expression_type_checking.md` (2026-05-10) and the
earlier draft v1/v2 of this file (rule-based blacklist approach).

---

## 1. Problem

MDL expressions today flow through ANTLR but their **semantic correctness** is
never verified. Mistakes produce CE errors only when Studio Pro opens the MPR.
The single consumer of `mxcli` is an AI Agent with no Studio Pro feedback loop;
it cannot self-correct.

## 2. Core Insight (what changed)

Earlier drafts approached this as **blacklist of known bad patterns**, with rules
mined from `Mx2026AIDay/mdlsource/*.mdl` (15 hand/AI-written files). That source
is **untrusted** — the .mdl files may themselves contain bugs.

**The truth lives in the MPR.** Anything Studio Pro accepts is, by definition,
legal Mendix. `mxcli describe` reads MPR and emits canonical MDL. Therefore:

> **Ground truth = `mxcli describe` output of every microflow in a working MPR.**
> The Mx2026AIDay MPR contains **1637 microflows**; that is our corpus.

From this corpus we mine:
- **BNF productions** for expressions
- **Slot expectations** (which AST positions accept which `TypeKind`)
- **Function signatures** (every called function with arg/return types)

The checker is then **a robust recursive-descent parser built from the mined
grammar**. Errors are not enumerated; they fall out of failed parses.

## 3. Goals

1. **Two entry points, one engine**: `mxcli check` (static) and `mxcli exec`
   (pre-BSON-write) share the same robust parser.
2. **Mined, not handwritten**: grammar/slot/function tables generated from MPR
   ground truth. Rebuildable as Mendix evolves.
3. **Max-match parsing**: any input returns a tree (with `RecoveredExpr`
   nodes); legal sub-structure preserved alongside hint-flagged regions.
4. **AI-consumable hint format**: every hint includes `slot`, `source`,
   `parsed-tree`, `expect`, optional `values`. Greppable, machine-readable.
5. **Round-trip guarantee**: `describe → check` must produce 0 hints. This is
   the Stage 3 acceptance gate.

## 4. Non-Goals

- LSP / VS Code / REPL UX. AI Agents are the only consumer.
- Page-level / OQL / XPath expressions. Microflow + nanoflow expressions only.
- Auto-fix. Hints are sufficient for the AI to regenerate.
- Replacing ANTLR for statement-level parsing. ANTLR keeps that job.

## 5. Architecture

### 5.1 Pipeline

```
         ┌──────────────────────────────────────────────────────┐
         │ Stage 0  [Robust AST grammar mining — offline]       │
         │   MPR → describe → MDL → ANTLR-AST →                 │
         │   walk Expression slots → cluster →                  │
         │   emit generated/exprgrammar/mined.go                │
         │   (BNF + SlotExpectations + Functions tables)        │
         └──────────────────────────────────────────────────────┘
                                 │
                                 ▼
         ┌──────────────────────────────────────────────────────┐
         │ Stage 1  [MDL ANTLR — unchanged]                     │
         │   .mdl → ANTLR statement parse → ast.* nodes         │
         │   (visitor patched to capture source_text on Expr)   │
         └─────────────────┬────────────────────────────────────┘
                           │
                           ▼
         ┌──────────────────────────────────────────────────────┐
         │ Stage 2  [Robust AST — new]                          │
         │   For each Expression slot:                          │
         │     RobustParse(source_text, slot_path, ctx)         │
         │       → RobustExpr + []Hint                          │
         │   Recursive descent with max-match recovery,         │
         │   hints emitted inline at parse failure points.      │
         └─────────────────┬────────────────────────────────────┘
                           │
        ┌──────────────────┴──────────────────┐
        ▼                                     ▼
┌─────────────────────────┐    ┌──────────────────────────────┐
│ Stage 4a  mxcli check   │    │ Stage 4b  mxcli exec         │
│   hints → Violations    │    │   hints → HINT lines stdout  │
│   exit non-zero on err  │    │   serialize Robust → BSON    │
│   (no MPR side effect)  │    │   skip slot if severity=err  │
└─────────────────────────┘    └──────────────────────────────┘
                                         ▲
                                         │
         ┌──────────────────────────────────────────────────────┐
         │ Stage 3  [Round-trip self-test — CI gate]            │
         │   for each microflow in MPR:                         │
         │     for each Expression slot:                        │
         │       RobustParse(describe-source) → 0 hints required│
         │   Failures: either Stage 0 grammar incomplete, or    │
         │   Stage 2 parser bug.                                │
         └──────────────────────────────────────────────────────┘
```

### 5.2 Layer attribution

| Stage | Tool | Status |
|-------|------|--------|
| 0a. MPR → describe MDL | **MDL ANTLR** (existing `cmd describe`) | reuse |
| 0b. ANTLR walk → expression inventory | **MDL ANTLR + ast walker** | new walker |
| 0c. Cluster → BNF/Slot/Func tables | **Robust AST infra** | new codegen tool |
| 1. MDL statement parse | **MDL ANTLR** | unchanged + 1 visitor patch |
| 2. Expression parse + check | **Robust AST** | new core |
| 3. Round-trip self-test | Robust AST | test harness |
| 4a. mxcli check adapter | linter integration | thin adapter |
| 4b. mxcli exec adapter | flowBuilder integration | thin adapter |

## 6. SOLID Mapping

**Single Responsibility (SRP)**

| Type | Responsibility |
|------|---------------|
| `Miner` (Stage 0) | Walk MPR, emit grammar/slot/func tables. Nothing else. |
| `Lexer` | Tokenize source text, error-tolerant. |
| `Parser` | Recursive descent over BNF; max-match recovery. |
| `SlotChecker` | Apply mined `SlotExpectations` to parsed tree. |
| `FuncChecker` | Apply `Functions` table to `CallExpr` nodes. |
| `Hint` | Carry one diagnostic + suggested fix. |
| `Adapter` (check / exec) | Wire parser to consumer; format hints for output. |

Each component has one reason to change. Mining strategy changes do not touch
the parser. Hint formatting changes do not touch the slot checker.

**Open/Closed (OCP)**

The grammar is **data**, not code. Adding support for a new Mendix construct
means:
- Re-run `make mine-exprgrammar` over an updated MPR → new productions appear
  in `generated/exprgrammar/mined.go`
- Or hand-augment `mdl/exprcheck/grammar/manual.go` for cases the corpus
  doesn't cover

The parser itself is closed for modification. New rules / new functions /
new slot kinds extend without code changes.

**Liskov Substitution (LSP)**

Both adapters consume `Parser.Parse(source, ctx) → (RobustExpr, []Hint)`.
A test double parser can replace the production parser without changes to
either adapter.

**Interface Segregation (ISP)**

```go
// Minimal contract for callers
type Parser interface {
    Parse(source string, ctx Context) (RobustExpr, []Hint)
}

// Slot lookup is a separate, optional capability
type SlotResolver interface {
    Expect(slotPath string) (SlotConstraint, bool)
}

// Catalog access is a third, optional capability (not all callers have it)
type CatalogReader interface {
    AttributeKind(entityQN, attrName string) (TypeKind, bool)
    EnumCases(enumQN string) ([]string, bool)
    MicroflowReturn(qn string) (TypeKind, bool)
    MicroflowParam(qn, paramName string) (TypeKind, bool)
}
```

Adapters compose only the interfaces they need. The check adapter without
`--references` uses `Parser` + `SlotResolver`; with `--references`, also
`CatalogReader`.

**Dependency Inversion (DIP)**

The parser depends on three abstractions: `SlotResolver`, `CatalogReader`,
`HintSink`. The mined grammar is loaded into a concrete `SlotResolver` impl
at startup, but the parser type knows only the interface. Tests inject
in-memory implementations.

### 6.1 Package layout

```
mdl/exprcheck/
├── doc.go
├── parser.go               # Parser (Stage 2 core, recursive descent)
├── lexer.go                # Lexer with error-tolerant tokens
├── ast.go                  # RobustExpr, RecoveredExpr, BinExpr, ...
├── hint.go                 # Hint, Severity, formatter
├── interfaces.go           # Parser, SlotResolver, CatalogReader, HintSink
├── slot_resolver.go        # Concrete impl backed by mined table
├── func_checker.go
├── slot_checker.go
├── recovery.go             # consumeUntilSafe, max-match helpers
├── adapters/
│   ├── check.go            # used by ValidateMicroflow / linter
│   └── exec.go             # used by flowBuilder
└── *_test.go

mdl/exprcheck/grammar/
├── manual.go               # hand-augmented productions (Mendix ref docs)
└── ...                     # any handcrafted tables

generated/exprgrammar/
└── mined.go                # GENERATED (DO NOT EDIT) — Stage 0 output

cmd/exprgrammar-mine/
├── main.go                 # CLI: mxcli internal cmd or standalone tool
├── walker.go               # walk ANTLR-parsed describe MDL
├── cluster.go              # group token sequences into productions
└── emit.go                 # render to mined.go template
```

## 7. Data Model

```go
// hint.go
type Hint struct {
    Severity Severity        // info / warning / error
    SlotPath string          // e.g., "IfStmt.Condition"
    Location SourceLoc       // microflow + line + col
    Source   string          // exact wrong fragment
    Parsed   string          // pretty-printed RobustExpr at point of failure
    Expect   string          // suggested correct fragment
    Values   []string        // optional — legal enum values, etc.
    Reason   string          // one-line diagnostic
    RuleTag  string          // e.g., "slot-mismatch", "func-arity",
                             //       "concat-type", "null-to-empty"
}

// ast.go
type RobustExpr interface{ isRobustExpr() }

type StringLit struct { Value string; Pos SourceLoc }
type NumberLit struct { Value string; Kind TypeKind; Pos SourceLoc }
type BoolLit   struct { Value bool; Pos SourceLoc }
type EmptyExpr struct { Pos SourceLoc }
type VariableExpr struct { Name string; Pos SourceLoc }
type AttributePathExpr struct { Variable string; Path []PathSeg; Pos SourceLoc }
type QNameExpr struct { Module, Name, Sub string; Pos SourceLoc } // 2 or 3 part
type CallExpr struct { Name string; Args []RobustExpr; Pos SourceLoc }
type BinExpr struct { Op string; L, R RobustExpr; Pos SourceLoc }
type UnaryExpr struct { Op string; Operand RobustExpr; Pos SourceLoc }
type ParenExpr struct { Inner RobustExpr; Pos SourceLoc }
type IfThenElseExpr struct { Cond, Then, Else RobustExpr; Pos SourceLoc }
type TokenExpr struct { Token string; Arg string; Pos SourceLoc }
type ConstantRef struct { QName string; Pos SourceLoc }

type RecoveredExpr struct {
    SourceFragment string    // unparsed bytes
    Pos            SourceLoc
    Reason         string    // why recovery triggered here
}

// slot_resolver.go
type SlotConstraint struct {
    Kind        TypeKind          // expected type kind
    ResolveBy   string            // "" | "AttributeOf:Parent" | "MicroflowReturn" | "TargetParameter"
    Frequency   int               // mining count (signal of confidence)
}

type TypeKind int
const (
    KindUnknown TypeKind = iota
    KindAny
    KindBoolean
    KindString
    KindInteger
    KindLong
    KindDecimal
    KindDateTime
    KindBinary
    KindObject       // QualifiedName-bearing
    KindList
    KindEnumeration
    KindEmpty
)

// generated/exprgrammar/mined.go (sketch)
var SlotExpectations = map[string]SlotConstraint{
    "IfStmt.Condition":         {Kind: KindBoolean, Frequency: 891},
    "WhileStmt.Condition":      {Kind: KindBoolean, Frequency: 12},
    "RetrieveStmt.LimitExpr":   {Kind: KindInteger, Frequency: 47},
    "RetrieveStmt.OffsetExpr":  {Kind: KindInteger, Frequency: 8},
    "ChangeItem.Value":         {Kind: KindUnknown, ResolveBy: "AttributeOf:Parent", Frequency: 3812},
    "ReturnStmt.Value":         {Kind: KindUnknown, ResolveBy: "MicroflowReturn", Frequency: 1102},
    "CallArgument.Value":       {Kind: KindUnknown, ResolveBy: "TargetParameter", Frequency: 4203},
    // ... full list mined automatically
}

var Functions = map[string]FunctionSig{
    "length":       {Args: []TypeKind{KindString}, Returns: KindInteger, Mined: 47},
    "toString":     {Args: []TypeKind{KindAny},    Returns: KindString,  Mined: 203},
    "parseInteger": {Args: []TypeKind{KindString}, Returns: KindInteger, Mined: 12},
    // ...
}
```

## 8. Algorithm — Stage 2 (Robust Parser)

```
Parse(source, ctx) → (RobustExpr, []Hint):
  toks = lex(source)               // never throws; produces ErrorToken inline
  return parseExpr(toks, ctx)

parseExpr(toks, ctx):  return parseOr(toks, ctx)

parseOr / parseAnd / parseNot / parseCmp / parseAdd / parseMul: standard
recursive descent over mined precedence; on operator-position non-match,
propagate up.

parsePrimary(toks, ctx):
  switch toks.peek():
    case String           → StringLit + slot/type checks (★)
    case "null"           → EmptyExpr + Hint(null→empty)            (★)
    case Number           → NumberLit
    case "true"/"false"   → BoolLit
    case "empty"          → EmptyExpr
    case "$"              → parseVariable
    case "@"              → parseConstantRef
    case "[%"             → parseToken
    case Ident "."        → parseQName (2- or 3-part)
    case Ident "("        → parseFuncCall (uses Functions table)
    case "("              → parseParen
    case "if"             → parseIfThenElse
    case "not"            → parseNot
    default               → RecoveryAtPrimary (★)

★ Hint emission rules (semantic, applied at the right level):
  - slot/type checks (StringLit / VariableExpr / etc.):
      lookup ctx.ExpectedKind from SlotResolver
      compare against inferred kind of node
      mismatch → Hint{slot-mismatch, expect: <derived from slot kind>}
  - null literal: always Hint{null-to-empty}
  - call: arity + arg type checks via Functions table

RecoveryAtPrimary:
  unknown_tok = toks.consume()
  salvage     = toks.consumeUntilSafeBoundary()
                // safe: ')', ',', ';', 'then', 'else', 'end',
                //       'AND', 'OR', '+', '-', '*', '=', '!=', EOF
  return RecoveredExpr{SourceFragment: unknown_tok+salvage}, Hint{unknown-primary}

Max-match invariant:
  At every binary level, after recovery in one operand, continue parsing the
  rest of the chain. Hints accumulate; AST shape stays maximal.
```

## 9. Functions Delivered (AI-facing)

| ID | Trigger | Output | Audience |
|----|---------|--------|----------|
| **F1** | `mxcli exec` | HINT lines per bad expression with `slot`, `source`, `expect`, `values` | AI fix loop |
| **F2** | `mxcli check [--references]` | Same hints as `linter.Violation` (text/JSON/SARIF) | AI fix loop, no MPR side effect |
| **F3** | F1/F2 sub-capability | Max-match: partial parse with pinpoint hints, recovered fragments preserved | AI on complex exprs |
| **F4** | `mxcli show expr-slot <SlotPath>` | Slot expected `Kind` + sample expressions mined from MPR | AI lookup before write |
| **F5** | `mxcli show functions [name]` | Function table with mined signatures + example calls | AI lookup before write |
| **F6** | (implicit) `describe → check` | 0 hints guaranteed by Stage 3 self-test | AI safe template copy |
| **F7** | `mxcli check --coverage` | Coverage report: matched / recovered / novel-shape counts | Health audit |
| **F8** | `make mine-exprgrammar MPR=...` | Re-mine on new MPR; show diff vs current grammar | Maintenance |
| **F9** | `mxcli explain expression <text> --slot <slotPath>` | Single-shot debug of a literal expression string | AI debug |

## 10. Hint Format (concrete examples)

### F1 — `mxcli exec` stdout

```
HINT [SlotPath=IfStmt.Condition] FraudDetection.SUB_UpdateAlertStatus@line 36:
  source:   $Alert/Status = 'NewAlert'
  parsed:   BinExpr(=, AttributePathExpr($Alert/Status), StringLit('NewAlert'))
            ★ slot expects Boolean; comparison RHS String against LHS Enumeration
  expect:   $Alert/Status = FraudDetection.AlertStatus.NewAlert
  values:   NewAlert | Validated | Processing | Closed | Error
```

### F2 — `mxcli check --format json`

```json
{
  "ruleId":  "EXPR-SLOT-MISMATCH",
  "severity":"error",
  "slotPath":"IfStmt.Condition",
  "location":{"document":"FraudDetection.SUB_UpdateAlertStatus","line":36,"col":5},
  "source":  "$Alert/Status = 'NewAlert'",
  "expect":  "$Alert/Status = FraudDetection.AlertStatus.NewAlert",
  "values":  ["NewAlert","Validated","Processing","Closed","Error"]
}
```

### F3 — Max-match with recovery

```
HINT [parsed-tree-recovery]:
  source:   'count=' + length(@@@broken@@@) + ' items'
  parsed:   BinExpr(+,
              BinExpr(+, StringLit('count='), CallExpr(length, [RecoveredExpr])),
              StringLit(' items'))
  hints:
    - call.arg-1 (length): UNRECOGNIZED '@@@broken@@@' near col 28
        recovered to next safe boundary ')'
        expect: String expression
    - + concat: 'count=' + Integer + ' items'
        Integer needs toString()
        expect: 'count=' + toString(length(...)) + ' items'
```

## 11. Wiring

### 11.1 `mxcli check` (Stage 4a)

`mdl/executor/validate_microflow.go` adds, at the end of `ValidateMicroflow`:

```go
issues := exprcheck.NewCheckAdapter(catalog).
    CheckMicroflow(stmt)
v.violations = append(v.violations, issues.AsViolations()...)
```

`catalog` is nil unless `--references` is set; the adapter degrades gracefully.

### 11.2 `mxcli exec` (Stage 4b)

`flowBuilder.exprToString` is replaced with:

```go
func (fb *flowBuilder) exprToBSON(slotPath string, expr ast.Expression) string {
    src := exprSourceText(expr) // captured by Stage 1 visitor patch
    rExpr, hints := fb.exprAdapter.Parse(src, exprcheck.Context{
        SlotPath:    slotPath,
        Scope:       fb.scope(),
        Catalog:     fb.catalog,
    })
    fb.hintSink.Emit(hints...)
    if hints.HasError() {
        return ""   // skip — AI must fix
    }
    return rExpr.Serialize()  // canonical Mendix expression string
}
```

Existing `expressionToString` becomes `RobustExpr.Serialize()`'s back-end (or
is delegated to during transition).

### 11.3 `cmd/exprgrammar-mine` — Stage 0 generator

```bash
# rebuild grammar from MPR
go run ./cmd/exprgrammar-mine \
    --mpr "/path/to/Factory Management.mpr" \
    --out generated/exprgrammar/mined.go
```

Make target:

```makefile
mine-exprgrammar:
	go run ./cmd/exprgrammar-mine \
	    --mpr "$(MPR)" \
	    --out generated/exprgrammar/mined.go
```

### 11.4 `mxcli show expr-slot` / `show functions` / `explain expression`

Three new sub-commands under `cmd/mxcli/`. Each is < 60 LoC; they read
generated `mined.go` tables and (for `explain`) call the parser directly.

## 12. Implementation Phases

| Phase | Deliverable | Branch state |
|-------|-------------|--------------|
| **P0** | `cmd/exprgrammar-mine` walks MPR via `describe`, emits skeleton `mined.go` (counts only) | mining tool runs |
| **P1** | `mdl/exprcheck/` lexer + parser skeleton; load mined tables; F1+F2 for slot-kind mismatch only | hints visible |
| **P2** | Max-match recovery (F3); `RecoveredExpr`; complete `parsePrimary` arms | partial parse works |
| **P3** | `FuncChecker` + `SlotChecker` complete; all mined slots and functions used | full coverage |
| **P4** | Stage 3 round-trip test green over 1637 microflows | acceptance gate passes |
| **P5** | F4/F5/F9 sub-commands | AI lookup utilities ship |
| **P6** | F7 coverage report; modelsdk codegen FieldKind annotation | maintenance polish |

Each phase is self-contained, testable, mergeable in isolation.

## 13. Test Strategy

- `mdl/exprcheck/*_test.go` — table-driven unit tests per parser arm and
  recovery path.
- `cmd/exprgrammar-mine/*_test.go` — golden-file tests against a small
  fixture MPR (committed under `testdata/`).
- **Stage 3 round-trip CI gate**: `go test -tags=roundtrip` walks the
  Mx2026AIDay MPR (or a smaller embedded fixture), describes every
  microflow, parses every Expression slot, asserts 0 hints. Fails CI on
  regressions.
- Integration: `mdl-examples/expr-checker/` golden-file tests for both
  `mxcli check` and `mxcli exec` paths.

## 14. Open Questions

1. **Mining MPR source**: the Mx2026AIDay MPR is large (1637 microflows) but
   is one Mendix 11.6 project. Should P0 also mine an embedded "system module"
   MPR shipped by Mendix, to capture rarer constructs? Proposal: embed a
   minimal reference MPR under `testdata/` for CI, allow `--mpr` flag for
   richer corpora.
2. **Hint severity defaults**: slot-kind mismatch with catalog confirmation =
   `error`; without catalog (heuristic) = `warning`. `null→empty` =
   `warning` (auto-corrected). Function arity = `error`. Subject to
   refinement during P3.
3. **modelsdk integration**: P6's `FieldKind` annotation lets the exec
   adapter resolve `SlotConstraint.ResolveBy` against the live model when
   the `mdl/catalog` is unavailable. Open: whether to backfill or only
   ship for new domains.
4. **Coverage report novelty threshold**: F7 flags expressions that don't
   match any mined production. What's "novel enough" to warn? Proposal:
   exact AST-shape match required; structural near-misses get `info` level.
5. **`mxcli show expr-slot` granularity**: sample expressions in F4 — show
   raw text or pretty-printed normalised form? Proposal: raw, capped at
   5 examples sorted by frequency.

## 15. Out of Scope (explicit)

- LSP diagnostics, REPL hints, code-actions.
- Page expressions, OQL, XPath constraints (existing `expressionToXPath`
  handles XPath separately and is not regressed by this work).
- Java action / JavaScript action body validation.
- Auto-fix or rewriting source MDL.
