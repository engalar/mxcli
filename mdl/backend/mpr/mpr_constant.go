// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
)

type constantBackend struct {
	reader *modelsdkmpr.Reader
}

func newConstantBackend(reader *modelsdkmpr.Reader) *constantBackend {
	return &constantBackend{reader: reader}
}

func (b *constantBackend) ListConstants() ([]*model.Constant, error) {
	units, err := mprread.ListUnitsWithContainer[*genConst.Constant](b.reader)
	if err != nil {
		return nil, err
	}
	return constUnitsToModel(units), nil
}

func (b *constantBackend) GetConstant(id model.ID) (*model.Constant, error) {
	consts, err := b.ListConstants()
	if err != nil {
		return nil, err
	}
	for _, c := range consts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("constant not found: %s", id)
}
