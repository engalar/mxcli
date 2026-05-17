// SPDX-License-Identifier: Apache-2.0

// convert.go — conversion helpers for sdk/mpr → mdl/types.
// All mpr.* types referenced here are now type aliases to types.*, so these
// helpers are identity functions. They are kept for call-site compatibility.

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	msdkversion "github.com/mendixlabs/mxcli/modelsdk/mpr/version"
)

// buildSystemModuleForBackend returns the virtual System module appended
// to every ListModules result. Mirrors sdk/mpr.BuildSystemModule but uses
// the modelsdk-native meta.SystemModuleID constant.
func buildSystemModuleForBackend() *model.Module {
	m := &model.Module{Name: "System"}
	m.ID = model.ID(meta.SystemModuleID)
	return m
}

func convertMPRVersion(v types.MPRVersion) types.MPRVersion { return v }

// convertProjectVersionFromMsdk maps modelsdk/mpr/version.ProjectVersion to
// mdl/types.ProjectVersion. Both structs hold identical fields; the two types
// exist only because modelsdk/mpr cannot depend on mdl/types.
func convertProjectVersionFromMsdk(in *msdkversion.ProjectVersion) *types.ProjectVersion {
	if in == nil {
		return nil
	}
	return &types.ProjectVersion{
		ProductVersion: in.ProductVersion,
		BuildVersion:   in.BuildVersion,
		FormatVersion:  in.FormatVersion,
		SchemaHash:     in.SchemaHash,
		MajorVersion:   in.MajorVersion,
		MinorVersion:   in.MinorVersion,
		PatchVersion:   in.PatchVersion,
	}
}

func convertRawCustomWidgetTypePtr(in *types.RawCustomWidgetType, err error) (*types.RawCustomWidgetType, error) {
	return in, err
}

func convertRawCustomWidgetTypeSlice(in []*types.RawCustomWidgetType, err error) ([]*types.RawCustomWidgetType, error) {
	return in, err
}

// msdkUnitInfoSliceToTypes converts modelsdk/mpr.UnitInfo slice to
// mdl/types.UnitInfo. Identical field layout; only model.ID vs string differs.
func msdkUnitInfoSliceToTypes(in []*modelsdkmpr.UnitInfo) []*types.UnitInfo {
	if in == nil {
		return nil
	}
	out := make([]*types.UnitInfo, 0, len(in))
	for _, u := range in {
		if u == nil {
			continue
		}
		out = append(out, &types.UnitInfo{
			ID:              model.ID(u.ID),
			ContainerID:     model.ID(u.ContainerID),
			ContainmentName: u.ContainmentName,
			Type:            u.Type,
		})
	}
	return out
}
