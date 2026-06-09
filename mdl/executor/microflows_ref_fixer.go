// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// RefFixAction 描述一次成功的引用修复操作。
type RefFixAction struct {
	CallerMF string // 被修改的调用方微流限定名
	TargetMF string // 被调用的目标微流限定名
	OldParam string // 被修复的完整参数限定名（旧）
	NewParam string // 替换后的参数名；空字符串表示已删除
}

// RefFixReport 汇总一次修复操作的结果。
type RefFixReport struct {
	Fixed  []RefFixAction
	Errors []error
}

// FixedCount 返回成功修复的数量。
func (r *RefFixReport) FixedCount() int { return len(r.Fixed) }

// MFCallerRefFixer 通过 ctx.Microflows 读写微流，批量修复悬空的
// MicroflowCallParameterMapping 引用。
type MFCallerRefFixer struct {
	ctx *ExecContext
}

// NewMFCallerRefFixer 创建修复器。
func NewMFCallerRefFixer(ctx *ExecContext) *MFCallerRefFixer {
	return &MFCallerRefFixer{ctx: ctx}
}

// RemoveStaleMappings 删除所有调用方中指向 targetMFQN 上已被移除参数的
// MicroflowCallParameterMapping 条目，并写回 MPR。
//
// 典型使用场景：CREATE OR REPLACE MICROFLOW 去掉了某个参数后，
// 调用此方法自动清理所有调用方，避免 CE1613。
func (f *MFCallerRefFixer) RemoveStaleMappings(targetMFQN string, removedParams []string) (*RefFixReport, error) {
	return f.applyFix(targetMFQN, removedParams, "")
}

// RemapParam 将所有调用方中对 targetMFQN.oldParamName 的引用改为
// targetMFQN.newParamName，并写回 MPR。
//
// 典型使用场景：参数被重命名，调用此方法批量更新调用方。
func (f *MFCallerRefFixer) RemapParam(targetMFQN, oldParamName, newParamName string) (*RefFixReport, error) {
	return f.applyFix(targetMFQN, []string{oldParamName}, newParamName)
}

// applyFix 是 RemoveStaleMappings 和 RemapParam 的共同实现。
// newParamName == "" 表示删除；否则表示重命名。
func (f *MFCallerRefFixer) applyFix(targetMFQN string, staleParams []string, newParamName string) (*RefFixReport, error) {
	if len(staleParams) == 0 || f.ctx.Microflows == nil {
		return &RefFixReport{}, nil
	}

	allMFs, err := f.ctx.Microflows.ListAll()
	if err != nil {
		return nil, fmt.Errorf("list microflows: %w", err)
	}

	// 构建 caller 限定名映射
	names := f.buildCallerNames(allMFs)
	// 找到所有破损引用
	broken := scanBrokenCallerRefs(allMFs, names, targetMFQN, staleParams)
	if len(broken) == 0 {
		return &RefFixReport{}, nil
	}

	// 按调用方分组，减少写操作次数
	byCaller := make(map[string][]BrokenCallerRef)
	for _, b := range broken {
		byCaller[b.CallerName] = append(byCaller[b.CallerName], b)
	}

	// 建立 callerQN → mf 的映射（反向索引）
	mfByName := make(map[string]*genMf.Microflow, len(allMFs))
	for _, mf := range allMFs {
		if mf == nil {
			continue
		}
		mfByName[names[mf]] = mf
	}

	report := &RefFixReport{}
	prefix := targetMFQN + "."

	for callerQN, refs := range byCaller {
		mf := mfByName[callerQN]
		if mf == nil {
			report.Errors = append(report.Errors, fmt.Errorf("caller %q not found in mf map", callerQN))
			continue
		}

		brokenSet := make(map[string]bool, len(refs))
		for _, r := range refs {
			brokenSet[r.BrokenParam] = true
		}

		fixed, err := f.patchMFMappings(mf, targetMFQN, prefix, brokenSet, newParamName)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Errorf("patch %s: %w", callerQN, err))
			continue
		}
		if len(fixed) == 0 {
			continue
		}

		if err := f.ctx.Microflows.Update(mf); err != nil {
			report.Errors = append(report.Errors, fmt.Errorf("update %s: %w", callerQN, err))
			continue
		}
		for _, pqn := range fixed {
			report.Fixed = append(report.Fixed, RefFixAction{
				CallerMF: callerQN,
				TargetMF: targetMFQN,
				OldParam: pqn,
				NewParam: newParamName,
			})
		}
	}
	return report, nil
}

// patchMFMappings 修改单个微流对象中的 ParameterMappings（不写磁盘）。
// 返回被修改的 ParameterQualifiedName 列表。
func (f *MFCallerRefFixer) patchMFMappings(
	mf *genMf.Microflow,
	targetMFQN, prefix string,
	brokenSet map[string]bool,
	newParamName string,
) ([]string, error) {
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil, nil
	}

	var patched []string
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

		// 倒序删除/替换，避免 index 偏移
		items := call.ParameterMappingsItems()
		for i := len(items) - 1; i >= 0; i-- {
			pm, ok := items[i].(*genMf.MicroflowCallParameterMapping)
			if !ok || pm == nil {
				continue
			}
			pqn := pm.ParameterQualifiedName()
			if !strings.HasPrefix(pqn, prefix) {
				continue
			}
			if !brokenSet[pqn] {
				continue
			}

			if newParamName == "" {
				// 删除
				call.RemoveParameterMappings(i)
			} else {
				// 重命名
				pm.SetParameterQualifiedName(targetMFQN + "." + newParamName)
			}
			patched = append(patched, pqn)
		}
	}
	return patched, nil
}

// buildCallerNames 为所有微流构建 *genMf.Microflow → "Module.Name" 的映射。
func (f *MFCallerRefFixer) buildCallerNames(allMFs []*genMf.Microflow) map[*genMf.Microflow]string {
	names := make(map[*genMf.Microflow]string, len(allMFs))
	h, _ := getHierarchy(f.ctx)
	for _, mf := range allMFs {
		if mf == nil {
			continue
		}
		if h != nil {
			if cid, err := f.ctx.Microflows.GetContainerUUID(model.ID(mf.ID())); err == nil {
				modID := h.FindModuleID(cid)
				if mod := h.GetModuleName(modID); mod != "" {
					names[mf] = mod + "." + mf.Name()
					continue
				}
			}
		}
		names[mf] = mf.Name()
	}
	return names
}
