# Mendix Microflow Skill

This skill provides comprehensive guidance for writing Mendix microflows in MDL (Mendix Definition Language) syntax.

## When to Use This Skill

Use this skill when:
- Writing CREATE MICROFLOW statements
- Debugging microflow syntax errors
- Converting Studio Pro microflows to MDL
- Understanding microflow control flow and structure

## Microflow Structure

**CRITICAL: All microflows MUST have JavaDoc-style documentation**

```mdl
/**
 * Microflow description explaining what it does
 *
 * Detailed explanation of the business logic, use cases,
 * and any important implementation notes.
 *
 * @param $Parameter1 Description of first parameter
 * @param $Parameter2 Description of second parameter
 * @returns Description of return value
 * @since 1.0.0
 * @author Team Name
 */
CREATE MICROFLOW Module.MicroflowName (
  $Parameter1: Type,
  $Parameter2: Type
)
RETURNS ReturnType AS $ReturnVariable
[FOLDER 'FolderPath']
BEGIN
  -- Microflow logic here
  RETURN $ReturnVariable;
END;
```

### FOLDER Option

Place microflows in folders for organization:

```mdl
CREATE MICROFLOW MyModule.ACT_ProcessOrder ($Order: MyModule.Order)
RETURNS Boolean AS $Success
FOLDER 'Orders/Processing'
BEGIN
  -- logic
  RETURN true;
END;
```

**Key Rules:**
- Parameters start with `$` prefix
- Return variable must be declared or used
- Every microflow must end with `RETURN` statement
- Statements end with semicolon `;`
- Microflow ends with `/` separator

### Parameter Types

```mdl
-- Primitive types
$Name: String
$Count: Integer
$Amount: Decimal
$IsActive: Boolean
$Date: DateTime

-- Entity types
$Customer: Module.Entity

-- List types
$ProductList: List of Module.Product

-- Enumeration types
$Status: ENUM Module.OrderStatus
```

## Variable Declarations

### ✅ CORRECT Syntax

```mdl
-- Primitive types with initialization
DECLARE $Counter Integer = 0;
DECLARE $Message String = 'Hello';
DECLARE $IsValid Boolean = true;
DECLARE $Today DateTime = [%CurrentDateTime%];

-- Entity types (no initialization needed)
DECLARE $Product Test.Product;
DECLARE $Order Shop.Order;

-- Lists
DECLARE $ProductList List of Test.Product = empty;
```

### ❌ INCORRECT Syntax

```mdl
-- WRONG: Using AS keyword (not supported in mxcli)
DECLARE $Product AS Test.Product;  -- ERROR: parse error

-- WRONG: Missing type
DECLARE $Counter = 0;  -- Type inference not always supported

-- WRONG: Using 'OF' instead of 'of'
DECLARE $List List OF Test.Product;  -- Case sensitive

-- WRONG: Using = empty for entity types
DECLARE $Product Test.Product = empty;  -- Use without initialization
```

## Common Pitfalls

### 1. Entity Type Declarations

**Error**: Parse error or CE0053 - "Selected type is not allowed"

❌ **INCORRECT:**
```mdl
DECLARE $Product AS Test.Product;      -- AS keyword not supported
DECLARE $Product Test.Product = empty; -- = empty not needed for entities
```

✅ **CORRECT:**
```mdl
DECLARE $Product Test.Product;
DECLARE $Order Shop.Order;
```

**Explanation**: Entity types are declared with just the type name, no `AS` keyword and no `= empty` initialization.

### 2. XPath Association Navigation

**Error**: CE0117 - "Error in expression"

❌ **INCORRECT:**
```mdl
-- Using simple association name
DECLARE $CustomerName String = $Order/Customer/Name;
SET $Name = $Product/Category/Name;
```

✅ **CORRECT:**
```mdl
-- Use fully qualified association name: Module.AssociationName
DECLARE $CustomerName String = $Order/Shop.Order_Customer/Name;
SET $Name = $Product/Shop.Product_Category/Name;
```

**Explanation**: XPath navigation requires the full qualified association name in the format `Module.AssociationName`.

### 3. Missing Attributes

**Error**: Attribute references must exist in entity definition

❌ **INCORRECT:**
```mdl
-- Referencing Status when it doesn't exist in Order entity
CHANGE $Order (
  Status = 'PROCESSING',
  ProcessedDate = [%CurrentDateTime%]);
```

