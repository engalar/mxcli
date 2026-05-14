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

// concreteWriter returns the underlying *mmpr.Writer that the modelsdk
// repos require, or (nil, false) if msdkWriter is unset or non-concrete.
// Used by every Stage 3 repo accessor on this type.
func (b *MprBackend) concreteWriter() (*mmpr.Writer, bool) {
	if b.msdkWriter == nil {
		return nil, false
	}
	w, ok := b.msdkWriter.(*mmpr.Writer)
	return w, ok
}

// Microflows returns the modelsdk-native MicroflowRepository, or nil if
// the backend has no modelsdk writer (Connect failed or backend was
// constructed in a way that left msdkWriter unset).
//
// Returning a fresh repo per call is intentional — the repo is a thin
// adapter and the cost is negligible. Stage 3.x phases that exercise
// this method heavily can promote it to a memoized field if profiling
// shows allocation pressure.
func (b *MprBackend) Microflows() repos.MicroflowRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewMicroflowRepository(w)
}

// Nanoflows returns the modelsdk-native NanoflowRepository, or nil with
// the same conditions as Microflows().
func (b *MprBackend) Nanoflows() repos.NanoflowRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewNanoflowRepository(w)
}

// Security returns the modelsdk-native SecurityRepository, or nil with
// the same conditions as Microflows().
func (b *MprBackend) Security() repos.SecurityRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewSecurityRepository(w)
}

// JavaActions returns the modelsdk-native JavaActionRepository, or nil
// with the same conditions as Microflows() (Stage 3.3.2 A0).
func (b *MprBackend) JavaActions() repos.JavaActionRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewJavaActionRepository(w)
}

// JavaScriptActions returns the modelsdk-native JavaScriptActionRepository,
// or nil with the same conditions as Microflows().
func (b *MprBackend) JavaScriptActions() repos.JavaScriptActionRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewJavaScriptActionRepository(w)
}

// DomainModels returns the modelsdk-native DomainModelRepository, or nil
// with the same conditions as Microflows() (Stage 3.3.4 A0).
func (b *MprBackend) DomainModels() repos.DomainModelRepository {
	w, ok := b.concreteWriter()
	if !ok {
		return nil
	}
	return mprrepos.NewDomainModelRepository(w)
}
