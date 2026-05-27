# SOP: 03-protect-golden-mpr

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: testdata/helpdesk-golden/ MPR staged without describe-snapshot.mdl."

## Context variables
- `{STAGED_MPR}` — the MPR or mprcontents/ file that was directly modified

## Steps
1. Run: `git restore testdata/helpdesk-golden/` — undo any direct edits to the golden MPR
2. Make your change via the MDL source instead:
   - Edit `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
   - Run: `make update-helpdesk-golden`
3. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/ testdata/helpdesk-golden-clean/describe-snapshot.mdl`
4. Re-attempt commit
