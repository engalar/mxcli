// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b — gen-typed workflow write helpers.
//
// The legacy autoBindCallMicroflow in cmd_workflows_write.go pulls
// every microflow via Backend.ListMicroflows() to look up the named
// target's parameter list and generate ParameterMappings on a
// workflow CallMicroflowTask. This file provides the modelsdk/gen
// equivalent that consumes ctx.Microflows and reads parameters out of
// the gen MicroflowObjectCollection — which is the same pattern
// Stage 3.2.1's genMicroflowParameters established.
//
// The CallMicroflowTask, ParameterMapping and outcome construction
// live in sdk/workflows and are unchanged here: workflow document
// types are not part of the microflow migration. This file only
// replaces the microflow lookup half of autoBindCallMicroflow.
//
// Other helpers in cmd_workflows_write.go (autoBindCallWorkflow,
// buildWorkflowActivities, deduplicateActivityNames, etc.) do not
// touch sdk/microflows so they stay in the legacy file.
//
// The dispatch layer (Stage 3.2.6) will route autoBindActivitiesInFlow
// to autoBindCallMicroflowGen and then delete the sdk-typed original.

package executor

import (
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// autoBindCallMicroflowGen mirrors autoBindCallMicroflow but resolves
// the called microflow + its parameters via ctx.Microflows (gen path).
//
// Behaviour parity:
//   - Sanitize task.Name.
//   - Inject a VoidConditionOutcome when none is present (Mendix runtime
//     CE6686 — every CallMicroflowTask must own at least one outcome).
//   - Skip parameter binding if explicit mappings already exist.
//   - Look up the microflow by qualified name; if not found (or the
//     gen repo is unavailable), return without injecting mappings —
//     same silent-fallback shape as the legacy function.
//
// Parameters are stored on the gen Microflow as MicroflowParameter
// objects in the ObjectCollection (no typed accessor exists on the
// Microflow type itself); we walk the collection and pick up any
// element whose TypeName is `Microflows$MicroflowParameter`, exactly
// as Stage 3.2.1's genMicroflowParameters does.
func autoBindCallMicroflowGen(ctx *ExecContext, task *workflows.CallMicroflowTask) {
	task.Name = sanitizeActivityName(task.Name)

	if len(task.Outcomes) == 0 {
		outcome := &workflows.VoidConditionOutcome{
			Flow: &workflows.Flow{},
		}
		outcome.BaseElement.ID = model.ID(types.GenerateID())
		outcome.Flow.BaseElement.ID = model.ID(types.GenerateID())
		task.Outcomes = append(task.Outcomes, outcome)
	}

	if len(task.ParameterMappings) > 0 {
		return
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil || len(mfs) == 0 {
		return
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		qualifiedName := modName + "." + mf.Name()
		if qualifiedName != task.Microflow {
			continue
		}

		for _, paramName := range genMicroflowParameterNames(mf) {
			paramQualifiedName := qualifiedName + "." + paramName
			mapping := &workflows.ParameterMapping{
				Parameter:  paramQualifiedName,
				Expression: "$WorkflowContext",
			}
			mapping.BaseElement.ID = model.ID(types.GenerateID())
			task.ParameterMappings = append(task.ParameterMappings, mapping)
		}
		return
	}
}

// genMicroflowParameterNames returns the parameter names of a gen
// Microflow, walking the ObjectCollection for MicroflowParameter
// elements. Mirrors the inline scan in genMicroflowParameters
// (cmd_microflows_show_gen.go) but returns just the names — that's
// all the workflow auto-binder needs.
func genMicroflowParameterNames(mf *genMf.Microflow) []string {
	if mf == nil {
		return nil
	}
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}
	var out []string
	for _, obj := range oc.ObjectsItems() {
		if obj == nil {
			continue
		}
		if obj.TypeName() != "Microflows$MicroflowParameter" {
			continue
		}
		if nv, ok := obj.(interface{ NameValue() string }); ok {
			out = append(out, nv.NameValue())
		}
	}
	return out
}
