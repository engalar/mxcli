// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

func TestRecordingSnippetRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingSnippetRepository{}
	s := genPg.NewSnippet()
	if err := rec.Create("parent-uuid", "Documents", s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 {
		t.Fatalf("Created length = %d, want 1", len(rec.Created))
	}
	got := rec.Created[0]
	if got.ParentUUID != "parent-uuid" {
		t.Errorf("ParentUUID = %q, want parent-uuid", got.ParentUUID)
	}
	if got.Snippet != s {
		t.Errorf("Snippet != input pointer")
	}
}

func TestRecordingSnippetRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingSnippetRepository{
		CreateFunc: func(SnippetCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genPg.NewSnippet())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}

func TestRecordingSnippetRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingSnippetRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genPg.NewSnippet())
	_ = rec.Delete("del-1")
	_ = rec.Move("mv-1", "newp")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
	if len(rec.Updated) != 1 || len(rec.Deleted) != 1 || len(rec.Moved) != 1 {
		t.Errorf("counts: U=%d D=%d M=%d", len(rec.Updated), len(rec.Deleted), len(rec.Moved))
	}
}
