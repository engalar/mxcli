# Built-in Rules

mxcli ships with 14 built-in lint rules implemented in Go. These rules are fast and always available.

## MDL Rules

| Rule | Description |
|------|-------------|
| **MDL001** | Naming conventions -- Checks entity, attribute, and microflow naming patterns |
| **MDL002** | Empty microflows -- Detects microflows with no activities |
| **MDL003** | Domain model size -- Warns when a module has too many entities |
| **MDL004** | Validation feedback -- Checks for proper validation feedback usage |
| **MDL005** | Image source -- Validates image widget source configuration |
| **MDL006** | Empty containers -- Detects container widgets with no children |
| **MDL007** | Page navigation security -- Checks that pages called from microflows have appropriate access rules |

## Security Rules

| Rule | Description |
|------|-------------|
| **SEC001** | Entity access rules -- Ensures persistent entities have access rules defined |
| **SEC002** | Password policy -- Checks for secure password configuration |
| **SEC003** | Demo users -- Warns about demo users in production security level |

## Convention Rules

| Rule | Description |
|------|-------------|
| **CONV011** | No commit in loop -- Detects COMMIT statements inside LOOP blocks (performance anti-pattern) |
| **CONV012** | Exclusive split captions -- Checks that decision branches have meaningful captions |
| **CONV013** | Error handling on external calls -- Ensures external service calls have error handling |
| **CONV014** | No continue error handling -- Warns against using CONTINUE error handling without logging |

## Running Built-in Rules

Built-in rules run automatically with `mxcli lint`:

```bash
# Run all rules (built-in + Starlark)
mxcli lint -p app.mpr

# List all rules to see which are built-in
mxcli lint -p app.mpr --list-rules
```

## Excluding Modules

System and marketplace modules often trigger false positives. Exclude them:

```bash
mxcli lint -p app.mpr --exclude System --exclude Administration
```
