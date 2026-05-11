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

### 7.1 Hint design principle (AI-self-contained)

The shipped `mxcli` is consumed by an AI Agent that has **no access** to:
- mxcli source code
- the mined grammar tables (`mined.go`)
- internal AST type names (`BinExpr`, `IfStmt.Condition`, etc.)

The AI **does** have:
- Mendix domain vocabulary from training (Enumeration, attribute, microflow)
- the MDL it just generated (in its own context)
- the hint output

Therefore every `Hint` field that crosses the boundary to AI must use
**MDL-source vocabulary or Mendix-concept vocabulary**. Internal SlotPath
strings (`IfStmt.Condition`) are translated to user-facing context
(`in IF condition`) before emission. Internal AST type names never leak.

### 7.2 Hint code stability table

Each hint carries a stable code (E0xx) and a descriptive slug. AI prompts
can reliably grep on either. New codes are append-only; never re-used.

| Code | Slug | Trigger | Severity |
|------|------|---------|----------|
| E001 | `enum-string-mismatch` | Enumeration slot/comparison receives string literal | error |
| E002 | `bool-string-mismatch` | Boolean slot receives `'true'`/`'false'` string | error |
| E003 | `null-to-empty` | `null` keyword used where `empty` expected | warning |
| E004 | `concat-type` | `+` operands of incompatible kinds | error |
| E005 | `func-arg-type` | Built-in function arg has wrong kind | error |
| E006 | `func-arg-arity` | Built-in function call has wrong arg count | error |
| E007 | `unknown-token` | Max-match recovery triggered | warning |
| E008 | `enum-missing-module` | `Enum.Value` written without module prefix | error |
| E009 | `slot-type-mismatch` | Generic slot kind mismatch (catch-all) | error |
| E010 | `attribute-not-found` | `$x/Attr` where Attr doesn't exist on entity | error |

The catalog of codes is documented in `docs/06-mdl-reference/expr-hints.md`
(generated). `mxcli help hint <code>` (see §11.5) prints the same content
on demand.

### 7.3 Types

```go
// hint.go — fields named for the AI's mental model, not internal AST
type Hint struct {
    Code     string         // e.g., "E001"
    Slug     string         // e.g., "enum-string-mismatch"
    Severity Severity       // info / warning / error

    Where    HintLocation   // user-facing location

    YouWrote string         // verbatim source line(s) — exactly what the AI sent
    Problem  string         // one-paragraph explanation in Mendix vocabulary
    Fix      string         // corrected MDL fragment, ready to paste

    Reference *HintReference // optional: enum values, function signature, etc.
}

// HintLocation uses MDL/Mendix words, not Go-AST class names.
type HintLocation struct {
    File      string         // .mdl filename (or "<exec>" when running exec on string)
    Line      int
    Column    int
    Microflow string         // qualified microflow name
    Context   string         // e.g., "IF condition", "RETURN value",
                             //       "argument of length()", "Status field of CHANGE"
}

// HintReference holds optional context: enum values, function signature,
// attribute type, etc. Always serialises with self-explanatory keys.
type HintReference struct {
    Enum             string   `json:"enum,omitempty"`
    EnumValues       []string `json:"values,omitempty"`
    FunctionName     string   `json:"function,omitempty"`
    FunctionArgs     []string `json:"function_args,omitempty"`     // textual descriptions
    FunctionReturns  string   `json:"function_returns,omitempty"`
    AttributeName    string   `json:"attribute,omitempty"`
    AttributeType    string   `json:"attribute_type,omitempty"`
    EntityType       string   `json:"entity_type,omitempty"`
}

// SlotPath remains internal to the parser/checker. Translated to
// HintLocation.Context via slot_to_context.go before emission.
type slotPath = string  // package-private, never exposed in Hint output

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
| **F1** | `mxcli exec` | Self-contained HINT blocks per bad expression (text mode default; `--hint-format json` for machine) | AI fix loop |
| **F2** | `mxcli check [--references] [--format text\|json\|sarif]` | Same hint payload as `linter.Violation` (with `Code`, `Slug`, `Problem`, `Fix`, `Reference`) | AI fix loop, no MPR side effect |
| **F3** | F1/F2 sub-capability | Max-match: partial parse with pinpoint hints; recovered fragments preserved and surfaced via `E007 unknown-token` hints | AI on complex exprs |
| **F4** | `mxcli show expr-slot <slot-context>` | Slot expected kind in plain English + 5 highest-frequency mined sample expressions | AI lookup before write |
| **F5** | `mxcli show functions [name]` | Function table with mined signatures + example calls in MDL | AI lookup before write |
| **F6** | (implicit) `describe → check` | 0 hints guaranteed by Stage 3 self-test | AI safe template copy |
| **F7** | `mxcli check --coverage` | Coverage report: matched / recovered / novel-shape counts | Health audit |
| **F8** | `make mine-exprgrammar MPR=...` | Re-mine on new MPR; show diff vs current grammar | Maintenance |
| **F9** | `mxcli explain expression <text> --in <slot-context>` | Single-shot debug of an expression string | AI debug |
| **F10** | `mxcli help hint <code>` | Static reference page per hint code (E001-E0xx): when, why wrong, how to fix, MDL examples | AI fallback when in-line hint isn't enough |

## 10. Hint Format (concrete examples — AI-self-contained, no internal jargon)

### 10.1 Mode A — JSON (machine-first, `--hint-format json`)

```json
{
  "code": "E001",
  "slug": "enum-string-mismatch",
  "severity": "error",
  "where": {
    "file": "fraud.mdl",
    "line": 36,
    "column": 21,
    "microflow": "FraudDetection.SUB_UpdateAlertStatus",
    "context": "IF condition"
  },
  "you_wrote": "IF $Alert/Status = 'NewAlert' THEN ...",
  "problem": "Comparing an Enumeration attribute against a string literal. In Mendix expressions, enumeration values must be written as Module.Enum.Value, never as a quoted string.",
  "fix": "IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...",
  "reference": {
    "enum": "FraudDetection.AlertStatus",
    "values": ["NewAlert", "Validated", "Processing", "Closed", "Error"]
  }
}
```

### 10.2 Mode B — indented text (default stdout, grep-friendly)

```
HINT [E001 enum-string-mismatch] error
  WHERE:
    fraud.mdl line 36, in IF condition of microflow
    FraudDetection.SUB_UpdateAlertStatus

  YOU WROTE:
    IF $Alert/Status = 'NewAlert' THEN ...

  PROBLEM:
    Comparing an Enumeration attribute against a string literal.
    In Mendix expressions, enumeration values must be written as
    Module.Enum.Value, never as a quoted string.

  FIX:
    IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...

  LEGAL VALUES for FraudDetection.AlertStatus:
    NewAlert, Validated, Processing, Closed, Error
