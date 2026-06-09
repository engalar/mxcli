// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// BrokenCallerRef describes a MicroflowCallParameterMapping that references
// a parameter which no longer exists on the target microflow.
type BrokenCallerRef struct {
	CallerName  string // qualified name of the microflow containing the broken call
	TargetMF    string // qualified name of the called microflow
	BrokenParam string // the full ParameterQualifiedName that is stale
}

// scanBrokenCallerRefs walks all provided microflows and returns every
// MicroflowCallParameterMapping whose ParameterQualifiedName starts with
// targetMFQN+"." and whose parameter name (last dot segment) is in removedParams.
func scanBrokenCallerRefs(
	allMFs []*genMf.Microflow,
	callerNames map[*genMf.Microflow]string,
	targetMFQN string,
	removedParams []string,
) []BrokenCallerRef {
	if len(removedParams) == 0 {
		return nil
	}
	removed := make(map[string]bool, len(removedParams))
	for _, p := range removedParams {
		removed[p] = true
	}
	prefix := targetMFQN + "."

	var out []BrokenCallerRef
	for _, mf := range allMFs {
		if mf == nil {
			continue
		}
		oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if !ok || oc == nil {
			continue
		}
		callerQN := callerNames[mf]
		for _, obj := range oc.ObjectsItems() {
			callAction, ok := obj.(*genMf.MicroflowCallAction)
			if !ok {
				continue
			}
			call, ok := callAction.MicroflowCall().(*genMf.MicroflowCall)
			if !ok || call == nil {
				continue
			}
			if call.MicroflowQualifiedName() != targetMFQN {
				continue
			}
			for _, pmElem := range call.ParameterMappingsItems() {
				pm, ok := pmElem.(*genMf.MicroflowCallParameterMapping)
				if !ok || pm == nil {
					continue
				}
				pqn := pm.ParameterQualifiedName()
				if !strings.HasPrefix(pqn, prefix) {
					continue
				}
				paramName := pqn[len(prefix):]
				if removed[paramName] {
					out = append(out, BrokenCallerRef{
						CallerName:  callerQN,
						TargetMF:    targetMFQN,
						BrokenParam: pqn,
					})
				}
			}
		}
	}
	return out
}

// microflowParamNamesFromOC extracts parameter names from a microflow's
// ObjectCollection using the typed Name() method — works for both freshly
// constructed gen elements (no raw BSON) and decoded-from-BSON elements.
// Contrast with genMicroflowParameterNames which relies on NameValue() (raw BSON only).
func microflowParamNamesFromOC(mf *genMf.Microflow) []string {
	if mf == nil {
		return nil
	}
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}
	var out []string
	for _, obj := range oc.ObjectsItems() {
		if param, ok := obj.(*genMf.MicroflowParameter); ok && param != nil {
			if n := param.Name(); n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

// warnBrokenCallerRefs 扫描所有调用方，对即将产生 CE1613 的破损引用打印警告。
// 使用 ctx.Microflows.ListAll()；若不可用则静默跳过。
// 注意：这里只警告，不修复。如需修复，使用 MFCallerRefFixer.RemoveStaleMappings。
func warnBrokenCallerRefs(ctx *ExecContext, targetMFQN string, removedParams []string) {
	if len(removedParams) == 0 || ctx.Microflows == nil {
		return
	}
	allMFs, err := ctx.Microflows.ListAll()
	if err != nil || len(allMFs) == 0 {
		return
	}

	names := make(map[*genMf.Microflow]string, len(allMFs))
	h, _ := getHierarchy(ctx)
	for _, mf := range allMFs {
		if mf == nil {
			continue
		}
		if h != nil {
			if cid, err2 := ctx.Microflows.GetContainerUUID(model.ID(mf.ID())); err2 == nil {
				modID := h.FindModuleID(cid)
				if mod := h.GetModuleName(modID); mod != "" {
					names[mf] = mod + "." + mf.Name()
					continue
				}
			}
		}
		names[mf] = mf.Name()
	}

	refs := scanBrokenCallerRefs(allMFs, names, targetMFQN, removedParams)
	if len(refs) == 0 {
		return
	}

	fmt.Fprintf(ctx.Output,
		"warning: microflow %s signature changed — %d caller(s) reference removed parameter(s):\n",
		targetMFQN, len(refs))
	for _, r := range refs {
		fmt.Fprintf(ctx.Output,
			"  CE1613 risk: %s → %s (parameter %q no longer exists)\n",
			r.CallerName, r.TargetMF, r.BrokenParam)
	}
	fmt.Fprintf(ctx.Output,
		"  Fix: use MFCallerRefFixer.RemoveStaleMappings() or update callers manually.\n")
}
