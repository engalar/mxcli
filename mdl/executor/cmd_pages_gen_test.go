// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.A1 unit tests for listPagesGen / listLayoutsGen /
// listSnippetsGen. Mirrors cmd_workflows_gen_test.go.

package executor

import (
	"bytes"
	"strings"
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// --- pages ---

func newListPagesTestCtx(t *testing.T, pages []*genPg.Page) *ExecContext {
	t.Helper()
	repo := &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return pages, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return "MOD", nil
		},
	}
	var buf bytes.Buffer
	ctx := &ExecContext{
		Pages:  repo,
		Output: &buf,
		Format: FormatTable,
	}
	ctx.ensureCache()
	return ctx
}

func TestListPagesGen_OutputsHeaderAndSummary(t *testing.T) {
	pg := genPg.NewPage()
	pg.SetID("PID1")
	pg.SetName("Login")
	pg.SetUrl("/login")

	ctx := newListPagesTestCtx(t, []*genPg.Page{pg})
	if err := listPagesGen(ctx, ""); err != nil {
		t.Fatalf("listPagesGen: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "Qualified Name") {
		t.Errorf("missing Qualified Name header: %q", out)
	}
	if !strings.Contains(out, "Login") {
		t.Errorf("missing page name: %q", out)
	}
	if !strings.Contains(out, "/login") {
		t.Errorf("missing url value: %q", out)
	}
	if !strings.Contains(out, "1 pages") {
		t.Errorf("missing summary: %q", out)
	}
}

func TestListPagesGen_FilterByModule_ExcludesNonMatch(t *testing.T) {
	pg := genPg.NewPage()
	pg.SetID("PID1")
	pg.SetName("Login")

	ctx := newListPagesTestCtx(t, []*genPg.Page{pg})
	if err := listPagesGen(ctx, "OtherModule"); err != nil {
		t.Fatalf("listPagesGen with filter: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "0 pages") {
		t.Errorf("filter must exclude all entries; got %q", out)
	}
}

func TestListPagesGen_NilPageSkipped(t *testing.T) {
	ctx := newListPagesTestCtx(t, []*genPg.Page{nil, nil})
	if err := listPagesGen(ctx, ""); err != nil {
		t.Fatalf("listPagesGen: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "0 pages") {
		t.Errorf("nil entries must be skipped; got %q", out)
	}
}

// --- layouts ---

func TestListLayoutsGen_OutputsHeaderAndSummary(t *testing.T) {
	l := genPg.NewLayout()
	l.SetID("LID1")
	l.SetName("Atlas_Default")
	l.SetLayoutType("Responsive")

	repo := &repostesting.RecordingLayoutRepository{
		ListAllFunc: func() ([]*genPg.Layout, error) { return []*genPg.Layout{l}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return "MOD", nil
		},
	}
	var buf bytes.Buffer
	ctx := &ExecContext{Layouts: repo, Output: &buf, Format: FormatTable}
	ctx.ensureCache()

	if err := listLayoutsGen(ctx, ""); err != nil {
		t.Fatalf("listLayoutsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Atlas_Default") {
		t.Errorf("missing layout name: %q", out)
	}
	if !strings.Contains(out, "Responsive") {
		t.Errorf("missing layout type: %q", out)
	}
	if !strings.Contains(out, "1 layouts") {
		t.Errorf("missing summary: %q", out)
	}
}

// --- snippets ---

func TestListSnippetsGen_OutputsHeaderAndSummary(t *testing.T) {
	s := genPg.NewSnippet()
	s.SetID("SID1")
	s.SetName("HeaderSnippet")

	repo := &repostesting.RecordingSnippetRepository{
		ListAllFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{s}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return "MOD", nil
		},
	}
	var buf bytes.Buffer
	ctx := &ExecContext{Snippets: repo, Output: &buf, Format: FormatTable}
	ctx.ensureCache()

	if err := listSnippetsGen(ctx, ""); err != nil {
		t.Fatalf("listSnippetsGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "HeaderSnippet") {
		t.Errorf("missing snippet name: %q", out)
	}
	if !strings.Contains(out, "1 snippets") {
		t.Errorf("missing summary: %q", out)
	}
}
