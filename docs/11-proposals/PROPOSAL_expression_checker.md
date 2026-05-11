# Proposal: Expression Checker for `mxcli check` and `mxcli exec`

**Status:** Draft
**Date:** 2026-05-11
**Branch:** `feature/expression-checker` (based on `feat/modelsdk-core`)
**Supersedes/refines:** `PROPOSAL_expression_type_checking.md` (2026-05-10) — refocuses scope to AI-first, exec-time first.

---

## 1. Problem

MDL expressions today are parsed into a structured AST but never semantically checked.
Mistakes pass `mxcli exec` silently, surface as CE errors only when the project is
opened in Studio Pro. The only consumer of this tool is an AI Agent that iteratively
generates MDL: it has no Studio Pro feedback loop, so it cannot self-correct.

The eight error cases below were derived from sampling the `Mx2026AIDay` repository
plus Mendix expression semantics. They are the actual bugs an AI produces.

| ID | Pattern | Sampled | Effect |
|----|---------|---------|--------|
| EX001 | `= null` in expression context | validation.mdl:31,41,60,70 | Auto-corrected to `empty`; source rots |
| EX002 | `Status = 'NewAlert'` in CREATE/CHANGE | create-alert.mdl:43 | CE0109 in Studio Pro |
| EX003 | `IF $x/Status = 'Open'` (enum vs string) | derived | Always-false comparison |
| EX004 | `'text' + $intVar` | polling.mdl:113 | CE0109 type mismatch |
| EX005 | `length($intAttr)` | derived | CE0109 wrong arg type |
| EX006 | `$x/IsActive = 'true'` | derived | Always-false comparison |
| EX007 | `AlertStatus.NewAlert` (no module prefix) | derived | Cannot resolve enum |
| EX008 | `CALL Mf($EnumParam = 'Validated')` | derived | CE param type mismatch |

## 2. Goals

1. **Two checking entry points, one engine**: `mxcli check` (static) and `mxcli exec`
   (pre-BSON-write) share the same `ExprChecker`. No code duplication.
2. **AI-consumable hint format**: every issue includes a `Got` / `Expect` / optional
   `Values` triple that the agent can string-match and apply directly.
3. **Progressive precision**: rules degrade gracefully without scope or catalog
   (pattern-only) and tighten with each context source available.
4. **SOLID-by-construction**: rule set extends without modifying the engine.

## 3. Non-Goals

- LSP / VS Code integration. Human developers are not in scope.
- REPL interactive mode. AI-only consumption.
- Type checking for OQL / XPath / page expressions. Microflow + nanoflow only.
- Auto-fix / code-action. Hint format is sufficient for the AI to regenerate.

## 4. Architecture (SOLID)

### 4.1 Package layout

```
mdl/exprcheck/                          # standalone, no executor or linter dependency
├── doc.go
├── checker.go                          # ExprChecker (orchestrator)
├── interfaces.go                       # Rule, ScopeReader, CatalogReader, IssueSink
├── issue.go                            # Issue + Hint formatting
├── scope.go                            # ExprScope concrete impl
├── walker.go                           # AST walker
├── functable.go                        # built-in function signatures
├── rules/
│   ├── registry.go                     # init()-based rule registration
│   ├── ex001_null_in_expr.go
│   ├── ex002_string_to_enum_assign.go
│   ├── ex003_compare_enum_string.go
│   ├── ex004_string_concat_type.go
│   ├── ex005_function_arg_type.go
│   ├── ex006_boolean_string.go
│   ├── ex007_enum_qualified.go
│   └── ex008_call_enum_arg.go
└── adapters/
    ├── check.go                        # CheckMicroflowExpressions(stmt) → []linter.Violation
    └── exec.go                         # ExecExprWatcher used inside flowBuilder
```

### 4.2 SOLID mapping

**Single Responsibility (SRP)**

| Type | Responsibility |
|------|---------------|
| `ExprChecker` | Drives walker + rule loop; nothing else |
| `Walker` | AST traversal; emits `(node, path)` events |
| `ExprScope` | Variable name → `TypeKind` lookup |
| `Rule` impl | Detects ONE pattern, returns `Issue` |
| `Issue` | Holds rule ID, location, hint triple |
| `HintFormatter` | Renders `Issue` to text or `linter.Violation` |
| `FuncTable` | Built-in function signature lookup |

