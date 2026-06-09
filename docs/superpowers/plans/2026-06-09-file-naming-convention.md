# File Naming Convention Unification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename ~149 Go source files across `backend/mock/`, `backend/mpr/`, and `mdl/executor/` so file names reflect their actual role, in 4 pure-rename commits with zero code changes.

**Architecture:** Four sequential `git mv` commits, one per directory/concern. Each commit is fully buildable and testable on its own. No import paths change (Go imports are directory-scoped, not file-scoped).

**Tech Stack:** Go 1.21+, git mv, make build, make test

**Spec:** `docs/superpowers/specs/2026-06-09-file-naming-convention-design.md`

---

## Rename Rules Reference

| Pattern | Old | New | Meaning |
|---------|-----|-----|---------|
| `_modelsdk.go` | `foo_modelsdk.go` | `foo.go` | Drop stale migration marker |
| `_modelsdk_gen.go` | `foo_modelsdk_gen.go` | `foo_v2.go` | Drop marker + hand-written gen suffix |
| `_compat.go` | `foo_compat.go` | `foo_legacy.go` | Explicit "retained legacy code" |
| `_gen.go` (hand-written) | `foo_gen.go` | `foo_v2.go` | Uses `modelsdk/gen/*`; not code-generated |
| `cmd_` non-command | `cmd_foo.go` | `foo.go` | Keep `cmd_` only for files with `exec*` functions |
| `cmd_` + `_gen` non-command | `cmd_foo_gen.go` | `foo_v2.go` | Both rules together |

---

## Task 1: backend/mock/ (1 file)

**Files:**
- Rename: `mdl/backend/mock/backend.go` → `mdl/backend/mock/mock_backend.go`

- [ ] **Step 1: Run the rename**

```bash
cd /path/to/repo
git mv mdl/backend/mock/backend.go mdl/backend/mock/mock_backend.go
```

- [ ] **Step 2: Verify build passes**

```bash
make build
```

Expected: `Build succeeded` (or equivalent success output, no errors)

- [ ] **Step 3: Verify tests pass**

```bash
make test
```

Expected: all tests pass, zero failures

- [ ] **Step 4: Verify the commit contains only renames**

```bash
git diff --stat HEAD
```

Expected: only `renamed` lines, no `modified` lines

- [ ] **Step 5: Commit**

