// SPDX-License-Identifier: Apache-2.0

// Stage 3.1 cutover plumbing: extract modelsdk-native repositories from
// a backend.FullBackend implementation that supports them. Defined as a
// standalone interface check so the executor doesn't import the concrete
// mprbackend package — keeps the dependency direction
// (executor → backend interface, never executor → concrete impl).

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
)

// microflowsRepoProvider is the duck-typed interface a backend must
// satisfy to expose the modelsdk-native MicroflowRepository. MprBackend
// implements this in repos_provider.go.
type microflowsRepoProvider interface {
	Microflows() repos.MicroflowRepository
}

// extractMicroflowsRepo returns the modelsdk-native MicroflowRepository
// if the backend supports it, else nil. Callers (handlers) MUST check
// nil and fall back to ctx.Backend during the Stage 3.x incremental
// cutover — once every microflow handler migrates, this fallback can
// be removed and the field can be made non-nil-required.
func extractMicroflowsRepo(b backend.FullBackend) repos.MicroflowRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(microflowsRepoProvider); ok {
		return p.Microflows()
	}
	return nil
}

// deleteMicroflowViaRepoOrBackend is the Stage 3.1 cutover bridge for
// delete-by-ID. Uses the modelsdk-native MicroflowRepository when
// available (production paths through MprBackend), else falls back to
// the legacy ctx.Backend.DeleteMicroflow (test contexts using
// MockBackend or hand-built ExecContext literals).
//
// Only delete-by-ID migrates in Stage 3.1: it has no sdk-vs-gen type
// mismatch in its parameter or return surface. List/Get/Create/Update
// stay on ctx.Backend because handlers consume sdk/microflows.Microflow
// pervasively (DescribeMicroflowToString, renderMicroflowMDL, all 60+
// activity emitters). Migrating those is Stage 3.2+ work.
func (ctx *ExecContext) deleteMicroflowViaRepoOrBackend(id model.ID) error {
	if ctx.Microflows != nil {
		return ctx.Microflows.Delete(id)
	}
	return ctx.Backend.DeleteMicroflow(id)
}
