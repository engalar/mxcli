// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// javaBackend implements the read-only gen-typed Java/JavaScript action
// surface by wrapping modelsdk-native repos. Write methods (Create/Update/
// Delete) remain on MprBackend.
type javaBackend struct {
	writer *mmpr.Writer
}

func newJavaBackend(writer *mmpr.Writer) *javaBackend {
	return &javaBackend{writer: writer}
}

func (b *javaBackend) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
	return mprrepos.NewJavaActionRepository(b.writer).ListAll()
}

func (b *javaBackend) ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error) {
	return mprrepos.NewJavaActionRepository(b.writer).FindByQualifiedName(qualifiedName)
}

func (b *javaBackend) ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error) {
	return mprrepos.NewJavaScriptActionRepository(b.writer).ListAll()
}

func (b *javaBackend) ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error) {
	return mprrepos.NewJavaScriptActionRepository(b.writer).FindByQualifiedName(qualifiedName)
}
