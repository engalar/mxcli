package executor

type microflowCache struct {
	microflowsWithContainerGen []MicroflowGenWithContainer
	nanoflowsWithContainerGen  []NanoflowGenWithContainer
}

func (c *microflowCache) MicroflowsWithContainer() []MicroflowGenWithContainer {
	return c.microflowsWithContainerGen
}

func (c *microflowCache) SetMicroflowsWithContainer(v []MicroflowGenWithContainer) {
	c.microflowsWithContainerGen = v
}

func (c *microflowCache) NanoflowsWithContainer() []NanoflowGenWithContainer {
	return c.nanoflowsWithContainerGen
}

func (c *microflowCache) SetNanoflowsWithContainer(v []NanoflowGenWithContainer) {
	c.nanoflowsWithContainerGen = v
}
