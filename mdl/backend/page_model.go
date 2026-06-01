// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// PageModelBackend provides PageModel-level read and write access to page,
// snippet, and layout units.
type PageModelBackend interface {
	GetPageModel(id model.ID) (*types.PageModel, error)
	GetSnippetModel(id model.ID) (*types.PageModel, error)
	GetLayoutModel(id model.ID) (*types.PageModel, error)
	WritePageModel(id model.ID, m *types.PageModel) error
	WriteSnippetModel(id model.ID, m *types.PageModel) error
}
