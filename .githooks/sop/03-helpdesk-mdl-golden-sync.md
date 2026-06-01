# SOP: 03-helpdesk-mdl-golden-sync

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: BSON-affecting file staged without rebuilding"
OR: "COMMIT BLOCKED: testdata/helpdesk-golden-11.6.6/ staged without mdl-examples/..."

## Context variables
- `{TRIGGER_FILE}` — the file that triggered the sync requirement

## Steps (Rule A — golden not staged after BSON change)
1. Confirm the change in `{TRIGGER_FILE}` is complete and correct
2. Add a comment to `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` documenting what changed:
   - Find the relevant section (e.g. `-- MARK: Pages`) and add a `-- Note:` line
3. Run: `make update-helpdesk-golden`
4. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl`
5. Re-attempt commit

## Steps (Rule B — golden staged without MDL)
1. Open `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
2. Add or update a comment that explains why the golden was rebuilt (e.g. fix description, new example)
3. Run: `make update-helpdesk-golden`
4. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl`
5. Re-attempt commit
