package executor

type microflowCache struct {
	microflowsWithContainerGen *domainCache[MicroflowGenWithContainer]
	nanoflowsWithContainerGen  *domainCache[NanoflowGenWithContainer]
}
