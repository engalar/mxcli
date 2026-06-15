# Prompt Template: Review Generated MDL

## When to Use

The AI has generated some MDL and you want to confirm its quality before executing.

## Template

```
Please review the following MDL code and check for issues:

**Code:**
[paste the MDL to review]

**Business requirements:**
[paste the corresponding business requirement description]

**Please check:**
1. Are the business rules complete (edge cases, status validation)?
2. Are the attribute types correct (do strings have a length? do booleans have a default?)?
3. Is the microflow missing a commit? Missing a return?
4. Are the association directions correct?
5. Are there potential null-pointer issues (e.g. accessing a property of an object that may be empty)?

For each issue found, please provide the corrected code.
```
