# SOP: 05-mx-check-golden

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: mx check found N new error(s)"

## Context variables
- `{ERROR_COUNT}` — total errors found by mx check
- `{BASELINE}` — accepted baseline error count
- `{MX_VERSION}` — Mendix version of the golden MPR

## Steps
1. Run: `~/.mxcli/mxbuild/{MX_VERSION}/modeler/mx check testdata/helpdesk-golden-*/minimal.mpr 2>&1 | grep '^\[error\]'`
2. Compare output with baseline ({BASELINE} errors expected). The excess lines are the new errors.
3. For each new `[error]` line, identify the domain:
   - `CE0463` → widget definition mismatch (see `.claude/skills/debug-bson.md`)
   - `CE1613` → wrong attribute ref (check `AttributeRef` in BSON write path)
   - `CE0003` → missing version prefix on array field
   - Other → search error code in `docs/` and `.claude/skills/`
4. Fix the relevant executor/backend/gen code
5. Run: `make update-helpdesk-golden`
6. Run: `git add testdata/helpdesk-golden-*/ mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
7. Re-attempt commit (if still blocked, repeat from step 1)
