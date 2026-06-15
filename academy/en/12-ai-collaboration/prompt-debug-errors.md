# Prompt Template: Debug an mxcli Error

## When to Use

When running `mxcli exec` or `mxcli check` reports an error.

## Template

```
I ran into an error while running mxcli. Please help me analyze the cause and the fix.

**Command executed:**
mxcli exec my-file.mdl -p app.mpr

**Error message:**
[paste the full error output, including line numbers]

**Relevant MDL code (the part that errored):**
[paste the code around the failing line, 5 lines above and below]

**My expectation:**
[describe what this code is supposed to do]

Please:
1. Explain the cause of the error
2. Provide the corrected code
3. If there is a similar gotcha, tell me how to avoid it next time
```

## Common Error Types Quick Reference

| Error Keyword | Usual Cause |
|---------------|-------------|
| `unknown type` | An enumeration or entity name is misspelled, or not yet created |
| `StorageLoadException` | BSON structure error, usually a mismatched attribute type |
| `association not found` | Wrong association name, or the association direction is reversed |
| `parse error` | MDL syntax error (mismatched parentheses/semicolons/quotes) |
| `CE0463` | Widget properties don't match the def.json definition |
```
