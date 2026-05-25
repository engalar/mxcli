#!/bin/sh
# L1: unit tests — fast, no external deps, no Docker.
LOG_FILE="$(git rev-parse --show-toplevel)/.test-fail.log"
rm -f "$LOG_FILE"

echo "pre-commit: running unit tests..."
if ! CGO_ENABLED=0 go test -timeout 120s ./... > "$LOG_FILE" 2>&1; then
    echo "" >&2
    echo "COMMIT BLOCKED: unit tests failed." >&2
    echo "" >&2
    cat "$LOG_FILE" >&2
    exit 1
fi

rm -f "$LOG_FILE"
echo "pre-commit: unit tests passed."