✅ **CORRECT:**
```mdl
-- First, ensure entity has the attributes
CREATE PERSISTENT ENTITY Shop.Order (
  OrderNumber: String(50),
  Status: String(50),          -- ← Must be defined
  ProcessedDate: DateTime       -- ← Must be defined
);

-- Then reference them
CHANGE $Order (
  Status = 'PROCESSING',
  ProcessedDate = [%CurrentDateTime%]);
```

### 4. Flow Must End with RETURN

**Error**: CE0105 - "Activity cannot be the last object of a flow"

❌ **INCORRECT:**
```mdl
BEGIN
  DECLARE $Success Boolean = true;
  LOG INFO 'Done';
  -- Missing RETURN!
END;
```

✅ **CORRECT:**
```mdl
BEGIN
  DECLARE $Success Boolean = true;
  LOG INFO 'Done';
  RETURN $Success;  -- ← Always required
END;
```

### 5. Unreachable Code After RETURN

**Error**: CE0104 - "Action activity is unreachable"

❌ **INCORRECT:**
```mdl
IF $Value < 0 THEN
  RETURN false;
  LOG INFO 'This will never execute';  -- ← Unreachable!
END IF;
```

✅ **CORRECT:**
```mdl
IF $Value < 0 THEN
  LOG INFO 'Value is negative';
  RETURN false;
END IF;
```

### 6. Unused Variables

**Warning**: CW0094 - "Variable 'X' is never used"

```mdl
-- Studio Pro will warn if parameters/variables are declared but never used
CREATE MICROFLOW Test.Example (
  $ProductCode: String  -- ← Warning if never referenced
)
RETURNS Boolean AS $Success
BEGIN
  SET $Success = true;  -- ProductCode never used
  RETURN $Success;
END;
```

### 7. Using SET on Undeclared Variables

**Error**: MDL executor validates that all variables used with `SET` are declared first.

❌ **INCORRECT:**
```mdl
BEGIN
  IF $Value > 10 THEN
    SET $Message = 'High';  -- ERROR: $Message not declared!
  END IF;
  RETURN true;
END;
```

✅ **CORRECT:**
```mdl
BEGIN
  DECLARE $Message String = '';  -- Declare first
  IF $Value > 10 THEN
    SET $Message = 'High';  -- Now SET works
  END IF;
  RETURN true;
END;
```

**Note**: Parameters are automatically declared by the parameter list. The `RETURNS Type AS $Var` syntax names the return variable but does NOT declare it - you must still use `DECLARE $Var Type = value;` if you want to use SET on it.

## Control Flow

### IF Statements

```mdl
-- Simple IF
IF $Value > 10 THEN
  SET $Message = 'Greater than 10';
END IF;

-- IF/ELSE
IF $Value > 100 THEN
  SET $Category = 'High';
ELSE
  SET $Category = 'Low';
END IF;

-- Nested IF
IF $Score >= 90 THEN
  SET $Grade = 'A';
ELSE
  IF $Score >= 80 THEN
    SET $Grade = 'B';
  ELSE
    SET $Grade = 'C';
  END IF;
END IF;
```

**Important**: Always close with `END IF` (not just `END`).

### Enumeration Comparisons

**CRITICAL**: When comparing enumeration values, use the fully qualified enumeration value, NOT a string literal.

```mdl
-- CORRECT: Use fully qualified enumeration value
IF $Task/Status = Module.TaskStatus.Completed THEN
  SET $IsComplete = true;
END IF;

IF $Order/OrderStatus != Module.OrderStatus.Cancelled THEN
  -- Process the order
END IF;

-- WRONG: Do NOT use string literals
-- IF $Task/Status = 'Completed' THEN  -- INCORRECT!
```

**Checking for empty enumeration:**
```mdl
IF $Entity/Status = empty THEN
  -- Enumeration is not set
END IF;
```

### LOOP Statements

```mdl
-- Basic loop
LOOP $Product IN $ProductList
BEGIN
  SET $Count = $Count + 1;
END LOOP;

-- Loop with object modification
LOOP $Product IN $ProductList
BEGIN
  CHANGE $Product (IsActive = true);
  COMMIT $Product;
END LOOP;

-- Loop with conditional logic
LOOP $Product IN $ProductList
BEGIN
  IF $Product/IsActive THEN
    SET $ActiveCount = $ActiveCount + 1;
  END IF;
END LOOP;
```

