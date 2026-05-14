// SPDX-License-Identifier: Apache-2.0

// Package mprread provides read-only, gen-typed list helpers on top of
// *modelsdk/mpr.Reader without forcing callers to open a Writer.
//
// The package exists because modelsdk/codec already imports
// modelsdk/mpr (encoder uses mpr.IDToBsonBinary, store/resolver use
// mpr.Reader). Adding methods on *mpr.Reader that decode into
// modelsdk/gen/* types would import codec back into mpr and create a
// cycle. mprread sits one layer up and is free to depend on both.
//
// Use this package for read-only consumers — project tree printers,
// example programs, lint walkers — that previously reached into
// sdk/mpr to enumerate microflows/nanoflows. Write paths still go
// through mdl/backend/mpr/repos.
//
// As more read-only gen-typed listers land (pages, entities,
// workflows, …) they belong here too.
package mprread
