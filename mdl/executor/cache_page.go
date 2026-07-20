package executor

import genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"

type pageCache struct {
	pagesWithContainerGen    *domainCache[ContainerWithGen[*genPg.Page]]
	layoutsWithContainerGen  *domainCache[ContainerWithGen[*genPg.Layout]]
	snippetsWithContainerGen *domainCache[ContainerWithGen[*genPg.Snippet]]
}
