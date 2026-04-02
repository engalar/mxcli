# Customizing AI Generation

How to make AI assistants (Claude Code, OpenCode, Cursor, etc.) generate Mendix code that follows your team's conventions.

## How it works

When you run `mxcli init` in your Mendix project, it creates a `.claude/` folder with **skills** -- markdown files that teach AI assistants how to write MDL for your project. The AI reads these before generating code.

```
your-app/
  .claude/
    CLAUDE.md          # Project-level instructions
    skills/
      write-microflows.md
      create-page.md
      generate-domain-model.md
      check-syntax.md
      ...
    commands/
      mendix/
        explore.md
        create-entity.md
        ...
```

## Quick setup

```bash
# Initialize AI context in your project
mxcli init -p your-app.mpr

# This creates .claude/ with skills, commands, and a CLAUDE.md
```

Commit the `.claude/` folder to your repo so all team members get the same AI behavior.

## Customizing CLAUDE.md

The `CLAUDE.md` file is the **top-level instruction file** that AI assistants read first. Add your team's conventions here:

```markdown
# Project Instructions

## Naming Conventions
- Entity names: PascalCase, singular (Customer, not Customers)
- Microflow names: ACT_ prefix for button actions, SUB_ for sub-microflows
- Page names: P_ prefix followed by action and entity (P_Customer_Overview)
- Attribute names: PascalCase (FirstName, not first_name)

## Architecture Rules
- All database queries go through a DAL_ (Data Access Layer) microflow
- Never commit objects inside a loop
- Use sub-microflows for logic blocks over 10 activities
- Every entity must have a Name or Description attribute for display

## Security
- Default: no access. Explicitly grant access for each role.
- Admin role gets full access
- User role gets read-only on reference data

## Module Structure
- Core: domain model, enumerations, constants
- API: REST services, consumed services
- UI: pages, snippets, layouts
- Logic: microflows, nanoflows
```

## Writing custom skills

Skills are markdown files in `.claude/skills/` that provide focused guidance for specific tasks. The AI reads the relevant skill when performing that type of work.

### Example: Company-specific entity skill

Create `.claude/skills/create-entity.md`:

```markdown
# Creating Entities

## Standard attributes
Every persistent entity MUST include:
- CreatedDate: DateTime (set by before-commit microflow)
- ModifiedDate: DateTime (set by before-commit microflow)
- CreatedBy: Association to Administration.Account
- IsDeleted: Boolean DEFAULT false (soft delete pattern)

## Example
CREATE PERSISTENT ENTITY CRM.Customer (
  Name: String(200) NOT NULL,
  Email: String(200),
  Phone: String(50),
  IsActive: Boolean DEFAULT true,
  -- Standard audit fields (required)
  CreatedDate: DateTime,
  ModifiedDate: DateTime,
  IsDeleted: Boolean DEFAULT false
);

CREATE ASSOCIATION CRM.Customer_CreatedBy
FROM CRM.Customer TO Administration.Account
TYPE Reference
OWNER Default;

## Naming rules
- Singular names: Customer, not Customers
- No abbreviations: InvoiceLine, not InvLine
- Boolean attributes: Is/Has prefix (IsActive, HasAddress)
```

### Example: Microflow naming skill

Create `.claude/skills/microflow-naming.md`:

```markdown
# Microflow Naming Convention

## Prefixes
| Prefix | Use | Example |
|--------|-----|---------|
| ACT_ | Button/page actions | ACT_Customer_Save |
| SUB_ | Sub-microflows (called by others) | SUB_ValidateEmail |
| SCH_ | Scheduled events | SCH_CleanupExpiredSessions |
| DS_ | Data sources for widgets | DS_Customer_GetActive |
| DAL_ | Data access layer | DAL_Customer_FindByEmail |
| VAL_ | Validation rules | VAL_Customer_CheckRequired |
| BCO_ | Before commit | BCO_Customer_SetAuditFields |
| ACO_ | After commit | ACO_Order_SendConfirmation |
| BDE_ | Before delete | BDE_Customer_CheckReferences |

## Parameters
- Always use $ prefix: $Customer, $OrderList
- Entity parameters: use entity name ($Customer, not $Cust)
- List parameters: add List suffix ($CustomerList)
- Primitive parameters: descriptive name ($SearchQuery, $MaxResults)
```

### Example: Page patterns skill

Create `.claude/skills/page-patterns.md`:

```markdown
# Page Patterns

## Standard layouts
- Overview pages: DataGrid2 with search, pagination, new/edit buttons
- Edit pages: Popup layout with form fields and Save/Cancel footer
- Detail pages: Full-width layout with tabs for related data

## Widget standards
- Always set Label on input widgets
- Use COMBOBOX for enums with < 10 values, RADIOBUTTONS for < 5
- DataGrid2 for lists, not ListView (unless card layout needed)
- Always add AlternativeText on IMAGE widgets

## Example overview page
CREATE PAGE CRM.P_Customer_Overview (
  Title: 'Customer Overview',
  Layout: Atlas_Core.Atlas_Default
) {
  LAYOUTGRID lgMain {
    ROW row1 {
      COLUMN col1 (DesktopWidth: 12) {
        DATAGRID2 dgCustomers (
          DataSource: DATABASE CRM.Customer
            WHERE [IsDeleted = false]
            SORT BY Name ASC
        ) {
          ...columns...
        }
      }
    }
  }
}
```

## Using custom commands

Custom commands in `.claude/commands/` let users invoke common workflows with a slash command. Create `.claude/commands/new-entity.md`:

```markdown
Create a new entity with standard audit fields in module $ARGUMENTS:

1. Create the entity with Name, standard audit fields, and soft delete
2. Create the CreatedBy association
3. Create the BCO_ before-commit microflow that sets audit fields
4. Grant Admin full access, User read access
5. Validate with mxcli check
```

Then invoke with: `/new-entity CRM.Product`

## Enforcing conventions with linting

Custom Starlark linter rules in `.claude/lint-rules/` can automatically check conventions:

```python
# .claude/lint-rules/naming_prefix.star
RULE_ID = "CUSTOM001"
RULE_NAME = "Microflow Naming Prefix"
DESCRIPTION = "Microflows must use standard naming prefixes"
CATEGORY = "convention"
SEVERITY = "warning"

VALID_PREFIXES = ["ACT_", "SUB_", "SCH_", "DS_", "DAL_", "VAL_", "BCO_", "ACO_", "BDE_"]

def check():
    violations = []
    for mf in microflows():
        name = mf.name.split(".")[-1]  # Get name without module
        has_prefix = False
        for prefix in VALID_PREFIXES:
            if name.startswith(prefix):
                has_prefix = True
                break
        if not has_prefix:
            violations.append(violation(
                message="Microflow '{}' does not use a standard prefix ({})".format(
                    mf.qualified_name, ", ".join(VALID_PREFIXES)),
                location=location(module=mf.module_name, document_type="microflow",
                                  document_name=mf.name)
            ))
    return violations
```

Run with:
```bash
mxcli lint -p your-app.mpr
```

## Best practices

1. **Commit `.claude/` to git** -- everyone on the team gets the same AI behavior
2. **Start small** -- add one skill at a time, test it, then add more
3. **Use examples** -- one good example teaches more than paragraphs of rules
4. **Run `mxcli lint`** -- automated enforcement is better than hoping the AI remembers
5. **Update skills when conventions change** -- outdated skills cause inconsistency
6. **Use `mxcli check` after generation** -- always validate AI output before committing