**Note**:
- Loop variable (`$Product`) is scoped to the loop body
- The loop variable type is **automatically derived** from the list type (e.g., `List of Test.Product` → `Test.Product`)
- CHANGE statements inside loops use the derived type to resolve attribute names

## Object Operations

### CREATE Object

```mdl
$NewProduct = CREATE Test.Product (
  Name = $Name,
  Code = $Code,
  IsActive = true,
  CreateDate = [%CurrentDateTime%]);
```

**Syntax Rules:**
- Variable assignment on left side (`$NewProduct =`)
- Entity type is fully qualified
- Attributes in parentheses, comma separated
- Closing `)` followed by semicolon
- Syntax aligned with CALL MICROFLOW/CALL JAVA ACTION

### CHANGE Object

```mdl
CHANGE $Product (
  Name = $NewName,
  ModifiedDate = [%CurrentDateTime%]);
```

**Note**: Only specify attributes you want to change. Syntax aligned with CREATE.

### COMMIT Object

```mdl
-- Commit without events
COMMIT $Product;

-- Commit with events (triggers event handlers)
COMMIT $Product WITH EVENTS;

-- Commit with refresh in client (updates UI after commit)
COMMIT $Product REFRESH;

-- Commit with events and refresh
COMMIT $Product WITH EVENTS REFRESH;
```

**Best Practice**: Use `WITH EVENTS` when you want before/after commit event handlers to execute. Use `REFRESH` when the committed object is displayed in the client and you want the UI to update immediately.

## Database Operations

### RETRIEVE Statement

```mdl
-- Retrieve all
RETRIEVE $ProductList FROM Test.Product;

-- Retrieve with WHERE
RETRIEVE $ProductList FROM Test.Product
  WHERE Code = $SearchCode;

-- Retrieve with multiple conditions
RETRIEVE $ProductList FROM Test.Product
  WHERE IsActive = true
    AND Price > 100;

-- Retrieve single object
RETRIEVE $Product FROM Test.Product
  WHERE Code = $ProductCode;
```

**Important**:
- Use `FROM Module.Entity` (fully qualified)
- RETRIEVE with `LIMIT 1` returns a **single entity**
- RETRIEVE without `LIMIT 1` returns a **list** (`List of Module.Entity`)
- Use `LIMIT 1` when you expect exactly one result (e.g., lookup by unique key)

## XPath Navigation

### Attribute Access

```mdl
-- Read attribute
DECLARE $ProductName String = $Product/Name;
DECLARE $Price Decimal = $Product/Price;

-- Write attribute (alternative to CHANGE)
SET $Product/Price = $NewPrice;
SET $Product/ModifiedDate = [%CurrentDateTime%];
```

### Association Navigation

```mdl
-- Navigate to related object
DECLARE $CustomerName String = $Order/Shop.Order_Customer/Name;
DECLARE $CategoryName String = $Product/Shop.Product_Category/Name;

-- Set association
SET $Order/Shop.Order_Customer = $Customer;
SET $Order/Shop.Order_Product = $Product;
```

**Critical**: Always use fully qualified association names (`Module.AssociationName`).

### XPath in Expressions

```mdl
-- Use in calculations
DECLARE $MonthlyTotal Decimal = $Product/MonthlyTotal;
DECLARE $DailyAverage Decimal = $MonthlyTotal div 30;

-- Use in conditions
IF $Product/IsActive THEN
  SET $Count = $Count + 1;
END IF;

-- Combine with operators
SET $TotalPrice = $Product/Price * $Quantity;
```

## Operators

### Arithmetic

```mdl
$Result = $A + $B;      -- Addition
$Result = $A - $B;      -- Subtraction
$Result = $A * $B;      -- Multiplication
$Result = $A div $B;    -- Division (use 'div', not '/')
```

**Important**: Use `div` for division, NOT `/`.

### Comparison

```mdl
$A = $B       -- Equals
$A != $B      -- Not equals
$A > $B       -- Greater than
$A >= $B      -- Greater than or equal
$A < $B       -- Less than
$A <= $B      -- Less than or equal
$A = empty    -- Check if empty/null
$A != empty   -- Check if not empty
```

### Boolean Logic

```mdl
$Result = $A AND $B;    -- Logical AND
$Result = $A OR $B;     -- Logical OR
$Result = NOT $A;       -- Logical NOT

-- Complex expressions
IF $IsActive AND $IsValid AND $HasStock THEN
  SET $CanProcess = true;
END IF;
```

