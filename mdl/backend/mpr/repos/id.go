// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type idGen struct{}

func NewIDGenerator() repos.IDGenerator { return idGen{} }

func (idGen) NewID() model.ID { return model.ID(mmpr.GenerateID()) }
