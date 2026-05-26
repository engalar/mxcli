#!/bin/sh
# Guard: when helpdesk-app.mdl is staged, testdata/helpdesk-golden/ must
# also be staged — proof that the golden MPR was rebuilt after the MDL change.
#
# Rule: MDL staged alone → block (golden is stale)
#       MDL + golden both staged → allow (rebuild was run)
#       golden staged alone → block (handled by 03-protect-golden-mpr.sh)
#       neither staged → allow
#
# Correct workflow after editing helpdesk-app.mdl:
#   make update-helpdesk-golden   # rebuilds MPR + describe-snapshot.mdl
#   git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/
#   git commit

staged_mdl=$(git diff --cached --name-only | \
  grep '^mdl-examples/use-cases/helpdesk/helpdesk-app\.mdl$' | head -1)

if [ -z "$staged_mdl" ]; then
    exit 0
fi

staged_golden=$(git diff --cached --name-only | \
  grep '^testdata/helpdesk-golden/' | head -1)

if [ -z "$staged_golden" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: helpdesk-app.mdl staged without rebuilding testdata/helpdesk-golden/." >&2
    echo "" >&2
    echo "  After editing helpdesk-app.mdl, rebuild and stage the golden MPR:" >&2
    echo "    make update-helpdesk-golden" >&2
    echo "    git add testdata/helpdesk-golden/" >&2
    echo "" >&2
    echo "  This ensures the golden MPR, mprcontents/, and describe-snapshot.mdl" >&2
    echo "  stay in sync with the MDL source." >&2
    echo "" >&2
    exit 1
fi
