// SPDX-License-Identifier: Apache-2.0

// Package-level wrappers for sdk/mpr functions used in repos/*.go files.
// Consolidates the sdk/mpr dependency to this single file.
package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

func sdkPatchNavigationProfile(rawBytes []byte, profileName string, spec types.NavigationProfileSpec) ([]byte, error) {
	return sdkmpr.PatchNavigationProfile(rawBytes, profileName, types.NavigationProfileSpec(spec))
}

// sdkOpenBSONScanner opens an sdk/mpr reader at path and returns it as a
// types.BSONScanner. The caller must call the returned close function when done.
// Used by tests that need a BSONScanner without importing sdk/mpr directly.
func sdkOpenBSONScanner(path string) (types.BSONScanner, func(), error) {
	r, err := sdkmpr.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	return r, func() { _ = r.Close() }, nil
}