```bash
git commit -m "chore: rename backend/mock/backend.go to mock_backend.go

File name now matches the mock_ prefix convention used by all other
files in this package.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: backend/mpr/ — drop `_modelsdk` and rename `_compat` → `_legacy` (25 files)

**Files modified:** all in `mdl/backend/mpr/`

- [ ] **Step 1: Rename the 19 `_modelsdk` files**

```bash
D=mdl/backend/mpr
git mv $D/agenteditor_modelsdk.go            $D/agenteditor.go
git mv $D/constants_modelsdk.go              $D/constants.go
git mv $D/create_services_modelsdk.go        $D/create_services.go
git mv $D/delete_move_modelsdk.go            $D/delete_move.go
git mv $D/domainmodel_modelsdk.go            $D/domainmodel.go
git mv $D/domainmodel_modelsdk_gen.go        $D/domainmodel_v2.go
git mv $D/enums_modelsdk.go                  $D/enums.go
git mv $D/modules_modelsdk.go                $D/modules.go
git mv $D/navigation_modelsdk.go             $D/navigation.go
git mv $D/refs_modelsdk.go                   $D/refs.go
git mv $D/rename_modelsdk.go                 $D/rename.go
git mv $D/security_allowed_roles_modelsdk.go $D/security_allowed_roles.go
git mv $D/security_entity_access_modelsdk.go $D/security_entity_access.go
git mv $D/security_modelsdk.go               $D/security.go
git mv $D/security_module_modelsdk.go        $D/security_module.go
git mv $D/security_project_modelsdk.go       $D/security_project.go
git mv $D/services_modelsdk.go               $D/services.go
git mv $D/settings_modelsdk.go               $D/settings.go
git mv $D/update_services_modelsdk.go        $D/update_services.go
```

- [ ] **Step 2: Rename the 3 `_modelsdk` test files**

```bash
D=mdl/backend/mpr
git mv $D/domainmodel_modelsdk_test.go  $D/domainmodel_test.go
git mv $D/security_modelsdk_test.go     $D/security_test.go
git mv $D/services_modelsdk_test.go     $D/services_test.go
```

- [ ] **Step 3: Rename the 6 `_compat` files to `_legacy`**

```bash
D=mdl/backend/mpr
git mv $D/image_compat.go    $D/image_legacy.go
git mv $D/nav_compat.go      $D/navigation_legacy.go
git mv $D/odata_compat.go    $D/odata_legacy.go
git mv $D/rename_compat.go   $D/rename_legacy.go
git mv $D/rest_compat.go     $D/rest_legacy.go
git mv $D/settings_compat.go $D/settings_legacy.go
```

- [ ] **Step 4: Verify no content changes crept in**

```bash
git diff HEAD
```

Expected: empty output (only renames, no content diff)

- [ ] **Step 5: Verify build passes**

```bash
make build
```

Expected: success

- [ ] **Step 6: Verify tests pass**

```bash
make test
```

Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git commit -m "chore: unify backend/mpr file naming conventions

- Drop _modelsdk suffix (19 files): migration complete, suffix no longer
  distinguishes anything in this package
- domainmodel_modelsdk_gen.go → domainmodel_v2.go: _gen hand-written using
  gen types becomes _v2
- Rename _compat → _legacy (6 files): makes lifecycle intent explicit

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: executor/ — remove `cmd_` prefix from non-command files (70 prod + test files)

Non-command files = files without a top-level `func exec` dispatcher. Removing `cmd_` prefix makes it clear these are helpers, builders, and formatters — not command entry points.

Where the file also has `_gen` suffix (hand-written), rename `_gen` → `_v2` in the same step.

**Files modified:** all in `mdl/executor/`

- [ ] **Step 1: Rename the 70 production files**

```bash
D=mdl/executor
git mv $D/cmd_agenteditor_agents.go             $D/agenteditor_agents.go
git mv $D/cmd_agenteditor_kbs.go                $D/agenteditor_kbs.go
git mv $D/cmd_agenteditor_mcpservices.go        $D/agenteditor_mcpservices.go
git mv $D/cmd_associations_write_gen.go         $D/associations_write_v2.go
git mv $D/cmd_businessevents.go                 $D/businessevents.go
git mv $D/cmd_constants.go                      $D/constants.go
git mv $D/cmd_contract.go                       $D/contract.go
git mv $D/cmd_dbconnection.go                   $D/dbconnection.go
git mv $D/cmd_describe_translations.go          $D/describe_translations.go
git mv $D/cmd_diff.go                           $D/diff.go
git mv $D/cmd_diff_gen.go                       $D/diff_v2.go
git mv $D/cmd_diff_local.go                     $D/diff_local.go
git mv $D/cmd_diff_mdl.go                       $D/diff_mdl.go
git mv $D/cmd_diff_output.go                    $D/diff_output.go
git mv $D/cmd_domainmodel_elk.go                $D/domainmodel_elk.go
git mv $D/cmd_domainmodel_elk_gen.go            $D/domainmodel_elk_v2.go
git mv $D/cmd_entities_access.go                $D/entities_access.go
git mv $D/cmd_entities_describe.go              $D/entities_describe.go
git mv $D/cmd_entities_event_indexes_gen.go     $D/entities_event_indexes_v2.go
git mv $D/cmd_entities_gen.go                   $D/entities_v2.go
git mv $D/cmd_entities_refs.go                  $D/entities_refs.go
git mv $D/cmd_entities_validation_gen.go        $D/entities_validation_v2.go
git mv $D/cmd_entities_write_gen.go             $D/entities_write_v2.go
git mv $D/cmd_export_project.go                 $D/export_project.go
git mv $D/cmd_import_project.go                 $D/import_project.go
git mv $D/cmd_languages.go                      $D/languages.go
git mv $D/cmd_layouts.go                        $D/layouts.go
git mv $D/cmd_mermaid.go                        $D/mermaid.go
git mv $D/cmd_mermaid_dm_gen.go                 $D/mermaid_dm_v2.go
git mv $D/cmd_mermaid_gen.go                    $D/mermaid_v2.go
git mv $D/cmd_microflow_elk_gen.go              $D/microflow_elk_v2.go
git mv $D/cmd_microflows_caller_scan.go         $D/microflows_caller_scan.go
git mv $D/cmd_microflows_entity_param_scan.go   $D/microflows_entity_param_scan.go
git mv $D/cmd_microflows_format_action_gen.go   $D/microflows_format_action_v2.go
git mv $D/cmd_microflows_format_calls_gen.go    $D/microflows_format_calls_v2.go
git mv $D/cmd_microflows_format_data_gen.go     $D/microflows_format_data_v2.go
git mv $D/cmd_microflows_format_external_gen.go $D/microflows_format_external_v2.go
git mv $D/cmd_microflows_format_workflow_gen.go $D/microflows_format_workflow_v2.go
git mv $D/cmd_microflows_helpers.go             $D/microflows_helpers.go
git mv $D/cmd_microflows_ref_fixer.go           $D/microflows_ref_fixer.go
git mv $D/cmd_microflows_show_gen.go            $D/microflows_show_v2.go
git mv $D/cmd_microflows_show_list_gen.go       $D/microflows_show_list_v2.go
git mv $D/cmd_microflows_viz_helpers_gen.go     $D/microflows_viz_helpers_v2.go
git mv $D/cmd_module_overview.go                $D/module_overview.go
git mv $D/cmd_modules_gen.go                    $D/modules_v2.go
git mv $D/cmd_nanoflow_elk_gen.go               $D/nanoflow_elk_v2.go
git mv $D/cmd_nanoflows_show_gen.go             $D/nanoflows_show_v2.go
git mv $D/cmd_odata_gen.go                      $D/odata_v2.go
git mv $D/cmd_oql_plan.go                       $D/oql_plan.go
git mv $D/cmd_page_wireframe.go                 $D/page_wireframe.go
git mv $D/cmd_pages_ast_to_model.go             $D/pages_ast_to_model.go
git mv $D/cmd_pages_builder_input.go            $D/pages_builder_input.go
git mv $D/cmd_pages_builder_input_filters.go    $D/pages_builder_input_filters.go
git mv $D/cmd_pages_builder_v3.go               $D/pages_builder_v3.go
git mv $D/cmd_pages_describe.go                 $D/pages_describe.go
git mv $D/cmd_pages_describe_output.go          $D/pages_describe_output.go
git mv $D/cmd_pages_describe_parse.go           $D/pages_describe_parse.go
git mv $D/cmd_pages_describe_pluggable.go       $D/pages_describe_pluggable.go
git mv $D/cmd_pages_gen.go                      $D/pages_v2.go
git mv $D/cmd_pages_model_to_mdl.go             $D/pages_model_to_mdl.go
git mv $D/cmd_pages_show.go                     $D/pages_show.go
git mv $D/cmd_rest_clients.go                   $D/rest_clients.go
git mv $D/cmd_security_access_check.go          $D/security_access_check.go
git mv $D/cmd_security_defaults.go              $D/security_defaults.go
git mv $D/cmd_security_gen.go                   $D/security_v2.go
git mv $D/cmd_settings.go                       $D/settings.go
git mv $D/cmd_snippets.go                       $D/snippets.go
git mv $D/cmd_structure.go                      $D/structure.go
git mv $D/cmd_translate.go                      $D/translate.go
git mv $D/cmd_workflows_gen.go                  $D/workflows_v2.go
```

- [ ] **Step 2: Rename the corresponding test files**

Apply the same transformation to `_test.go` counterparts. Each test file that shares a prefix with a renamed production file gets the same rename:

```bash
D=mdl/executor
git mv $D/cmd_agenteditor_mock_test.go             $D/agenteditor_mock_test.go
git mv $D/cmd_alter_page_mock_test.go              $D/alter_page_mock_test.go
git mv $D/cmd_associations_gen_test.go             $D/associations_write_v2_test.go
git mv $D/cmd_associations_mock_test.go            $D/associations_mock_test.go
git mv $D/cmd_businessevents_mock_test.go          $D/businessevents_mock_test.go
git mv $D/cmd_constants_mock_test.go               $D/constants_mock_test.go
git mv $D/cmd_context_entity_test.go               $D/context_entity_test.go
git mv $D/cmd_contract_test.go                     $D/contract_test.go
git mv $D/cmd_datatransformer_mock_test.go         $D/datatransformer_mock_test.go
git mv $D/cmd_dbconnection_mock_test.go            $D/dbconnection_mock_test.go
git mv $D/cmd_describe_translations_mock_test.go   $D/describe_translations_mock_test.go
git mv $D/cmd_diff_gen_test.go                     $D/diff_v2_test.go
git mv $D/cmd_diff_local_test.go                   $D/diff_local_test.go
git mv $D/cmd_domainmodel_elk_gen_test.go          $D/domainmodel_elk_v2_test.go
git mv $D/cmd_entities_gen_test.go                 $D/entities_v2_test.go
git mv $D/cmd_entities_mock_test.go                $D/entities_mock_test.go
git mv $D/cmd_entities_write_gen_test.go           $D/entities_write_v2_test.go
git mv $D/cmd_enumerations_mock_test.go            $D/enumerations_mock_test.go
git mv $D/cmd_error_mock_test.go                   $D/error_mock_test.go
git mv $D/cmd_export_mappings_mock_test.go         $D/export_mappings_mock_test.go
git mv $D/cmd_export_project_test.go               $D/export_project_test.go
git mv $D/cmd_features_mock_test.go                $D/features_mock_test.go
git mv $D/cmd_folders_mock_test.go                 $D/folders_mock_test.go
git mv $D/cmd_fragments_mock_test.go               $D/fragments_mock_test.go
git mv $D/cmd_imagecollections_mock_test.go        $D/imagecollections_mock_test.go
git mv $D/cmd_import_mappings_mock_test.go         $D/import_mappings_mock_test.go
git mv $D/cmd_import_mock_test.go                  $D/import_mock_test.go
git mv $D/cmd_import_project_test.go               $D/import_project_test.go
git mv $D/cmd_json_mock_test.go                    $D/json_mock_test.go
git mv $D/cmd_jsonstructures_mock_test.go          $D/jsonstructures_mock_test.go
git mv $D/cmd_languages_mock_test.go               $D/languages_mock_test.go
git mv $D/cmd_lint_mock_test.go                    $D/lint_mock_test.go
git mv $D/cmd_mermaid_dm_gen_test.go               $D/mermaid_dm_v2_test.go
git mv $D/cmd_mermaid_gen_test.go                  $D/mermaid_v2_test.go
git mv $D/cmd_mermaid_mock_test.go                 $D/mermaid_mock_test.go
git mv $D/cmd_microflow_elk_gen_test.go            $D/microflow_elk_v2_test.go
git mv $D/cmd_microflows_annotation_escape_test.go $D/microflows_annotation_escape_test.go
git mv $D/cmd_microflows_caller_scan_test.go       $D/microflows_caller_scan_test.go
```

- [ ] **Step 3: Rename remaining executor test files (continued)**

```bash
D=mdl/executor
git mv $D/cmd_microflows_expr_literal_escape_test.go   $D/microflows_expr_literal_escape_test.go
git mv $D/cmd_microflows_format_action_gen_test.go     $D/microflows_format_action_v2_test.go
git mv $D/cmd_microflows_format_calls_gen_test.go      $D/microflows_format_calls_v2_test.go
git mv $D/cmd_microflows_format_data_gen_test.go       $D/microflows_format_data_v2_test.go
git mv $D/cmd_microflows_format_external_gen_test.go   $D/microflows_format_external_v2_test.go
git mv $D/cmd_microflows_format_workflow_gen_test.go   $D/microflows_format_workflow_v2_test.go
git mv $D/cmd_microflows_show_gen_test.go              $D/microflows_show_v2_test.go
git mv $D/cmd_modules_gen_test.go                      $D/modules_v2_test.go
git mv $D/cmd_nanoflow_elk_gen_test.go                 $D/nanoflow_elk_v2_test.go
git mv $D/cmd_navigation_mock_test.go                  $D/navigation_mock_test.go
git mv $D/cmd_notconnected_mock_test.go                $D/notconnected_mock_test.go
git mv $D/cmd_odata_gen_test.go                        $D/odata_v2_test.go
git mv $D/cmd_odata_mock_test.go                       $D/odata_mock_test.go
git mv $D/cmd_odata_test.go                            $D/odata_test.go
git mv $D/cmd_pages_ast_to_model_test.go               $D/pages_ast_to_model_test.go
git mv $D/cmd_pages_builder_fileinput_test.go          $D/pages_builder_fileinput_test.go
git mv $D/cmd_pages_builder_input_test.go              $D/pages_builder_input_test.go
git mv $D/cmd_pages_builder_v3_test.go                 $D/pages_builder_v3_test.go
git mv $D/cmd_pages_describe_action_test.go            $D/pages_describe_action_test.go
git mv $D/cmd_pages_describe_container_test.go         $D/pages_describe_container_test.go
git mv $D/cmd_pages_describe_pluggable_test.go         $D/pages_describe_pluggable_test.go
git mv $D/cmd_pages_gen_test.go                        $D/pages_v2_test.go
git mv $D/cmd_pages_mock_test.go                       $D/pages_mock_test.go
git mv $D/cmd_pages_model_to_mdl_test.go               $D/pages_model_to_mdl_test.go
git mv $D/cmd_published_rest_mock_test.go              $D/published_rest_mock_test.go
git mv $D/cmd_rename_mock_test.go                      $D/rename_mock_test.go
git mv $D/cmd_rest_clients_mock_test.go                $D/rest_clients_mock_test.go
git mv $D/cmd_rest_openapi_mock_test.go                $D/rest_openapi_mock_test.go
git mv $D/cmd_security_access_check_test.go            $D/security_access_check_test.go
git mv $D/cmd_security_defaults_test.go                $D/security_defaults_test.go
git mv $D/cmd_security_gen_test.go                     $D/security_v2_test.go
git mv $D/cmd_security_mock_test.go                    $D/security_mock_test.go
git mv $D/cmd_settings_mock_test.go                    $D/settings_mock_test.go
git mv $D/cmd_snippetcall_params_test.go               $D/snippetcall_params_test.go
git mv $D/cmd_styling_mock_test.go                     $D/styling_mock_test.go
git mv $D/cmd_translate_microflow_mock_test.go         $D/translate_microflow_mock_test.go
git mv $D/cmd_translate_mock_test.go                   $D/translate_mock_test.go
git mv $D/cmd_widgets_installed_test.go                $D/widgets_installed_test.go
git mv $D/cmd_workflows_describe_test.go               $D/workflows_describe_test.go
git mv $D/cmd_workflows_gen_test.go                    $D/workflows_v2_test.go
git mv $D/cmd_workflows_mock_test.go                   $D/workflows_mock_test.go
git mv $D/cmd_write_handlers_mock_test.go              $D/write_handlers_mock_test.go
```

- [ ] **Step 4: Verify no content changes**

```bash
git diff HEAD
```

Expected: empty (only renames)

- [ ] **Step 5: Verify build passes**

```bash
make build
```

Expected: success

- [ ] **Step 6: Verify tests pass**

```bash
make test
```

Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git commit -m "chore: remove cmd_ prefix from non-command executor files

Files without an exec* dispatcher function are helpers, builders,
formatters, and scanners — not command entry points. The cmd_ prefix
was misleading. Also renames hand-written _gen files to _v2 where they
use modelsdk/gen/* types (not code-generated output).

~110 files renamed (70 prod + test counterparts).

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: executor/ — rename `_gen` → `_v2` for all remaining files (57 prod + test files)

Covers: command entry files that keep their `cmd_` prefix, `flowbuilder_*`, `helpers_*`, and top-level `autocomplete_gen` / `flowbuilder_gen` / `helpers_gen` files.

**Files modified:** all in `mdl/executor/`

- [ ] **Step 1: Rename command entry files (keep `cmd_` prefix)**

```bash
D=mdl/executor
git mv $D/cmd_alter_association_gen.go         $D/cmd_alter_association_v2.go
git mv $D/cmd_alter_entity_gen.go              $D/cmd_alter_entity_v2.go
git mv $D/cmd_create_association_gen.go        $D/cmd_create_association_v2.go
git mv $D/cmd_create_entity_gen.go             $D/cmd_create_entity_v2.go
git mv $D/cmd_create_view_entity_gen.go        $D/cmd_create_view_entity_v2.go
git mv $D/cmd_drop_association_gen.go          $D/cmd_drop_association_v2.go
git mv $D/cmd_drop_entity_gen.go               $D/cmd_drop_entity_v2.go
git mv $D/cmd_javaactions_gen.go               $D/cmd_javaactions_v2.go
git mv $D/cmd_microflows_create_gen.go         $D/cmd_microflows_create_v2.go
git mv $D/cmd_nanoflows_create_gen.go          $D/cmd_nanoflows_create_v2.go
git mv $D/cmd_nanoflows_drop_gen.go            $D/cmd_nanoflows_drop_v2.go
git mv $D/cmd_security_write_demouser_gen.go   $D/cmd_security_write_demouser_v2.go
git mv $D/cmd_security_write_entity_gen.go     $D/cmd_security_write_entity_v2.go
git mv $D/cmd_security_write_extservice_gen.go $D/cmd_security_write_extservice_v2.go
git mv $D/cmd_security_write_gen.go            $D/cmd_security_write_v2.go
git mv $D/cmd_security_write_modulerole_gen.go $D/cmd_security_write_modulerole_v2.go
git mv $D/cmd_security_write_page_gen.go       $D/cmd_security_write_page_v2.go
git mv $D/cmd_security_write_project_gen.go    $D/cmd_security_write_project_v2.go
git mv $D/cmd_security_write_update_gen.go     $D/cmd_security_write_update_v2.go
git mv $D/cmd_security_write_userrole_gen.go   $D/cmd_security_write_userrole_v2.go
git mv $D/cmd_structure_gen.go                 $D/cmd_structure_v2.go
git mv $D/cmd_workflows_write_gen2.go          $D/cmd_workflows_write_v2.go
```

- [ ] **Step 2: Rename `flowbuilder_*_gen` files (27 files)**

```bash
D=mdl/executor
git mv $D/flowbuilder_actions_change_gen.go    $D/flowbuilder_actions_change_v2.go
git mv $D/flowbuilder_actions_feedback_gen.go  $D/flowbuilder_actions_feedback_v2.go
git mv $D/flowbuilder_actions_gen.go           $D/flowbuilder_actions_v2.go
git mv $D/flowbuilder_actions_listop_gen.go    $D/flowbuilder_actions_listop_v2.go
git mv $D/flowbuilder_actions_retrieve_gen.go  $D/flowbuilder_actions_retrieve_v2.go
git mv $D/flowbuilder_annotations_gen.go       $D/flowbuilder_annotations_v2.go
git mv $D/flowbuilder_assoc_lookup_gen.go      $D/flowbuilder_assoc_lookup_v2.go
git mv $D/flowbuilder_calls_code_gen.go        $D/flowbuilder_calls_code_v2.go
git mv $D/flowbuilder_calls_external_gen.go    $D/flowbuilder_calls_external_v2.go
git mv $D/flowbuilder_calls_flow_gen.go        $D/flowbuilder_calls_flow_v2.go
git mv $D/flowbuilder_calls_page_gen.go        $D/flowbuilder_calls_page_v2.go
git mv $D/flowbuilder_calls_rest_gen.go        $D/flowbuilder_calls_rest_v2.go
git mv $D/flowbuilder_calls_send_rest_gen.go   $D/flowbuilder_calls_send_rest_v2.go
git mv $D/flowbuilder_calls_webservice_gen.go  $D/flowbuilder_calls_webservice_v2.go
git mv $D/flowbuilder_calls_xml_gen.go         $D/flowbuilder_calls_xml_v2.go
git mv $D/flowbuilder_control_if_gen.go        $D/flowbuilder_control_if_v2.go
git mv $D/flowbuilder_control_loop_gen.go      $D/flowbuilder_control_loop_v2.go
git mv $D/flowbuilder_control_split_gen.go     $D/flowbuilder_control_split_v2.go
git mv $D/flowbuilder_dispatch_gen.go          $D/flowbuilder_dispatch_v2.go
git mv $D/flowbuilder_eh_body_gen.go           $D/flowbuilder_eh_body_v2.go
git mv $D/flowbuilder_eh_queue_gen.go          $D/flowbuilder_eh_queue_v2.go
git mv $D/flowbuilder_flows_gen.go             $D/flowbuilder_flows_v2.go
git mv $D/flowbuilder_gen.go                   $D/flowbuilder_v2.go
git mv $D/flowbuilder_graph_gen.go             $D/flowbuilder_graph_v2.go
git mv $D/flowbuilder_raw_setter_gen.go        $D/flowbuilder_raw_setter_v2.go
git mv $D/flowbuilder_synchronize_gen.go       $D/flowbuilder_synchronize_v2.go
git mv $D/flowbuilder_workflow_gen.go          $D/flowbuilder_workflow_v2.go
```

- [ ] **Step 3: Rename `helpers_*_gen`, `autocomplete_gen`, and `helpers_gen_container`**

```bash
D=mdl/executor
git mv $D/autocomplete_gen.go          $D/autocomplete_v2.go
git mv $D/helpers_gen.go               $D/helpers_v2.go
git mv $D/helpers_gen_container.go     $D/helpers_container_v2.go
git mv $D/helpers_domainmodels_gen.go  $D/helpers_domainmodels_v2.go
git mv $D/helpers_javaactions_gen.go   $D/helpers_javaactions_v2.go
git mv $D/helpers_pages_gen.go         $D/helpers_pages_v2.go
git mv $D/helpers_security_gen.go      $D/helpers_security_v2.go
git mv $D/helpers_workflows_gen.go     $D/helpers_workflows_v2.go
```

- [ ] **Step 4: Rename corresponding test files**

```bash
D=mdl/executor
# command file tests
git mv $D/cmd_microflows_create_gen_test.go         $D/cmd_microflows_create_v2_test.go
git mv $D/cmd_nanoflows_create_gen_test.go          $D/cmd_nanoflows_create_v2_test.go
git mv $D/cmd_nanoflows_drop_gen_test.go            $D/cmd_nanoflows_drop_v2_test.go
git mv $D/cmd_security_write_gen_test.go            $D/cmd_security_write_v2_test.go
git mv $D/cmd_security_write_modulerole_gen_test.go $D/cmd_security_write_modulerole_v2_test.go
git mv $D/cmd_structure_gen_test.go                 $D/cmd_structure_v2_test.go
git mv $D/cmd_workflows_write_gen2_test.go          $D/cmd_workflows_write_v2_test.go
# cmd_workflows_write_gen_test.go tests autoBindCallMicroflowGen helper
# (distinct from cmd_workflows_write_gen2_test.go — cannot share the same v2 name)
git mv $D/cmd_workflows_write_gen_test.go           $D/cmd_workflows_write_autobind_v2_test.go

