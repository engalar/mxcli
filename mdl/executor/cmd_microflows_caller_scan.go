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
	// TODO: implement in Task 2
	_ = strings.HasPrefix // suppress unused import
	return nil
}
