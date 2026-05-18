// SPDX-License-Identifier: Apache-2.0

package typecheck

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

type defaultFuncReg struct{}

// NewFuncReg returns a FuncReg backed by the exprcheck built-in function table.
func NewFuncReg() FuncReg { return &defaultFuncReg{} }

func (r *defaultFuncReg) ReturnType(name string) (exprcheck.TypeKind, bool) {
	return exprcheck.FuncReturnKind(name)
}
