package executor

import genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"

type securityCache struct {
	projectSecurityGen             *genSec.ProjectSecurity
	moduleSecurityWithContainerGen []ModuleSecurityGenWithContainer
}

func (c *securityCache) ProjectSecurityGen() *genSec.ProjectSecurity { return c.projectSecurityGen }
func (c *securityCache) SetProjectSecurityGen(v *genSec.ProjectSecurity) {
	c.projectSecurityGen = v
}

func (c *securityCache) ModuleSecurityWithContainer() []ModuleSecurityGenWithContainer {
	return c.moduleSecurityWithContainerGen
}

func (c *securityCache) SetModuleSecurityWithContainer(v []ModuleSecurityGenWithContainer) {
	c.moduleSecurityWithContainerGen = v
}
