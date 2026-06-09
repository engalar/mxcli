# File Naming Convention Unification

**Date:** 2026-06-09  
**Scope:** Pure rename — no code changes, no logic moves  
**Total files affected:** ~149  

## Background

Historical growth introduced four naming inconsistencies that make it hard to understand a file's role at a glance:

1. `_gen` suffix used for both "code-generated output" and "hand-written code that uses `modelsdk/gen/*` types"
2. `cmd_` prefix on 70 files that are not command entry points
3. `_modelsdk` suffix on 19 files that is no longer meaningful (migration complete)
4. `_compat` suffix with no clear lifecycle or deletion intent

## Naming Convention Rules

### Global suffix rules

| Old suffix | New suffix | When to use |
|-----------|-----------|-------------|
| `_gen.go` | `_gen.go` (keep) | **Only** if the file is output by `go generate` and has `// Code generated ... DO NOT EDIT.` in line 1 |
| `_gen.go` (hand-written) | `_v2.go` | File uses `modelsdk/gen/*` new type path; migration-period marker |
| `_modelsdk.go` | `.go` (drop suffix) | Suffix no longer distinguishes anything in `backend/mpr/` — drop it |
| `_modelsdk_gen.go` | `_v2.go` | Both rules apply: drop `_modelsdk`, rename `_gen` → `_v2` |
| `_compat.go` | `_legacy.go` | Explicit "this is retained legacy code, not the current path" |

### `executor/` prefix rules

```
cmd_{domain}.go               # Command handler: contains exec* dispatcher functions
cmd_{domain}_{aspect}.go      # Command handler split file (still has exec* functions)

{domain}_builder.go           # Builds BSON/gen structures (no exec* functions)
{domain}_formatter.go         # Formats MDL text output
{domain}_scanner.go           # Scans codebase for patterns
{domain}_helpers.go           # Internal utilities
{domain}_ast.go               # AST → model transforms

# Existing patterns — do not change:
validate_{domain}.go          # Validation logic
flowbuilder_{aspect}.go       # Flow builder helpers
widget_fmt_{aspect}.go        # Widget formatting
```

Non-command files that happened to have `cmd_` prefix: **drop the prefix, keep the rest of the name.**

### `backend/mock/` rules

All files must be `mock_{domain}.go`. The single exception (`backend.go`) is renamed to `mock_backend.go`.

## Commit Sequence

### Commit 1 — `backend/mock/` (1 file)

```
mock/backend.go  →  mock/mock_backend.go
```

### Commit 2 — `backend/mpr/` (25 files)

**Drop `_modelsdk` suffix (19 files):**

```
agenteditor_modelsdk.go           →  agenteditor.go
constants_modelsdk.go             →  constants.go
create_services_modelsdk.go       →  create_services.go
delete_move_modelsdk.go           →  delete_move.go
domainmodel_modelsdk.go           →  domainmodel.go
domainmodel_modelsdk_gen.go       →  domainmodel_v2.go
enums_modelsdk.go                 →  enums.go
modules_modelsdk.go               →  modules.go
navigation_modelsdk.go            →  navigation.go
refs_modelsdk.go                  →  refs.go
rename_modelsdk.go                →  rename.go
security_allowed_roles_modelsdk.go →  security_allowed_roles.go
security_entity_access_modelsdk.go →  security_entity_access.go
security_modelsdk.go              →  security.go
security_module_modelsdk.go       →  security_module.go
security_project_modelsdk.go      →  security_project.go
services_modelsdk.go              →  services.go
settings_modelsdk.go              →  settings.go
update_services_modelsdk.go       →  update_services.go
```

**Rename `_compat` → `_legacy` (6 files):**

```
image_compat.go    →  image_legacy.go
nav_compat.go      →  navigation_legacy.go
odata_compat.go    →  odata_legacy.go
rename_compat.go   →  rename_legacy.go
rest_compat.go     →  rest_legacy.go
settings_compat.go →  settings_legacy.go
```

### Commit 3 — `executor/` drop `cmd_` prefix from non-command files (70 files)

Rule: files without any `^func exec` function lose the `cmd_` prefix. Where the file also has `_gen` suffix, that becomes `_v2`.

