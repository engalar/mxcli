// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type navigationBackend struct {
	reader *modelsdkmpr.Reader
}

func newNavigationBackend(reader *modelsdkmpr.Reader) *navigationBackend {
	return &navigationBackend{reader: reader}
}

func (b *navigationBackend) listNavigationUnits() ([]*types.NavigationDocument, error) {
	rawUnits, err := b.reader.ListRawUnitsByType(navigationDocumentBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*types.NavigationDocument, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		nav, err := parseNavigationDocumentRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse navigation %s: %w", ru.ID, err)
		}
		out = append(out, nav)
	}
	return out, nil
}

func (b *navigationBackend) ListNavigationDocuments() ([]*types.NavigationDocument, error) {
	return b.listNavigationUnits()
}

func (b *navigationBackend) GetNavigation() (*types.NavigationDocument, error) {
	docs, err := b.listNavigationUnits()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no navigation document found")
	}
	return docs[0], nil
}
