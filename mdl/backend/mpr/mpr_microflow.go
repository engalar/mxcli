// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// microflowBackend implements the read-only gen-typed Microflow/Nanoflow
// surface by wrapping the modelsdk-native repos. Write methods (DeleteMicroflow,
// DeleteNanoflow) remain on MprBackend.
type microflowBackend struct {
	writer *mmpr.Writer
}

func newMicroflowBackend(writer *mmpr.Writer) *microflowBackend {
	return &microflowBackend{writer: writer}
}

func (b *microflowBackend) ListMicroflowsGen() ([]*genMf.Microflow, error) {
	return mprrepos.NewMicroflowRepository(b.writer).ListAll()
}

func (b *microflowBackend) ListNanoflowsGen() ([]*genMf.Nanoflow, error) {
	return mprrepos.NewNanoflowRepository(b.writer).List("")
}

func (b *microflowBackend) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	return mprrepos.NewMicroflowRepository(b.writer).Get(id)
}

func (b *microflowBackend) IsRule(qualifiedName string) (bool, error) {
	return mprrepos.NewMicroflowRepository(b.writer).IsRule(qualifiedName)
}
