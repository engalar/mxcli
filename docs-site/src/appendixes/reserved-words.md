# Reserved Words

Complete reference for MDL reserved words and quoting rules.

## Quoting Rules

Most MDL keywords now work **unquoted** as entity names, attribute names, parameter names, and module names. Common words like `Caption`, `Check`, `Content`, `Format`, `Index`, `Label`, `Range`, `Select`, `Source`, `Status`, `Text`, `Title`, `Type`, `Value`, `Item`, `Version`, `Production`, etc. are all valid without quoting.

Only structural MDL keywords require quoting: `Create`, `Delete`, `Begin`, `End`, `Return`, `Entity`, `Module`.

## Quoted Identifiers

**Quoted identifiers** escape any reserved word using double-quotes (ANSI SQL style) or backticks (MySQL style):

```sql
DESCRIBE ENTITY "ComboBox"."CategoryTreeVE";
SHOW ENTITIES IN "ComboBox";
CREATE PERSISTENT ENTITY Module.VATRate ("Create": DateTime, Rate: Decimal);
```

Both double-quote and backtick styles are supported. You can mix quoted and unquoted parts: `"ComboBox".CategoryTreeVE`.

## Words That Do NOT Require Quoting

The following common words can be used as identifiers without quoting:

| Category | Words |
|----------|-------|
| UI terms | `Caption`, `Content`, `Label`, `Text`, `Title`, `Format`, `Style` |
| Data terms | `Index`, `Range`, `Select`, `Source`, `Status`, `Type`, `Value`, `Item`, `Version` |
| Security terms | `Production` |
| Other | `Check`, and most domain-specific terms |

## Words That Require Quoting

The following structural MDL keywords **must** be quoted when used as identifiers:

| Keyword | Example with quoting |
|---------|---------------------|
| `Create` | `"Create": DateTime` |
| `Delete` | `"Delete": Boolean` |
| `Begin` | `"Begin": DateTime` |
| `End` | `"End": DateTime` |
| `Return` | `"Return": String(200)` |
| `Entity` | `"Entity": String(200)` |
| `Module` | `"Module": String(200)` |

## Boolean Attribute Defaults

Boolean attributes auto-default to `false` when no `DEFAULT` is specified. Mendix Studio Pro requires Boolean attributes to have a default value.

## CALCULATED Attributes

`CALCULATED` marks an attribute as calculated (not stored). Use `CALCULATED BY Module.Microflow` to specify the calculation microflow. Calculated attributes derive their value from a microflow at runtime.

## ButtonStyle Values

`ButtonStyle` supports all values: `Primary`, `Default`, `Success`, `Danger`, `Warning`, `Info`.
