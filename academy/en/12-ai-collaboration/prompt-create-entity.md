# Prompt Template: Create an Entity

## When to Use

When you need to create a new data entity in a Mendix project.

## Template

```
I have a Mendix project that uses MDL (Mendix Definition Language).

Current context:
[paste the output of `mxcli -p app.mpr -c "show structure"`, or describe the existing modules]

I need to create a new entity, with the following business requirements:
[paste the business requirement description]

Please:
1. First list the entity attributes as you understand them (type + constraints + default value)
2. List the associations needed (direction + many-to-one/many-to-many + nullable or not)
3. Then generate the complete MDL

Requirements:
- Use the `create or modify persistent entity` syntax
- string types must specify a length (e.g. string(200))
- boolean attributes must have a default value
- enumeration attributes use the `ModuleName.EnumName default Value` format
- association format: `create or modify association X from A to B type reference owner default;`
```

## Tips

- Provide enough information in "Current context" to avoid the AI generating code that conflicts with existing modules
- Keep step 1 (list attributes) and step 2 (generate code) separate, so you can confirm the design before executing
