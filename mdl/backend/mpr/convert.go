// SPDX-License-Identifier: Apache-2.0

// convert.go — conversion helpers for sdk/mpr → mdl/types.
// All mpr.* types referenced here are now type aliases to types.*, so these
// helpers are identity functions. They are kept for call-site compatibility.

package mprbackend

import "github.com/mendixlabs/mxcli/mdl/types"

func convertMPRVersion(v types.MPRVersion) types.MPRVersion { return v }

func convertRawCustomWidgetTypePtr(in *types.RawCustomWidgetType, err error) (*types.RawCustomWidgetType, error) {
	return in, err
}

func convertRawCustomWidgetTypeSlice(in []*types.RawCustomWidgetType, err error) ([]*types.RawCustomWidgetType, error) {
	return in, err
}
