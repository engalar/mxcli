// SPDX-License-Identifier: Apache-2.0

// Package repos defines per-domain Repository / Service / Auxiliary
// interfaces used by the modelsdk-native executor.
//
// Layering (spec section 4):
//
//	executor → repos (this package, interfaces only)
//	             ↑
//	mdl/backend/mpr/repos (MPR implementations, separate package)
//
// This package MUST NOT import any *implementation* package. It depends
// only on:
//   - github.com/mendixlabs/mxcli/model        (model.ID)
//   - github.com/mendixlabs/mxcli/modelsdk/gen/*  (gen types — interface signatures)
//   - github.com/mendixlabs/mxcli/modelsdk/element (element.Element)
//
// Stage 2 (this plan) defines all per-domain interfaces. Microflows + Pages
// receive full implementations in mdl/backend/mpr/repos. The remainder are
// signature-only stubs marked `// TODO Stage 3 cutover` so Stage 3 handlers
// can compile.
package repos
