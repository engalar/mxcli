# Module 01: AI Collaboration Guide — Domain Modeling

## What This Module Produces

After running this module's reference implementation, your project will have:
- An HD module
- 2 enumerations (ticket status, ticket priority)
- 3 entities (Customer, Agent, Ticket) + 1 non-persistent entity (TicketSearch)
- 2 associations (Ticket→Customer, Ticket→Agent)
- 2 constants (SLA hours)

## Steps for Collaborating with Claude

### Step 1: Have Claude Read the Business Requirements

In Claude Code, enter:

```
Read academy/zh/01-领域建模/业务需求.md and help me design the Mendix domain model (entities, enumerations, associations), implemented in MDL
```

### Step 2: Confirm the Design Step by Step

Claude will propose a design. Before confirming, check:

1. Do the enumeration values cover all statuses and priorities mentioned in the requirements?
2. Are each entity's attributes complete? Are the types appropriate?
3. Are the association directions correct? (Ticket→Customer, Ticket→Agent)

If something is wrong, just tell Claude:

```
The ticket also needs a "resolved time" attribute, of type DateTime
```

### Step 3: Generate MDL and Validate

```bash
# Save Claude's generated MDL as my-domain.mdl, then:
mxcli check my-domain.mdl
mxcli exec  my-domain.mdl -p MyProject.mpr
~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr 2>&1 | grep -c "StorageLoadException"
# Expected: 0
```

### Step 4: Common Pitfalls

| Pitfall | Symptom | Solution |
|---------|---------|----------|
| Enum reference error | `mxcli check` reports "unknown type" | Make sure the enum is defined before the entity |
| Association direction reversed | mx check reports CE0XXX | from = the side that owns the foreign key (Ticket), to = the referenced side (Customer) |
| Wrong attribute type | mx check reports StorageLoadException | String types need a length: `string(200)`, you cannot write `string` directly |

## Reference Implementation

If you get stuck, look at `参考实现/domain-model.mdl`. Note: the entity syntax is `create or modify persistent entity` — do not write `create entity`.
