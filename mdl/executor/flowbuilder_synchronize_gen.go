// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addSynchronizeActionGen adds a Microflows$SynchronizeAction to the flow.
//
//	synchronize $Order;  → Type="SelectedObjects", VariableNames="Order"
//	synchronize;         → Type="All", VariableNames=""
func (fb *flowBuilderGen) addSynchronizeActionGen(s *ast.SynchronizeStmt) element.ID {
	action := genMf.NewSynchronizeAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(nil))

	if s.Variable != "" {
		action.SetType("SelectedObjects")
		action.SetVariableNames(s.Variable)
	} else {
		action.SetType("All")
		action.SetVariableNames("")
	}

	return fb.genActivityWrap(action, nil, "")
}
