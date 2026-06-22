package executor

import (
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

type javaCache struct {
	javaActionsWithContainerGen       []ContainerWithGen[*genJA.JavaAction]
	javaScriptActionsWithContainerGen []ContainerWithGen[*genJSA.JavaScriptAction]
}

func (c *javaCache) JavaActionsWithContainer() []ContainerWithGen[*genJA.JavaAction] {
	return c.javaActionsWithContainerGen
}

func (c *javaCache) SetJavaActionsWithContainer(v []ContainerWithGen[*genJA.JavaAction]) {
	c.javaActionsWithContainerGen = v
}

func (c *javaCache) JavaScriptActionsWithContainer() []ContainerWithGen[*genJSA.JavaScriptAction] {
	return c.javaScriptActionsWithContainerGen
}

func (c *javaCache) SetJavaScriptActionsWithContainer(v []ContainerWithGen[*genJSA.JavaScriptAction]) {
	c.javaScriptActionsWithContainerGen = v
}
