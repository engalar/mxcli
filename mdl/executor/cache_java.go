package executor

import (
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

type javaCache struct {
	javaActionsWithContainerGen       *domainCache[ContainerWithGen[*genJA.JavaAction]]
	javaScriptActionsWithContainerGen *domainCache[ContainerWithGen[*genJSA.JavaScriptAction]]
}
