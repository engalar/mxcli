# Expression Checker Hint Reference

Generated from `mdl/exprcheck/hints/registry.go`. Do not edit by hand.

## E001 — enum-string-mismatch (error)

**When this appears:** Your MDL has a comparison or assignment where one side is an Enumeration attribute (or Enumeration parameter) and the other side is a quoted string literal.

**Why it's wrong:** Mendix expressions cannot compare an Enumeration value to a String. The comparison would always be false at runtime, or trigger CE0109 in Studio Pro.

**How to fix:** Replace the string literal with the fully-qualified enumeration value: 'NewAlert' → FraudDetection.AlertStatus.NewAlert

**Examples:**

*CREATE/CHANGE assignment:*

```mdl
CHANGE $Alert (Status = 'NewAlert')   -- wrong
CHANGE $Alert (Status = FraudDetection.AlertStatus.NewAlert)   -- right
```

*IF condition:*

```mdl
IF $Alert/Status = 'NewAlert' THEN ...   -- wrong
IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...   -- right
```

*CALL parameter:*

```mdl
CALL Mf($Status = 'Validated')   -- wrong
CALL Mf($Status = FraudDetection.AlertStatus.Validated)   -- right
```

## E002 — bool-string-mismatch (error)

**When this appears:** A Boolean attribute is compared against a string literal like 'true' or 'false'.

**Why it's wrong:** Mendix Boolean expressions use the unquoted literals true and false. Comparing a Boolean against a string is always false.

**How to fix:** Replace 'true'/'false' with the unquoted literals true/false.

**Examples:**

*IF condition:*

```mdl
IF $Config/IsActive = 'true' THEN ...   -- wrong
IF $Config/IsActive = true THEN ...   -- right
```

## E003 — null-to-empty (warning)

**When this appears:** The keyword null is used in a Mendix expression.

**Why it's wrong:** Mendix expressions use empty, not null. Tools auto-correct on write but the source becomes inconsistent on the next round-trip.

**How to fix:** Replace null with empty.

**Examples:**

```mdl
IF $Alert = null THEN ...   -- wrong
IF $Alert = empty THEN ...   -- right
```

## E004 — concat-type (error)

**When this appears:** The '+' operator is used between values of incompatible kinds (e.g. String and Integer).

**Why it's wrong:** '+' concatenates Strings only. Mixing kinds raises CE0109 in Studio Pro.

**How to fix:** Wrap the non-String operand in toString().

**Examples:**

*$n is Integer:*

```mdl
'count=' + $n   -- wrong
'count=' + toString($n)   -- right
```

## E005 — func-arg-type (error)

**When this appears:** A built-in function received an argument of the wrong kind.

**Why it's wrong:** Built-in functions have fixed argument signatures.

**How to fix:** Cast the argument to the expected kind, e.g. wrap with toString() or toInteger().

**Examples:**

*RiskScore is Decimal; length expects String:*

```mdl
length($Alert/RiskScore)   -- wrong
length(toString($Alert/RiskScore))   -- right
```

## E006 — func-arg-arity (error)

**When this appears:** A built-in function was called with the wrong number of arguments.

**Why it's wrong:** Each built-in expects a fixed number of arguments.

**How to fix:** Provide the exact number of arguments listed in the function signature.

**Examples:**

```mdl
substring('hello')   -- wrong
substring('hello', 0, 3)   -- right
```

## E007 — unknown-token (warning)

**When this appears:** The parser encountered tokens it does not recognise as a valid Mendix expression and skipped to the next safe boundary.

**Why it's wrong:** The unrecognised text is not part of the Mendix expression grammar — typos, foreign characters, or stray punctuation usually cause this.

**How to fix:** Replace the unrecognised fragment with a valid expression: a literal, a variable, a function call, or a qualified name.

**Examples:**

*argument of length():*

```mdl
SET $msg = 'count=' + length(@@@broken@@@) + ' items';   -- wrong
SET $msg = 'count=' + toString(length($list)) + ' items';   -- right
```

## E008 — enum-missing-module (error)

**When this appears:** An enum value was written without its module prefix.

**Why it's wrong:** Mendix requires fully-qualified Module.Enum.Value references.

**How to fix:** Add the module prefix.

**Examples:**

```mdl
$Status = AlertStatus.NewAlert   -- wrong
$Status = FraudDetection.AlertStatus.NewAlert   -- right
```

## E009 — slot-type-mismatch (error)

**When this appears:** An expression's inferred kind does not match the slot's expected kind (catch-all).

**Why it's wrong:** The surrounding statement requires a specific kind (Boolean for IF condition, Integer for LIMIT, etc.).

**How to fix:** Adjust the expression so its result matches the slot's expected kind.

**Examples:**

```mdl
IF 'active' THEN ...   -- wrong
IF $obj/IsActive THEN ...   -- right
```

## E010 — attribute-not-found (error)

**When this appears:** An attribute path references an attribute that does not exist on the entity.

**Why it's wrong:** Catalog lookup confirmed the entity does not have the requested attribute.

**How to fix:** Use the correct attribute name from the entity definition.

**Examples:**

```mdl
$Customer/EmialAddress   -- wrong
$Customer/EmailAddress   -- right
```

## E011 — not-missing-parens (error)

**When this appears:** The 'not' keyword is used without parentheses around its operand.

**Why it's wrong:** Mendix expression syntax requires not(expr) — 'not expr' without parentheses is rejected by Studio Pro with CE0117.

**How to fix:** Wrap the operand in parentheses: not(expr).

**Examples:**

```mdl
not $Validation/IsValid   -- wrong
not($Validation/IsValid)   -- right
```

```mdl
not isMatch($Value, '^[0-9]+$')   -- wrong
not(isMatch($Value, '^[0-9]+$'))   -- right
```

```mdl
$x != empty and not contains($s, '@')   -- wrong
$x != empty and not(contains($s, '@'))   -- right
```

## E012 — id-attribute-illegal (error)

**When this appears:** The path '$Object/id' is used in a microflow expression or MDL SET statement.

**Why it's wrong:** Mendix reserves 'id' as a system attribute name — it cannot be accessed via '$Object/id' in microflow expressions. It is only valid in XPath constraints (e.g. '[id = $Variable]'), not in expressions.

**How to fix:** Option A (preferred): change the microflow return type to the entity object itself instead of Long, and let callers use the object directly.
Option B: add an AutoNumber attribute to the entity (e.g. 'WorkHistoryNo') and return '$Object/WorkHistoryNo' instead.

**Examples:**

*Option A — return the object:*

```mdl
SET $Id = $WorkHistory/id;   -- wrong
RETURN $WorkHistory;  -- change RETURNS type to the entity   -- right
```

*Option B — use a dedicated AutoNumber attribute:*

```mdl
SET $Id = $WorkHistory/id;   -- wrong
SET $Id = $WorkHistory/WorkHistoryNo;  -- WorkHistoryNo is AutoNumber   -- right
```