# flowbuilder tests
git mv $D/flowbuilder_actions_change_gen_test.go    $D/flowbuilder_actions_change_v2_test.go
git mv $D/flowbuilder_actions_feedback_gen_test.go  $D/flowbuilder_actions_feedback_v2_test.go
git mv $D/flowbuilder_actions_gen_test.go           $D/flowbuilder_actions_v2_test.go
git mv $D/flowbuilder_actions_listop_gen_test.go    $D/flowbuilder_actions_listop_v2_test.go
git mv $D/flowbuilder_actions_retrieve_gen_test.go  $D/flowbuilder_actions_retrieve_v2_test.go
git mv $D/flowbuilder_annotations_gen_test.go       $D/flowbuilder_annotations_v2_test.go
git mv $D/flowbuilder_assoc_lookup_gen_test.go      $D/flowbuilder_assoc_lookup_v2_test.go
git mv $D/flowbuilder_calls_code_gen_test.go        $D/flowbuilder_calls_code_v2_test.go
git mv $D/flowbuilder_calls_external_gen_test.go    $D/flowbuilder_calls_external_v2_test.go
git mv $D/flowbuilder_calls_flow_gen_test.go        $D/flowbuilder_calls_flow_v2_test.go
git mv $D/flowbuilder_calls_page_gen_test.go        $D/flowbuilder_calls_page_v2_test.go
git mv $D/flowbuilder_calls_rest_gen_test.go        $D/flowbuilder_calls_rest_v2_test.go
git mv $D/flowbuilder_calls_send_rest_gen_test.go   $D/flowbuilder_calls_send_rest_v2_test.go
git mv $D/flowbuilder_calls_webservice_gen_test.go  $D/flowbuilder_calls_webservice_v2_test.go
git mv $D/flowbuilder_calls_xml_gen_test.go         $D/flowbuilder_calls_xml_v2_test.go
git mv $D/flowbuilder_control_if_gen_test.go        $D/flowbuilder_control_if_v2_test.go
git mv $D/flowbuilder_control_loop_gen_test.go      $D/flowbuilder_control_loop_v2_test.go
git mv $D/flowbuilder_control_split_gen_test.go     $D/flowbuilder_control_split_v2_test.go
git mv $D/flowbuilder_dispatch_gen_test.go          $D/flowbuilder_dispatch_v2_test.go
git mv $D/flowbuilder_eh_body_gen_test.go           $D/flowbuilder_eh_body_v2_test.go
git mv $D/flowbuilder_eh_queue_gen_test.go          $D/flowbuilder_eh_queue_v2_test.go
git mv $D/flowbuilder_flows_gen_test.go             $D/flowbuilder_flows_v2_test.go
git mv $D/flowbuilder_gen_test.go                   $D/flowbuilder_v2_test.go
git mv $D/flowbuilder_graph_gen_test.go             $D/flowbuilder_graph_v2_test.go
git mv $D/flowbuilder_raw_setter_gen_test.go        $D/flowbuilder_raw_setter_v2_test.go
git mv $D/flowbuilder_synchronize_gen_test.go       $D/flowbuilder_synchronize_v2_test.go
git mv $D/flowbuilder_workflow_gen_test.go          $D/flowbuilder_workflow_v2_test.go

