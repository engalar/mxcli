#!/bin/sh
# Guard: if modelsdk/gen/ is staged, codegen source must also be staged.
# Generated files must only change when the generator is re-run.

GEN_DIR="modelsdk/gen"
CODEGEN_DIRS="internal/codegen cmd/modelsdk-codegen"

gen_staged=$(git diff --cached --name-only | grep -c "^${GEN_DIR}/" || true)
if [ "$gen_staged" -eq 0 ]; then
    exit 0
fi

codegen_staged=0
for dir in $CODEGEN_DIRS; do
    count=$(git diff --cached --name-only | grep -c "^${dir}/" || true)
    codegen_staged=$((codegen_staged + count))
done

if [ "$codegen_staged" -eq 0 ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: ${GEN_DIR}/ changed but codegen source unchanged." >&2
    echo "" >&2
    echo "Generated files must not be edited manually." >&2
    echo "Re-run the generator to produce a consistent diff:" >&2
    echo "  go run ./cmd/modelsdk-codegen ..." >&2
    echo "" >&2
    echo "Staged gen files:" >&2
    git diff --cached --name-only | grep "^${GEN_DIR}/" >&2
    exit 1
fi