Each rule file `ex0xx_*.go` is < 80 LoC: AST predicate + hint construction.

**Open/Closed (OCP)**

```go
// rules/registry.go
var registered []exprcheck.Rule

func Register(r exprcheck.Rule) { registered = append(registered, r) }
func All() []exprcheck.Rule    { return registered }

// rules/ex001_null_in_expr.go
func init() { Register(&nullInExprRule{}) }
```

Adding EX009 = new file + `init() { Register(...) }`. Engine untouched.

**Liskov Substitution (LSP)**

All rules implement the SAME `Rule` interface. The walker treats them
uniformly; replacing one with another (or with a no-op test double) cannot
break the engine.

**Interface Segregation (ISP)**

```go
// Core: every rule implements this
type Rule interface {
    ID() string
    Severity() Severity
    Check(node ast.Expression, ctx Context) []Issue
}

// Optional: rule opts in if it needs scope
type ScopeAware interface {
    Rule
    UsesScope() bool   // marker, lets engine skip when scope absent
}

// Optional: rule opts in if it needs catalog
type CatalogAware interface {
    Rule
    UsesCatalog() bool
}
```

A pattern-only rule (e.g. EX001, EX006) does NOT depend on scope or catalog
interfaces — they need only the AST.

**Dependency Inversion (DIP)**

The engine depends on three abstractions:

```go
type ScopeReader interface {
    Lookup(name string) (TypeKind, bool)
}

type CatalogReader interface {
    AttributeType(entityQN, attrName string) (TypeKind, bool)
    EnumCases(enumQN string) ([]string, bool)
    MicroflowReturnType(qn string) (TypeKind, bool)
    MicroflowParamType(qn, paramName string) (TypeKind, bool)
}

type IssueSink interface {
    Emit(Issue)
}
```

`mxcli check` adapter wires a `nopScope`, an optional `catalogReader` (only when
`--references` is set), and a `linter.Violation`-collecting sink.

`mxcli exec` adapter wires the live `flowBuilder` scope, the live catalog
(always available — project is open), and a stdout-printing sink.

Neither adapter knows about specific rules. Rules don't know about either path.

### 4.3 Data types

```go
// issue.go
type Issue struct {
    RuleID   string
    Severity Severity
    Loc      Location          // microflow / activity / position
    Got      string            // exact wrong source fragment
    Expect   string            // suggested correct fragment
    Values   []string          // optional — list of legal enum values, etc.
    Reason   string            // one-line human-readable diagnostic
}

// scope.go
type TypeKind int
const (
    KindUnknown TypeKind = iota
    KindString
    KindInteger
    KindLong
    KindDecimal
    KindBoolean
    KindDateTime
    KindBinary
    KindObject       // QualifiedName held in extra field
    KindList         // element QualifiedName held in extra field
    KindEnumeration  // enum QualifiedName held in extra field
    KindEmpty
)

type ExprScope struct {
    bindings map[string]Binding
    parent   *ExprScope         // for nested loops / branches
}

type Binding struct {
    Kind          TypeKind
    QualifiedName string         // for Object / List / Enumeration
}
```

`ExprScope` is populated by a one-pass walker from `DECLARE`, `PARAMETER`,
`RETRIEVE`, `CREATE OBJECT`, `CALL ... AS $x`, `LOOP $x IN $list`. This
populator lives in `scope.go` and is the only producer of `ExprScope`.

### 4.4 Hint format (AI-consumable)

`exec` adapter prints to stdout:

```
HINT [EX002] FraudDetection.SUB_CreateAlert/CREATE@Status:
  Reason: String literal assigned to Enumeration attribute.
  Got:    Status = 'NewAlert'
  Expect: Status = FraudDetection.AlertStatus.NewAlert
  Values: NewAlert | Validated | Processing | Closed | Error
```

`check` adapter converts each `Issue` to `linter.Violation`:

```go
linter.Violation{
    RuleID:     "EX002",
    Severity:   linter.SeverityError,
    Message:    issue.Reason,
    Location:   ...,
    Suggestion: issue.Expect,   // existing field
}
```

Both consume the same `Issue` produced by rules — no double-rendering.

## 5. Context Sources (precision ladder)

```
Pattern only ─────────► Scope-aware ─────────► Catalog-backed
(no project,            (executor knows         (entity/attr/enum
 no DECLARE info)        DECLARE types)          types resolvable)
```

