// SPDX-License-Identifier: Apache-2.0

package backend

import (
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// JavaScriptBackend provides JavaScript action operations.
type JavaScriptBackend interface {
	ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)
	ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error)
	CreateJavaScriptActionGen(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error
	UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error
}
