# SOP: 03-gen-consistency

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: modelsdk/gen/ changed but codegen source unchanged."

## Context variables
- `{GEN_STAGED}` — number of staged files in modelsdk/gen/

## Steps
1. Run: `git diff --cached --name-only | grep '^modelsdk/gen/'` — list the staged gen files
2. Determine what manual edits were made (gen files must not be hand-edited)
3. If the edit was intentional (e.g. fixing a bug in gen output):
   - Find the generator template in `internal/codegen/` that produces the affected type
   - Fix the template instead
   - Run: `go run ./cmd/modelsdk-codegen <args>` to regenerate
   - `git add internal/codegen/ modelsdk/gen/` and re-attempt commit
4. If the gen file change was accidental: `git restore modelsdk/gen/` and re-attempt commit
