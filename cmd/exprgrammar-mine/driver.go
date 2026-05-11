// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func MineMPR(m *Miner, mprPath string) error {
	be := mprbackend.New()
	if err := be.Connect(mprPath); err != nil {
		return fmt.Errorf("open mpr: %w", err)
	}
	defer be.Disconnect()
	ctx := newMiningContext(be)

	mfs, err := be.ListMicroflows()
	if err != nil {
		return fmt.Errorf("list microflows: %w", err)
	}
	h, err := executor.GetHierarchyForMining(ctx)
	if err != nil {
		return fmt.Errorf("hierarchy: %w", err)
	}
	for _, mf := range mfs {
		modID := h.FindModuleID(mf.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == "" || mf.Name == "" {
			continue
		}
		qn := ast.QualifiedName{Module: modName, Name: mf.Name}
		mdlText, err := executor.DescribeMicroflowToString(ctx, qn)
		if err != nil {
			fmt.Printf("skip %s: %v\n", qn.String(), err)
			continue
		}
		if err := WalkMDL(m, qn.String(), mdlText); err != nil {
			return fmt.Errorf("walk %s: %w", qn.String(), err)
		}
	}
	return nil
}

func newMiningContext(be backend.FullBackend) *executor.ExecContext {
	return &executor.ExecContext{
		Context: context.Background(),
		Backend: be,
		Output:  io.Discard,
		Quiet:   true,
	}
}
