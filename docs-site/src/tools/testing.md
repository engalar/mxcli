# Testing

mxcli includes a testing framework for validating Mendix projects using MDL test files. Tests verify that MDL scripts execute correctly and that the resulting project passes validation.

## Overview

The testing framework supports two test file formats:

| Format | Extension | Description |
|--------|-----------|-------------|
| MDL test files | `.test.mdl` | Pure MDL scripts with test annotations |
| Markdown test files | `.test.md` | Literate tests with prose and embedded MDL code blocks |

## Prerequisites

Running tests requires **Docker** for Mendix runtime validation. The test runner:

1. Creates a fresh Mendix project using `mx create-project`
2. Executes the MDL test script against the project
3. Validates the result with `mx check`
4. Reports pass/fail results

## Quick Start

```bash
# Run all tests in a directory
mxcli test tests/ -p app.mpr

# Run a specific test file
mxcli test tests/sales.test.mdl -p app.mpr
```

## Test Workflow

1. Write test files using `.test.mdl` or `.test.md` format
2. Add `@test` and `@expect` annotations for assertions
3. Run tests with `mxcli test`
4. Review results

## Example Test

```sql
-- tests/customer.test.mdl

-- @test Create customer entity
CREATE PERSISTENT ENTITY MyFirstModule.Customer (
    Name: String(200) NOT NULL,
    Email: String(200)
);
-- @expect 0 errors

-- @test Add association
CREATE ASSOCIATION MyFirstModule.Order_Customer
    FROM MyFirstModule.Order
    TO MyFirstModule.Customer
    TYPE Reference;
-- @expect 0 errors
```

## Related Pages

- [Test Formats](test-formats.md) -- `.test.mdl` and `.test.md` file formats
- [Test Annotations](test-annotations.md) -- `@test` and `@expect` annotations
- [Running Tests](running-tests.md) -- `mxcli test` command and Docker requirements
- [Diff](diff.md) -- Comparing scripts against project state
