// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// execAlterWorkflowDepsImpl is the HandlerDeps implementation for ALTER WORKFLOW.
func execAlterWorkflowDepsImpl(ctx context.Context, s *ast.AlterWorkflowStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	if err := checkFeatureDeps(deps, "workflows", "basic",
		"alter workflow",
		"upgrade your project to Mendix 9.12+ to use workflows"); err != nil {
		return err
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Use temp ExecContext for deep workflow helper chains
	tmpCtx := NewExecContext(ctx, deps)

	pairs, err := listWorkflowsWithContainerGen(tmpCtx)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}

	var wfID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			wfID = model.ID(p.Elem.ID())
			break
		}
	}
	if wfID == "" {
		return mdlerrors.NewNotFound("workflow", s.Name.Module+"."+s.Name.Name)
	}

	mutator, err := deps.WorkflowMutationOperator.OpenWorkflowForMutation(wfID)
	if err != nil {
		return mdlerrors.NewBackend("open workflow for mutation", err)
	}

	for _, op := range s.Operations {
		switch o := op.(type) {
		case *ast.SetWorkflowPropertyOp:
			switch o.Property {
			case "overview_page":
				qn := o.Entity.Module + "." + o.Entity.Name
				if qn == "." {
					qn = ""
				}
				if err := mutator.SetPropertyWithEntity(o.Property, qn, qn); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			case "parameter":
				qn := o.Entity.Module + "." + o.Entity.Name
				if qn == "." {
					qn = ""
				}
				if err := mutator.SetPropertyWithEntity(o.Property, o.Value, qn); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			default:
				if err := mutator.SetProperty(o.Property, o.Value); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			}

		case *ast.SetActivityPropertyOp:
			value := o.Value
			switch o.Property {
			case "page":
				value = o.PageName.Module + "." + o.PageName.Name
			case "targeting_microflow":
				value = o.Microflow.Module + "." + o.Microflow.Name
			}
			if err := mutator.SetActivityProperty(o.ActivityRef, o.AtPosition, o.Property, value); err != nil {
				return mdlerrors.NewBackend("set activity", err)
			}

		case *ast.InsertAfterOp:
			acts := buildAndBindActivitiesGen(tmpCtx, []ast.WorkflowActivityNode{o.NewActivity})
			if len(acts) == 0 {
				return mdlerrors.NewValidation("failed to build new activity")
			}
			if err := mutator.InsertAfterActivityGen(o.ActivityRef, o.AtPosition, acts); err != nil {
				return mdlerrors.NewBackend("insert after", err)
			}

		case *ast.DropActivityOp:
			if err := mutator.DropActivity(o.ActivityRef, o.AtPosition); err != nil {
				return mdlerrors.NewBackend("drop activity", err)
			}

		case *ast.ReplaceActivityOp:
			acts := buildAndBindActivitiesGen(tmpCtx, []ast.WorkflowActivityNode{o.NewActivity})
			if len(acts) == 0 {
				return mdlerrors.NewValidation("failed to build replacement activity")
			}
			if err := mutator.ReplaceActivityGen(o.ActivityRef, o.AtPosition, acts); err != nil {
				return mdlerrors.NewBackend("replace activity", err)
			}

		case *ast.InsertOutcomeOp:
			acts := buildAndBindActivitiesGen(tmpCtx, o.Activities)
			if err := mutator.InsertOutcomeGen(o.ActivityRef, o.AtPosition, o.OutcomeName, acts); err != nil {
				return mdlerrors.NewBackend("insert outcome", err)
			}

		case *ast.DropOutcomeOp:
			if err := mutator.DropOutcome(o.ActivityRef, o.AtPosition, o.OutcomeName); err != nil {
				return mdlerrors.NewBackend("drop outcome", err)
			}

		case *ast.InsertPathOp:
			acts := buildAndBindActivitiesGen(tmpCtx, o.Activities)
			if err := mutator.InsertPathGen(o.ActivityRef, o.AtPosition, "", acts); err != nil {
				return mdlerrors.NewBackend("insert path", err)
			}

		case *ast.DropPathOp:
			if err := mutator.DropPath(o.ActivityRef, o.AtPosition, o.PathCaption); err != nil {
				return mdlerrors.NewBackend("drop path", err)
			}

		case *ast.InsertBranchOp:
			acts := buildAndBindActivitiesGen(tmpCtx, o.Activities)
			if err := mutator.InsertBranchGen(o.ActivityRef, o.AtPosition, o.Condition, acts); err != nil {
				return mdlerrors.NewBackend("insert branch", err)
			}

		case *ast.DropBranchOp:
			if err := mutator.DropBranch(o.ActivityRef, o.AtPosition, o.BranchName); err != nil {
				return mdlerrors.NewBackend("drop branch", err)
			}

		case *ast.InsertBoundaryEventOp:
			acts := buildAndBindActivitiesGen(tmpCtx, o.Activities)
			if o.EventType == "NonInterruptingTimer" {
				endPath := genWf.NewEndOfBoundaryEventPathActivity()
				endPath.SetID(element.ID(types.GenerateID()))
				endPath.SetCaption("End of boundary path")
				endPath.SetName("endOfBoundaryEventPath1")
				acts = append(acts, endPath)
			} else if !endsWithTerminalWorkflowActivity(acts) {
				end := genWf.NewEndWorkflowActivity()
				end.SetID(element.ID(types.GenerateID()))
				end.SetCaption("End")
				end.SetName("End")
				acts = append(acts, end)
			}
			if err := mutator.InsertBoundaryEventGen(o.ActivityRef, o.AtPosition, o.EventType, o.Delay, acts); err != nil {
				return mdlerrors.NewBackend("insert boundary event", err)
			}

		case *ast.DropBoundaryEventOp:
			if err := mutator.DropBoundaryEvent(o.ActivityRef, o.AtPosition); err != nil {
				return mdlerrors.NewBackend("drop boundary event", err)
			}

		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unknown alter workflow operation type: %T", op))
		}
	}

	if err := mutator.Save(); err != nil {
		return mdlerrors.NewBackend("save modified workflow", err)
	}

	invalidateHierarchyDeps(deps)
	fmt.Fprintf(deps.Output, "Altered workflow %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// ExecAlterWorkflow handles ALTER WORKFLOW Module.Name { operations }.
func ExecAlterWorkflow(ctx *ExecContext, s *ast.AlterWorkflowStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	// Version pre-check: workflows require Mendix 9.12+
	if err := checkFeature(ctx, "workflows", "basic",
		"alter workflow",
		"upgrade your project to Mendix 9.12+ to use workflows"); err != nil {
		return err
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Find workflow by qualified name (gen-typed lookup via cache helper).
	pairs, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}

	var wfID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			wfID = model.ID(p.Elem.ID())
			break
		}
	}
	if wfID == "" {
		return mdlerrors.NewNotFound("workflow", s.Name.Module+"."+s.Name.Name)
	}

	// Open mutator
	mutator, err := ctx.WorkflowMutationOperator.OpenWorkflowForMutation(wfID)
	if err != nil {
		return mdlerrors.NewBackend("open workflow for mutation", err)
	}

	// Apply operations sequentially
	for _, op := range s.Operations {
		switch o := op.(type) {
		case *ast.SetWorkflowPropertyOp:
			switch o.Property {
			case "overview_page":
				// overview_page uses Entity as the page qualified name (Value is unused).
				qn := o.Entity.Module + "." + o.Entity.Name
				if qn == "." {
					qn = ""
				}
				if err := mutator.SetPropertyWithEntity(o.Property, qn, qn); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			case "parameter":
				// PARAMETER uses Value as the variable name and Entity as the entity qualified name.
				qn := o.Entity.Module + "." + o.Entity.Name
				if qn == "." {
					qn = ""
				}
				if err := mutator.SetPropertyWithEntity(o.Property, o.Value, qn); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			default:
				if err := mutator.SetProperty(o.Property, o.Value); err != nil {
					return mdlerrors.NewBackend("set "+o.Property, err)
				}
			}

		case *ast.SetActivityPropertyOp:
			value := o.Value
			switch o.Property {
			case "page":
				value = o.PageName.Module + "." + o.PageName.Name
			case "targeting_microflow":
				value = o.Microflow.Module + "." + o.Microflow.Name
			}
			if err := mutator.SetActivityProperty(o.ActivityRef, o.AtPosition, o.Property, value); err != nil {
				return mdlerrors.NewBackend("set activity", err)
			}

		case *ast.InsertAfterOp:
			acts := buildAndBindActivitiesGen(ctx, []ast.WorkflowActivityNode{o.NewActivity})
			if len(acts) == 0 {
				return mdlerrors.NewValidation("failed to build new activity")
			}
			if err := mutator.InsertAfterActivityGen(o.ActivityRef, o.AtPosition, acts); err != nil {
				return mdlerrors.NewBackend("insert after", err)
			}

		case *ast.DropActivityOp:
			if err := mutator.DropActivity(o.ActivityRef, o.AtPosition); err != nil {
				return mdlerrors.NewBackend("drop activity", err)
			}

		case *ast.ReplaceActivityOp:
			acts := buildAndBindActivitiesGen(ctx, []ast.WorkflowActivityNode{o.NewActivity})
			if len(acts) == 0 {
				return mdlerrors.NewValidation("failed to build replacement activity")
			}
			if err := mutator.ReplaceActivityGen(o.ActivityRef, o.AtPosition, acts); err != nil {
				return mdlerrors.NewBackend("replace activity", err)
			}

		case *ast.InsertOutcomeOp:
			acts := buildAndBindActivitiesGen(ctx, o.Activities)
			if err := mutator.InsertOutcomeGen(o.ActivityRef, o.AtPosition, o.OutcomeName, acts); err != nil {
				return mdlerrors.NewBackend("insert outcome", err)
			}

		case *ast.DropOutcomeOp:
			if err := mutator.DropOutcome(o.ActivityRef, o.AtPosition, o.OutcomeName); err != nil {
				return mdlerrors.NewBackend("drop outcome", err)
			}

		case *ast.InsertPathOp:
			acts := buildAndBindActivitiesGen(ctx, o.Activities)
			if err := mutator.InsertPathGen(o.ActivityRef, o.AtPosition, "", acts); err != nil {
				return mdlerrors.NewBackend("insert path", err)
			}

		case *ast.DropPathOp:
			if err := mutator.DropPath(o.ActivityRef, o.AtPosition, o.PathCaption); err != nil {
				return mdlerrors.NewBackend("drop path", err)
			}

		case *ast.InsertBranchOp:
			acts := buildAndBindActivitiesGen(ctx, o.Activities)
			if err := mutator.InsertBranchGen(o.ActivityRef, o.AtPosition, o.Condition, acts); err != nil {
				return mdlerrors.NewBackend("insert branch", err)
			}

		case *ast.DropBranchOp:
			if err := mutator.DropBranch(o.ActivityRef, o.AtPosition, o.BranchName); err != nil {
				return mdlerrors.NewBackend("drop branch", err)
			}

		case *ast.InsertBoundaryEventOp:
			acts := buildAndBindActivitiesGen(ctx, o.Activities)
			if o.EventType == "NonInterruptingTimer" {
				// CE1844: EndWorkflowActivity forbidden in non-interrupting boundary events.
				// Use EndOfBoundaryEventPathActivity instead — ends only the boundary path.
				endPath := genWf.NewEndOfBoundaryEventPathActivity()
				endPath.SetID(element.ID(types.GenerateID()))
				endPath.SetCaption("End of boundary path")
				endPath.SetName("endOfBoundaryEventPath1")
				acts = append(acts, endPath)
			} else if !endsWithTerminalWorkflowActivity(acts) {
				// InterruptingTimer: inject EndWorkflowActivity at the boundary event level.
				end := genWf.NewEndWorkflowActivity()
				end.SetID(element.ID(types.GenerateID()))
				end.SetCaption("End")
				end.SetName("End")
				acts = append(acts, end)
			}
			if err := mutator.InsertBoundaryEventGen(o.ActivityRef, o.AtPosition, o.EventType, o.Delay, acts); err != nil {
				return mdlerrors.NewBackend("insert boundary event", err)
			}

		case *ast.DropBoundaryEventOp:
			if err := mutator.DropBoundaryEvent(o.ActivityRef, o.AtPosition); err != nil {
				return mdlerrors.NewBackend("drop boundary event", err)
			}

		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unknown alter workflow operation type: %T", op))
		}
	}

	// Save
	if err := mutator.Save(); err != nil {
		return mdlerrors.NewBackend("save modified workflow", err)
	}

	invalidateHierarchy(ctx)
	fmt.Fprintf(ctx.Output, "Altered workflow %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// buildAndBindActivitiesGen is the gen-typed twin used by ALTER WORKFLOW.
// Mirrors buildAndBindActivities semantically but returns []element.Element
// for the gen mutator surface (D7).
func buildAndBindActivitiesGen(ctx *ExecContext, nodes []ast.WorkflowActivityNode) []element.Element {
	wbc := newWfBuildCtx(ctx)
	acts := buildWorkflowActivitiesGen(wbc, nodes)
	autoBindWorkflowGen(ctx, acts)
	deduplicateActivityNamesGen(acts)
	return acts
}
