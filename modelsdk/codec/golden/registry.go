// SPDX-License-Identifier: Apache-2.0

package golden

import (
	_ "embed"
)

//go:embed testdata/MyFirstModule.Nanoflow.mxunit
var nanoflowBSON []byte

// Registry returns all registered golden entries.
func Registry() []GoldenEntry {
	return []GoldenEntry{
		{
			Name:    "Nanoflow",
			Source:  "Studio Pro 11.12.1 — MyFirstModule.Nanoflow",
			BSON:    nanoflowBSON,
			Builder: BuildNanoflow,
			SetupMDL: `create module MyFirstModule;
create module Administration;
create entity Administration.Account (FullName: String(200));
create entity Administration.AccountPasswordData (Password: String(200));
create association Administration.AccountPasswordData_Account
  from Administration.AccountPasswordData to Administration.Account;`,
			SkipFields: nil, // $ID is auto-skipped
		},
	}
}
