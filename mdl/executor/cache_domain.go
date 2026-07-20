package executor

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

type domainModelCache struct {
	domainModelsWithContainerGen *domainCache[DomainModelGenWithContainer]
	domainModels                 []*genDm.DomainModel
	domainModelsGen              []*genDm.DomainModel
	domainModelByModule          map[model.ID]*genDm.DomainModel
}

func (c *domainModelCache) DomainModels() []*genDm.DomainModel { return c.domainModels }
func (c *domainModelCache) SetDomainModels(v []*genDm.DomainModel) {
	c.domainModels = v
}

func (c *domainModelCache) DomainModelsGen() []*genDm.DomainModel { return c.domainModelsGen }
func (c *domainModelCache) SetDomainModelsGen(v []*genDm.DomainModel) {
	c.domainModelsGen = v
}

func (c *domainModelCache) DomainModelByModule() map[model.ID]*genDm.DomainModel {
	return c.domainModelByModule
}

func (c *domainModelCache) SetDomainModelByModule(v map[model.ID]*genDm.DomainModel) {
	c.domainModelByModule = v
}
