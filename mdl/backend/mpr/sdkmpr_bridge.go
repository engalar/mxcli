// SPDX-License-Identifier: Apache-2.0

// Package-level wrappers for sdk/mpr functions used across *_modelsdk.go files.
// Consolidates the sdk/mpr dependency to this single file so that the individual
// *_modelsdk.go files do not need to import sdk/mpr directly.
package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// ── Serialize wrappers ────────────────────────────────────────────────────────

func sdkSerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	return modelsdkmpr.SerializeProjectSettings(ps)
}

func sdkSerializeDataTransformer(dt *model.DataTransformer) ([]byte, error) {
	return modelsdkmpr.SerializeDataTransformer(dt)
}

func sdkSerializeImportMapping(im *model.ImportMapping) ([]byte, error) {
	return sdkmpr.SerializeImportMapping(im)
}

func sdkSerializeExportMapping(em *model.ExportMapping) ([]byte, error) {
	return sdkmpr.SerializeExportMapping(em)
}

func sdkSerializeConsumedODataService(svc *model.ConsumedODataService) ([]byte, error) {
	return sdkmpr.SerializeConsumedODataService(svc)
}

func sdkSerializePublishedODataService(svc *model.PublishedODataService) ([]byte, error) {
	return sdkmpr.SerializePublishedODataService(svc)
}

func sdkSerializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	return sdkmpr.SerializeConsumedRestService(svc)
}

func sdkSerializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	return sdkmpr.SerializePublishedRestService(svc)
}

func sdkSerializeImageCollection(ic *types.ImageCollection) ([]byte, error) {
	return modelsdkmpr.SerializeImageCollection(ic)
}

// ── Scanner helper ────────────────────────────────────────────────────────────

// sdkOpenBSONScanner opens a modelsdk/mpr reader at path and returns it as a
// types.BSONScanner. The caller must call the returned close function when done.
func sdkOpenBSONScanner(path string) (types.BSONScanner, func(), error) {
	r, err := modelsdkmpr.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	return r, func() { _ = r.Close() }, nil
}

// ── Patch wrappers ────────────────────────────────────────────────────────────

func sdkPatchNavigationProfile(rawBytes []byte, profileName string, spec types.NavigationProfileSpec) ([]byte, error) {
	return modelsdkmpr.PatchNavigationProfile(rawBytes, profileName, spec)
}

func sdkPatchReconcileMemberAccesses(rawBytes []byte, moduleName string) ([]byte, int, error) {
	return modelsdkmpr.PatchReconcileMemberAccesses(rawBytes, moduleName)
}

// ── Reader bridge ─────────────────────────────────────────────────────────────

// sdkReader is a type alias for sdk/mpr.Reader, making it accessible in files
// that import sdkmpr_bridge but not sdk/mpr directly.
type sdkReader = sdkmpr.Reader

// sdkOpenReader opens a project MPR file for read-write access and returns an
// sdk/mpr Reader. Used by Connect() in backend.go.
func sdkOpenReader(path string) (*sdkReader, error) {
	return sdkmpr.OpenWithOptions(path, sdkmpr.OpenOptions{ReadOnly: false})
}
