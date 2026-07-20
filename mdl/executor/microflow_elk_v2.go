// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Microflow ELK visualization (gen-typed track).
//
// This file is the gen-typed parallel of legacy `cmd_microflow_elk.go`.
// It is intentionally thin: every elk graph builder
// (`buildFlowELKGen`, node/edge constructors, size formula, JSON
// emitter) was already introduced in Stage 3.2.4 commit 1 alongside
// `cmd_nanoflow_elk_gen.go`, so this file only adds the Microflow
// resolution / entry plumbing.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// MicroflowELKGen is the gen-typed parallel of Executor.MicroflowELK.
// Resolves the microflow via the gen MicroflowRepository's
// FindByQualifiedName helper and renders the same JSON ELK graph
// schema legacy produces.
func (e *Executor) MicroflowELKGen(name string) error {
	return microflowELKGenDeps(e.buildHandlerDeps(), name)
}

// MicroflowELK is the public Executor wrapper consumed by
// cmd/mxcli/cmd_describe.go. Stage 3.2.6.3a moved this method out of
// the (deleted) cmd_microflow_elk.go; the body always routes through
// the gen path now that legacy `microflowELK` is gone.
func (e *Executor) MicroflowELK(name string) error {
	return microflowELKGenDeps(e.buildHandlerDeps(), name)
}

func microflowELKGen(ctx *ExecContext, name string) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflow repository", fmt.Errorf("ctx.Microflows is nil"))
	}

	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return mdlerrors.NewValidationf("expected qualified name Module.Microflow, got: %s", name)
	}
	qn := ast.QualifiedName{Module: parts[0], Name: parts[1]}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	entityNames, err := buildEntityNames(ctx, h)
	if err != nil {
		return err
	}

	targetMf, err := ctx.Microflows.FindByQualifiedName(name)
	if err != nil {
		return mdlerrors.NewBackend("find microflow", err)
	}
	if targetMf == nil {
		return mdlerrors.NewNotFound("microflow", name)
	}

	mdlSource, sourceMap, _ := describeMicroflowGenToString(ctx, qn)

	return buildFlowELKGen(ctx, flowELKInputGen{
		FlowType:         "microflow",
		QualifiedName:    name,
		ReturnType:       targetMf.ReturnType(),
		Parameters:       genFlowParametersFromCollection(targetMf.ObjectCollection()),
		ObjectCollection: targetMf.ObjectCollection(),
		TopLevelFlows:    targetMf.FlowsItems(),
		EntityNames:      entityNames,
		MdlSource:        mdlSource,
		SourceMap:        sourceMap,
	})
}

func microflowELKGenDeps(deps *HandlerDeps, name string) error {
	return microflowELKGen(execContextFromDeps(deps), name)
}


