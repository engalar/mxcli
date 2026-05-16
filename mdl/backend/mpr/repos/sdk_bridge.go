// SPDX-License-Identifier: Apache-2.0

// Package-level wrappers for sdk/mpr functions used in repos/*.go files.
// Consolidates the sdk/mpr dependency to this single file.
package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

func sdkPatchNavigationProfile(rawBytes []byte, profileName string, spec types.NavigationProfileSpec) ([]byte, error) {
	return sdkmpr.PatchNavigationProfile(rawBytes, profileName, types.NavigationProfileSpec(spec))
}

// sdkOpenBSONScanner opens a modelsdk/mpr reader at path and returns it as a
// types.BSONScanner. The caller must call the returned close function when done.
func sdkOpenBSONScanner(path string) (types.BSONScanner, func(), error) {
	r, err := modelsdkmpr.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	return r, func() { _ = r.Close() }, nil
}
