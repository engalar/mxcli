# Proposals Index

Status of all mxcli feature proposals. See [archive/](archive/) for fully implemented proposals.

## Status Legend

| Status | Meaning |
|--------|---------|
| Done | Fully implemented |
| Partial | Some phases/features implemented |
| Proposed | Design complete, not yet started |
| Draft | Early design, not finalized |
| Reference | Analysis or reference document (not actionable) |
| Superseded | Replaced by a newer proposal |

---

## Active Proposals

### Schema & Version Management

These proposals form a dependency chain for multi-version Mendix support.

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [BSON Schema Registry](BSON_SCHEMA_REGISTRY_PROPOSAL.md) | Proposed | Runtime schema registry for version-aware BSON serialization, driven by reflection data | [Multi-Version Support](#) |
| [Multi-Version Support](MULTI_VERSION_SUPPORT.md) | Partial | Unified architecture for Mendix version differences. Phase W (widget augmentation) done; Phase 1 (schema registry core) not started | BSON Schema Registry |
| [Version-Aware MDL](version-aware-mdl.md) | Proposed | Read any Mendix version, write to target version with hybrid schema-driven serialization | BSON Schema Registry, Multi-Version Support |
| [Update Built-in Widget Properties](PROPOSAL_update_builtin_widget_properties.md) | Draft | Bulk property updates on built-in widgets using reflection-data schema metadata | BSON Schema Registry |

```
BSON schema Registry ◄──── multi-version Support
        ▲                          ▲
        │                          │
        ├── version-Aware MDL ─────┘
        │
        └── update Built-in widget properties
```

### MDL Language Evolution

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [MDL Syntax Improvements v1](PROPOSAL_mdl_syntax_improvements.md) | Draft | Go-style assignment, C-style braces, fluent list APIs | — |
| [MDL Syntax Improvements v2](PROPOSAL_mdl_syntax_improvements_v2.md) | Proposed | Consolidated v2: unified variable declaration, C-style braces, fluent list ops | Syntax Improvements v1 |
| [Microflow Free Annotation](PROPOSAL_microflow_free_annotation.md) | Draft | Order-sensitive `@annotation` handling for free-floating visual notes in microflows | — |
| [Microflow Download File Statement](PROPOSAL_microflow_download_file_statement.md) | Draft | `download file $FileDocument [show in browser]` for `DownloadFileAction` round-trip and authoring | — |
| [Empty Java Action Argument](PROPOSAL_microflow_empty_java_action_argument.md) | Implemented | `empty` for unbound Java-action parameters; round-trip preservation of empty `BasicCodeActionParameterValue` bindings | — |
| [Microflow Call Web Service Statement](PROPOSAL_microflow_call_web_service_statement.md) | Draft | Structured and raw MDL syntax for legacy SOAP `CallWebServiceAction` round-trip preservation | — |
| [Page Syntax V2](PROPOSAL_page_syntax_v2.md) | Superseded | Page/widget syntax with `{}` blocks and `->` binding. Superseded by V3 (archived) | — |
| [Page Styling Support](page-styling-support.md) | Partial | CSS classes, inline styles, dynamic classes, design properties. Phase 1 (Class/Style) done | — |
| [Page Composition](proposal_page_composition.md) | Proposed | Fragment definitions and ALTER PAGE for partial page editing | Page Syntax V2, Page Styling |
| [XPath Gaps](xpath-gaps-proposal.md) | Partial | XPath constraint support gap analysis. ~85% complete, association paths and nested predicates remain | — |
| [Microflow ENUM SPLIT Statement](PROPOSAL_microflow_enum_split_statement.md) | Implemented | Enumeration decision splits via `case $Var when Value then … end case;` | — |
| [Microflow CHANGE Refresh Modifier](PROPOSAL_microflow_change_refresh_modifier.md) | Draft | Preserve `RefreshInClient` on change-object actions | — |
| [Microflow ADD Expression To List](PROPOSAL_microflow_add_expression_to_list.md) | Draft | Preserve expression-valued list-add actions in microflow round-trips | — |
| [Microflow Inheritance Split And Cast Statements](PROPOSAL_microflow_inheritance_split_statement.md) | Draft | Preserve type-based microflow decisions and cast actions in round-trips | — |
| [LLM MDL Assistance](PROPOSAL_llm_mdl_assistance.md) | Proposed | Enhanced error messages with examples, reorganized skills by use case | — |

### Testing & Evaluation

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [MDL Test Framework](proposal-mdl-test-framework.md) | Draft | Testing framework using MDL with javadoc annotations for test definitions | — |
| [Playwright Testing](proposal-playwright-testing.md) | Proposed | Playwright UI testing + PostgreSQL data validation in generation pipeline | — |
| [Playwright CLI](proposal-playwright-cli.md) | Draft | Replace generated Playwright tests with direct playwright-cli browser control | Playwright Testing |
| [Eval Framework](proposal-eval-framework.md) | Partial | Systematic eval for Mendix app generation. Phase 1 (structural checks, scoring) done | Playwright Testing |
| [GitHub MDL Integration](proposal-github-mdl-integration.md) | Draft | CI workflow validating MDL scripts against real Mendix projects | MDL Test Framework |

```
MDL Test Framework ◄──── GitHub MDL Integration

Playwright Testing ◄──── Playwright CLI
        ▲
        └──── Eval Framework
```

### VS Code Extension

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [VS Code Visualizations](PROPOSAL_vscode_visualizations.md) | Proposed | Visual diagram previews: ER diagrams, flowcharts, wireframes, dependency graphs | — |
| [Sprotty Visualization](proposal_sprotty_visualization.md) | Draft | Sprotty-based interactive domain model diagrams (PoC) | VS Code Visualizations |
| [VS Code Search](PROPOSAL_vscode_search.md) | Proposed | Quick Pick full-text search UI + workspace symbol (Ctrl+T) | — |

### Navigation & Visualization

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [Navigation Support](navigation-support.md) | Partial | MDL support for navigation profiles, menus, role-based home pages. Parser support started | — |
| [Architecture Diagram](architecture-diagram-plan.md) | Proposed | Layered architecture diagrams per module (pages, microflows, entities, external services) | Navigation Support |
| [Journey Architecture Viz](journey-architecture-viz.md) | Proposed | Customer journey visualization with user roles, pages, microflows. Replaced by Architecture Diagram | Navigation Support |

### Runtime & External Integration

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [Admin API Runtime](admin_api_runtime.md) | Reference | Technical findings on M2EE admin API and XAS protocol (reverse-engineered endpoints) | — |
| [Runtime Admin Port](proposal-runtime-admin-port.md) | Proposed | Expose M2EE admin API: hot reload, CSS reload, microflow debugging | Admin API Runtime |
| [OData Services](odata-services-proposal.md) | Proposed | Consumed + published OData services, external entities | — |
| [Import Associations](PROPOSAL_import_associations.md) | Draft | LINK clause in IMPORT for mapping source columns to entity associations | — |
| [Marketplace Modules](PROPOSAL_marketplace_modules.md) | Draft | `mxcli marketplace install/search/info` for downloading marketplace modules | — |

### Infrastructure & Code Quality

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [Concurrent Access](PROPOSAL_concurrent_access.md) | Proposed | Project-level file locking for multiple simultaneous mxcli processes | — |
| [Session Logging](PROPOSAL_session_logging.md) | Proposed | JSON Lines session logging to `~/.mxcli/logs/` for debugging and bug reports | — |
| [Refactor Large Files](refactor-large-files.md) | Proposed | Split 6 large source files (visitor.go, writer.go, etc.) for maintainability | — |
| [Starlark Security API](starlark-security.md) | Proposed | Expose entity access rule data to Starlark lint rules | — |
| [Bulk Widget Property Updates](PROPOSAL_bulk_widget_property_updates.md) | Draft | Bulk find/modify custom widget properties across pages and snippets | — |
| [Structure Command](mxcli-structure-proposal.md) | Partial | Token-efficient project structure overview. SHOW STRUCTURE exists but some gaps remain | — |

### SHOW/DESCRIBE for Missing Document Types

Add read-only SHOW and DESCRIBE commands for Mendix document types that lack MDL support. Prioritized by prevalence across real-world projects. All follow the same implementation pattern (AST constant, grammar rule, visitor, executor handler).

**Tier 1 — High Priority** (heavily used, present in all analyzed projects):

| Proposal | Status | Summary | Reader Exists |
|----------|--------|---------|---------------|
| [JavaScript Actions](show-describe-javascript-actions.md) | Proposed | SHOW/DESCRIBE for `JavaScriptActions$JavaScriptAction` (283 across 3 projects) | Yes |
| [Building Blocks](show-describe-building-blocks.md) | Proposed | SHOW/DESCRIBE for `Forms$BuildingBlock` (233) | Yes |
| [Page Templates](show-describe-page-templates.md) | Proposed | SHOW/DESCRIBE for `Forms$PageTemplate` (215) | Yes |
| [Nanoflow DESCRIBE](show-describe-nanoflows.md) | Proposed | Enhanced DESCRIBE for `microflows$nanoflow` (227). Very low complexity — pure wiring | Yes (full) |
| [JSON Structures](show-describe-json-structures.md) | Proposed | SHOW/DESCRIBE for `JsonStructures$JsonStructure` (96). Recursive element tree | No |
| [Import Mappings](show-describe-import-mappings.md) | Proposed | SHOW/DESCRIBE for `ImportMappings$ImportMapping` (83). High complexity | No |
| [Export Mappings](show-describe-export-mappings.md) | Proposed | SHOW/DESCRIBE for `ExportMappings$ExportMapping` (67). Shares impl with Import Mappings | No |
| [Published REST Services](show-describe-published-rest-services.md) | Proposed | SHOW/DESCRIBE for `rest$PublishedRestService` (16) | Yes |

**Tier 2 — Medium Priority** (common, useful for project understanding):

| Proposal | Status | Summary | Reader Exists |
|----------|--------|---------|---------------|
| [Module Settings](show-describe-module-settings.md) | Proposed | SHOW/DESCRIBE for `Projects$ModuleSettings` (97). Low complexity | No |
| [Image Collections](show-describe-image-collections.md) | Proposed | SHOW/DESCRIBE for `Images$ImageCollection` (65) | Yes |
| [Rules](show-describe-rules.md) | Proposed | SHOW/DESCRIBE for `microflows$rule` (49). Reuses microflow parsing | No |
| [Message Definitions](show-describe-message-definitions.md) | Proposed | SHOW/DESCRIBE for `MessageDefinitions$MessageDefinitionCollection` (28) | No |
| [Scheduled Events](show-describe-scheduled-events.md) | Proposed | SHOW/DESCRIBE for `ScheduledEvents$ScheduledEvent` (19) | Yes |
| [Consumed REST Services](show-describe-consumed-rest-services.md) | Proposed | SHOW/DESCRIBE for `rest$ConsumedRestService` (2) | No |

**Tier 3 — Low Priority** (small counts, simple types):

| Proposal | Status | Summary | Reader Exists |
|----------|--------|---------|---------------|
| [Regular Expressions](show-describe-regular-expressions.md) | Proposed | SHOW/DESCRIBE for `RegularExpressions$RegularExpression` (13). Very low complexity | No |
| [Custom Icon Collections](show-describe-custom-icon-collections.md) | Proposed | SHOW/DESCRIBE for `CustomIcons$CustomIconCollection` (8) | No |
| [Menu Documents](show-describe-menu-documents.md) | Proposed | SHOW/DESCRIBE for `Menus$MenuDocument` (6). Recursive menu items | No |
| [Queues](show-describe-queues.md) | Proposed | SHOW/DESCRIBE for `Queues$Queue` (5). Very low complexity | No |
| [Pluggable Widgets](show-describe-pluggable-widgets.md) | Proposed | Improve DESCRIBE output for pluggable (React) widgets that fall through to generic formatting | Yes |

### Linting

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [Linter Framework](PROPOSAL_mxcli_linter.md) | Partial | Built-in Go rules (14) + Starlark extensible engine done. Some advanced rules remain | — |

### Documentation & Analysis

| Proposal | Status | Summary | Depends On |
|----------|--------|---------|------------|
| [Documentation Site](proposal-documentation-site.md) | Proposed | Unified docs site (mdBook + Typst PDF), PostgreSQL-style structure with tutorial, language guide, per-statement reference | — |
| [LLM Training Docs](github-for-llms.md) | Proposed | Improve documentation for LLM training: examples, common mistakes, doc index | Documentation Site |
| [SDK Equivalence](SDK_EQUIVALENCE.md) | Reference | TypeScript SDK vs Go implementation gap analysis | — |
| [Missing Capabilities](PROPOSAL_missing_capabilities.md) | Reference | Analysis of unsupported document types (REST, JSON structures, import/export mappings) | — |
| [Case Study: MxGraphStudioDemo](CASE_STUDY_MxGraphStudioDemo.md) | Reference | Real-world project analysis showing MDL coverage gaps | — |

---

## Archived Proposals (Fully Implemented)

| Proposal | Summary |
|----------|---------|
| [Pages V3](archive/proposal_pages_v3.md) | V3 page syntax — current standard |
| [High-Level API](archive/PROPOSAL_high_level_api.md) | Fluent builder API in `api/` package |
| [MDL Security](archive/PROPOSAL_mdl_security.md) | Module roles, user roles, access control, GRANT/REVOKE |
| [Business Events](archive/PROPOSAL_business_events_support.md) | SHOW/DESCRIBE/CREATE/DROP business event services |
| [Docker Integration](archive/PROPOSAL_mxcli_docker.md) | Docker PAD build/run/check integration |
| [External SQL](archive/PROPOSAL_mxcli_sql.md) | SQL CONNECT, query, import for PostgreSQL/Oracle/SQL Server |
| [Workflow Support](archive/PROPOSAL_workflow_support.md) | CREATE/DROP WORKFLOW with all activity types |
| [Connector Generation](archive/PROPOSAL_generate_connector.md) | SQL GENERATE CONNECTOR from external schema |
| [LSP Server](archive/lsp-language-server.md) | Language Server Protocol with hover, completion, diagnostics |
| [Code Search](archive/code-search.md) | SHOW CALLERS/CALLEES/REFERENCES/IMPACT/CONTEXT |
| [Code Search Implementation](archive/code-search-implementation.md) | References table and cross-reference catalog |
| [Catalog Tables](archive/catalog-tables.md) | SQLite catalog with SQL querying |
| [MPK Widget Augmentation](archive/PROPOSAL_mpk_widget_augmentation.md) | Dynamic widget template augmentation from .mpk files |
| [Page Variables](archive/PROPOSAL_page_variables.md) | Page variable support |