## Logging

```mdl
-- Log levels
LOG INFO 'Information message';
LOG WARNING 'Warning message';
LOG ERROR 'Error message';

-- With node name
LOG INFO NODE 'OrderService' 'Processing order';
LOG WARNING NODE 'ValidationService' 'Invalid data detected';

-- With variables (use concatenation)
LOG INFO NODE 'OrderService' 'Order processed: ' + $OrderNumber;
LOG ERROR NODE 'Service' 'Error: ' + $ErrorMessage;
```

## Activity Annotations

Annotations use `@` prefix syntax placed before the activity they apply to:

```mdl
-- Canvas position (always shown in DESCRIBE output)
@position(200, 200)
COMMIT $Order WITH EVENTS;

-- Custom caption (overrides auto-generated caption)
@caption 'Save the order'
COMMIT $Order WITH EVENTS;

-- Background color (Blue, Green, Red, Yellow, Purple, Gray)
@color Green
LOG INFO NODE 'App' 'Success';

-- Visual note attached to the next activity (creates AnnotationFlow)
@annotation 'Validate the order before processing'
COMMIT $Order WITH EVENTS;

-- Multiple annotations stacked on a single activity
@position(400, 200)
@caption 'Persist product'
@color Blue
@annotation 'Step 2: Save to database'
COMMIT $Product;
```

**Rules:**
- `@annotation` before an activity attaches the note to that activity
- `@annotation` at the end (no following activity) creates a free-floating note
- Escape single quotes by doubling: `@annotation 'Don''t forget'`
- `@position` always appears in DESCRIBE output; `@caption` only when custom; `@color` only when not Default
- DESCRIBE MICROFLOW shows `@` annotations before their activities

## Special Values

```mdl
empty                      -- Null/empty value
[%CurrentDateTime%]        -- Current date/time
[%CurrentUser%]            -- Current user object
toString($Value)           -- Convert to string
randomInt($Max)            -- Random integer
```

## Complete Example

```mdl
CREATE MICROFLOW Shop.ProcessOrder (
  $OrderNumber: String
)
RETURNS Boolean AS $Success
COMMENT 'Process order with validation and status update'
BEGIN
  DECLARE $Success Boolean = false;
  DECLARE $Order Shop.Order;

  -- Find the order
  RETRIEVE $Order FROM Shop.Order
    WHERE OrderNumber = $OrderNumber;

  -- Validate order exists
  IF $Order = empty THEN
    LOG WARNING NODE 'OrderService' 'Order not found: ' + $OrderNumber;
    RETURN false;
  END IF;

  -- Validate customer association
  IF $Order/Shop.Order_Customer = empty THEN
    LOG ERROR NODE 'OrderService' 'Order has no customer';
    RETURN false;
  END IF;

  -- Update order status
  CHANGE $Order (
    Status = 'PROCESSING',
    ProcessedDate = [%CurrentDateTime%]);

  COMMIT $Order WITH EVENTS;

  -- Log success
  LOG INFO NODE 'OrderService' 'Order processed: ' + $OrderNumber;
  SET $Success = true;
  RETURN $Success;
END;
/
```

## Calling Microflows

### ✅ CORRECT Syntax

```mdl
-- Call with result assignment (no SET keyword)
$Result = CALL MICROFLOW Module.ProcessOrder(Order = $Order);

-- Call without result (void microflow)
CALL MICROFLOW Module.SendNotification(Message = $Message);

-- Call with error handling
$Result = CALL MICROFLOW Module.ExternalService(Data = $Data) ON ERROR CONTINUE;
```

### ❌ INCORRECT Syntax

```mdl
-- WRONG: Do NOT use SET with CALL MICROFLOW
SET $Result = CALL MICROFLOW Module.ProcessOrder(Order = $Order);  -- ERROR!

-- CORRECT: Direct variable assignment
$Result = CALL MICROFLOW Module.ProcessOrder(Order = $Order);
```

**Important**: The `SET` keyword is for changing existing variable values, NOT for capturing microflow return values. Use direct assignment (`$var = CALL MICROFLOW ...`).

### Parameter Name Matching

**CRITICAL**: Parameter names in `CALL MICROFLOW` must **exactly match** the parameter names declared in the target microflow's signature (without the `$` prefix). A mismatch causes a build error (MxBuild) but may fail silently at MDL execution time.

