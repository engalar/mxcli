# Capabilities Overview

Everything mxcli can do, organized by use case.

## Read: Explore and understand your project

| Capability | Command | Notes |
|---|---|---|
| List modules | `SHOW MODULES` | With marketplace version info |
| List entities | `SHOW ENTITIES [IN Module]` | With attribute/association counts |
| List microflows | `SHOW MICROFLOWS [IN Module]` | With parameter/activity counts |
| List pages | `SHOW PAGES [IN Module]` | With widget counts |
| List enumerations | `SHOW ENUMERATIONS [IN Module]` | |
| List constants | `SHOW CONSTANTS [IN Module]` | |
| List workflows | `SHOW WORKFLOWS [IN Module]` | |
| List nanoflows | `SHOW NANOFLOWS [IN Module]` | |
| List layouts | `SHOW LAYOUTS [IN Module]` | |
| List snippets | `SHOW SNIPPETS [IN Module]` | |
| Compact overview | `SHOW STRUCTURE [DEPTH 1\|2\|3]` | Tree view of entire project |
| Describe any document | `DESCRIBE ENTITY\|MICROFLOW\|PAGE ...` | Full MDL output (re-executable) |
| Full-text search | `SEARCH 'keyword'` | Across all strings and source |
| Show languages | `SHOW LANGUAGES` | All languages in the project |
| Show project security | `SHOW PROJECT SECURITY` | Security overview |
| Show access rules | `SHOW ACCESS ON Module.Entity` | Entity/microflow/page access |
| Show settings | `SHOW SETTINGS` | Project-level settings |

## Write: Create and modify documents

### Domain Model

| Capability | Command | Status |
|---|---|---|
| Create entity | `CREATE PERSISTENT ENTITY` | Full support |
| Create non-persistent entity | `CREATE NON-PERSISTENT ENTITY` | Full support |
| Create view entity (OQL) | `CREATE VIEW ENTITY ... AS SELECT` | 10.18+ |
| Modify entity | `ALTER ENTITY ... ADD/MODIFY/DROP ATTRIBUTE` | Full support |
| Create association | `CREATE ASSOCIATION ... FROM ... TO` | Full support |
| Create enumeration | `CREATE ENUMERATION` | Full support |
| Create/modify idempotently | `CREATE OR MODIFY ENTITY` | Full support |

### Microflows and Nanoflows

| Capability | Command | Status |
|---|---|---|
| Create microflow | `CREATE MICROFLOW ... BEGIN ... END` | 60+ activity types |
| Create nanoflow | `CREATE NANOFLOW ... BEGIN ... END` | Client-side flows |
| CRUD operations | `CREATE`, `CHANGE`, `COMMIT`, `DELETE` | Object manipulation |
| Retrieve | `RETRIEVE ... FROM ... WHERE` | Database/association queries |
| Control flow | `IF/THEN/ELSE`, `LOOP`, `WHILE` | Including nested |
| Call flows | `CALL MICROFLOW`, `CALL NANOFLOW` | With parameters |
| Show page | `SHOW PAGE Module.Page(...)` | With page parameters (11.0+) |
| REST requests | `SEND REST REQUEST` | GET/POST/PUT/DELETE |
| Database queries | `EXECUTE DATABASE QUERY` | External databases |
| Log messages | `LOG INFO\|WARNING\|ERROR` | With templates |
| Error handling | `RAISE ERROR`, error handlers | On activities |
| Validation | `VALIDATION FEEDBACK` | Field-level validation |

### Pages

| Capability | Command | Status |
|---|---|---|
| Create page | `CREATE PAGE ... { widgets }` | 50+ widget types |
| Create snippet | `CREATE SNIPPET` | Reusable widget fragments |
| Modify page | `ALTER PAGE ... SET/INSERT/DROP/REPLACE` | In-place modifications |
| Built-in widgets | TEXTBOX, TEXTAREA, DATEPICKER, etc. | All standard widgets |
| Layout widgets | LAYOUTGRID, CONTAINER, GROUPBOX | With responsive columns |
| Display widgets | DYNAMICTEXT, STATICTEXT, IMAGE | Including pluggable Image |
| Data widgets | DATAVIEW, LISTVIEW | With datasource binding |
| Pluggable widgets | DATAGRID2, GALLERY, COMBOBOX | Template-based |
| Action buttons | ACTIONBUTTON | Save, cancel, microflow, page |
| Navigation | NAVIGATIONLIST | With items and actions |

