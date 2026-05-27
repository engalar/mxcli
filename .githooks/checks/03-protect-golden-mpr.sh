#!/bin/sh
# Guard: testdata/helpdesk-golden/ MPR files must only be updated via
# `make update-helpdesk-golden` (TestHelpdeskGolden_Update), never hand-edited.
#
# Rule: if minimal.mpr or any mprcontents/ file is staged,
# describe-snapshot.mdl must also be staged — proof that the official
# rebuild path was used (it regenerates both together).
#
# What this blocks:
#   mxcli exec ... -p testdata/helpdesk-golden/minimal.mpr  → changes mprcontents/ only
#   direct binary edits to minimal.mpr                       → no snapshot update
#
# What this allows:
#   make update-helpdesk-golden  → updates mprcontents/ + minimal.mpr + describe-snapshot.mdl

staged_mpr=$(git diff --cached --name-only | \
  grep -E '^testdata/helpdesk-golden/(minimal\.mpr|mprcontents/)' | head -1)

if [ -z "$staged_mpr" ]; then
    exit 0
fi

snapshot_path="testdata/helpdesk-golden/describe-snapshot.mdl"
staged_snapshot=$(git diff --cached --name-only | grep "^${snapshot_path}$" | head -1)

# Allow when snapshot is staged OR when snapshot is already up-to-date (no pending changes).
# The latter happens when make update-helpdesk-golden regenerates the same content,
# e.g. when a gen type BSON field rename changes the MPR binary but not the describe text.
if [ -z "$staged_snapshot" ]; then
    snapshot_dirty=$(git status --porcelain "${snapshot_path}" 2>/dev/null | head -1)
    if [ -n "$snapshot_dirty" ]; then
        echo "" >&2
        echo "COMMIT BLOCKED: testdata/helpdesk-golden/ MPR staged without describe-snapshot.mdl." >&2
        echo "" >&2
        echo "  Golden MPR must ONLY be rebuilt via:" >&2
        echo "    make update-helpdesk-golden" >&2
        echo "" >&2
        echo "  This runs TestHelpdeskGolden_Update (FUSE overlay + helpdesk-app.mdl)" >&2
        echo "  and regenerates minimal.mpr, mprcontents/, AND describe-snapshot.mdl" >&2
        echo "  together. Do NOT run mxcli exec against the golden MPR directly." >&2
        echo "" >&2
        echo "SOP: .githooks/sop/03-protect-golden-mpr.md" >&2
        echo "CONTEXT: STAGED_MPR=${staged_mpr}" >&2
        exit 1
    fi
    # Snapshot is clean (content unchanged) — nothing to stage, allow commit.
fi
