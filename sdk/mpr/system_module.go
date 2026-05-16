// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/model"
)

// System module constants — deterministic IDs for the virtual System module.
const (
	SystemModuleID = "00000000-0000-0000-0000-000000000001"
)

// BuildSystemModule returns a virtual Module for the System module.
func BuildSystemModule() *model.Module {
	m := &model.Module{
		Name: "System",
	}
	m.ID = model.ID(SystemModuleID)
	return m
}
