#!/bin/sh
# Guard: executor/ must not import raw bson libs directly.
# All BSON construction must go through gen modelsdk types.
# Exceptions: test files, cmd_pages_builder_v3.go (legacy page BSON, grandfathered).

EXECUTOR_DIR="mdl/executor"
VIOLATIONS=""

for f in $(git diff --cached --name-only | grep "^${EXECUTOR_DIR}/.*\.go$" | grep -v "_test\.go"); do
    # Grandfathered files: read-only BSON use (format/describe) or legacy page BSON
    case "$f" in
        *cmd_pages_builder_v3.go) continue ;;
        *datagrid*) continue ;;
        *format_action*) continue ;;
        *format_data*) continue ;;
        *format_calls*) continue ;;
        *format_workflow*) continue ;;
        *format_external*) continue ;;
        *describe*) continue ;;
    esac

    # Only flag files that newly ADD the bson import (not pre-existing)
    if git diff --cached -- "$f" | grep '^+' | grep -q '"go.mongodb.org/mongo-driver/bson"' 2>/dev/null; then
        VIOLATIONS="$VIOLATIONS $f"
    fi
done

if [ -n "$VIOLATIONS" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: executor files must use gen modelsdk, not raw bson:" >&2
    for f in $VIOLATIONS; do
        echo "  $f" >&2
    done
    echo "" >&2
    echo "Replace bson.D/bson.M construction with gen types from modelsdk/gen/." >&2
    echo "Use setRawBSONField() only as a last resort with a comment explaining why." >&2
    exit 1
fi
