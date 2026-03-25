# Running Tests

The `mxcli test` command executes test files and reports results.

## Prerequisites

Running tests requires **Docker** for Mendix runtime validation. The test runner uses:

- `mx create-project` to create a fresh blank Mendix project
- `mx check` to validate the project after applying MDL changes

The `mx` binary is located at:

| Environment | Path |
|-------------|------|
| Dev container | `~/.mxcli/mxbuild/{version}/modeler/mx` |
| Repository | `reference/mxbuild/modeler/mx` |

To auto-download mxbuild for the project's Mendix version:

```bash
mxcli setup mxbuild -p app.mpr
```

## Basic Usage

```bash
# Run all tests in a directory
mxcli test tests/ -p app.mpr

# Run a specific test file
mxcli test tests/sales.test.mdl -p app.mpr

# Run markdown test files
mxcli test tests/integration.test.md -p app.mpr
```

## Test Execution Flow

1. **Create project** -- A fresh Mendix project is created in a temporary directory using `mx create-project`
2. **Execute MDL** -- The test script is executed against the fresh project using `mxcli exec`
3. **Validate** -- `mx check` validates the resulting project for errors
4. **Report** -- Results are reported with pass/fail status per test case

## Isolated Testing Pattern

For manual testing or debugging, you can replicate the test runner's workflow:

```bash
# Create a fresh project
cd /tmp/test-workspace
~/.mxcli/mxbuild/*/modeler/mx create-project

# Apply MDL changes
mxcli exec script.mdl -p /tmp/test-workspace/App.mpr

# Validate
~/.mxcli/mxbuild/*/modeler/mx check /tmp/test-workspace/App.mpr
```

Expected output for a passing test:

```
Checking app for errors...
The app contains: 0 errors.
```

## Test Validation Checklist

Before marking tests as complete:

- MDL script executes without errors
- `mx check` passes with 0 errors
- Created elements appear correctly if inspected with `DESCRIBE`
- Security rules are valid (no CE0066 errors)

## Debugging Test Failures

### TypeCacheUnknownTypeException

```
The type cache does not contain a type with qualified name DomainModels$Index
```

This means a BSON `$Type` field uses the wrong name. Check the reflection data for the correct `storageName`.

### CE0066 "Entity access is out of date"

Security access rules don't match the entity's current structure. Make sure:
- All attributes in the entity are included in access rules
- Association member access is only on the FROM entity
- Run `GRANT` after adding new attributes

### Common Fixes

```bash
# Re-run with verbose output
mxcli exec script.mdl -p app.mpr --verbose

# Inspect the generated BSON
mxcli dump-bson -p app.mpr --doc "Module.EntityName"
```
