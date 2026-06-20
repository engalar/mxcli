// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

type folderBackend struct {
	reader *modelsdkmpr.Reader
}

func newFolderBackend(reader *modelsdkmpr.Reader) *folderBackend {
	return &folderBackend{reader: reader}
}

func (b *folderBackend) ListFolders() ([]*types.FolderInfo, error) {
	units, err := mprread.ListFolders(b.reader)
	if err != nil {
		return nil, err
	}
	return folderUnitsToTypes(units), nil
}