```
cmd_agenteditor_agents.go              →  agenteditor_agents.go
cmd_agenteditor_kbs.go                 →  agenteditor_kbs.go
cmd_agenteditor_mcpservices.go         →  agenteditor_mcpservices.go
cmd_associations_write_gen.go          →  associations_write_v2.go
cmd_businessevents.go                  →  businessevents.go
cmd_constants.go                       →  constants.go
cmd_contract.go                        →  contract.go
cmd_dbconnection.go                    →  dbconnection.go
cmd_describe_translations.go           →  describe_translations.go
cmd_diff.go                            →  diff.go
cmd_diff_gen.go                        →  diff_v2.go
cmd_diff_local.go                      →  diff_local.go
cmd_diff_mdl.go                        →  diff_mdl.go
cmd_diff_output.go                     →  diff_output.go
cmd_domainmodel_elk.go                 →  domainmodel_elk.go
cmd_domainmodel_elk_gen.go             →  domainmodel_elk_v2.go
cmd_entities_access.go                 →  entities_access.go
cmd_entities_describe.go               →  entities_describe.go
cmd_entities_event_indexes_gen.go      →  entities_event_indexes_v2.go
cmd_entities_gen.go                    →  entities_v2.go
cmd_entities_refs.go                   →  entities_refs.go
cmd_entities_validation_gen.go         →  entities_validation_v2.go
cmd_entities_write_gen.go              →  entities_write_v2.go
cmd_export_project.go                  →  export_project.go
cmd_import_project.go                  →  import_project.go
cmd_languages.go                       →  languages.go
cmd_layouts.go                         →  layouts.go
cmd_mermaid.go                         →  mermaid.go
cmd_mermaid_dm_gen.go                  →  mermaid_dm_v2.go
cmd_mermaid_gen.go                     →  mermaid_v2.go
cmd_microflow_elk_gen.go               →  microflow_elk_v2.go
cmd_microflows_caller_scan.go          →  microflows_caller_scan.go
cmd_microflows_entity_param_scan.go    →  microflows_entity_param_scan.go
cmd_microflows_format_action_gen.go    →  microflows_format_action_v2.go
cmd_microflows_format_calls_gen.go     →  microflows_format_calls_v2.go
cmd_microflows_format_data_gen.go      →  microflows_format_data_v2.go
cmd_microflows_format_external_gen.go  →  microflows_format_external_v2.go
cmd_microflows_format_workflow_gen.go  →  microflows_format_workflow_v2.go
cmd_microflows_helpers.go              →  microflows_helpers.go
cmd_microflows_ref_fixer.go            →  microflows_ref_fixer.go
cmd_microflows_show_gen.go             →  microflows_show_v2.go
cmd_microflows_show_list_gen.go        →  microflows_show_list_v2.go
cmd_microflows_viz_helpers_gen.go      →  microflows_viz_helpers_v2.go
cmd_module_overview.go                 →  module_overview.go
cmd_modules_gen.go                     →  modules_v2.go
cmd_nanoflow_elk_gen.go                →  nanoflow_elk_v2.go
cmd_nanoflows_show_gen.go              →  nanoflows_show_v2.go
cmd_odata_gen.go                       →  odata_v2.go
cmd_oql_plan.go                        →  oql_plan.go
cmd_page_wireframe.go                  →  page_wireframe.go
cmd_pages_ast_to_model.go              →  pages_ast_to_model.go
cmd_pages_builder_input.go             →  pages_builder_input.go
cmd_pages_builder_input_filters.go     →  pages_builder_input_filters.go
cmd_pages_builder_v3.go                →  pages_builder_v3.go
cmd_pages_describe.go                  →  pages_describe.go
cmd_pages_describe_output.go           →  pages_describe_output.go
cmd_pages_describe_parse.go            →  pages_describe_parse.go
cmd_pages_describe_pluggable.go        →  pages_describe_pluggable.go
cmd_pages_gen.go                       →  pages_v2.go
cmd_pages_model_to_mdl.go             →  pages_model_to_mdl.go
cmd_pages_show.go                      →  pages_show.go
cmd_rest_clients.go                    →  rest_clients.go
cmd_security_access_check.go           →  security_access_check.go
cmd_security_defaults.go               →  security_defaults.go
cmd_security_gen.go                    →  security_v2.go
cmd_settings.go                        →  settings.go
cmd_snippets.go                        →  snippets.go
cmd_structure.go                       →  structure.go
cmd_translate.go                       →  translate.go
cmd_workflows_gen.go                   →  workflows_v2.go
```

### Commit 4 — `executor/` `_gen` → `_v2` on command files + flowbuilder + helpers (53 files)

**Command entry files (22 files) — keep `cmd_` prefix:**

