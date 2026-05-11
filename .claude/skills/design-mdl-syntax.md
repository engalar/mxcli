# Design New MDL Syntax

This skill provides guardrails for designing new MDL statements. Read this **before** writing grammar rules, AST types, or executor code for any new MDL feature.

## When to Use This Skill

- Adding a new document type to MDL (e.g., scheduled events, message definitions, REST services)
- Adding a new action type to microflows (e.g., new activity, new operation)
- Extending existing syntax with new clauses or keywords
- Reviewing a PR that adds or modifies MDL syntax
- Resolving a syntax design disagreement

## Core Principles (Priority Order)

When principles conflict, higher-priority ones win.

### 1. Read Like English

MDL targets citizen developers and business analysts, not software engineers. Statements should read as natural English sentences.

- Use keyword words (`from`, `where`, `in`), not symbols (`->`, `|>`, `=>`)
- Spell out full words (`microflow`, `association`), not abbreviations (`MF`, `ASSOC`)
- Use prepositions to clarify relationships: `grant read on entity to role`

**Test**: Read it aloud. A business analyst should understand on first hearing.

### 2. One Way to Do Each Thing

Reuse existing patterns. Never create a second syntax for the same concept.

| Operation | Pattern | Example |
|-----------|---------|---------|
| Create | `create [MODIFIERS] <type> Module.Name (...)` | `create persistent entity Shop.Product (...)` |
| Modify | `alter <type> Module.Name <operation>` | `alter entity Shop.Product add (...)` |
| Remove | `drop <type> Module.Name` | `drop entity Shop.Product` |
| List | `list <type>S [in module]` | `list entities in Shop` |
| Inspect | `describe <type> Module.Name` | `describe entity Shop.Product` |
| Security | `grant/revoke <perm> on <target> to/from <role>` | `grant read on Shop.Product to Shop.User` |

Do NOT use alternative verbs: `add` instead of `create`, `remove` instead of `drop`, `show` instead of `list`, `view` instead of `describe`. Note: `show` is the legacy verb — new commands use `list`.

### 3. Optimize for LLMs

- Keep patterns regular so one example is sufficient for generation
- Statements must be self-contained (no implicit state from prior statements)
- Use consistent keyword order: `<VERB> [MODIFIERS] <type> <NAME> [CLAUSES] [body]`
- Prefer flat statement sequences over deeply nested structures

### 4. Make Diffs Reviewable

- One property per line in multi-property constructs
- Allow trailing commas
- `describe` output uses deterministic property order
- Default values omitted unless non-obvious

### 5. Token Efficiency (Without Sacrificing Clarity)

- Omit noise words: `create entity` not `create A NEW entity`
- Support `or modify` to avoid check-then-create
- Allow type inference for obvious cases: `declare $count = 0`
- Do NOT use symbols to save tokens at the cost of readability

## Design Workflow

Follow these steps when designing syntax for a new MDL feature.

### Step 1: Check Existing Patterns

Read the MDL Quick Reference: `docs/01-project/MDL_QUICK_REFERENCE.md`

Does an existing pattern cover this? If yes, extend it. Don't invent new syntax.

```
New feature: "image collections"
Existing pattern: create/alter/drop/list/describe
design: create image collection Module.Name (...)
        describe image collection Module.Name
        list image COLLECTIONS [in module]
```

### Step 2: Pick the Statement Shape

Every MDL statement fits one of these shapes:

```
DDL:   <VERB> [MODIFIERS] <type> <QualifiedName> [CLAUSES] [body];
DML:   <action> <TARGET> [CLAUSES];
DQL:   <query-VERB> <type>S [FILTERS];
```

If your feature doesn't fit any shape, it may belong as a CLI command (`mxcli <subcommand>`) rather than MDL syntax.

### Step 3: Choose Keywords

1. Reuse existing keywords first (check reserved words in grammar)
2. Use SQL/DDL verbs: `create`, `alter`, `drop`, `show`, `describe`, `grant`, `revoke`, `set`
3. Use Mendix terminology: `entity` not `table`, `microflow` not `FUNCTION`, `page` not `view`
4. Prepositions clarify structure: `from`, `to`, `in`, `on`, `by`, `with`, `as`, `where`, `into`

