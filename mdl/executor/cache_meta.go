package executor

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

type metadataCache struct {
	modules   []*model.Module
	units     []*types.UnitInfo
	folders   []*types.FolderInfo
	hierarchy *ContainerHierarchy

	entityNames    map[model.ID]string
	microflowNames map[model.ID]string
	pageNames      map[model.ID]string
}

func (c *metadataCache) EntityNames() map[model.ID]string        { return c.entityNames }
func (c *metadataCache) SetEntityNames(v map[model.ID]string)    { c.entityNames = v }
func (c *metadataCache) MicroflowNames() map[model.ID]string     { return c.microflowNames }
func (c *metadataCache) SetMicroflowNames(v map[model.ID]string) { c.microflowNames = v }
func (c *metadataCache) PageNames() map[model.ID]string          { return c.pageNames }
func (c *metadataCache) SetPageNames(v map[model.ID]string)      { c.pageNames = v }