```mdl
-- Target microflow declaration:
CREATE MICROFLOW Module.SendEmail ($Recipient: String, $Subject: String)
BEGIN ... END;

-- CORRECT: parameter names match the declaration
CALL MICROFLOW Module.SendEmail(Recipient = $Email, Subject = $Title);

-- WRONG: parameter name does not match (EmailAddress vs Recipient)
CALL MICROFLOW Module.SendEmail(EmailAddress = $Email, Subject = $Title);  -- BUILD ERROR!
```

When calling microflows, always check the target's parameter list. Use `DESCRIBE MICROFLOW Module.Name` to see the exact parameter names.

## Page Navigation

### SHOW PAGE

```mdl
-- Open page with parameter (canonical syntax)
SHOW PAGE Module.EditPage($Product = $Product);

-- Widget-style syntax also accepted in microflows
SHOW PAGE Module.EditPage(Product: $Product);
```

Both `($Param = $value)` and `(Param: $value)` syntaxes are accepted in microflow SHOW PAGE statements. Similarly, widget Action: properties accept both `SHOW_PAGE Module.Page(Param: $value)` and `SHOW_PAGE Module.Page($Param = $value)`.

### CLOSE PAGE

```mdl
CLOSE PAGE;
```

### SHOW HOME PAGE

```mdl
SHOW HOME PAGE;
```

## Implicit Variable Creation (CE0111 Duplicate Variable)

These statements **implicitly create a new variable** with the name on the left side:

- `$Var = CALL MICROFLOW ...`
- `$Var = CALL JAVA ACTION ...`
- `$Var = CALL NANOFLOW ...`
- `$Var = CREATE Module.Entity (...)`
- `RETRIEVE $Var FROM Module.Entity ...`

**Do NOT use `DECLARE` before these** — it creates a duplicate variable (CE0111):

```mdl
-- WRONG: Duplicate variable — DECLARE + CALL both create $Result
DECLARE $Result Boolean = false;
$Result = CALL JAVA ACTION Module.DoSomething();  -- CE0111!

-- CORRECT: Let CALL create the variable, use a different name if you need a default
DECLARE $Success Boolean = false;
$CallResult = CALL JAVA ACTION Module.DoSomething();
SET $Success = $CallResult;

-- CORRECT: Simple pass-through (no default needed)
$Result = CALL JAVA ACTION Module.DoSomething();
RETURN $Result;
```

The same applies to RETRIEVE:

```mdl
-- WRONG: Duplicate variable
DECLARE $Items List OF Module.Entity = empty;
RETRIEVE $Items FROM Module.Entity WHERE Active = true;  -- CE0111!

-- CORRECT: Let RETRIEVE create the variable
RETRIEVE $Items FROM Module.Entity WHERE Active = true;
```

**Note**: `RETURNS Type AS $Var` in the microflow signature does NOT create an activity variable — it only names the return value. So `$Var = CALL JAVA ACTION ...` after `RETURNS AS $Var` is fine (one creation).

## REST Service Calls

MDL supports two patterns for calling REST APIs from microflows:

### SEND REST REQUEST — Consumed REST Service Operations

Calls an operation defined in a consumed REST service (created via `CREATE REST CLIENT`). The URL, headers, authentication, and response mapping are configured in the REST client document — the microflow only references the operation.

```mdl
-- Fire and forget (RESPONSE NONE operation)
SEND REST REQUEST Module.ServiceName.OperationName;

-- With output variable (RESPONSE JSON operation — maps to entity)
$Result = SEND REST REQUEST Module.ServiceName.OperationName;

-- With request body (POST/PUT operations)
$Result = SEND REST REQUEST Module.ServiceName.CreateItem
    BODY $NewItem;
```

**CRITICAL: `$latestHttpResponse` system variable**

After every `SEND REST REQUEST`, Mendix automatically populates `$latestHttpResponse` (type `System.HttpResponse`). Use this to check call success — do **NOT** check the output variable directly:

```mdl
-- ✅ CORRECT: check $latestHttpResponse
$RootResult = SEND REST REQUEST Module.Service.GetData;
IF $latestHttpResponse/Content != empty THEN
  -- Process $RootResult (the mapped entity)
END IF;

-- ❌ WRONG: checking the output variable directly causes CE0117
IF $RootResult != empty THEN  -- ERROR!
```

**Key attributes on `$latestHttpResponse`:**
- `Content` (String) — response body as string
- `StatusCode` (Integer) — HTTP status code (200, 404, etc.)