```

### 10.3 Max-match recovery (E007 + cascading)

```
HINT [E007 unknown-token] warning
  WHERE:
    fraud.mdl line 12, in argument of length() in microflow
    FraudDetection.SomeMicroflow

  YOU WROTE:
    SET $msg = 'count=' + length(@@@broken@@@) + ' items';
                                 ^^^^^^^^^^^^^^

  PROBLEM:
    The token '@@@broken@@@' is not a valid Mendix expression. The parser
    skipped to the next ')' to continue parsing. The rest of the line
    parsed successfully but produced additional hints below.

  FIX:
    Replace '@@@broken@@@' with a valid String expression — a string
    literal '...', a variable like $someText, or a function call returning
    String.

HINT [E004 concat-type] error
  WHERE:
    fraud.mdl line 12, in '+' concatenation in microflow
    FraudDetection.SomeMicroflow

  YOU WROTE:
    SET $msg = 'count=' + length(...) + ' items';

  PROBLEM:
    The '+' operator concatenates Strings. length() returns Integer, which
    cannot be concatenated with a String directly.

  FIX:
    SET $msg = 'count=' + toString(length(...)) + ' items';
```

### 10.4 Internal-context translation table

Internal slot names are translated to user-facing context strings before
emission. The complete table lives in
`mdl/exprcheck/slot_to_context.go`:

| Internal SlotPath | User-facing context |
|-------------------|---------------------|
| `IfStmt.Condition` | `IF condition` |
| `WhileStmt.Condition` | `WHILE condition` |
| `ChangeItem.Value` | `<AttributeName> field of CHANGE` |
| `CreateItem.Value` | `<AttributeName> field of CREATE` |
| `ReturnStmt.Value` | `RETURN value` |
| `CallArgument.Value` | `argument <ParamName> of CALL <microflow>` |
| `RetrieveStmt.LimitExpr` | `LIMIT clause` |
| `RetrieveStmt.OffsetExpr` | `OFFSET clause` |
| `LogStmt.Message` | `LOG message` |
| `MfSetStmt.Value` | `right-hand side of SET` |
| `DeclareStmt.InitialValue` | `initial value of DECLARE <var>` |
| `FuncCall.Arg[i]` | `argument <i> of <funcName>()` |

If a user-facing context doesn't fit any pattern, the fallback is the
Mendix-vocabulary statement name plus position
(e.g., `expression in microflow body line N`).

### 10.5 `mxcli help hint <code>` — static reference (F10)

```
$ mxcli help hint E001

HINT CODE E001 — enum-string-mismatch (severity: error)

WHEN THIS APPEARS:
  Your MDL has a comparison or assignment where one side is an
  Enumeration attribute (or Enumeration parameter) and the other side
  is a quoted string literal like 'NewAlert'.

