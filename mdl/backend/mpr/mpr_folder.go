// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	"github.com/mendixlabs/mxcli/mdl/types"
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
