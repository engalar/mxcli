package executor

import genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"

type workflowCache struct {
	workflowsWithContainerGen []ContainerWithGen[*genWf.Workflow]
}

func (c *workflowCache) WorkflowsWithContainer() []ContainerWithGen[*genWf.Workflow] {
	return c.workflowsWithContainerGen
}

func (c *workflowCache) SetWorkflowsWithContainer(v []ContainerWithGen[*genWf.Workflow]) {
	c.workflowsWithContainerGen = v
}
