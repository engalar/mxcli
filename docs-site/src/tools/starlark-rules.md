# Starlark Rules

In addition to the built-in Go rules, mxcli bundles 27 Starlark-based lint rules. Starlark is a Python-like language that allows rules to be extended and customized without recompiling mxcli.

## Bundled Starlark Rules

### Security Rules (SEC004-SEC009)

| Rule | Description |
|------|-------------|
| **SEC004** | Guest access -- Warns about overly permissive guest/anonymous access |
| **SEC005** | Strict mode -- Checks for strict security mode settings |
| **SEC006** | PII exposure -- Detects potentially sensitive data without restricted access |
| **SEC007** | Anonymous access -- Flags entities accessible to anonymous users |
| **SEC008** | Member restrictions -- Checks for overly broad member-level access |
| **SEC009** | Additional security rules |

### Architecture Rules (ARCH001-ARCH003)

| Rule | Description |
|------|-------------|
| **ARCH001** | Cross-module data -- Detects tight coupling between modules via direct data access |
| **ARCH002** | Microflow-based writes -- Ensures data modifications go through microflows |
| **ARCH003** | Entity business keys -- Checks that entities have meaningful business key attributes |

### Quality Rules (QUAL001-QUAL004)

| Rule | Description |
|------|-------------|
| **QUAL001** | McCabe complexity -- Flags microflows with high cyclomatic complexity |
| **QUAL002** | Documentation -- Checks for missing documentation on public elements |
| **QUAL003** | Long microflows -- Warns about microflows with too many activities |
| **QUAL004** | Orphaned elements -- Detects unused entities, microflows, or pages |

### Design Rules (DESIGN001)

| Rule | Description |
|------|-------------|
| **DESIGN001** | Entity attribute count -- Warns when entities have too many attributes |

### Convention Rules (CONV001-CONV010, CONV015-CONV017)

| Rule | Description |
|------|-------------|
| **CONV001-CONV010** | Best practice conventions including boolean naming, page suffixes, enumeration prefixes, snippet prefixes |
| **CONV015** | Validation rules -- Checks for consistent validation patterns |
| **CONV016** | Event handlers -- Validates event handler configuration |
| **CONV017** | Calculated attributes -- Checks calculated attribute patterns |

Additional convention rules cover access rule constraints, role mapping, microflow size and content.

## Where Starlark Rules Live

When you run `mxcli init`, Starlark rules are installed to:

```
your-project/
└── .claude/
    └── lint-rules/
        ├── sec004_guest_access.star
        ├── arch001_cross_module.star
        ├── qual001_complexity.star
        └── ...
```

## Running Starlark Rules

Starlark rules run automatically alongside built-in rules:

```bash
mxcli lint -p app.mpr
```

Use `--list-rules` to see all available rules:

```bash
mxcli lint -p app.mpr --list-rules
```
