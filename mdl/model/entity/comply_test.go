// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
)

// Compile-time interface compliance assertions.
// These have no runtime cost — they are checked at compile time by go test.
//
// GREEN baseline: EntityModel currently satisfies both interfaces.
// Turns RED if someone removes ToMDL() or Persist() from EntityModel, or if
// the Document/Persistable interface contract changes.
//
// When a new domain is added (e.g. mdl/model/association/), create
// mdl/model/association/comply_test.go with the equivalent two lines.
// No other file needs modification.
var _ model.Document = (*entity.EntityModel)(nil)
var _ model.Persistable = (*entity.EntityModel)(nil)
