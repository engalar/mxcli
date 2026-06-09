// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a — gen-typed microflow / nanoflow diff helpers.
//
// `mxcli diff` compares an MDL script against project state. The
// per-statement dispatcher in cmd_diff.go calls diffMicroflow /
// diffNanoflow, both of which reach into Backend.ListMicroflows and
// Backend.ListNanoflows to locate the existing flow. This file provides
// the modelsdk/gen-native equivalents that consume ctx.Microflows /
// ctx.Nanoflows.
//
// The "current" MDL rendering still flows through the legacy
// describeMicroflow / describeNanoflow helpers — that surface is owned
// by Stages 3.2.1–3.2.2 (describe) and 3.2.5c (nanoflow describe), not
// 3.2.5a. Stage 3.2.6 will rotate diff dispatch to these *Gen variants
// and delete the originals once describe is gen-typed end-to-end.

package executor

import (
	"bytes"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

// diffMicroflowGen compares a CREATE MICROFLOW statement against the
// project, locating the existing microflow via ctx.Microflows + the
// SQL-backed container chain rather than Backend.ListMicroflows.
func diffMicroflowGen(ctx *ExecContext, s *ast.CreateMicroflowStmt) (*DiffResult, error) {
	result := &DiffResult{
		ObjectType: "Microflow",
		ObjectName: s.Name,
		Proposed:   microflowStmtToMDL(ctx, s),
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		result.IsNew = true
		return result, nil
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		result.IsNew = true
		return result, nil
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if modName == s.Name.Module && mf.Name() == s.Name.Name {
			var buf bytes.Buffer
			oldOutput := ctx.Output
			ctx.Output = &buf
			describeMicroflowGen(ctx, s.Name)
			ctx.Output = oldOutput
			result.Current = strings.TrimSuffix(buf.String(), "\n")
			result.Changes = compareMicroflows(ctx, result.Current, result.Proposed)
			return result, nil
		}
	}

	result.IsNew = true
	return result, nil
}

// diffNanoflowGen mirrors diffMicroflowGen for nanoflows.
func diffNanoflowGen(ctx *ExecContext, s *ast.CreateNanoflowStmt) (*DiffResult, error) {
	result := &DiffResult{
		ObjectType: "Nanoflow",
		ObjectName: s.Name,
		Proposed:   nanoflowStmtToMDL(ctx, s),
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		result.IsNew = true
		return result, nil
	}

	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		result.IsNew = true
		return result, nil
	}

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == s.Name.Module && nf.Name() == s.Name.Name {
			var buf bytes.Buffer
			if err := func() error {
				oldOutput := ctx.Output
				ctx.Output = &buf
				defer func() { ctx.Output = oldOutput }()
				return describeNanoflowGen(ctx, s.Name)
			}(); err != nil {
				return nil, err
			}
			result.Current = strings.TrimSuffix(buf.String(), "\n")
			result.Changes = compareMicroflows(ctx, result.Current, result.Proposed)
			return result, nil
		}
	}

	result.IsNew = true
	return result, nil
}
