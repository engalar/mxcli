# SOP: 03-describe-datasource-arch

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: ... adds raw BSON field access by string literal."

## Context variables
- `{AFFECTED_FILE}` — the executor Go file that added raw field access
- `{BAD_FIELDS}` — e.g. `["NanoflowSettings"] ["Nanoflow"]`

## Steps
1. Open `{AFFECTED_FILE}` and find the lines accessing `{BAD_FIELDS}` via `ds["Field"]` or `w["Field"]`
2. Identify the BSON type (e.g. `Forms$NanoflowSource`) from the surrounding `case` or `if` block
3. Find the corresponding gen type in `modelsdk/gen/pages/types.go` — search for the struct whose `SetTypeName` matches the BSON type
4. Replace the raw map access with gen-type decode:
   ```go
   raw, _ := bson.Marshal(ds)
   elem, _ := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(raw))
   typed, ok := elem.(*genPg.TheType)
   if !ok { return "" }
   return typed.TheQualifiedName()
   ```
5. Run: `CGO_ENABLED=0 go test ./mdl/executor/... -v` — confirm tests pass
6. `git add {AFFECTED_FILE}` and re-attempt commit
