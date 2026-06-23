# SOP: 04-no-raw-bson-in-executor

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: executor files must use gen modelsdk, not raw bson:"

## Context variables
- `{VIOLATIONS}` — colon-separated list of executor Go files that added raw bson import

## Steps
1. Open each file in `{VIOLATIONS}` (split on `:`)
2. Find the `bson.D` / `bson.M` construction site
3. Identify the Mendix type being built (look for `$Type` key or surrounding comments)
4. Find the gen constructor in `modelsdk/gen/` — search for `New<TypeName>()` e.g. `NewMicroflowSource()`
5. Replace the raw bson.D construction with:
   ```go
   elem := genPg.NewTheType()
   elem.SetTheProperty(value)
   // serialise via codec if needed: codec.NewEncoder().Encode(elem)
   ```
6. Run: `CGO_ENABLED=0 go build ./...` — confirm no compile errors
7. Run: `CGO_ENABLED=0 go test ./mdl/executor/... -v`
8. `git add` the fixed files and re-attempt commit
