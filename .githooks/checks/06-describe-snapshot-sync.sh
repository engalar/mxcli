#!/bin/sh
# Guard: describe-snapshot.mdl must move together with MPR mprcontents/;
#        staged describe-snapshot.mdl must pass mxcli check --references.
#
# Rule A — MPR mprcontents/ staged without describe-snapshot.mdl → block.
# Rule B — describe-snapshot.mdl staged → run mxcli check --references.
#
# Versions checked: 11.6.6 11.10.0
# mxcli binary: bin/mxcli (skipped if not built)

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
MXCLI="${REPO_ROOT}/bin/mxcli"

if [ ! -x "$MXCLI" ]; then
    exit 0  # mxcli not built — skip (normal for non-describe changes)
fi

for version in 11.6.6 11.10.0; do
    golden_dir="testdata/helpdesk-golden-${version}"
    snapshot="${golden_dir}/describe-snapshot.mdl"
    mpr="${golden_dir}/minimal.mpr"

    staged_mpr=$(git diff --cached --name-only | \
      grep -E "^${golden_dir}/(minimal\\.mpr|mprcontents/)" | head -1)
    staged_snapshot=$(git diff --cached --name-only | \
      grep "^${snapshot}\$" | head -1)

    # Rule A: MPR staged without corresponding describe-snapshot.mdl.
    if [ -n "$staged_mpr" ] && [ -z "$staged_snapshot" ]; then
        echo "" >&2
        echo "COMMIT BLOCKED: ${golden_dir}/mprcontents/ staged without describe-snapshot.mdl." >&2
        echo "" >&2
        echo "  Run: make update-snapshots" >&2
        echo "  Then: git add ${snapshot}" >&2
        echo "" >&2
        echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
        echo "CONTEXT: TRIGGER_FILE=${staged_mpr}" >&2
        exit 1
    fi

    # Rule B: describe-snapshot.mdl staged → validate with mxcli check.
    if [ -n "$staged_snapshot" ]; then
        if [ ! -f "$mpr" ]; then
            continue  # MPR not present locally — skip check
        fi
        echo "pre-commit: mxcli check describe-snapshot.mdl (Mendix ${version})..." >&2
        check_output=$("$MXCLI" check "$snapshot" -p "$mpr" --references 2>&1)
        check_rc=$?
        if [ $check_rc -ne 0 ]; then
            echo "" >&2
            echo "COMMIT BLOCKED: describe-snapshot.mdl failed mxcli check (Mendix ${version})." >&2
            echo "" >&2
            echo "$check_output" >&2
            echo "" >&2
            echo "  Fix the describe output, then: make update-snapshots" >&2
            echo "" >&2
            echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
            exit 1
        fi
        echo "pre-commit: mxcli check PASS (${version})" >&2
    fi
done
