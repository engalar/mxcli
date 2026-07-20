package executor

import (
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// initLoadFns initialises all *domainCache fields on the executorCache.
// Always recreates load functions so callers see updated repos.
func (c *executorCache) initLoadFns(deps *HandlerDeps) {
	if deps == nil {
		return
	}
	c.microflowsWithContainerGen = newDomainCache(func() ([]MicroflowGenWithContainer, error) {
		return loadMicroflowsWithContainerGen(deps)
	})
	c.nanoflowsWithContainerGen = newDomainCache(func() ([]NanoflowGenWithContainer, error) {
		return loadNanoflowsWithContainerGen(deps)
	})
	c.pagesWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Page], error) {
		return loadPagesWithContainerGen(deps)
	})
	c.layoutsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Layout], error) {
		return loadLayoutsWithContainerGen(deps)
	})
	c.snippetsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Snippet], error) {
		return loadSnippetsWithContainerGen(deps)
	})
	c.workflowsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genWf.Workflow], error) {
		return loadWorkflowsWithContainerGen(deps)
	})
	c.javaActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJA.JavaAction], error) {
		return loadJavaActionsWithContainerGen(deps)
	})
	c.javaScriptActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
		return loadJavaScriptActionsWithContainerGen(deps)
	})
	c.domainModelsWithContainerGen = newDomainCache(func() ([]DomainModelGenWithContainer, error) {
		return loadDomainModelsWithContainerGen(deps)
	})
}

// newExecutorCache creates an executorCache without wiring load functions.
func newExecutorCache() *executorCache {
	return &executorCache{}
}
