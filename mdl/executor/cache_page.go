package executor

import genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"

type pageCache struct {
	pagesWithContainerGen    []ContainerWithGen[*genPg.Page]
	layoutsWithContainerGen  []ContainerWithGen[*genPg.Layout]
	snippetsWithContainerGen []ContainerWithGen[*genPg.Snippet]
}

func (c *pageCache) PagesWithContainer() []ContainerWithGen[*genPg.Page] {
	return c.pagesWithContainerGen
}

func (c *pageCache) SetPagesWithContainer(v []ContainerWithGen[*genPg.Page]) {
	c.pagesWithContainerGen = v
}

func (c *pageCache) LayoutsWithContainer() []ContainerWithGen[*genPg.Layout] {
	return c.layoutsWithContainerGen
}

func (c *pageCache) SetLayoutsWithContainer(v []ContainerWithGen[*genPg.Layout]) {
	c.layoutsWithContainerGen = v
}

func (c *pageCache) SnippetsWithContainer() []ContainerWithGen[*genPg.Snippet] {
	return c.snippetsWithContainerGen
}

func (c *pageCache) SetSnippetsWithContainer(v []ContainerWithGen[*genPg.Snippet]) {
	c.snippetsWithContainerGen = v
}
