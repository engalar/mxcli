# SOP: 06-describe-snapshot-sync

## Trigger A: MPR staged without describe-snapshot.mdl

**Message:** `COMMIT BLOCKED: testdata/helpdesk-golden-X/mprcontents/ staged without describe-snapshot.mdl`

**Steps:**
1. Run: `make update-snapshots`
2. Stage the updated snapshot: `git add testdata/helpdesk-golden-X/describe-snapshot.mdl`
3. Re-attempt commit

**Why:** The MPR and its describe snapshot must always move together. A stale
snapshot means CI tests (TestHelpdeskGolden_DescribeSnapshot) will fail.

---

## Trigger B: describe-snapshot.mdl fails mxcli check

**Message:** `COMMIT BLOCKED: describe-snapshot.mdl failed mxcli check`

**Steps:**
1. Run manually to see the full error:
   ```bash
   bin/mxcli check testdata/helpdesk-golden-X/describe-snapshot.mdl \
     -p testdata/helpdesk-golden-X/minimal.mpr --references
   ```
2. Identify the failing statement (entity ref, page param syntax, etc.)
3. Fix the describe logic in `mdl/executor/` or `mdl/backend/mpr/`
4. Rebuild and re-validate: `make update-snapshots`
5. Stage and re-commit

**Common causes:**
- Renamed entity/attribute not updated in describe output
- Page parameter syntax changed in MDL grammar
- New reserved keyword conflicting with widget/entity name in snapshot

---

## Idempotency failure (make validate-snapshots)

**Test:** `TestHelpdeskGolden_DescribeSnapshot_Idempotent`

**Meaning:** Executing `describe-snapshot.mdl` on the clean MPR and re-describing
produces different output. The snapshot is not a complete, self-contained description.

**Steps:**
1. Run to see the diff:
   ```bash
   HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
     -tags linux,integration \
     -run TestHelpdeskGolden_DescribeSnapshot_Idempotent -v
   ```
2. Identify which statements change on re-describe (typically lossy IR describe output)
3. Fix the relevant describe path in `mdl/executor/cmd_pages_describe.go` or
   `mdl/executor/cmd_pages_model_to_mdl.go`
4. Run `make update-snapshots` and re-validate

**Known gaps (as of 2026-06-02):** PageModel IR does not fully cover button actions,
checkbox, radiobuttons, and pluggable widget Object trees. These fall back to the
legacy describe path which produces correct output. See `pageModelHasLossyWidget`
in `mdl/executor/cmd_pages_create_v3.go`.