**Restrictions:**
- `SEND REST REQUEST` does **NOT** support custom error handling (`ON ERROR CONTINUE/ROLLBACK` causes CE6035). Errors are always handled by aborting.
- The operation must be defined via `CREATE REST CLIENT` with a three-part qualified name: `Module.ServiceDocument.OperationName`.

### REST CALL — Inline HTTP Calls

Direct HTTP call with URL, headers, auth, body, and response handling specified inline. Useful for one-off calls or when no REST client document exists.

```mdl
-- Simple GET returning string
$Response = REST CALL GET 'https://api.example.com/data'
    HEADER Accept = 'application/json'
    TIMEOUT 30
    RETURNS String;

-- POST with JSON body
$Response = REST CALL POST 'https://api.example.com/items'
    HEADER 'Content-Type' = 'application/json'
    HEADER Accept = 'application/json'
    BODY '{{"name": "{1}", "value": {2}}' WITH (
        {1} = $ItemName,
        {2} = toString($ItemValue)
    )
    TIMEOUT 30
    RETURNS String
    ON ERROR CONTINUE;

-- GET with URL template parameters
$Response = REST CALL GET 'https://api.example.com/users/{1}' WITH (
    {1} = toString($UserId)
)
    HEADER Accept = 'application/json'
    RETURNS String;

-- With basic authentication
$Response = REST CALL GET 'https://api.example.com/secure'
    HEADER Accept = 'application/json'
    AUTH BASIC $Username PASSWORD $Password
    TIMEOUT 30
    RETURNS String;

-- DELETE (no response)
REST CALL DELETE 'https://api.example.com/items/{1}' WITH (
    {1} = $ItemId
)
    RETURNS Nothing
    ON ERROR CONTINUE;
```

**REST CALL response types:**
- `RETURNS String` — response body as string variable
- `RETURNS Nothing` / `RETURNS None` — ignore response
- `RETURNS Response` — returns `System.HttpResponse` object
- `RETURNS MAPPING Module.ImportMapping AS Module.Entity` — import mapping

**REST CALL supports full error handling** (`ON ERROR CONTINUE`, `ON ERROR ROLLBACK`, custom error handlers).

## Error Handling

MDL supports error handling for activities that may fail (microflow calls, commits, external service calls, etc.).

### Error Handling Types

```mdl
-- ON ERROR CONTINUE: Ignore error and continue execution
CALL MICROFLOW Module.RiskyOperation() ON ERROR CONTINUE;

-- ON ERROR ROLLBACK: Rollback transaction and propagate error
COMMIT $Order WITH EVENTS ON ERROR ROLLBACK;

-- ON ERROR { ... }: Custom error handler with rollback
$Result = CALL MICROFLOW Module.ExternalService(Data = $Data) ON ERROR {
  LOG ERROR NODE 'ServiceError' 'External service failed';
  RETURN $DefaultResult;
};

-- ON ERROR WITHOUT ROLLBACK { ... }: Custom handler, keep changes
COMMIT $Order ON ERROR WITHOUT ROLLBACK {
  LOG WARNING NODE 'CommitError' 'Commit failed, using fallback';
  CHANGE $Order (Status = 'PENDING');
};
```

### Error Handling Semantics

| Syntax | Behavior |
|--------|----------|
| `ON ERROR CONTINUE` | Catch error silently, continue normal flow |
| `ON ERROR ROLLBACK` | Rollback database changes, propagate error |
| `ON ERROR { ... }` | Execute handler block, then continue (with rollback) |
| `ON ERROR WITHOUT ROLLBACK { ... }` | Execute handler block, keep database changes |

### When to Use Each Type

- **CONTINUE**: Non-critical operations where failure is acceptable
- **ROLLBACK**: Critical operations where data integrity must be preserved
- **Custom handlers**: When you need to log errors, set fallback values, or notify users

### Example: Robust External Call

```mdl
/**
 * Calls external service with error handling
 */
CREATE MICROFLOW Module.SafeExternalCall (
  $RequestData: String
)
RETURNS Module.Response AS $Response
BEGIN
  DECLARE $Response Module.Response;

  -- Try external call with custom error handler
  $Response = CALL MICROFLOW Module.CallExternalAPI(Data = $RequestData)
    ON ERROR WITHOUT ROLLBACK {
      LOG ERROR NODE 'ExternalAPI' 'API call failed for: ' + $RequestData;
      -- Create error response
      $Response = CREATE Module.Response (
        Success = false,
        Message = 'External service unavailable');
    };

  RETURN $Response;
END;
/
```

