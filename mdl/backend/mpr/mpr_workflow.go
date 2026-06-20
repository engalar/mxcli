// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// workflowBackend implements the read-only gen-typed Workflow surface by
// wrapping the modelsdk-native WorkflowRepository. Write methods remain on
// MprBackend.
type workflowBackend struct {
	writer *mmpr.Writer
}

func newWorkflowBackend(writer *mmpr.Writer) *workflowBackend {
	return &workflowBackend{writer: writer}
}

func (b *workflowBackend) ListWorkflowsGen() ([]*genWf.Workflow, error) {
	return mprrepos.NewWorkflowRepository(b.writer).ListAll()
}

func (b *workflowBackend) GetWorkflowGen(id model.ID) (*genWf.Workflow, error) {
	return mprrepos.NewWorkflowRepository(b.writer).Get(id)
}