### Security

| Capability | Command | Status |
|---|---|---|
| Create module roles | `GRANT ... TO Role` | Entity/microflow/page access |
| Create user roles | via security commands | Map to module roles |
| Entity access rules | `GRANT READ/WRITE ON Entity TO Role` | With XPath constraints |
| Demo users | `CREATE DEMO USER` | For testing |

### Integration

| Capability | Command | Status |
|---|---|---|
| REST clients | `CREATE REST CLIENT` | Consumed REST services |
| Business events | `CREATE BUSINESS EVENT SERVICE` | Event definitions |
| Database connections | `CREATE DATABASE CONNECTION` | External SQL |
| OData services | `SHOW ODATA CLIENTS/SERVICES` | Read-only browsing |

### Navigation and Settings

| Capability | Command | Status |
|---|---|---|
| Navigation profiles | `ALTER NAVIGATION` | Home pages, menus |
| Project settings | `ALTER SETTINGS` | Java version, theme, etc. |

### Workflows

| Capability | Command | Status |
|---|---|---|
| Create workflow | `CREATE WORKFLOW` | User tasks, decisions |
| Parallel splits | Supported | Fork/join patterns |

## Analyze: Code intelligence and quality

| Capability | Command | Notes |
|---|---|---|
| Cross-references | `SHOW CALLERS/CALLEES OF` | Who calls what |
| Impact analysis | `SHOW IMPACT OF Module.Entity` | What breaks if I change this |
| Transitive callers | `SHOW CALLERS OF ... TRANSITIVE` | Full call chain |
| Linting | `mxcli lint -p app.mpr` | 14 built-in + 27 Starlark rules |
| Best practices report | `mxcli report -p app.mpr` | Scored report with categories |
| Missing translations | QUAL005 linter rule | Detects incomplete translations |
| Catalog queries | `SELECT ... FROM CATALOG.tables` | SQL over project metadata |
| Validation | `mxcli check -p app.mpr` | Syntax + optional `mx check` |
| Diff | `mxcli diff -p app.mpr changes.mdl` | Compare script vs project |

## Automate: CI/CD and scripting

| Capability | Command | Notes |
|---|---|---|
| Batch execution | `mxcli -p app.mpr -c "..."` | Non-interactive |
| Script files | `mxcli exec script.mdl -p app.mpr` | Run MDL files |
| Docker build | `mxcli docker build -p app.mpr` | Build MDA in container |
| Docker check | `mxcli docker check -p app.mpr` | Validate in container |
| Testing | `mxcli test tests/ -p app.mpr` | `.test.mdl` / `.test.md` |
| SARIF output | `mxcli lint --format sarif` | For CI integration |
| Init project | `mxcli init` | Set up `.claude/` with skills |

## Known Limitations

| Area | Limitation | Workaround |
|---|---|---|
| Page parameters | Requires Mendix 11.0+ | Use non-persistent entity pattern on 10.x |
| Design properties (Atlas v3) | Requires Mendix 11.0+ | Use CSS classes on 10.x |
| REST query parameters | Requires Mendix 11.0+ | Build query string manually on 10.x |
| Pluggable widget ImageUrl mode | Cannot set imageUrl from MDL | Configure in Studio Pro |
| Concurrent editing | Not supported | Close Studio Pro before mxcli writes |
| Widget template drift | CE0463 on version mismatch | MPK augmentation handles most cases |
| 47 of 52 metamodel domains | Not yet implemented | REST, OData write, etc. pending |

## Version Compatibility

See [Version Compatibility](../appendixes/version-compatibility.md) for detailed per-version feature availability.
