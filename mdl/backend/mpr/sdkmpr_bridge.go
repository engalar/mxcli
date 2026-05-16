// SPDX-License-Identifier: Apache-2.0

// Package-level wrappers for sdk/mpr functions used across *_modelsdk.go files.
// Consolidates the sdk/mpr dependency to this single file so that the individual
// *_modelsdk.go files do not need to import sdk/mpr directly.
package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// ── Serialize wrappers ────────────────────────────────────────────────────────

func sdkSerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	return sdkmpr.SerializeProjectSettings(ps)
}

func sdkSerializeDataTransformer(dt *model.DataTransformer) ([]byte, error) {
	return sdkmpr.SerializeDataTransformer(dt)
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
	return sdkmpr.SerializeImageCollection(ic)
}

// ── Scanner helper ────────────────────────────────────────────────────────────

// sdkOpenBSONScanner opens an sdk/mpr reader at path and returns it as a
// types.BSONScanner. The caller must call the returned close function when done.
// Used by tests to obtain a BSONScanner without importing sdk/mpr directly.
func sdkOpenBSONScanner(path string) (types.BSONScanner, func(), error) {
	r, err := sdkmpr.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	return r, func() { _ = r.Close() }, nil
}

// ── Patch wrappers ────────────────────────────────────────────────────────────

func sdkPatchNavigationProfile(rawBytes []byte, profileName string, spec types.NavigationProfileSpec) ([]byte, error) {
	return sdkmpr.PatchNavigationProfile(rawBytes, profileName, spec)
}

func sdkPatchReconcileMemberAccesses(rawBytes []byte, moduleName string) ([]byte, int, error) {
	return sdkmpr.PatchReconcileMemberAccesses(rawBytes, moduleName)
}
