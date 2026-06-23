# SOP: 02-unit-tests

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: unit tests failed."

## Context variables
- `{LOG_FILE}` — path to the test failure log (always `.test-fail.log`)

## Steps
1. Run: `cat {LOG_FILE}` — read the full failure output
2. Identify the failing test package and test name from the `FAIL` lines
3. Run: `CGO_ENABLED=0 go test -timeout 120s -run TestXxx ./path/to/pkg -v` (substitute actual test name and package)
4. Fix the failing code (implementation or test, depending on root cause)
5. Run: `CGO_ENABLED=0 go test -timeout 120s ./...` — confirm all pass
6. `git add` the changed files and re-attempt commit