| Case | Pattern | Scope | Catalog |
|------|---------|-------|---------|
| EX001 | ✅ | — | — |
| EX002 | heuristic | — | ✅ (precise) |
| EX003 | heuristic | partial | ✅ (precise) |
| EX004 | partial (literal+literal) | ✅ (DECLARE-tracked vars) | ✅ |
| EX005 | param count | partial | ✅ |
| EX006 | ✅ | — | — |
| EX007 | ✅ (structural) | — | ✅ (validates enum exists) |
| EX008 | — | — | ✅ |

Pattern-only rules ship in P1 and run in `mxcli check` without project.

## 6. Wiring

### 6.1 `mxcli check` path

`ValidateMicroflow(stmt)` (in `mdl/executor/validate_microflow.go`) appends:

```go
issues := exprcheck.CheckMicroflow(stmt, exprcheck.CheckOptions{
    Catalog: optionalCatalogReader, // nil unless --references
})
v.violations = append(v.violations, exprcheck.ToViolations(issues)...)
```

Single line addition. `ValidateMicroflow` keeps its current responsibility.

### 6.2 `mxcli exec` path

`flowBuilder` gains an embedded `*exprcheck.ExecWatcher`. Replaces the bare
`expressionToString()` call site:

```go
// before
result := fb.exprToString(expr)

// after
result := fb.exprToStringWithCheck(expr, exprcheck.Context{
    Activity: "CREATE",
    Field:    "Status",
    Expected: fieldExpectedKind,
})
```

`exprToStringWithCheck` invokes the watcher (which runs the rules), prints
hints, then delegates to the existing `expressionToString`. No rule blocks
execution; all output is `HINT` lines.

### 6.3 `modelsdk` integration (P4)

`modelsdk/gen/microflows/types.go` declares per-property types. We extend the
codegen template with a `FieldKind` annotation distinguishing
`KindExpression`, `KindEnumRef`, `KindXPath`, `KindPlainString`. The exec
adapter consults this annotation to determine `Expected` automatically per
field, replacing manual mapping.

Example (post-P4):

```go
type AggregateListAction struct {
    expression *property.Primitive[string] // FieldKind: Expression
}
```

The codegen change is small and additive — does not touch generated
constructors.

## 7. Rule Specifications

Each rule file follows the template:

```go
type xxxRule struct{}

func (r *xxxRule) ID() string         { return "EX0xx" }
func (r *xxxRule) Severity() Severity { return SeverityWarning } // or Error

func (r *xxxRule) Check(node ast.Expression, ctx Context) []Issue {
    // 1. shape match (cheap predicate on AST)
    // 2. precision lift if scope/catalog available
    // 3. construct Issue with Got / Expect / Values
}

func init() { Register(&xxxRule{}) }
```

### EX001 — `null` in expression context

Trigger: `LiteralExpr{Kind: LiteralNull}` anywhere except a no-op normalization
context.

```
Got:    $Alert/Status = null
Expect: $Alert/Status = empty
Reason: 'null' is auto-corrected to 'empty' on BSON write; update the source.
```

### EX002 — String literal assigned to Enumeration attribute (CREATE/CHANGE)

Trigger: walking `CreateObjectStmt.Changes` or `ChangeObjectStmt.Changes`,
each `ChangeItem.Value` of kind `LiteralString`.

- Pattern (no catalog): value is a CamelCase identifier-shaped string AND
  attribute name matches enum-likely heuristic (`Status`, `Type`, `Level`,
  ends in `Kind`, etc.) → low-confidence hint.
- Catalog (preferred): resolve `Entity.Attr.Type`. If `Enumeration`, emit
  high-confidence Issue with `Values` populated from `EnumCases`.

```
Got:    Status = 'NewAlert'
Expect: Status = FraudDetection.AlertStatus.NewAlert
Values: NewAlert | Validated | Processing | Closed | Error
```

### EX003 — Comparing enum attribute to string literal in expression

Trigger: `BinaryExpr{Op: "=", "<>"}` where one side is `AttributePathExpr` and
the other is `LiteralExpr{Kind: LiteralString}`.

Pattern: same heuristic as EX002. Catalog: precise.

### EX004 — String concatenation with non-String operand

