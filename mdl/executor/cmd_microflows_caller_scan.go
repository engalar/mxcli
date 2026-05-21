// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

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
