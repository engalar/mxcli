# Fix Issue Skill

A fast-path workflow for diagnosing and fixing bugs in mxcli. Each fix appends
to the symptom table below, so the next similar issue costs fewer reads.

## How to Use

1. Match the issue symptom to a row in the table — go straight to that file.
2. Follow the fix pattern for that row.
3. Write a failing test first, then implement.
4. After the fix: **add a new row** to the table if the symptom is not already covered.

---

## Symptom → Layer → File Table

| Symptom | Root cause layer | First file to open | Fix pattern |
|---------|-----------------|-------------------|-------------|
| `describe` shows `$var = list operation ...;` | Missing parser case | `sdk/mpr/parser_microflow.go` → `parseListOperation()` | Add `case "microflows$XxxType":` returning the correct struct |
| `describe` shows `$var = action ...;` | Missing formatter case | `mdl/executor/cmd_microflows_format_action.go` → `formatActionStatement()` | Add `case *microflows.XxxAction:` with `fmt.Sprintf` output |
| `describe` shows `$var = list operation %T;` (with type name) | Missing formatter case | `mdl/executor/cmd_microflows_format_action.go` → `formatListOperation()` | Add `case *microflows.XxxOperation:` before the `default` |
| Compile error: `undefined: microflows.XxxOperation` | Missing SDK struct | `sdk/microflows/microflows_actions.go` | Add struct + `func (XxxOperation) isListOperation() {}` marker |
| `TypeCacheUnknownTypeException` in Studio Pro | Wrong `$type` storage name in BSON write | `sdk/mpr/writer_microflow.go` | Check the storage name table in CLAUDE.md; verify against `reference/mendixmodellib/reflection-data/` |
| CE0066 "Entity access is out of date" | MemberAccess added to wrong entity | `sdk/mpr/writer_domainmodel.go` | MemberAccess must only be on the FROM entity (`ParentPointer`), not the TO entity — see CLAUDE.md association semantics |
| CE0463 "widget definition changed" | Object property structure doesn't match Type PropertyTypes | `sdk/widgets/templates/` | Re-extract template from Studio Pro; see `sdk/widgets/templates/README.md` |
| Parser returns `nil` for a known BSON type | Unhandled `default` in a `parseXxx()` switch | `sdk/mpr/parser_microflow.go` or `parser_page.go` | Find the switch by grepping for `default: return nil`; add the missing case |
| MDL check gives "unexpected token" on valid-looking syntax | Grammar missing rule or token | `mdl/grammar/MDLParser.g4` + `MDLLexer.g4` | Add rule/token, run `make grammar` |
| CE7054 "parameters updated" / CE7067 "does not support body entity" after `send rest request` | `addSendRestRequestAction` emitted wrong BSON: all params as query params, BodyVariable set for JSON bodies | `mdl/executor/cmd_microflows_builder_calls.go` → `addSendRestRequestAction` | Look up operation via `fb.restServices`; route path/query params with `buildRestParameterMappings`; suppress BodyVariable for JSON/TEMPLATE/FILE via `shouldSetBodyVariable` |
| `CREATE X` returns "already exists — use create or replace to overwrite" but OR REPLACE is not valid for that type | Error message in executor points to wrong keyword | `mdl/executor/cmd_<type>_*.go` — find the `NewAlreadyExistsMsg` call | Change hint from `or replace` to `or modify`; verify the AST stmt uses `CreateOrModify` not `CreateOrReplace` |
| `mx check` CE0126 "Missing value for parameter X" on `call java action ... ($Param = empty)` for typed (non-entity, non-microflow) parameters | Builder emitted `BasicCodeActionParameterValue.Argument: ""` instead of the literal `"empty"` keyword | `mdl/executor/cmd_microflows_builder_calls.go` → `addCallJavaActionAction` | Capture all resolved BasicParameterType params into `resolvedBasicParams`; when bound to MDL `empty`, emit `Argument: "empty"` so Studio Pro recognises an explicit empty literal rather than treating the slot as missing |
| `DESCRIBE microflow` puts shared activities inside an `if … then` block — they should appear after `end if;` | Nested guard split inside `traverseFlowUntilMerge` crosses the outer merge boundary | `mdl/executor/cmd_microflows_show_helpers.go` — guard path in `traverseFlowUntilMerge` (~line 854) | Add `if contID != mergeID` guard before the `isMerge` skip-through so the guard continuation never crosses the outer merge |

---

## TDD Protocol

Always follow this order — never implement before the test exists:

```
Step 1: write a failing unit test (parser test or formatter test)
Step 2: Confirm it fails to compile or fails at runtime
Step 3: Implement the minimum code to make it pass
Step 4: run: /c/users/Ylber.Sadiku/go/go/bin/go test ./mdl/executor/... ./sdk/mpr/...
Step 5: add the symptom row to the table above if not already present
```

Parser tests go in `sdk/mpr/parser_<domain>_test.go`.
Formatter tests go in `mdl/executor/cmd_<domain>_format_<area>_test.go`.

---

## Issue #212 — Reference Fix (seeding example)

**Symptom:** `describe microflow` showed `$var = list operation ...;` for
`microflows$find`, `microflows$filter`, `microflows$ListRange`.

**Root cause:** `parseListOperation()` in `sdk/mpr/parser_microflow.go` had no
cases for these three BSON types — they fell to `default: return nil`.

**Files changed:**
| File | Change |
|------|--------|
| `sdk/microflows/microflows_actions.go` | Added `FindByAttributeOperation`, `FilterByAttributeOperation`, `RangeOperation` |
| `sdk/mpr/parser_microflow.go` | Added 3 parser cases |
| `mdl/executor/cmd_microflows_format_action.go` | Added 3 formatter cases |
| `mdl/executor/cmd_microflows_format_listop_test.go` | Added 4 formatter tests |
| `sdk/mpr/parser_listoperation_test.go` | New file, 4 parser tests |

**Key insight:** `microflows$ListRange` stores offset/limit inside a nested
`CustomRange` map — must cast `raw["CustomRange"].(map[string]any)` before
extracting `OffsetExpression`/`LimitExpression`.

---

## After Every Fix — Checklist

- [ ] Failing test written before implementation
- [ ] `go test ./mdl/executor/... ./sdk/mpr/...` passes
- [ ] New symptom row added to the table above (if not already covered)
- [ ] PR title: `fix: <one-line description matching the symptom>`
