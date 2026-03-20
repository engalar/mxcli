# Task Plan: Fix GitHub Issue #1 - MDL Round-Trip Validation

## Goal
Fix 4 bugs from https://github.com/engalar/mxcli/issues/1 and verify with roundtrip tests.

## Phases
- [x] Phase 1: Fix DEMO USER DESCRIBE missing ENTITY clause
- [x] Phase 2: Add CALCULATED BY keyword + persistent-only validation
- [x] Phase 3: Implement CREATE WORKFLOW support
- [x] Phase 4: Update documentation and skills
- [x] Phase 5: Fix BSON type names (Workflows$Parameter, EmptyUserSource)
- [ ] Phase 6: Verify roundtrip tests pass (in progress)
- [ ] Phase 7: Commit and create PR

## Active Agent Team: roundtrip-verify
Team config: `~/.claude/teams/roundtrip-verify/config.json`

### Agents:
- **test-calculated**: COMPLETED ✅ - CALCULATED BY roundtrip passes
- **test-demouser**: IN PROGRESS - waiting for response, may be stuck
- **test-workflow**: IN PROGRESS - notified about binary rebuild with type fixes

### Remaining Work:
1. Wait for test-demouser and test-workflow results
2. If test-demouser is stuck, run the test manually:
   ```bash
   ./bin/mxcli -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr -c "SHOW DEMO USERS;"
   ./bin/mxcli -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr -c "DESCRIBE DEMO USER 'demo_admin';"
   # Check output includes ENTITY clause
   ```
3. If test-workflow has issues, check writer_workflow.go BSON against mendixmodelsdk:
   ```bash
   # SDK types reference:
   grep "structureTypeName" /mnt/data_sdd/home_eg/.nvm/versions/node/v20.19.0/lib/node_modules/mendixmodelsdk/src/gen/workflows.js
   ```
4. After all tests pass, commit all changes and create PR to main

## Key Fixes Made
| Fix | Files Changed |
|-----|--------------|
| DESCRIBE DEMO USER ENTITY | `mdl/executor/cmd_security.go:701-703` |
| CALCULATED BY + persistent validation | `MDLParser.g4`, `visitor_helpers.go`, `visitor_entity.go`, `cmd_entities.go`, `visitor_test.go` |
| CREATE WORKFLOW | `MDLParser.g4`, `MDLLexer.g4`, `mdl/ast/ast_workflow.go`, `mdl/visitor/visitor_workflow.go`, `mdl/executor/cmd_workflows_write.go`, `sdk/mpr/writer_workflow.go` + grammar regeneration |
| BSON type fix | `sdk/mpr/writer_workflow.go` (Parameter, EmptyUserSource) |
| Docs/Skills | `CLAUDE.md`, `MDL_FEATURE_MATRIX.md`, `MDL_QUICK_REFERENCE.md`, `workflow.txt`, `security.txt`, `entity.txt`, `generate-domain-model.md` |

## BSON Type Issues Found (via mendixmodelsdk + mx check + Studio Pro BSON analysis)

### Already Fixed:
- `Workflows$WorkflowParameter` → `Workflows$Parameter` (FIXED)
- `Workflows$NoUserSource` → `Workflows$EmptyUserSource` (FIXED)

### Still Need Fixing (from test-workflow agent roundtrip):

| Component | Writer Uses (WRONG) | Correct (from Studio Pro BSON) |
|-----------|-------------------|-------------------------------|
| Parameter structure | Nested `EntityRef` → `DomainModels$IndirectEntityRef` | Direct `Entity` (BY_NAME string) + `Name` field |
| String Template | `Texts$StringTemplate` with `Parameters: [3]` | `Microflows$StringTemplate` with `Parameters: [2]` |
| User Task | `Workflows$UserTask` | `Workflows$SingleUserTaskActivity` |
| User Source (XPath) | `Workflows$XPathBasedUserSource` | `Workflows$XPathUserTargeting` |
| User Source (Microflow) | `Workflows$MicroflowBasedUserSource` | `Workflows$MicroflowUserTargeting` |
| Events | (missing) | `Workflows$NoEvent` for OnCreatedEvent |
| Task Page | Direct `Page` string | `Workflows$PageReference` object |
| Missing fields | — | `PersistentId`, `BoundaryEvents`, `AutoAssignSingleTargetUser`, `Annotation`, `RelativeMiddlePoint`, `Size` |

### Correct Parameter Structure (from Studio Pro):
```bson
Parameter: {
  $ID: <binary>,
  $Type: "Workflows$Parameter",
  Entity: "Module.Entity",   // direct BY_NAME string, NOT nested EntityRef
  Name: "WorkflowContext"
}
```

### Reference SDK:
- mendixmodelsdk installed globally: `/mnt/data_sdd/home_eg/.nvm/versions/node/v20.19.0/lib/node_modules/mendixmodelsdk/src/gen/workflows.js`
- Use `grep "structureTypeName\|ByNameReference\|PartProperty\|PrimitiveProperty" workflows.js` to find correct types
- Compare writer output BSON against existing working workflows in the test project

## ALTER ENTITY
Already fully implemented - issue report was incorrect. No changes needed.

## Status
**Currently in Phase 6** - Waiting for roundtrip test agents to report back