# helpers + autocomplete tests
git mv $D/autocomplete_gen_test.go         $D/autocomplete_v2_test.go
git mv $D/helpers_gen_test.go              $D/helpers_v2_test.go
git mv $D/helpers_gen_container_test.go    $D/helpers_container_v2_test.go
git mv $D/helpers_domainmodels_gen_test.go $D/helpers_domainmodels_v2_test.go
git mv $D/helpers_javaactions_gen_test.go  $D/helpers_javaactions_v2_test.go
git mv $D/helpers_pages_gen_test.go        $D/helpers_pages_v2_test.go
git mv $D/helpers_security_gen_test.go     $D/helpers_security_v2_test.go
git mv $D/helpers_workflows_gen_test.go    $D/helpers_workflows_v2_test.go
```

- [ ] **Step 5: Verify no `_gen.go` files remain (except truly code-generated ones)**

```bash
find mdl/executor mdl/backend -name "*_gen.go" ! -name "*_test.go" \
  | xargs grep -L "// Code generated" 2>/dev/null
```

Expected: empty output — all hand-written `_gen` files have been renamed to `_v2`

- [ ] **Step 6: Verify build passes**

```bash
make build
```

Expected: success

- [ ] **Step 7: Verify tests pass**

```bash
make test
```

Expected: all tests pass

- [ ] **Step 8: Commit**

```bash
git commit -m "chore: rename _gen → _v2 in executor command, flowbuilder, helpers

_gen suffix on hand-written files was misleading (looks like go:generate
output). Files using modelsdk/gen/* types are now _v2 to express their
role as the new-path implementation during migration.

Special case: cmd_workflows_write_gen_test.go → cmd_workflows_write_autobind_v2_test.go
to avoid conflict with cmd_workflows_write_gen2_test.go → cmd_workflows_write_v2_test.go.

~114 files renamed (57 prod + test counterparts).

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Verification: Confirm Naming Rules Are Clean

After all 4 commits, run this final check:

```bash
# No hand-written _gen files remain
find mdl/executor mdl/backend -name "*_gen.go" | xargs grep -L "// Code generated" 2>/dev/null
# Expected: empty

# No _modelsdk files remain
find mdl/backend/mpr -name "*_modelsdk*"
# Expected: empty

# No _compat files remain
find mdl/backend/mpr -name "*_compat*"
# Expected: empty

# All mock files have mock_ prefix in mock/
ls mdl/backend/mock/*.go | grep -v "mock_"
# Expected: empty

# make build and make test still pass
make build && make test
```
