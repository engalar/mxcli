// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

type enumerationBackend struct {
	reader *modelsdkmpr.Reader
}

func newEnumerationBackend(reader *modelsdkmpr.Reader) *enumerationBackend {
	return &enumerationBackend{reader: reader}
}

func (b *enumerationBackend) ListEnumerations() ([]*model.Enumeration, error) {
	units, err := mprread.ListUnitsWithContainer[*genEnum.Enumeration](b.reader)
	if err != nil {
		return nil, err
	}
	enums := enumUnitsToModel(units)
	return append(enums, builtinSystemEnumerations()...), nil
}

func (b *enumerationBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) {
	enums, err := b.ListEnumerations()
	if err != nil {
		return nil, err
	}
	for _, e := range enums {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, fmt.Errorf("enumeration not found: %s", id)
}
