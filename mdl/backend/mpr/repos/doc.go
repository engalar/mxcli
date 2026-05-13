// SPDX-License-Identifier: Apache-2.0

// Package mprrepos provides MPR-backed implementations of the
// mdl/repos interfaces. Every constructor accepts the shared
// *modelsdk/mpr.Writer (and derived dependencies) so a single Writer
// drives the entire repo set.
//
// Package name is "mprrepos" (not "repos") to avoid collision with the
// imported github.com/mendixlabs/mxcli/mdl/repos package.
package mprrepos