### Step 4: Write the Property List

All property-bearing constructs use this format:

```mdl
create <type> Module.Name (
    Property1: value,
    Property2: value,
);
```

Rules:
- Parentheses `()` delimit property lists
- Colon `:` separates key from value
- Comma `,` separates properties
- Trailing comma allowed
- One property per line (single line acceptable for 1-2 properties)

#### Colon `:` vs `as` — When to Use Each

Use **colon** for property definitions (assigning a value to a named property):

```mdl
create entity Shop.Product (
    Name: string(200),          -- property: type/value
    Price: decimal,
);
textbox txtName (label: 'Name', attribute: title)
```

Use **`as`** for name-to-name mappings (renaming, aliasing, mapping one name to another):

```mdl
CUSTOM NAME map (
    'kvkNummer' as 'ChamberOfCommerceNumber',   -- old name AS new name
    'naam' as 'CompanyName',
)
alter entity Shop.Product rename Code as ProductCode   -- old attr AS new attr
```

**Rule of thumb**: if the left side is a *fixed property key* defined by the syntax, use `:`. If the left side is a *user-provided name* being mapped to another name, use `as`.

### Step 5: Validate

Run these checks before finalizing syntax design:

1. **Read aloud test** — Does it read as English? Can a business analyst understand it?
2. **LLM generation test** — Give one example to an LLM, ask for a variant. Does it get it right?
3. **Diff test** — Change one property. Is the diff exactly one line?
4. **Pattern test** — Does it follow CREATE/ALTER/DROP/SHOW/DESCRIBE? If not, why?
5. **Roundtrip test** — Can `describe` output be fed back as input?

## Anti-Patterns (DO NOT)

### Custom Verbs for Standard Operations

```mdl
-- WRONG: custom verb
SCHEDULE event Shop.Cleanup ...
REGISTER WEBHOOK Shop.OnOrder ...

-- RIGHT: standard CREATE
create SCHEDULED event Shop.Cleanup (...)
create WEBHOOK Shop.OnOrder (...)
```

### Implicit Module Context

```mdl
-- WRONG: implicit state
use module Shop;
create entity Customer (...);

-- RIGHT: explicit qualified name
create entity Shop.Customer (...);
```

### Symbolic Syntax

```mdl
-- WRONG: requires learning symbol meanings
$items |> filter($.active) |> map($.name)

-- RIGHT: keyword-based
filter $Items where Active = true
```

### Positional Arguments

```mdl
-- WRONG: meaning unclear without docs
create rule Shop Process Order ACT_ProcessOrder

-- RIGHT: labeled properties
create rule Shop.ProcessOrder (
    type: validation,
    microflow: Shop.ACT_ProcessOrder,
);
```

### Keyword Overloading

```mdl
-- CAUTION: SET already means variable assignment in microflows
-- Don't reuse it to mean property modification elsewhere unless established
```

## Checklist

Before merging any PR that adds new MDL syntax, verify:

- [ ] Follows `create`/`alter`/`drop`/`list`/`describe` pattern
- [ ] Uses `Module.Element` qualified names (no bare names)
- [ ] Property lists use `( key: value, ... )` format
- [ ] Keywords are full English words (no abbreviations)
- [ ] Statement reads as English (aloud test passed)
- [ ] One example sufficient for LLM generation
- [ ] Small change = one-line diff
- [ ] No new keyword overloading
- [ ] No implicit context dependency
- [ ] `describe` roundtrips to valid MDL
- [ ] Grammar regenerated (`make grammar`)
- [ ] Quick reference updated (`docs/01-project/MDL_QUICK_REFERENCE.md`)
- [ ] Full-stack wired: grammar, AST, visitor, executor, DESCRIBE

## Related Resources

- Full design rationale: `docs/11-proposals/PROPOSAL_mdl_syntax_design_guidelines.md`
- MDL Quick Reference: `docs/01-project/MDL_QUICK_REFERENCE.md`
- Implementation workflow: `.claude/skills/implement-mdl-feature.md`
- Existing syntax proposals: `docs/11-proposals/PROPOSAL_mdl_syntax_improvements.md`
- Grammar file: `mdl/grammar/MDLParser.g4`
