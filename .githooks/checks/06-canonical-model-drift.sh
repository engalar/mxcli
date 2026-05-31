#!/bin/sh
# Warn when staged changes touch non-migrated canonical model serialization functions.
# Non-blocking: always exit 0.
# Spec: docs/superpowers/specs/2026-05-31-canonical-drift-guard-design.md

STAGED=$(git diff --cached --name-only | grep "^mdl/executor/.*\.go$" | grep -v "_test\.go")
[ -z "$STAGED" ] && exit 0

git diff --cached --unified=0 | go run ./tools/check-canonical-drift/ >&2
exit 0
