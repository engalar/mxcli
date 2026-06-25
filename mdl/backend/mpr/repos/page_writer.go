// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// pageWriter mirrors microflowWriter for the Pages domain.
type pageWriter struct {
	sink writeSink
	enc  *encoder
}

func newPageWriterWithSink(sink writeSink, enc *encoder) *pageWriter {
	return &pageWriter{sink: sink, enc: enc}
}

// NewPageWriter is the public direct-mode constructor.
func NewPageWriter(w *mmpr.Writer) repos.PageWriter {
	return newPageWriterWithSink(newWriterSink(w), newEncoder())
}

func (r *pageWriter) Create(parentUUID string, containmentName string, page *genPg.Page) error {
	if page.ID() == "" {
		page.SetID(element.ID(mmpr.GenerateID()))
	}
	if page.TypeName() == "" {
		page.SetTypeName("Forms$Page")
	}
	contents, err := r.enc.EncodePage(page)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(page.ID()), parentUUID, containmentName, page.TypeName(), contents)
}

func (r *pageWriter) Update(page *genPg.Page) error {
	contents, err := r.enc.EncodePage(page)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(page.ID()), contents)
}

func (r *pageWriter) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *pageWriter) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}
