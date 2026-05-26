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

staged_snapshot=$(git diff --cached --name-only | \
  grep '^testdata/helpdesk-golden/describe-snapshot\.mdl$' | head -1)

if [ -z "$staged_snapshot" ]; then
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
    exit 1
fi
