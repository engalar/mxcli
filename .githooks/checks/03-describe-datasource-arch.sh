#!/bin/sh
# Architectural guard: mdl/executor/ must not accumulate new raw BSON field accesses.
#
# Rule: new lines added to executor files must not access BSON data via string literals
#   e.g.  ds["MicroflowSettings"]  value["DataSource"]  w["EntityRef"]
#   These bypass gen modelsdk types and cause silent nil bugs (see: MicroflowSource fix).
#
# Use gen modelsdk types instead:
#   ms.(*genPg.MicroflowSource).MicroflowSettings().(*genPg.MicroflowSettings).MicroflowQualifiedName()
#
# Existing code is grandfathered — only NEWLY ADDED lines are checked.
# Rationale: the executor read path is tech debt. This hook prevents it from growing worse
# and forces new code through gen types.
#
# Pattern detected: ["UppercaseName"] — Mendix BSON field names always start with uppercase.
# (Go map keys in non-BSON code are typically lowercase; this heuristic has near-zero false positives.)

# Collect staged executor Go files (exclude test files — they legitimately build mock BSON).
staged_files=$(git diff --cached --name-only | grep '^mdl/executor/.*\.go$' | grep -v '_test\.go$')

[ -z "$staged_files" ] && exit 0

failed=0
for f in $staged_files; do
    # Extract only newly added lines from the staged diff for this file.
    new_lines=$(git diff --cached -U0 -- "$f" 2>/dev/null \
        | grep '^+' \
        | grep -v '^+++')

    [ -z "$new_lines" ] && continue

    # Detect raw BSON field access: map indexed by a capitalized string literal.
    # Pattern: ["AnyCapitalizedName"] — e.g. ds["DataSource"], w["MicroflowSettings"]
    bad=$(echo "$new_lines" | grep -oE '\["[A-Z][a-zA-Z]*"\]' | sort -u)

    if [ -n "$bad" ]; then
        echo "COMMIT BLOCKED: $f (mdl/executor/) adds raw BSON field access by string literal." >&2
        echo "  Fields: $bad" >&2
        echo "" >&2
        echo "  Raw map access (ds[\"Field\"], w[\"Field\"]) bypasses type safety and causes" >&2
        echo "  silent nil bugs. Use gen modelsdk types instead:" >&2
        echo "    import genPg \"github.com/mendixlabs/mxcli/modelsdk/gen/pages\"" >&2
        echo "    ms.(*genPg.MicroflowSource).MicroflowSettings()..." >&2
        echo "" >&2
        echo "  If you are migrating existing code (not adding new raw access), use:" >&2
        echo "    // nolint:describe-raw-bson" >&2
        echo "  on the line to opt out (requires justification in PR)." >&2
        echo "SOP: .githooks/sop/03-describe-datasource-arch.md" >&2
        echo "CONTEXT: AFFECTED_FILE=${f} BAD_FIELDS=${bad}" >&2
        failed=1
    fi
done

[ "$failed" -eq 1 ] && exit 1

echo "pre-commit: describe BSON arch check passed."
