# Test Formats

mxcli supports two test file formats: `.test.mdl` for pure MDL tests and `.test.md` for literate tests with documentation.

## .test.mdl Format

Pure MDL scripts with test annotations. Each test case is marked with `@test` and optionally `@expect`.

```sql
-- @test Create a new module
CREATE MODULE TestModule;
-- @expect 0 errors

-- @test Create entity with attributes
CREATE PERSISTENT ENTITY TestModule.Customer (
    Name: String(200) NOT NULL,
    Email: String(200) UNIQUE,
    IsActive: Boolean DEFAULT true
);
-- @expect 0 errors

-- @test Create enumeration
CREATE ENUMERATION TestModule.Status (
    Active 'Active',
    Inactive 'Inactive'
);
-- @expect 0 errors

-- @test Create microflow
CREATE MICROFLOW TestModule.ACT_Activate(
    $customer: TestModule.Customer
) RETURNS Boolean AS $result
BEGIN
    DECLARE $result Boolean = false;
    CHANGE $customer (IsActive = true);
    COMMIT $customer;
    SET $result = true;
    RETURN $result;
END;
-- @expect 0 errors
```

## .test.md Format

Literate test files that combine prose documentation with embedded MDL code blocks. The test runner extracts and executes the MDL code blocks while ignoring the prose.

````markdown
# Customer Module Tests

This test suite validates the customer management module.

## Entity Creation

Create the customer entity with standard fields:

```sql
CREATE PERSISTENT ENTITY TestModule.Customer (
    Name: String(200) NOT NULL,
    Email: String(200)
);
```

## Association Setup

Link customers to their orders:

```sql
CREATE ASSOCIATION TestModule.Order_Customer
    FROM TestModule.Order
    TO TestModule.Customer
    TYPE Reference;
```
````

The `.test.md` format is useful for:
- Documenting test intent alongside the test code
- Creating test suites that serve as tutorials
- Sharing test cases with non-technical stakeholders

## File Organization

```
tests/
├── domain-model.test.mdl     # Entity and association tests
├── microflows.test.mdl        # Microflow logic tests
├── security.test.mdl          # Access rule tests
├── pages.test.mdl             # Page creation tests
└── integration.test.md        # Full integration test with docs
```

## Naming Conventions

- Use descriptive file names that indicate what is being tested
- Group related tests in the same file
- Use the `@test` annotation to name individual test cases within a file
