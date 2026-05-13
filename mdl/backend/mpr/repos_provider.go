// SPDX-License-Identifier: Apache-2.0

// Stage 3 RepoProvider implementation: exposes the modelsdk-native
// repositories that the executor's ExecContext now carries alongside the
// legacy backend.FullBackend interface. Stage 3.1 wires only the
// MicroflowRepository (the single domain whose handlers can be cut
// over without resolving the sdk vs gen type mismatch — only the
// Delete-by-ID path migrates this stage).

package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// Microflows returns the modelsdk-native MicroflowRepository, or nil if
// the backend has no modelsdk writer (Connect failed or backend was
// constructed in a way that left msdkWriter unset).
//
// Returning a fresh repo per call is intentional — the repo is a thin
// adapter and the cost is negligible. Stage 3.x phases that exercise
// this method heavily can promote it to a memoized field if profiling
// shows allocation pressure.
func (b *MprBackend) Microflows() repos.MicroflowRepository {
	if b.msdkWriter == nil {
		return nil
	}
	w, ok := b.msdkWriter.(*mmpr.Writer)
	if !ok {
		// Defensive: in production msdkWriter is always *mmpr.Writer (set
		// by Connect/Wrap). A non-concrete value here means a test or a
		// future caller swapped in a different impl — return nil so the
		// executor falls back to ctx.Backend.
		return nil
	}
	return mprrepos.NewMicroflowRepository(w)
}
