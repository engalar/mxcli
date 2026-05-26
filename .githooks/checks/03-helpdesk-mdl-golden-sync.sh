#!/bin/sh
# Guard: when helpdesk-app.mdl OR BSON-affecting gen/backend files are staged,
# testdata/helpdesk-golden/ must also be staged — proof that the golden MPR
# was rebuilt after the change.
#
# Triggers (any one of these requires golden rebuild):
#   mdl-examples/use-cases/helpdesk/helpdesk-app.mdl  — MDL source changed
#   modelsdk/gen/microflows/types.go                   — BSON field/type names
#   mdl/backend/mpr/                                   — BSON write logic
#   mdl/executor/flowbuilder_actions_retrieve_gen.go   — retrieve BSON builder
#
# Rule: trigger staged alone → block (golden is stale)
#       trigger + golden both staged → allow (rebuild was run)
#
# Correct workflow:
#   make update-helpdesk-golden   # rebuilds MPR + describe-snapshot.mdl
#   git add <changed-files> testdata/helpdesk-golden/
#   git commit

staged_trigger=$(git diff --cached --name-only | grep -E \
  '^mdl-examples/use-cases/helpdesk/helpdesk-app\.mdl$|^modelsdk/gen/microflows/types\.go$|^mdl/backend/mpr/|^mdl/executor/flowbuilder_actions_retrieve_gen\.go$' \
  | head -1)

if [ -z "$staged_trigger" ]; then
    exit 0
fi

staged_golden=$(git diff --cached --name-only | \
  grep '^testdata/helpdesk-golden/' | head -1)

if [ -z "$staged_golden" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: BSON-affecting file staged without rebuilding testdata/helpdesk-golden/." >&2
    echo "" >&2
    echo "  Trigger: $staged_trigger" >&2
    echo "" >&2
    echo "  Rebuild and stage the golden MPR before committing:" >&2
    echo "    make update-helpdesk-golden" >&2
    echo "    git add testdata/helpdesk-golden/" >&2
    echo "" >&2
    exit 1
fi
