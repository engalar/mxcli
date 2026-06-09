// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// EntityParamRef 描述一个以某实体为类型的微流参数。
type EntityParamRef struct {
	MFName    string // 微流限定名
	ParamName string // 参数名
	IsList    bool   // true = "list of Entity"
}

// scanMicroflowsForEntityParam 返回所有以 entityQN 为参数类型的微流参数。
// 同时匹配单实体（Object）和列表（List of）两种情况。
func scanMicroflowsForEntityParam(
	allMFs []*genMf.Microflow,
	names map[*genMf.Microflow]string,
	entityQN string,
) []EntityParamRef {
	var out []EntityParamRef
	for _, mf := range allMFs {
		if mf == nil {
			continue
		}
		oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if !ok || oc == nil {
			continue
		}
		for _, obj := range oc.ObjectsItems() {
			param, ok := obj.(*genMf.MicroflowParameter)
			if !ok {
				continue
			}
			qn, isList := entityQNFromParamType(param)
			if qn == entityQN {
				out = append(out, EntityParamRef{
					MFName:    names[mf],
					ParamName: param.Name(),
					IsList:    isList,
				})
			}
		}
	}
	return out
}

// entityQNFromParamType 从 MicroflowParameter 的 ParameterType 中提取实体限定名。
// 返回 ("", false) 如果不是实体/列表类型。
func entityQNFromParamType(param *genMf.MicroflowParameter) (qn string, isList bool) {
	if param == nil {
		return "", false
	}
	vt := param.ParameterType()
	if vt == nil {
		return "", false
	}
	switch t := vt.(type) {
	case *genDt.ObjectType:
		return t.EntityQualifiedName(), false
	case *genDt.ListType:
		return t.EntityQualifiedName(), true
	}
	return "", false
}

// warnMicroflowEntityParamRefs 打印警告：entityQN 被 DROP 后，
// 哪些微流参数类型会失效（CE1613）。
// 不依赖 catalog；直接遍历 ctx.Microflows.ListAll()。
func warnMicroflowEntityParamRefs(ctx *ExecContext, entityQN string) {
	if ctx.Microflows == nil {
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

	refs := scanMicroflowsForEntityParam(allMFs, names, entityQN)
	if len(refs) == 0 {
		return
	}
	fmt.Fprintf(ctx.Output,
		"warning: entity %s has %d microflow parameter(s) typed to it:\n",
		entityQN, len(refs))
	for _, r := range refs {
		kind := "Object"
		if r.IsList {
			kind = "List"
		}
		fmt.Fprintf(ctx.Output,
			"  CE1613 risk: %s.$%s (%s %s)\n",
			r.MFName, r.ParamName, kind, entityQN)
	}
}