WHY IT'S WRONG:
  Mendix expressions cannot compare an Enumeration value to a String.
  The comparison would always be false at runtime, or trigger CE0109
  in Studio Pro.

HOW TO FIX:
  Replace the string literal with the fully-qualified enumeration
  value:
    'NewAlert'  →  FraudDetection.AlertStatus.NewAlert

EXAMPLES:

  CREATE / CHANGE assignment:
    CHANGE $Alert (Status = 'NewAlert')                                -- wrong
    CHANGE $Alert (Status = FraudDetection.AlertStatus.NewAlert)       -- right

  IF / WHILE / SET expression:
    IF $Alert/Status = 'NewAlert' THEN ...                             -- wrong
    IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...    -- right

  CALL parameter:
    CALL Mf($Status = 'Validated')                                     -- wrong
    CALL Mf($Status = FraudDetection.AlertStatus.Validated)            -- right
```

The `mxcli help hint` content is generated from the same source-of-truth
table that emits hints — guaranteed in sync. Stored under
`docs/06-mdl-reference/expr-hints.md` (also generated).

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

### 11.5 `mxcli help hint <code>` (F10)

Reads `docs/06-mdl-reference/expr-hints.md` (which itself is generated
from the same `HintRegistry` table that emits hints) and prints the
section for the requested code. `cmd/mxcli/cmd_help_hint.go`, < 40 LoC.

A single source of truth — `mdl/exprcheck/hints/registry.go` —
defines for each `Code`:
- `Slug`, `Severity`, `Trigger`, `WhyWrong`, `HowToFix`
- a list of `(wrong, right)` example pairs in MDL syntax

The hint emitter, the `mxcli help hint` command, and the generated
markdown reference all consume this same registry. Drift is impossible
by construction.

## 12. Implementation Phases

| Phase | Deliverable | Branch state |
|-------|-------------|--------------|
| **P0** | `cmd/exprgrammar-mine` walks MPR via `describe`, emits skeleton `mined.go` (counts only) | mining tool runs |
| **P1** | `mdl/exprcheck/` lexer + parser skeleton; load mined tables; F1+F2 for slot-kind mismatch only | hints visible |
| **P2** | Max-match recovery (F3); `RecoveredExpr`; complete `parsePrimary` arms | partial parse works |
| **P3** | `FuncChecker` + `SlotChecker` complete; all mined slots and functions used | full coverage |
| **P4** | Stage 3 round-trip test green over 1637 microflows | acceptance gate passes |
| **P5** | F4/F5/F9 sub-commands | AI lookup utilities ship |
| **P6** | F7 coverage report; F10 `help hint`; modelsdk codegen FieldKind annotation | maintenance polish |

Each phase is self-contained, testable, mergeable in isolation.

## 13. Test Strategy

### 13.1 Standard layers

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

### 13.2 Hint readability self-test (LLM-in-the-loop)

The hint format is a UX surface; correctness is "an AI Agent reads the hint
and produces a fix that passes". Asserted by:

```
mdl/exprcheck/hints/readability_test.go (build tag: -tags=llm-readability)

  for each (Code, sample wrong MDL, expected right MDL) in HintRegistry:
    1. run mxcli check --hint-format json on wrong MDL → capture hint payload
    2. send to a sidecar LLM (configurable: claude-sonnet via API)
       prompt: "You wrote this MDL. Here is the hint output. Produce the
                corrected MDL."
       payload: { wrong_mdl, hint_json }
    3. diff LLM output against expected right MDL
    4. require ≥ 95% match across the registry corpus
```

Requirements for this test to pass = hint is genuinely AI-self-contained.
Failures point to vague `Problem` text, missing `Fix`, or incomplete
`Reference` data. Run nightly (not on every PR — costs API calls), gates
hint format changes.

LLM is pluggable behind `mdl/exprcheck/hints/llm.go` with a
`type Reasoner interface { Fix(wrongMDL, hintJSON string) (string, error) }`
so a deterministic stub fills in for offline / unit-test runs.

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
6. **LLM readability test cost**: the hint readability test (§13.2) calls
   a sidecar LLM. Run cadence — nightly is proposed; monthly may be
   sufficient. Cost analysis pending P5.
7. **HintRegistry hand-curation**: codes E001-E010 are seeded manually with
   trigger / why-wrong / how-to-fix text. New codes added by future PRs need
   the same fields populated; PR template enforces this. Open: whether to
   automate parts of the registry from mined data.

## 15. Out of Scope (explicit)

- LSP diagnostics, REPL hints, code-actions.
- Page expressions, OQL, XPath constraints (existing `expressionToXPath`
  handles XPath separately and is not regressed by this work).
- Java action / JavaScript action body validation.
- Auto-fix or rewriting source MDL.
