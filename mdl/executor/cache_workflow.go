package executor

import genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"

type workflowCache struct {
	workflowsWithContainerGen *domainCache[ContainerWithGen[*genWf.Workflow]]
}
