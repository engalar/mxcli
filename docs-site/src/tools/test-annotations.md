# Test Annotations

Test annotations control test execution and define expectations. They are placed in comments within `.test.mdl` files.

## @test

Marks the start of a named test case.

**Syntax:**

```sql
-- @test <description>
```

**Example:**

```sql
-- @test Create customer entity
CREATE PERSISTENT ENTITY MyModule.Customer (
    Name: String(200) NOT NULL
);
```

Each `@test` annotation starts a new test case. All MDL statements between one `@test` and the next (or end of file) belong to that test case.

## @expect

Defines the expected outcome of a test case.

**Syntax:**

```sql
-- @expect <expectation>
```

### Expected Outcomes

| Expectation | Description |
|-------------|-------------|
| `0 errors` | The test should complete with no errors from `mx check` |
| `error` | The test is expected to produce an error |
| `<n> errors` | The test should produce exactly N errors |

**Examples:**

```sql
-- @test Valid entity creation
CREATE PERSISTENT ENTITY MyModule.Customer (
    Name: String(200) NOT NULL
);
-- @expect 0 errors

-- @test Missing module should fail
CREATE PERSISTENT ENTITY NonExistent.Entity (
    Name: String(200)
);
-- @expect error
```

## Complete Example

```sql
-- @test Setup: Create module and enumeration
CREATE MODULE TestSales;

CREATE ENUMERATION TestSales.OrderStatus (
    Draft 'Draft',
    Submitted 'Submitted',
    Completed 'Completed'
);
-- @expect 0 errors

-- @test Create entity with enum attribute
CREATE PERSISTENT ENTITY TestSales.Order (
    OrderNumber: String(50) NOT NULL,
    Status: Enumeration(TestSales.OrderStatus) DEFAULT 'Draft',
    TotalAmount: Decimal DEFAULT 0
);
-- @expect 0 errors

-- @test Create microflow with validation
CREATE MICROFLOW TestSales.ACT_SubmitOrder(
    $order: TestSales.Order
) RETURNS Boolean AS $success
BEGIN
    DECLARE $success Boolean = false;

    IF $order/TotalAmount <= 0 THEN
        VALIDATION FEEDBACK $order/TotalAmount MESSAGE 'Total must be positive';
        RETURN false;
    END IF;

    CHANGE $order (Status = 'Submitted');
    COMMIT $order;
    SET $success = true;
    RETURN $success;
END;
-- @expect 0 errors

-- @test Security: grant access
CREATE MODULE ROLE TestSales.User;
GRANT EXECUTE ON MICROFLOW TestSales.ACT_SubmitOrder TO TestSales.User;
GRANT TestSales.User ON TestSales.Order (CREATE, DELETE, READ *, WRITE *);
-- @expect 0 errors
```

## Notes

- Test cases run sequentially; earlier test cases set up state for later ones
- The test runner validates using `mx check` after executing each test case (or the full script)
- Tests that modify the project are run against an isolated copy, not the original MPR file