## UNSUPPORTED Syntax (Will Cause Parse Errors)

**CRITICAL**: The following syntax is NOT implemented and will cause parse errors. Do NOT use these patterns:

### ROLLBACK Statement (Supported!)

```mdl
-- CORRECT: ROLLBACK is now supported
ROLLBACK $Order;

-- With REFRESH to update client UI
ROLLBACK $Order REFRESH;
```

**Use Case**: Revert uncommitted changes to an object. Useful when validation fails and you want to restore the object to its database state.

### RETRIEVE with LIMIT (Supported!)

```mdl
-- CORRECT: LIMIT is supported
RETRIEVE $Product FROM Module.Product WHERE IsActive = true LIMIT 1;

-- LIMIT 1 returns a single entity (not a list)
-- Without LIMIT, returns a list
RETRIEVE $ProductList FROM Module.Product WHERE IsActive = true;
```

### WHILE Loop

```mdl
-- WHILE loops iterate while a condition is true
WHILE $Counter < 10
BEGIN
  SET $Counter = $Counter + 1;
END WHILE;

-- FOR EACH loops iterate over a list
LOOP $Item IN $ItemList
BEGIN
  -- Process each item
END LOOP;
```

### CASE/SWITCH Statement

```mdl
-- WRONG: CASE/SWITCH not supported
CASE $Status
  WHEN 'Active' THEN SET $Result = 1;
  WHEN 'Inactive' THEN SET $Result = 2;
  ELSE SET $Result = 0;
END CASE;

-- CORRECT: Use nested IF statements
IF $Status = 'Active' THEN
  SET $Result = 1;
ELSE
  IF $Status = 'Inactive' THEN
    SET $Result = 2;
  ELSE
    SET $Result = 0;
  END IF;
END IF;
```

### TRY/CATCH Block

```mdl
-- WRONG: TRY/CATCH not supported
TRY
  COMMIT $Order;
CATCH
  LOG ERROR 'Commit failed';
END TRY;

-- CORRECT: Use ON ERROR on specific activities
COMMIT $Order ON ERROR {
  LOG ERROR 'Commit failed';
};
```

### BREAK/CONTINUE in Loops

```mdl
-- WRONG: BREAK/CONTINUE not supported
LOOP $Item IN $ItemList
BEGIN
  IF $Item/Skip = true THEN
    CONTINUE;  -- NOT SUPPORTED
  END IF;
  IF $Item/Stop = true THEN
    BREAK;     -- NOT SUPPORTED
  END IF;
END LOOP;

-- CORRECT: Use conditional logic
LOOP $Item IN $ItemList
BEGIN
  IF $Item/Skip = false AND $Item/Stop = false THEN
    -- Process item
  END IF;
END LOOP;
```

### Reserved Words as Identifiers

**Best practice: Always quote all identifiers** (attribute names, parameter names, entity names) with double quotes. This eliminates all reserved keyword conflicts and is always safe — quotes are stripped automatically by the parser.

```mdl
CREATE PERSISTENT ENTITY Module."Item" (
  "Check": Boolean DEFAULT false,
  "Text": String(500),
  "Format": String(50),
  "Value": Decimal,
  "Create": DateTime,
  "Delete": DateTime
);
```

Quoted identifiers also work for microflow parameter names:
```mdl
CREATE MICROFLOW Module."Process" ("Select": String, "Type": Integer)
BEGIN
  LOG INFO 'Processing';
  RETURN;
END;
```

## Validation Checklist

Before executing a microflow script, verify:

- [ ] All entity types use `DECLARE $var AS EntityType` (not `EntityType = empty`)
- [ ] **All primitive variables are declared before SET** (`DECLARE $var Type = value;`)
- [ ] XPath association navigation uses qualified names (`Module.AssociationName`)
- [ ] All referenced attributes exist in entity definitions
- [ ] Every flow path ends with `RETURN`
- [ ] No code appears after `RETURN` statements
- [ ] Division uses `div` operator (not `/`)
- [ ] All entity/association names are fully qualified
- [ ] **CALL MICROFLOW parameter names exactly match target signature** (use `DESCRIBE MICROFLOW` to verify)
- [ ] Microflow ends with `/` separator
- [ ] Parameters start with `$` prefix
- [ ] Proper closing for control structures (`END IF`, `END LOOP`)