Trigger: `BinaryExpr{Op: "+", Left: LiteralExpr{Kind: LiteralString}}` or
right-side mirror.

- Pattern: if other side is `LiteralExpr{Kind: Integer/Decimal/Boolean}` →
  hard-flag.
- Scope: if other side is `VariableExpr` and scope says non-String → flag.
- Catalog: if other side is `AttributePathExpr` and resolved type is
  non-String → flag.

```
Got:    'Processed: ' + $ProcessedCount
Expect: 'Processed: ' + toString($ProcessedCount)
```

### EX005 — Function argument type mismatch

Trigger: `FunctionCallExpr` with name in `FuncTable`. Compare argument count
and type.

`FuncTable` (subset):

| Function | Arg types | Returns |
|----------|-----------|---------|
| `length` | (String) | Integer |
| `toString` | (Any) | String |
| `parseInteger` | (String) | Integer |
| `parseDecimal` | (String) | Decimal |
| `trim` / `toUpperCase` / `toLowerCase` / `substring` | (String, ...) | String |
| `find` | (String, String) | Integer |
| `contains` | (String, String) | Boolean |
| `startsWith` / `endsWith` | (String, String) | Boolean |

### EX006 — Boolean string literal

Trigger: `BinaryExpr{Op: "=" or "<>", Right: LiteralExpr{Kind: LiteralString,
Value: "true"|"false"|"True"|"False"}}`.

Pattern only — no scope or catalog needed.

```
Got:    $Config/IsPollingActive = 'true'
Expect: $Config/IsPollingActive = true
```

### EX007 — Enum reference missing module prefix

Trigger: `QualifiedNameExpr` with exactly two parts in a comparison or
assignment context. Heuristic: Module name lowercase resembles enum naming
(`AlertStatus`, ends in `Status`/`Type`/`Kind`/`Level`).

Catalog (preferred): verify the bare 2-part name doesn't resolve as an entity;
if no module exists with that name, suggest the obvious 3-part form by
searching catalog enumerations.

### EX008 — CALL with string for enum parameter

Trigger: `CallMicroflowStmt.Arguments[i].Value` is `LiteralString` and the
target microflow's parameter `i` is an `Enumeration` type per catalog.

Catalog-only rule. Skips silently when no catalog.

## 8. Implementation Phases

| Phase | Content | Branch state |
|-------|---------|--------------|
| **P1** | `ExprChecker` skeleton, `ExprScope`, `Walker`, `Issue`, EX001/EX006/EX007 (pattern-only); `check` adapter + `exec` adapter | first commit |
| **P2** | EX004 with scope-local; populate scope from DECLARE/PARAMETER/RETRIEVE/CREATE/CALL | second commit |
| **P3** | `CatalogReader` interface; EX002/EX003/EX005/EX008 catalog-backed | third commit |
| **P4** | `modelsdk` codegen `FieldKind` annotation; exec adapter uses it to pick `Expected` per field | depends on modelsdk codegen change |

Each phase is independently shippable, mergeable, and reverts cleanly.

## 9. Test Strategy

- `mdl/exprcheck/*_test.go` — table-driven tests per rule with snippets that
  trigger / don't trigger the rule.
- `mdl-examples/expr-checker/` — full MDL files that should produce the exact
  hint output (golden files).
- `mxcli check` integration tests using fixtures lifted from `Mx2026AIDay`
  (verbatim copies committed under `testdata/mx2026aiday/`).
- `mxcli exec` integration tests that capture stdout and assert on `HINT`
  lines.

## 10. Open Questions

- Should EX002 / EX003 emit `error` or `warning` severity? Proposed:
  `warning` until catalog-backed (P3), then promote to `error`.
- For EX007, when no catalog: should the pattern emit at all, or only with
  catalog? Proposed: emit `info` severity warning ("Module prefix
  unverifiable") when no catalog.
- Should the `exec` adapter respect a `MX_EXPRCHECK=off` env var to suppress
  hints? Proposed: yes, plus a `--no-expr-check` flag on `mxcli exec`.

## 11. Out-of-Scope (explicit)

- Page-level expressions (visibility, dynamic class, etc.) — separate proposal.
- XPath-only constructs (most are handled correctly by existing
  `expressionToXPath`).
- Static analysis of expressions in pluggable widget property defaults.
- Auto-fix or quick-fix code generation; AI consumes the hint and regenerates.
