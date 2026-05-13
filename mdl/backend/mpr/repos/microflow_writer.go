// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// microflowWriter implements repos.MicroflowWriter against a writeSink.
// It is used both standalone (UoW) and as the writer half of
// microflowRepo (Task 6 wires the reader half).
//
// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4); we never call ReaderCache.Invalidate manually.
type microflowWriter struct {
	sink writeSink
	enc  *encoder
}

// newMicroflowWriterWithSink constructs a writer routed through the
// given sink (direct or transactional). Used by Task 5 UoW and by
// Task 6 microflowRepo.
func newMicroflowWriterWithSink(sink writeSink, enc *encoder) *microflowWriter {
	return &microflowWriter{sink: sink, enc: enc}
}

// NewMicroflowWriter is the public direct-mode constructor (no UoW).
// Stage 3 may inline this once the executor is fully cut over.
func NewMicroflowWriter(w *mmpr.Writer) repos.MicroflowWriter {
	return newMicroflowWriterWithSink(newWriterSink(w), newEncoder())
}

func (r *microflowWriter) Create(parentUUID string, containmentName string, mf *genMf.Microflow) error {
	if mf.ID() == "" {
		mf.SetID(element.ID(mmpr.GenerateID()))
	}
	if mf.TypeName() == "" {
		mf.SetTypeName("Microflows$Microflow")
	}
	contents, err := r.enc.Encode(mf)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(mf.ID()), parentUUID, containmentName, mf.TypeName(), contents)
}

func (r *microflowWriter) Update(mf *genMf.Microflow) error {
	contents, err := r.enc.Encode(mf)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(mf.ID()), contents)
}

func (r *microflowWriter) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *microflowWriter) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}
