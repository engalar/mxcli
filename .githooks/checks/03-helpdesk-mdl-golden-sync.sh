#!/bin/sh
# Guard: helpdesk-app.mdl and testdata/helpdesk-golden/ must always move together.
#
# Rule A — BSON-affecting file staged without golden → block (golden is stale).
# Rule B — golden staged without MDL → block (MDL is the source of truth;
#           every golden rebuild must be traceable to an MDL change or explicit
#           acknowledgement that the MDL was verified after the fix).
#
# BSON-affecting triggers (any one requires golden + MDL to be staged):
#   mdl-examples/use-cases/helpdesk/helpdesk-app.mdl  — MDL source changed
#   modelsdk/gen/microflows/types.go                   — BSON field/type names
#   mdl/backend/mpr/                                   — BSON write logic
#   mdl/executor/flowbuilder_actions_retrieve_gen.go   — retrieve BSON builder
#
# Correct workflow after any BSON-affecting change:
#   1. Update helpdesk-app.mdl (add example / fix / comment documenting the change)
#   2. make update-helpdesk-golden
#   3. git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/
#   4. git commit

MDL_PATH="mdl-examples/use-cases/helpdesk/helpdesk-app.mdl"

staged_trigger=$(git diff --cached --name-only | grep -E \
  "^${MDL_PATH}\$|^modelsdk/gen/microflows/types\\.go\$|^mdl/backend/mpr/|^mdl/executor/flowbuilder_actions_retrieve_gen\\.go\$" \
  | head -1)

staged_golden=$(git diff --cached --name-only | \
  grep -E '^testdata/helpdesk-golden/(minimal\.mpr|mprcontents/)' | head -1)

staged_mdl=$(git diff --cached --name-only | \
  grep "^${MDL_PATH}\$" | head -1)

# Rule A: BSON-affecting file staged but golden not staged.
if [ -n "$staged_trigger" ] && [ -z "$staged_golden" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: BSON-affecting file staged without rebuilding testdata/helpdesk-golden/." >&2
    echo "" >&2
    echo "  Trigger: $staged_trigger" >&2
    echo "" >&2
    echo "  1. Update helpdesk-app.mdl to reflect / demonstrate the change." >&2
    echo "  2. make update-helpdesk-golden" >&2
    echo "  3. git add ${MDL_PATH} testdata/helpdesk-golden/" >&2
    echo "" >&2
    exit 1
fi

# Rule B: golden staged but MDL not staged.
if [ -n "$staged_golden" ] && [ -z "$staged_mdl" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: testdata/helpdesk-golden/ staged without ${MDL_PATH}." >&2
    echo "" >&2
    echo "  Every golden rebuild must be accompanied by an MDL change or update" >&2
    echo "  (add an example, fix a comment, or document the verified behaviour)." >&2
    echo "" >&2
    echo "  1. Update ${MDL_PATH}" >&2
    echo "  2. make update-helpdesk-golden" >&2
    echo "  3. git add ${MDL_PATH} testdata/helpdesk-golden/" >&2
    echo "" >&2
    exit 1
fi
