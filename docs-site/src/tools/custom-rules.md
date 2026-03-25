# Writing Custom Rules

You can extend the linting framework by writing custom rules in Starlark, a Python-like language. Custom rules are placed in the `.claude/lint-rules/` directory and are automatically picked up by `mxcli lint`.

## Rule File Structure

Each Starlark rule file defines a rule with metadata and a check function. Place your rule files in `.claude/lint-rules/` with a `.star` extension.

## Getting Started

1. Create a new `.star` file in `.claude/lint-rules/`
2. Define the rule metadata (ID, name, description, severity)
3. Implement the check function that inspects project elements
4. Run `mxcli lint -p app.mpr` to test your rule

## Example: Custom Naming Rule

```python
# .claude/lint-rules/custom001_entity_prefix.star

rule_id = "CUSTOM001"
rule_name = "Entity Prefix Convention"
rule_description = "All entities should be prefixed with their module abbreviation"
severity = "warning"

def check(context):
    findings = []
    for entity in context.entities:
        # Check if entity name starts with module prefix
        module_prefix = entity.module_name[:3]
        if not entity.name.startswith(module_prefix):
            findings.append({
                "message": "Entity '%s' should start with module prefix '%s'" % (entity.name, module_prefix),
                "element": entity.qualified_name,
            })
    return findings
```

## Rule Severity Levels

| Severity | Description |
|----------|-------------|
| `error` | Must be fixed; blocks CI pipelines |
| `warning` | Should be fixed; potential issue |
| `info` | Informational; suggestion for improvement |

## Testing Custom Rules

```bash
# Run lint to test your custom rule
mxcli lint -p app.mpr

# List rules to verify your rule is detected
mxcli lint -p app.mpr --list-rules
```

## Best Practices

- Use a unique rule ID prefix (e.g., `CUSTOM001`) to avoid conflicts with built-in rules
- Include clear, actionable messages that explain what to fix
- Test rules against a real project before deploying
- Keep rules focused on a single concern