## Common Studio Pro Errors

| Error Code | Message | Fix |
|------------|---------|-----|
| CE0053 | Selected type is not allowed | Use `DECLARE $var EntityType;` (no AS, no = empty) |
| CE0117 | Error in expression | Use qualified association names |
| CE0104 | Action activity is unreachable | Remove code after RETURN |
| CE0105 | Must end with end event | Add RETURN statement |
| CE0008 | No action defined | Define action for activity |
| CW0094 | Variable never used | Remove unused variables or use them |
| MDL | Variable not declared | Use `DECLARE $var Type = value;` before SET |

## Tips for Success

1. **Always use fully qualified names**: `Module.Entity`, `Module.Association`
2. **Test incrementally**: Create simple microflows first, then add complexity
3. **Check entity definitions**: Ensure all attributes exist before referencing
4. **Use meaningful variable names**: `$Customer` not `$c`, `$ProductList` not `$list`
5. **Comment complex logic**: Use `--` for inline comments
6. **Log important events**: Help with debugging and auditing
7. **Handle empty cases**: Check for `= empty` before using objects
8. **Use WITH EVENTS appropriately**: Only when you need event handlers
9. **Validate before executing**: Use `mxcli check script.mdl -p app.mpr --references` to catch errors

## Related Documentation

- [MDL Syntax Guide](../../docs/02-features/mdl-syntax.md)
- [OQL Syntax Guide](../../docs/syntax-proposals/OQL_SYNTAX_GUIDE.md)
- [Microflow Examples](../../examples/doctype-tests/microflow-examples.mdl)
- [Mendix Microflow Documentation](https://docs.mendix.com/refguide/microflows/)

## Quick Reference

### Variable Declaration Pattern
```mdl
DECLARE $primitive Type = value;              -- Primitives
DECLARE $entity Module.Entity;                -- Entities (no AS, no = empty)
DECLARE $list List of Module.Entity = empty;  -- Lists
```

### Object Operation Pattern
```mdl
$var = CREATE Module.Entity (attr = value);
CHANGE $var (attr = value);
COMMIT $var [WITH EVENTS] [REFRESH];
```

### Flow Control Pattern
```mdl
IF condition THEN ... [ELSE ...] END IF;
LOOP $var IN $list BEGIN ... END LOOP;
RETURN $value;
```

### XPath Pattern
```mdl
$var/AttributeName                      -- Attribute
$var/Module.AssociationName             -- Association
$var/Module.AssociationName/Attribute   -- Chained
```

### Annotation Pattern
```mdl
@position(200, 200)
@caption 'Persist order'
@color Green
@annotation 'Note about the next activity'
COMMIT $Order;                                          -- Annotations apply here
```

### Execute Database Query Pattern
```mdl
-- Static query (3-part name: Module.Connection.Query)
$Results = EXECUTE DATABASE QUERY Module.Conn.QueryName;

-- Dynamic SQL override
$Results = EXECUTE DATABASE QUERY Module.Conn.QueryName
  DYNAMIC 'SELECT * FROM table LIMIT 10';

-- Parameterized query (names must match query PARAMETER definitions)
$Results = EXECUTE DATABASE QUERY Module.Conn.QueryName
  (paramName = $Variable);

-- Runtime connection override
$Results = EXECUTE DATABASE QUERY Module.Conn.QueryName
  CONNECTION (DBSource = $Url, DBUsername = $User, DBPassword = $Pass);

-- Fire-and-forget (no output variable)
EXECUTE DATABASE QUERY Module.Conn.QueryName;
```
**Note:** Only `ON ERROR ROLLBACK` is supported (the default). `ON ERROR CONTINUE` is not available for this action.

### Page Navigation Pattern
```mdl
SHOW PAGE Module.Page($Param = $value);               -- Canonical
SHOW PAGE Module.Page(Param: $value);                  -- Widget-style (also valid)
CLOSE PAGE;
SHOW HOME PAGE;
```

### Error Handling Pattern
```mdl
CALL MICROFLOW ... ON ERROR CONTINUE;                  -- Ignore error
CALL MICROFLOW ... ON ERROR ROLLBACK;                  -- Rollback on error
CALL MICROFLOW ... ON ERROR { LOG ...; RETURN ...; };  -- Custom handler
CALL MICROFLOW ... ON ERROR WITHOUT ROLLBACK { ... };  -- No rollback
```