```
cmd_alter_association_gen.go          →  cmd_alter_association_v2.go
cmd_alter_entity_gen.go               →  cmd_alter_entity_v2.go
cmd_create_association_gen.go         →  cmd_create_association_v2.go
cmd_create_entity_gen.go              →  cmd_create_entity_v2.go
cmd_create_view_entity_gen.go         →  cmd_create_view_entity_v2.go
cmd_drop_association_gen.go           →  cmd_drop_association_v2.go
cmd_drop_entity_gen.go                →  cmd_drop_entity_v2.go
cmd_javaactions_gen.go                →  cmd_javaactions_v2.go
cmd_microflows_create_gen.go          →  cmd_microflows_create_v2.go
cmd_nanoflows_create_gen.go           →  cmd_nanoflows_create_v2.go
cmd_nanoflows_drop_gen.go             →  cmd_nanoflows_drop_v2.go
cmd_security_write_demouser_gen.go    →  cmd_security_write_demouser_v2.go
cmd_security_write_entity_gen.go      →  cmd_security_write_entity_v2.go
cmd_security_write_extservice_gen.go  →  cmd_security_write_extservice_v2.go
cmd_security_write_gen.go             →  cmd_security_write_v2.go
cmd_security_write_modulerole_gen.go  →  cmd_security_write_modulerole_v2.go
cmd_security_write_page_gen.go        →  cmd_security_write_page_v2.go
cmd_security_write_project_gen.go     →  cmd_security_write_project_v2.go
cmd_security_write_update_gen.go      →  cmd_security_write_update_v2.go
cmd_security_write_userrole_gen.go    →  cmd_security_write_userrole_v2.go
cmd_structure_gen.go                  →  cmd_structure_v2.go
cmd_workflows_write_gen2.go           →  cmd_workflows_write_v2.go
```

**`flowbuilder_` files (26 files):**

```
flowbuilder_actions_change_gen.go     →  flowbuilder_actions_change_v2.go
flowbuilder_actions_feedback_gen.go   →  flowbuilder_actions_feedback_v2.go
flowbuilder_actions_gen.go            →  flowbuilder_actions_v2.go
flowbuilder_actions_listop_gen.go     →  flowbuilder_actions_listop_v2.go
flowbuilder_actions_retrieve_gen.go   →  flowbuilder_actions_retrieve_v2.go
flowbuilder_annotations_gen.go        →  flowbuilder_annotations_v2.go
flowbuilder_assoc_lookup_gen.go       →  flowbuilder_assoc_lookup_v2.go
flowbuilder_calls_code_gen.go         →  flowbuilder_calls_code_v2.go
flowbuilder_calls_external_gen.go     →  flowbuilder_calls_external_v2.go
flowbuilder_calls_flow_gen.go         →  flowbuilder_calls_flow_v2.go
flowbuilder_calls_page_gen.go         →  flowbuilder_calls_page_v2.go
flowbuilder_calls_rest_gen.go         →  flowbuilder_calls_rest_v2.go
flowbuilder_calls_send_rest_gen.go    →  flowbuilder_calls_send_rest_v2.go
flowbuilder_calls_webservice_gen.go   →  flowbuilder_calls_webservice_v2.go
flowbuilder_calls_xml_gen.go          →  flowbuilder_calls_xml_v2.go
flowbuilder_control_if_gen.go         →  flowbuilder_control_if_v2.go
flowbuilder_control_loop_gen.go       →  flowbuilder_control_loop_v2.go
flowbuilder_control_split_gen.go      →  flowbuilder_control_split_v2.go
flowbuilder_dispatch_gen.go           →  flowbuilder_dispatch_v2.go
flowbuilder_eh_body_gen.go            →  flowbuilder_eh_body_v2.go
flowbuilder_eh_queue_gen.go           →  flowbuilder_eh_queue_v2.go
flowbuilder_flows_gen.go              →  flowbuilder_flows_v2.go
flowbuilder_graph_gen.go              →  flowbuilder_graph_v2.go
flowbuilder_raw_setter_gen.go         →  flowbuilder_raw_setter_v2.go
flowbuilder_synchronize_gen.go        →  flowbuilder_synchronize_v2.go
flowbuilder_workflow_gen.go           →  flowbuilder_workflow_v2.go
```

**`helpers_` files (5 files):**

```
helpers_domainmodels_gen.go  →  helpers_domainmodels_v2.go
helpers_javaactions_gen.go   →  helpers_javaactions_v2.go
helpers_pages_gen.go         →  helpers_pages_v2.go
helpers_security_gen.go      →  helpers_security_v2.go
helpers_workflows_gen.go     →  helpers_workflows_v2.go
```

## Out of Scope

- `cmd/mxcli/` — deferred; the naming chaos there is less impactful and more entangled with Cobra subcommand structure
- Merging paired files (e.g., `cmd_entities.go` + `entities_v2.go`) — requires code changes, separate task
- Deleting `*_legacy.go` files — requires code changes, separate task
- Any SOLID refactoring — separate from this rename pass

## Verification Checklist

After each commit:
1. `make build` — must pass (Go file names don't affect imports)
2. `make test` — must pass
3. `git diff --stat HEAD~1` — confirm only renames, no content changes (`git mv` produces `renamed` status)
4. `git log --follow <new-name>` — confirm history preserved

## Implementation Notes

- Use `git mv <old> <new>` for every rename — never delete + create
- Test files follow the same rename: `cmd_foo_test.go` → `foo_test.go`, `cmd_foo_gen_test.go` → `foo_v2_test.go`
- Commits must be pure renames with zero content changes — a reviewer should see only `renamed` lines in `git diff --stat`
