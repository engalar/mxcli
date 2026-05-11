# CREATE JAVA ACTION

## Synopsis

```sql
CREATE [ OR MODIFY ] JAVA ACTION module.Name ( parameters )
    RETURNS type
    [ EXPOSED AS 'caption' IN 'category' ]
    AS $$ java_code $$

DROP JAVA ACTION module.Name
```

## Description

Creates a Java action with inline Java code. Java actions allow custom server-side logic written in Java to be called from microflows.

The action parameters, return type, and Java body are all specified inline. When the action is created, the corresponding Java source file is generated with the parameter boilerplate and the provided code body.

> **The `AS $$ ... $$` body is mandatory.** Unlike microflows, Java actions require a code body in every `CREATE` statement. If the body is omitted the parser reports `no viable alternative at input '...'`. Use a minimal stub body if the implementation is not yet written: `AS $$ return false; $$`.

If `OR MODIFY` is specified and the Java action already exists, its parameters, return type, exposed-as settings, and Java body are updated in place. The UUID and any existing Java source file path are preserved.

### Type Parameters

Java actions support generic entity handling through type parameters. A parameter declared as `ENTITY <pEntity>` creates a type parameter selector -- the caller chooses which entity type to use. Other parameters declared as bare `pEntity` (without `ENTITY`) receive instances of the selected entity type.

### Exposed Actions

The optional `EXPOSED AS` clause makes the action visible in the Studio Pro toolbox under the specified category, allowing modelers to use it directly in microflows.

## Parameters

`module.Name`
:   The qualified name of the Java action.

`parameters`
:   Comma-separated parameter declarations. Each parameter has a name, colon, and type. Supported types:
    - Primitives: `String`, `Integer`, `Long`, `Decimal`, `Boolean`, `DateTime`
    - Entity: `Module.EntityName`
    - List: `List of Module.EntityName`
    - Enumeration: `ENUM Module.EnumName` or `Enumeration(Module.EnumName)`
    - String template: `StringTemplate(Sql)`, `StringTemplate(Oql)`
    - Type parameter declaration: `ENTITY <pEntity>` (declares the type parameter)
    - Type parameter reference: bare `pEntity` (uses the declared type parameter)
    - Add `NOT NULL` after the type to mark a parameter as required.

`RETURNS type`
:   The return type. Same type options as parameters.

`EXPOSED AS 'caption' IN 'category'`
:   Optional. Makes the action visible in the toolbox with the given caption and category.

`AS $$ java_code $$`
:   The Java code body, enclosed in `$$` delimiters. **Mandatory** — the body cannot be omitted.

`OR MODIFY`
:   Makes the statement idempotent. If the Java action already exists, updates its parameters, return type, exposed-as settings, and body. Without this clause, creating a duplicate Java action is an error.

## Examples

Simple Java action returning a string:

```sql
CREATE JAVA ACTION MyModule.JA_FormatCurrency (
    Amount: Decimal NOT NULL,
    CurrencyCode: String NOT NULL
) RETURNS String
AS $$
    java.text.NumberFormat formatter = java.text.NumberFormat.getCurrencyInstance();
    formatter.setCurrency(java.util.Currency.getInstance(CurrencyCode));
    return formatter.format(Amount);
$$;
```

Java action with type parameters for generic entity handling:

```sql
CREATE JAVA ACTION MyModule.JA_Validate (
    EntityType: ENTITY <pEntity> NOT NULL,
    InputObject: pEntity NOT NULL
) RETURNS Boolean
EXPOSED AS 'Validate Entity' IN 'Validation'
AS $$
    return InputObject != null;
$$;
```

Java action with a list parameter:

```sql
CREATE JAVA ACTION MyModule.JA_ExportToCsv (
    Records: List of MyModule.Customer NOT NULL,
    FilePath: String NOT NULL
) RETURNS Boolean
AS $$
    // Export logic here
    return true;
$$;
```

Drop a Java action:

```sql
DROP JAVA ACTION MyModule.JA_FormatCurrency;
```

### Idempotent upsert with OR MODIFY

```sql
-- Initial creation
CREATE JAVA ACTION MyModule.JA_FormatCurrency (
    Amount: Decimal NOT NULL,
    CurrencyCode: String NOT NULL
) RETURNS String
AS $$
    return Amount.toString();
$$;

-- Later: update to add proper formatting (UUID preserved)
CREATE OR MODIFY JAVA ACTION MyModule.JA_FormatCurrency (
    Amount: Decimal NOT NULL,
    CurrencyCode: String NOT NULL
) RETURNS String
AS $$
    java.text.NumberFormat formatter = java.text.NumberFormat.getCurrencyInstance();
    formatter.setCurrency(java.util.Currency.getInstance(CurrencyCode));
    return formatter.format(Amount);
$$;
```

### Minimal stub body

When a Java action needs to exist in the project but the implementation is not yet written:

```sql
CREATE JAVA ACTION MyModule.JA_Placeholder (
    Input: String NOT NULL
) RETURNS Boolean
AS $$ return false; $$;
```

## See Also

[CREATE MICROFLOW](create-microflow.md), [SHOW JAVA ACTIONS](/reference/query/show-microflows.md)
