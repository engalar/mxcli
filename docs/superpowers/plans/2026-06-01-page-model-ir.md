# Page Model IR Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the dual-path page architecture (separate rawWidget read path + direct BSON write path) with a shared `types.PageModel` IR, fixing empty `{}` DESCRIBE PAGE output and guaranteeing roundtrip stability.

**Architecture:** `types.PageModel`/`WidgetNode` lives in `mdl/types/page.go` as the shared contract. The backend layer (`mdl/backend/mpr/page_model.go`) converts BSON↔PageModel. The executor layer calls `ctx.Backend.GetPageModel()` for describe and `ctx.Backend.WritePageModel()` for create. Neither the executor nor the backend crosses into the other's domain.

**Tech Stack:** Go, `go.mongodb.org/mongo-driver/v2/bson`, existing `dGet*` helpers in `mdl/backend/mpr/page_mutator.go`, `visitor.Build()` for AST parsing in tests.

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `mdl/types/page.go` | PageModel, WidgetNode, all sub-types |
| Create | `mdl/backend/page_model.go` | PageModelBackend interface |
| Modify | `mdl/backend/backend.go` | Add PageModelBackend to FullBackend |
| Create | `mdl/backend/mock/mock_page_model.go` | Func-field stubs |
| Modify | `mdl/backend/mock/backend.go` | Func fields for new methods |
| Create | `mdl/backend/mpr/page_model.go` | fromBSON + toBSON (BSON↔PageModel) |
| Create | `mdl/executor/cmd_pages_model_to_mdl.go` | PageModel→MDL renderer |
| Create | `mdl/executor/roundtrip_page_model_test.go` | Roundtrip tests per widget kind |
| Modify | `mdl/executor/cmd_pages_describe.go` | Switch to GetPageModel() |
| Modify | `internal/goldenfs/helpdesk_regression_test.go` | Add page-body-nonempty assertion |
| Create | `mdl/executor/cmd_pages_ast_to_model.go` | AST→PageModel (Phase 4) |
| Modify | `mdl/executor/cmd_pages_create_v3.go` | Switch to WritePageModel() (Phase 4) |
| Modify | `mdl/backend/mpr/page_model.go` | Add toBSON (Phase 4) |
| Delete | `mdl/executor/cmd_pages_describe_parse.go` | Replaced by fromBSON |
| Delete | `mdl/executor/cmd_pages_describe_output.go` | Replaced by pageModelToMDL |
| Delete | `mdl/executor/cmd_pages_describe_pluggable.go` | Absorbed into fromBSON |
| Delete | `mdl/executor/cmd_pages_builder.go` | Replaced by pageASTToModel |
| Delete | `mdl/executor/cmd_pages_builder_input.go` | Logic moved to toBSON |
| Delete | `mdl/executor/cmd_pages_builder_input_filters.go` | Logic moved to toBSON |

---

## Task 1: Define PageModel Types

**Files:**
- Create: `mdl/types/page.go`

- [ ] **Step 1: Create the file**

```go
// SPDX-License-Identifier: Apache-2.0

package types

// PageModel is the MDL-level intermediate representation of a page, snippet,
// or layout. Both the write path (AST→PageModel→BSON) and the read path
// (BSON→PageModel→MDL) share this type, keeping the two paths in sync.
type PageModel struct {
	ModuleName string
	Name       string
	Title      string
	Layout     string        // e.g. "Atlas_Core.Atlas_Default"
	Folder     string        // e.g. "Ticket/Search"
	Params     []PageParam
	Variables  []PageVariable
	Widgets    []*WidgetNode
}

// PageParam represents a page parameter ($Foo: Module.Entity).
type PageParam struct {
	Name       string
	EntityName string
}

// PageVariable represents a page-level variable (non-persistent object).
type PageVariable struct {
	Name       string
	EntityName string
	IsList     bool
}

// WidgetKind identifies the MDL-level widget keyword.
type WidgetKind string

const (
	WidgetContainer    WidgetKind = "container"
	WidgetScrollView   WidgetKind = "scrollview"
	WidgetGroupBox     WidgetKind = "groupbox"
	WidgetLayoutGrid   WidgetKind = "layoutgrid"
	WidgetLayoutRow    WidgetKind = "row"
	WidgetLayoutCol    WidgetKind = "column"
	WidgetTabContainer WidgetKind = "tabcontainer"
	WidgetTabPage      WidgetKind = "tabpage"
	WidgetDataView     WidgetKind = "dataview"
	WidgetListView     WidgetKind = "listview"
	WidgetGallery      WidgetKind = "gallery"
	WidgetButton       WidgetKind = "button"
	WidgetTextBox      WidgetKind = "textbox"
	WidgetTextArea     WidgetKind = "textarea"
	WidgetDatePicker   WidgetKind = "datepicker"
	WidgetRadioButtons WidgetKind = "radiobuttons"
	WidgetCheckBox     WidgetKind = "checkbox"
	WidgetLabel        WidgetKind = "label"
	WidgetText         WidgetKind = "text"
	WidgetDynamicText  WidgetKind = "dynamictext"
	WidgetTitle        WidgetKind = "title"
	WidgetNavList      WidgetKind = "navigationlist"
	WidgetSnippet      WidgetKind = "snippet"
	WidgetDataGrid     WidgetKind = "datagrid"   // CustomWidget type=datagrid2
	WidgetComboBox     WidgetKind = "combobox"   // CustomWidget type=combobox
	WidgetImage        WidgetKind = "image"      // CustomWidget type=image
	WidgetUnknown      WidgetKind = "unknown"    // unrecognised pluggable widget
)

// WidgetNode is a single node in the page widget tree.
type WidgetNode struct {
	Kind     WidgetKind
	Name     string
	Children []*WidgetNode

	// Data binding
	DataSource *DataSourceDef
	EntityAttr string // attribute path (textbox, datepicker, …)
	EntityCtx  string // entity type provided to children by a dataview

	// Display
	Caption string // button/label caption
	Content string // static text content

	// Layout (column)
	ColWidth ColWidthDef

	// Actions
	OnClick     string // qualified microflow/nanoflow/page name
	ButtonStyle string // Primary | Success | Warning | Danger | Default | Link | Icon

	// Input widget properties
	Editable   string // Always | Never | Conditional
	EditableIf string // expression when Editable==Conditional
	ShowLabel  bool
	LabelPos   string // Left | Top
	ReadOnly   string // Inherit | Control | Text

	// Conditional visibility
	VisibleIf string

	// Appearance
	Class       string
	Style       string
	DesignProps []DesignProp

	// Kind-specific sub-structs (nil when not applicable)
	GroupBox *GroupBoxProps
	DataGrid *DataGridProps
	Gallery  *GalleryProps
	Image    *ImageProps
	Snippet  *SnippetProps
	Unknown  *UnknownProps
}

// ColWidthDef holds responsive column widths (1-12; 0 = not set).
type ColWidthDef struct {
	Desktop, Tablet, Phone int
}

// DataSourceDef describes a widget's data source.
type DataSourceDef struct {
	Kind            DataSourceKind
	Reference       string // qualified name for mf/nf/param
	Entity          string // entity for database sources
	XPathConstraint string
	SortColumns     []SortDef
}

// DataSourceKind enumerates supported data source types.
type DataSourceKind string

const (
	DataSourceDatabase  DataSourceKind = "database"
	DataSourceMicroflow DataSourceKind = "microflow"
	DataSourceNanoflow  DataSourceKind = "nanoflow"
	DataSourceParameter DataSourceKind = "parameter"
	DataSourceSelection DataSourceKind = "selection"
)

// SortDef represents a single sort column.
type SortDef struct {
	Attribute string
	Order     string // ASC | DESC
}

// DesignProp represents a Mendix design property.
type DesignProp struct {
	Key, Option string
	ValueType   string // toggle | option
}

// GroupBoxProps holds groupbox-specific properties.
type GroupBoxProps struct {
	Collapsible string // No | YesInitiallyExpanded | YesInitiallyCollapsed
	HeaderMode  string // Div | H1 … H6
}

// DataGridProps holds datagrid-specific properties.
type DataGridProps struct {
	Columns       []ColumnDef
	FilterWidgets []*WidgetNode
	ControlBar    []*WidgetNode
	PageSize      int
	Pagination    string // buttons | virtualScrolling | loadMore
	PagingPos     string // bottom | top | both
}

// ColumnDef describes a single DataGrid column.
type ColumnDef struct {
	Name, Attribute, Caption string
	ShowContentAs            string // attribute | customContent | dynamicText
	ContentWidgets           []*WidgetNode
	DynamicText              string
	Alignment                string // left | center | right
	WrapText, Sortable, Resizable, Draggable bool
	Hidable     string // yes | hidden | no
	ColumnWidth string // autoFill | autoFit | manual
	Size, Visible, CellClass, Tooltip string
}

// GalleryProps holds gallery-specific properties.
type GalleryProps struct {
	DesktopColumns, TabletColumns, PhoneColumns int
	Selection                                   string // Single | Multi | None
	FilterWidgets                               []*WidgetNode
	ContentWidgets                              []*WidgetNode
}

// ImageProps holds pluggable-image-specific properties.
type ImageProps struct {
	URL, AltText    string
	Width, Height   string
	WidthUnit       string // auto | pixels | percentage
	HeightUnit      string // auto | pixels | percentage | viewport
	DisplayAs       string // fullImage | thumbnail
	Responsive      bool
	ImageType       string // image | imageUrl | icon
	OnClickType     string // action | enlarge
}

// SnippetProps holds snippet-call-specific properties.
type SnippetProps struct {
	SnippetName string // qualified name
}

// UnknownProps holds data for unrecognised pluggable widgets.
type UnknownProps struct {
	WidgetID      string // e.g. com.mendix.widget.custom.switch.Switch
	ExplicitProps []ExplicitProp
}

// ExplicitProp is a single non-default property from an unknown widget.
type ExplicitProp struct {
	Key, Value string
	IsRef      bool
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./mdl/types/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add mdl/types/page.go
git commit -m "feat(types): add PageModel/WidgetNode IR types for page roundtrip"
```

---

## Task 2: Backend Interface + FullBackend + Mock Stubs

**Files:**
- Create: `mdl/backend/page_model.go`
- Modify: `mdl/backend/backend.go`
- Modify: `mdl/backend/mock/backend.go`
- Create: `mdl/backend/mock/mock_page_model.go`

- [ ] **Step 1: Create the interface file**

`mdl/backend/page_model.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// PageModelBackend provides PageModel-level read and write access to page,
// snippet, and layout units. All BSON conversion stays in the implementation
// (mdl/backend/mpr/page_model.go); callers see only types.PageModel.
type PageModelBackend interface {
	// GetPageModel decodes a page unit into a PageModel.
	GetPageModel(id model.ID) (*types.PageModel, error)
	// GetSnippetModel decodes a snippet unit into a PageModel.
	GetSnippetModel(id model.ID) (*types.PageModel, error)
	// GetLayoutModel decodes a layout unit into a PageModel.
	GetLayoutModel(id model.ID) (*types.PageModel, error)
	// WritePageModel encodes and stores the PageModel widget tree into an
	// existing page unit (metadata such as allowed roles is preserved).
	WritePageModel(id model.ID, m *types.PageModel) error
	// WriteSnippetModel is like WritePageModel but for snippet units.
	WriteSnippetModel(id model.ID, m *types.PageModel) error
}
```

- [ ] **Step 2: Add PageModelBackend to FullBackend**

In `mdl/backend/backend.go`, add `PageModelBackend` to the interface list (after `PageBackend`):

```go
type FullBackend interface {
	ConnectionBackend
	ModuleBackend
	ModuleSettingsBackend
	FolderBackend
	DomainModelBackend
	MicroflowBackend
	PageBackend
	PageModelBackend  // ← new
	EnumerationBackend
	// ... rest unchanged
```

- [ ] **Step 3: Add Func fields to MockBackend struct**

In `mdl/backend/mock/backend.go`, find the `// Stage 3.3.5.C1` PageBackend block and add after it:

```go
	// PageModelBackend
	GetPageModelFunc    func(id model.ID) (*types.PageModel, error)
	GetSnippetModelFunc func(id model.ID) (*types.PageModel, error)
	GetLayoutModelFunc  func(id model.ID) (*types.PageModel, error)
	WritePageModelFunc  func(id model.ID, m *types.PageModel) error
	WriteSnippetModelFunc func(id model.ID, m *types.PageModel) error
```

- [ ] **Step 4: Create mock implementations**

`mdl/backend/mock/mock_page_model.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

func (m *MockBackend) GetPageModel(id model.ID) (*types.PageModel, error) {
	if m.GetPageModelFunc != nil {
		return m.GetPageModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetPageModel not configured")
}

func (m *MockBackend) GetSnippetModel(id model.ID) (*types.PageModel, error) {
	if m.GetSnippetModelFunc != nil {
		return m.GetSnippetModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetSnippetModel not configured")
}

func (m *MockBackend) GetLayoutModel(id model.ID) (*types.PageModel, error) {
	if m.GetLayoutModelFunc != nil {
		return m.GetLayoutModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetLayoutModel not configured")
}

func (m *MockBackend) WritePageModel(id model.ID, pm *types.PageModel) error {
	if m.WritePageModelFunc != nil {
		return m.WritePageModelFunc(id, pm)
	}
	return fmt.Errorf("MockBackend.WritePageModel not configured")
}

func (m *MockBackend) WriteSnippetModel(id model.ID, pm *types.PageModel) error {
	if m.WriteSnippetModelFunc != nil {
		return m.WriteSnippetModelFunc(id, pm)
	}
	return fmt.Errorf("MockBackend.WriteSnippetModel not configured")
}
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./mdl/backend/... ./mdl/backend/mock/...
```

Expected: compile error "MprBackend does not implement PageModelBackend" — this is correct at this stage; Task 4 fixes it.

- [ ] **Step 6: Commit**

```bash
git add mdl/backend/page_model.go mdl/backend/backend.go \
        mdl/backend/mock/backend.go mdl/backend/mock/mock_page_model.go
git commit -m "feat(backend): add PageModelBackend interface and mock stubs"
```

---

## Task 3: Failing Roundtrip Tests (TDD Red)

**Files:**
- Create: `mdl/executor/roundtrip_page_model_test.go`
- Modify: `internal/goldenfs/helpdesk_regression_test.go`

- [ ] **Step 1: Create roundtrip test file**

`mdl/executor/roundtrip_page_model_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// roundtripPage is the core helper: create → describe → verify → re-execute → re-describe → stable.
func roundtripPage(t *testing.T, createMDL, pageName string, verify func(t *testing.T, described string)) {
	t.Helper()
	env := setupTestEnv(t)
	defer env.teardown()

	env.registerCleanup("page", pageName)

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("create page: %v", err)
	}

	described, err := env.describeMDL("describe page " + pageName + ";")
	if err != nil {
		t.Fatalf("describe page: %v", err)
	}
	if described == "" {
		t.Fatal("describe returned empty output")
	}

	// Semantic verification
	verify(t, described)

	// Stability: re-execute described MDL then re-describe — must be identical
	if err := env.executeMDL(described); err != nil {
		t.Fatalf("re-execute described MDL: %v", err)
	}
	redescribed, err := env.describeMDL("describe page " + pageName + ";")
	if err != nil {
		t.Fatalf("re-describe: %v", err)
	}
	if described != redescribed {
		t.Errorf("roundtrip not stable:\nfirst describe:\n%s\n\nsecond describe:\n%s", described, redescribed)
	}
}

func TestRoundtrip_PageModel_Container(t *testing.T) {
	entity := testModule + ".PMContainerEntity"
	page := testModule + ".PMContainerPage"
	roundtripPage(t, `
create or modify persistent entity `+entity+` (Title: String(200));
create or modify page `+page+` (
  title: 'Container Test',
  layout: Atlas_Core.Atlas_Default
) {
  container mainBox (class: 'spacing-outer') {
    button btn (caption: 'Click Me', action: call microflow `+testModule+`.ACT_Noop)
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "container") {
			t.Errorf("expected 'container' in describe output, got:\n%s", described)
		}
		if !strings.Contains(described, "button") {
			t.Errorf("expected 'button' in describe output, got:\n%s", described)
		}
	})
}

func TestRoundtrip_PageModel_DataGrid(t *testing.T) {
	entity := testModule + ".PMDataGridEntity"
	page := testModule + ".PMDataGridPage"
	roundtripPage(t, `
create or modify persistent entity `+entity+` (Name: String(200), Score: Integer);
create or modify page `+page+` (
  title: 'DataGrid Test',
  layout: Atlas_Core.Atlas_Default
) {
  layoutgrid grid {
    row r1 {
      column col1 (DesktopWidth: 12) {
        datagrid dg (DataSource: database `+entity+`) {
          column colName (Attribute: Name, Caption: 'Name')
          column colScore (Attribute: Score, Caption: 'Score')
        }
      }
    }
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "datagrid") {
			t.Errorf("expected 'datagrid' in describe output, got:\n%s", described)
		}
		if !strings.Contains(described, "colName") || !strings.Contains(described, "colScore") {
			t.Errorf("expected column names in describe output, got:\n%s", described)
		}
		// Verify parseable
		_, errs := visitor.Build(described)
		if len(errs) > 0 {
			t.Errorf("described MDL not parseable: %v", errs)
		}
	})
}

func TestRoundtrip_PageModel_DataView(t *testing.T) {
	entity := testModule + ".PMDataViewEntity"
	page := testModule + ".PMDataViewPage"
	roundtripPage(t, `
create or modify persistent entity `+entity+` (Title: String(200));
create or modify page `+page+` (
  title: 'DataView Test',
  layout: Atlas_Core.Atlas_Default,
  params: { $Item: `+entity+` }
) {
  dataview dv (DataSource: parameter $Item) {
    textbox tbTitle (Attribute: Title)
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "dataview") {
			t.Errorf("expected 'dataview' in describe output")
		}
		if !strings.Contains(described, "textbox") {
			t.Errorf("expected 'textbox' in describe output")
		}
	})
}

func TestRoundtrip_PageModel_TabContainer(t *testing.T) {
	entity := testModule + ".PMTabEntity"
	page := testModule + ".PMTabPage"
	roundtripPage(t, `
create or modify persistent entity `+entity+` (Name: String(200));
create or modify page `+page+` (
  title: 'Tab Test',
  layout: Atlas_Core.Atlas_Default
) {
  tabcontainer tabs {
    tab tab1 (caption: 'First Tab') {
      label lbl (caption: 'Hello')
    }
    tab tab2 (caption: 'Second Tab') {
      label lbl2 (caption: 'World')
    }
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "tabcontainer") {
			t.Errorf("expected 'tabcontainer' in describe output")
		}
		if !strings.Contains(described, "First Tab") {
			t.Errorf("expected tab caption in describe output")
		}
	})
}

func TestRoundtrip_PageModel_GroupBox(t *testing.T) {
	page := testModule + ".PMGroupBoxPage"
	roundtripPage(t, `
create or modify page `+page+` (
  title: 'GroupBox Test',
  layout: Atlas_Core.Atlas_Default
) {
  groupbox gb (caption: 'Details', collapsible: YesInitiallyExpanded) {
    label lbl (caption: 'Content')
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "groupbox") {
			t.Errorf("expected 'groupbox' in describe output")
		}
		if !strings.Contains(described, "YesInitiallyExpanded") {
			t.Errorf("expected collapsible setting in describe output")
		}
	})
}
```

- [ ] **Step 2: Add golden test assertion**

In `internal/goldenfs/helpdesk_regression_test.go`, find `TestHelpdeskGolden_DescribeSnapshot` and add after the `if string(want) == got { return }` check:

```go
	// Each page statement must have a non-empty body to catch silent empty-{} regression.
	assertPageBodiesNonEmpty(t, string(want))
```

Then add the helper function at the bottom of the file:

```go
// assertPageBodiesNonEmpty checks that every "create or modify page" statement
// in the snapshot has a non-empty body (i.e. contains at least one widget keyword).
func assertPageBodiesNonEmpty(t *testing.T, snapshot string) {
	t.Helper()
	widgetKeywords := []string{
		"container", "layoutgrid", "dataview", "datagrid", "gallery",
		"listview", "button", "textbox", "textarea", "datepicker",
		"tabcontainer", "groupbox", "label", "text ", "snippet",
	}
	lines := strings.Split(snapshot, "\n")
	inPage := false
	pageStart := 0
	pageHeader := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "create or modify page ") {
			inPage = true
			pageStart = i
			pageHeader = trimmed
		}
		if inPage && trimmed == "}" {
			// collect the body between pageStart and i
			body := strings.Join(lines[pageStart:i+1], "\n")
			hasWidget := false
			for _, kw := range widgetKeywords {
				if strings.Contains(body, kw) {
					hasWidget = true
					break
				}
			}
			if !hasWidget {
				t.Errorf("page has empty body (no widget keywords): %s (line %d)", pageHeader, pageStart+1)
			}
			inPage = false
		}
	}
}
```

- [ ] **Step 3: Run tests to confirm red**

```bash
go test -tags integration -run TestRoundtrip_PageModel ./mdl/executor/ -v 2>&1 | tail -30
```

Expected: FAIL (tests run but describe returns empty body or errors).

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/roundtrip_page_model_test.go \
        internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(pages): add failing roundtrip tests for PageModel IR (TDD red)"
```

---

## Task 4: MprBackend — GetPageModel (BSON → PageModel)

**Files:**
- Create: `mdl/backend/mpr/page_model.go`
backend/mpr/backend.go` (add compile-time guard)

- [ ] **Step 1: Create the file with fromBSON**

`mdl/backend/mpr/page_model.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ---------------------------------------------------------------------------
// GetPageModel / GetSnippetModel / GetLayoutModel
// ---------------------------------------------------------------------------

func (b *MprBackend) GetPageModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "page")
}

func (b *MprBackend) GetSnippetModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "snippet")
}

func (b *MprBackend) GetLayoutModel(id model.ID) (*types.PageModel, error) {
	return b.loadPageModel(id, "layout")
}

func (b *MprBackend) loadPageModel(id model.ID, kind string) (*types.PageModel, error) {
	raw, err := b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, fmt.Errorf("load unit bytes: %w", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	return pageDocToModel(doc), nil
}

// ---------------------------------------------------------------------------
// pageDocToModel: top-level page/snippet/layout BSON → PageModel
// ---------------------------------------------------------------------------

func pageDocToModel(doc bson.D) *types.PageModel {
	pm := &types.PageModel{}

	// Layout/form call — try LayoutCall (gen-encoded Cat-B) then FormCall (legacy)
	callDoc := dGetDoc(doc, "LayoutCall")
	if callDoc == nil {
		callDoc = dGetDoc(doc, "FormCall")
	}

	if callDoc != nil {
		// Extract layout name from the LayoutCall's $Type-referenced layout
		// The LayoutCall stores the layout reference as a qualified name.
		// In gen-encoded pages the layout name is stored in the gen object;
		// for describe purposes we extract it from the raw call document.
		if layoutRef := dGetString(callDoc, "LayoutQualifiedName"); layoutRef != "" {
			pm.Layout = layoutRef
		}

		// Extract widget tree from Arguments
		args := dGetArrayElements(dGet(callDoc, "Arguments"))
		for _, arg := range args {
			argDoc, ok := arg.(bson.D)
			if !ok {
				continue
			}
			// Gen-encoded: singular Widget field (a DivContainer wrapper)
			// Legacy: Widgets array
			if widgetDoc := dGetDoc(argDoc, "Widget"); widgetDoc != nil {
				widgets := widgetsFromDivContainer(widgetDoc)
				pm.Widgets = append(pm.Widgets, widgets...)
			} else {
				for _, w := range dGetArrayElements(dGet(argDoc, "Widgets")) {
					if wd, ok := w.(bson.D); ok {
						if node := widgetNodeFromBSON(wd); node != nil {
							pm.Widgets = append(pm.Widgets, node)
						}
					}
				}
			}
		}
	}

	return pm
}

// widgetsFromDivContainer unwraps a conditionalVisibilityWidget DivContainer.
// The gen-encoded write path wraps page content in a transparent DivContainer
// whose name starts with "conditionalVisibilityWidget". We unwrap it so the
// describe output omits the phantom container.
func widgetsFromDivContainer(doc bson.D) []*types.WidgetNode {
	name := dGetString(doc, "Name")
	typeName := dGetString(doc, "$Type")
	isWrapper := strings.HasPrefix(name, "conditionalVisibilityWidget") &&
		(typeName == "Pages$DivContainer" || typeName == "Forms$DivContainer")

	if isWrapper {
		var nodes []*types.WidgetNode
		for _, w := range dGetArrayElements(dGet(doc, "Widgets")) {
			if wd, ok := w.(bson.D); ok {
				if node := widgetNodeFromBSON(wd); node != nil {
					nodes = append(nodes, node)
				}
			}
		}
		return nodes
	}
	if node := widgetNodeFromBSON(doc); node != nil {
		return []*types.WidgetNode{node}
	}
	return nil
}

// ---------------------------------------------------------------------------
// BSON $type → WidgetKind mapping
// ---------------------------------------------------------------------------

var bsonTypeToKind = map[string]types.WidgetKind{
	"Forms$DivContainer":        types.WidgetContainer,
	"Pages$DivContainer":        types.WidgetContainer,
	"Forms$ScrollContainer":     types.WidgetScrollView,
	"Pages$ScrollContainer":     types.WidgetScrollView,
	"Forms$GroupBox":            types.WidgetGroupBox,
	"Pages$GroupBox":            types.WidgetGroupBox,
	"Forms$LayoutGrid":          types.WidgetLayoutGrid,
	"Pages$LayoutGrid":          types.WidgetLayoutGrid,
	"Forms$LayoutGridRow":       types.WidgetLayoutRow,
	"Pages$LayoutGridRow":       types.WidgetLayoutRow,
	"Forms$LayoutGridColumn":    types.WidgetLayoutCol,
	"Pages$LayoutGridColumn":    types.WidgetLayoutCol,
	"Forms$TabControl":          types.WidgetTabContainer,
	"Pages$TabControl":          types.WidgetTabContainer,
	"Pages$TabPage":             types.WidgetTabPage,
	"Forms$DataView":            types.WidgetDataView,
	"Pages$DataView":            types.WidgetDataView,
	"Forms$ListView":            types.WidgetListView,
	"Pages$ListView":            types.WidgetListView,
	"Forms$Gallery":             types.WidgetGallery,
	"Pages$Gallery":             types.WidgetGallery,
	"Forms$ActionButton":        types.WidgetButton,
	"Pages$ActionButton":        types.WidgetButton,
	"Forms$TextBox":             types.WidgetTextBox,
	"Pages$TextBox":             types.WidgetTextBox,
	"Forms$TextArea":            types.WidgetTextArea,
	"Pages$TextArea":            types.WidgetTextArea,
	"Forms$DatePicker":          types.WidgetDatePicker,
	"Pages$DatePicker":          types.WidgetDatePicker,
	"Forms$RadioButtons":        types.WidgetRadioButtons,
	"Pages$RadioButtons":        types.WidgetRadioButtons,
	"Forms$CheckBox":            types.WidgetCheckBox,
	"Pages$CheckBox":            types.WidgetCheckBox,
	"Forms$Label":               types.WidgetLabel,
	"Pages$Label":               types.WidgetLabel,
	"Forms$Text":                types.WidgetText,
	"Pages$Text":                types.WidgetText,
	"Forms$DynamicText":         types.WidgetDynamicText,
	"Pages$DynamicText":         types.WidgetDynamicText,
	"Forms$Title":               types.WidgetTitle,
	"Pages$Title":               types.WidgetTitle,
	"Forms$NavigationList":      types.WidgetNavList,
	"Pages$NavigationList":      types.WidgetNavList,
	"Forms$SnippetCallWidget":   types.WidgetSnippet,
	"Pages$SnippetCallWidget":   types.WidgetSnippet,
	"CustomWidgets$CustomWidget": types.WidgetUnknown, // refined below
}

// widgetNodeFromBSON converts a single raw BSON widget document to WidgetNode.
func widgetNodeFromBSON(doc bson.D) *types.WidgetNode {
	typeName := dGetString(doc, "$Type")
	name := dGetString(doc, "Name")

	kind, ok := bsonTypeToKind[typeName]
	if !ok {
		return nil // completely unknown structural element — skip
	}

	// Pluggable widget: refine kind from widget type string
	if typeName == "CustomWidgets$CustomWidget" {
		kind = customWidgetKind(doc)
	}

	node := &types.WidgetNode{Kind: kind, Name: name}

	// Extract appearance
	if app := dGetDoc(doc, "Appearance"); app != nil {
		node.Class = dGetString(app, "CSSClasses")
		node.Style = dGetString(app, "Style")
		node.DesignProps = extractDesignPropsFromBSON(app)
	}

	// Extract conditional visibility
	if cv := dGetDoc(doc, "ConditionalVisibilitySettings"); cv != nil {
		node.VisibleIf = dGetString(cv, "Expression")
	}

	// Kind-specific extraction
	switch kind {
	case types.WidgetContainer, types.WidgetScrollView:
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetLayoutGrid:
		node.Children = extractLayoutGridRows(doc)
	case types.WidgetLayoutRow:
		node.Children = extractChildWidgets(doc, "Columns")
	case types.WidgetLayoutCol:
		node.ColWidth = extractColWidth(doc)
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetGroupBox:
		node.Caption = extractTextFromTemplate(doc, "Caption")
		node.GroupBox = &types.GroupBoxProps{
			Collapsible: dGetString(doc, "Collapsible"),
			HeaderMode:  dGetString(doc, "HeaderMode"),
		}
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetTabContainer:
		node.Children = extractChildWidgets(doc, "TabPages")
	case types.WidgetTabPage:
		node.Caption = extractTextFromTemplate(doc, "Caption")
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetDataView:
		node.DataSource = extractBSONDataSource(doc)
		node.EntityCtx = extractDataViewEntityCtx(doc)
		node.Children = extractChildWidgets(doc, "Widgets")
	case types.WidgetListView:
		node.DataSource = extractBSONDataSource(doc)
		node.Children = extractChildWidgets(doc, "Templates")
	case types.WidgetButton:
		node.Caption = extractTextFromTemplate(doc, "Caption")
		node.ButtonStyle = dGetString(doc, "ButtonStyle")
		node.OnClick = extractButtonAction(doc)
	case types.WidgetLabel:
		node.Caption = extractTextFromTemplate(doc, "Caption")
	case types.WidgetText, types.WidgetTitle:
		node.Content = extractTextFromTemplate(doc, "Content")
	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		node.EntityAttr = extractAttributeRefStr(doc)
		node.Editable = extractEditableStr(doc)
		if ed := dGetDoc(doc, "ConditionalEditabilitySettings"); ed != nil {
			node.EditableIf = dGetString(ed, "Expression")
		}
	case types.WidgetSnippet:
		node.Snippet = &types.SnippetProps{SnippetName: extractSnippetName(doc)}
	case types.WidgetDataGrid:
		node.DataSource = extractPluggableDataSource(doc)
		node.DataGrid = extractDataGridProps(doc)
	case types.WidgetGallery:
		node.DataSource = extractPluggableDataSource(doc)
		node.Gallery = extractGalleryProps(doc)
	case types.WidgetImage:
		node.Image = extractImageProps(doc)
	case types.WidgetComboBox:
		node.DataSource = extractPluggableDataSource(doc)
		node.EntityAttr = extractCustomWidgetAttrRef(doc, "attribute")
	case types.WidgetUnknown:
		node.Unknown = &types.UnknownProps{
			WidgetID: extractCustomWidgetIDStr(doc),
		}
	}

	return node
}

// ---------------------------------------------------------------------------
// Helper extractors
// ---------------------------------------------------------------------------

func extractChildWidgets(doc bson.D, field string) []*types.WidgetNode {
	var nodes []*types.WidgetNode
	for _, w := range dGetArrayElements(dGet(doc, field)) {
		if wd, ok := w.(bson.D); ok {
			if node := widgetNodeFromBSON(wd); node != nil {
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

func extractLayoutGridRows(doc bson.D) []*types.WidgetNode {
	var nodes []*types.WidgetNode
	for _, r := range dGetArrayElements(dGet(doc, "Rows")) {
		if rd, ok := r.(bson.D); ok {
			if node := widgetNodeFromBSON(rd); node != nil {
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

func extractColWidth(doc bson.D) types.ColWidthDef {
	toInt := func(v any) int {
		switch n := v.(type) {
		case int32:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
		return 0
	}
	return types.ColWidthDef{
		Desktop: toInt(dGet(doc, "DesktopWeight")),
		Tablet:  toInt(dGet(doc, "TabletWeight")),
		Phone:   toInt(dGet(doc, "PhoneWeight")),
	}
}

func extractTextFromTemplate(doc bson.D, field string) string {
	tmpl := dGetDoc(doc, field)
	if tmpl == nil {
		return ""
	}
	// Translations are a versioned array; grab the first en_US translation.
	for _, t := range dGetArrayElements(dGet(tmpl, "Translations")) {
		if td, ok := t.(bson.D); ok {
			if dGetString(td, "LanguageCode") == "en_US" {
				return dGetString(td, "Text")
			}
		}
	}
	// Fallback: take first translation text regardless of language
	for _, t := range dGetArrayElements(dGet(tmpl, "Translations")) {
		if td, ok := t.(bson.D); ok {
			if txt := dGetString(td, "Text"); txt != "" {
				return txt
			}
		}
	}
	return ""
}

func extractAttributeRefStr(doc bson.D) string {
	aref := dGetDoc(doc, "AttributeRef")
	if aref == nil {
		return ""
	}
	return dGetString(aref, "AttributeQualifiedName")
}

func extractEditableStr(doc bson.D) string {
	return dGetString(doc, "Editable")
}

func extractDataViewEntityCtx(doc bson.D) string {
	// The entity type exposed by a DataView to its children is stored in
	// the data source's entity reference.
	ds := extractBSONDataSource(doc)
	if ds == nil {
		return ""
	}
	return ds.Entity
}

func extractBSONDataSource(doc bson.D) *types.DataSourceDef {
	dsd := dGetDoc(doc, "DataSource")
	if dsd == nil {
		return nil
	}
	typeName := dGetString(dsd, "$Type")
	ds := &types.DataSourceDef{}
	switch {
	case strings.Contains(typeName, "MicroflowSource"):
		ds.Kind = types.DataSourceMicroflow
		ds.Reference = dGetString(dsd, "MicroflowQualifiedName")
	case strings.Contains(typeName, "NanoflowSource"):
		ds.Kind = types.DataSourceNanoflow
		ds.Reference = dGetString(dsd, "NanoflowQualifiedName")
	case strings.Contains(typeName, "ContextSource") || strings.Contains(typeName, "ParameterSource"):
		ds.Kind = types.DataSourceParameter
		ds.Reference = dGetString(dsd, "ParameterName")
	case strings.Contains(typeName, "SelectionSource"):
		ds.Kind = types.DataSourceSelection
		ds.Reference = dGetString(dsd, "WidgetName")
	default:
		ds.Kind = types.DataSourceDatabase
		ds.Entity = dGetString(dsd, "EntityQualifiedName")
		ds.XPathConstraint = dGetString(dsd, "XPathConstraint")
	}
	return ds
}

func extractButtonAction(doc bson.D) string {
	act := dGetDoc(doc, "Action")
	if act == nil {
		return ""
	}
	typeName := dGetString(act, "$Type")
	switch {
	case strings.Contains(typeName, "MicroflowClientAction"):
		return dGetString(act, "MicroflowQualifiedName")
	case strings.Contains(typeName, "NanoflowClientAction"):
		return dGetString(act, "NanoflowQualifiedName")
	case strings.Contains(typeName, "OpenLinkClientAction"):
		return dGetString(act, "PageQualifiedName")
	}
	return ""
}

func extractSnippetName(doc bson.D) string {
	ref := dGetDoc(doc, "Snippet")
	if ref == nil {
		return ""
	}
	return dGetString(ref, "SnippetQualifiedName")
}

func extractDesignPropsFromBSON(app bson.D) []types.DesignProp {
	var props []types.DesignProp
	for _, dp := range dGetArrayElements(dGet(app, "DesignProperties")) {
		dpd, ok := dp.(bson.D)
		if !ok {
			continue
		}
		p := types.DesignProp{
			Key:       dGetString(dpd, "Key"),
			ValueType: dGetString(dpd, "Type"),
		}
		switch p.ValueType {
		case "option":
			p.Option = dGetString(dpd, "Value")
		}
		if p.Key != "" {
			props = append(props, p)
		}
	}
	return props
}

// customWidgetKind returns the WidgetKind for a CustomWidgets$CustomWidget
// by inspecting the widget type string stored in its object properties.
func customWidgetKind(doc bson.D) types.WidgetKind {
	widgetType := extractCustomWidgetIDStr(doc)
	switch {
	case strings.Contains(widgetType, "datagrid2"):
		return types.WidgetDataGrid
	case strings.Contains(widgetType, "gallery"):
		return types.WidgetGallery
	case strings.Contains(widgetType, "combobox"):
		return types.WidgetComboBox
	case strings.Contains(widgetType, "image"):
		return types.WidgetImage
	}
	return types.WidgetUnknown
}

func extractCustomWidgetIDStr(doc bson.D) string {
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ""
	}
	return dGetString(obj, "Type")
}

func extractCustomWidgetAttrRef(doc bson.D, propKey string) string {
	// Pluggable widget attribute references are stored as ObjectProperty with
	// key == propKey and type "attribute".
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ""
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		if dGetString(pd, "key") == propKey {
			val := dGetDoc(pd, "Value")
			if val == nil {
				return ""
			}
			return dGetString(val, "AttributeQualifiedName")
		}
	}
	return ""
}

func extractPluggableDataSource(doc bson.D) *types.DataSourceDef {
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return nil
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		if dGetString(pd, "key") == "datasource" {
			val := dGetDoc(pd, "Value")
			if val == nil {
				return nil
			}
			ds := &types.DataSourceDef{}
			typeName := dGetString(val, "$Type")
			switch {
			case strings.Contains(typeName, "NanoflowSource"):
				ds.Kind = types.DataSourceNanoflow
				ds.Reference = dGetString(val, "NanoflowQualifiedName")
			case strings.Contains(typeName, "MicroflowSource"):
				ds.Kind = types.DataSourceMicroflow
				ds.Reference = dGetString(val, "MicroflowQualifiedName")
			case strings.Contains(typeName, "ContextSource"):
				ds.Kind = types.DataSourceParameter
			default:
				ds.Kind = types.DataSourceDatabase
				ds.Entity = dGetString(val, "EntityQualifiedName")
				ds.XPathConstraint = dGetString(val, "XPathConstraint")
			}
			return ds
		}
	}
	return nil
}

func extractDataGridProps(doc bson.D) *types.DataGridProps {
	dgp := &types.DataGridProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return dgp
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "pageSize":
			if s := dGetString(val, "Value"); s != "" {
				if n, err := strconv.Atoi(s); err == nil {
					dgp.PageSize = n
				}
			}
		case "pagination":
			dgp.Pagination = dGetString(val, "Value")
		case "pagingPosition":
			dgp.PagingPos = dGetString(val, "Value")
		case "columns":
			dgp.Columns = extractDataGridColumns(val)
		}
	}
	return dgp
}

func extractDataGridColumns(columnsVal bson.D) []types.ColumnDef {
	var cols []types.ColumnDef
	for _, item := range dGetArrayElements(dGet(columnsVal, "ObjectItems")) {
		itemDoc, ok := item.(bson.D)
		if !ok {
			continue
		}
		col := types.ColumnDef{}
		col.Name = dGetString(itemDoc, "Name")
		for _, prop := range dGetArrayElements(dGet(itemDoc, "Properties")) {
			pd, ok := prop.(bson.D)
			if !ok {
				continue
			}
			key := dGetString(pd, "key")
			val := dGetDoc(pd, "Value")
			if val == nil {
				continue
			}
			switch key {
			case "attribute":
				col.Attribute = dGetString(val, "AttributeQualifiedName")
			case "header":
				col.Caption = dGetString(val, "Value")
			case "showContentAs":
				col.ShowContentAs = dGetString(val, "Value")
			case "alignment":
				col.Alignment = dGetString(val, "Value")
			case "wrapText":
				col.WrapText = dGetString(val, "Value") == "true"
			case "sortable":
				col.Sortable = dGetString(val, "Value") == "true"
			case "resizable":
				col.Resizable = dGetString(val, "Value") == "true"
			case "draggable":
				col.Draggable = dGetString(val, "Value") == "true"
			case "hidable":
				col.Hidable = dGetString(val, "Value")
			case "width":
				col.ColumnWidth = dGetString(val, "Value")
			case "size":
				col.Size = dGetString(val, "Value")
			}
		}
		cols = append(cols, col)
	}
	return cols
}

func extractGalleryProps(doc bson.D) *types.GalleryProps {
	gp := &types.GalleryProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return gp
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "desktopItems":
			gp.DesktopColumns = toInt(dGetString(val, "Value"))
		case "tabletItems":
			gp.TabletColumns = toInt(dGetString(val, "Value"))
		case "phoneItems":
			gp.PhoneColumns = toInt(dGetString(val, "Value"))
		case "itemSelection":
			gp.Selection = dGetString(val, "Value")
		}
	}
	return gp
}

func extractImageProps(doc bson.D) *types.ImageProps {
	ip := &types.ImageProps{}
	obj := dGetDoc(doc, "Object")
	if obj == nil {
		return ip
	}
	for _, prop := range dGetArrayElements(dGet(obj, "Properties")) {
		pd, ok := prop.(bson.D)
		if !ok {
			continue
		}
		key := dGetString(pd, "key")
		val := dGetDoc(pd, "Value")
		if val == nil {
			continue
		}
		switch key {
		case "imageUrl":
			ip.URL = dGetString(val, "Value")
		case "alternativeText":
			ip.AltText = dGetString(val, "Value")
		case "width":
			ip.Width = dGetString(val, "Value")
		case "height":
			ip.Height = dGetString(val, "Value")
		case "widthUnit":
			ip.WidthUnit = dGetString(val, "Value")
		case "heightUnit":
			ip.HeightUnit = dGetString(val, "Value")
		case "displayAs":
			ip.DisplayAs = dGetString(val, "Value")
		case "responsive":
			ip.Responsive = dGetString(val, "Value") == "true"
		case "type":
			ip.ImageType = dGetString(val, "Value")
		case "onClick":
			ip.OnClickType = dGetString(val, "Value")
		}
	}
	return ip
}
```

- [ ] **Step 2: Add compile-time guard**

In `mdl/backend/mpr/backend.go`, add below the existing `var _ backend.FullBackend` line:

```go
var _ backend.PageModelBackend = (*MprBackend)(nil)
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./mdl/backend/mpr/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/page_model.go mdl/backend/mpr/backend.go
git commit -m "feat(mpr): implement GetPageModel — BSON→PageModel conversion"
```

---

## Task 5: Executor — pageModelToMDL Renderer

**Files:**
- Create: `mdl/executor/cmd_pages_model_to_mdl.go`

- [ ] **Step 1: Create the renderer**

`mdl/executor/cmd_pages_model_to_mdl.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// pageModelToMDL renders a PageModel to MDL V3 text and writes it to w.
func pageModelToMDL(w io.Writer, pm *types.PageModel, modName, pageName string) {
	// Header: create or modify page Module.Name (...)
	fmt.Fprintf(w, "create or modify page %s.%s (", modName, pageName)
	fmt.Fprintf(w, "\n  title: '%s'", escapeMDLString(pm.Title))
	if pm.Layout != "" {
		fmt.Fprintf(w, ",\n  layout: %s", pm.Layout)
	}
	if pm.Folder != "" {
		fmt.Fprintf(w, ",\n  folder: '%s'", pm.Folder)
	}
	if len(pm.Params) > 0 {
		parts := make([]string, len(pm.Params))
		for i, p := range pm.Params {
			parts[i] = fmt.Sprintf("$%s: %s", p.Name, p.EntityName)
		}
		fmt.Fprintf(w, ",\n  params: { %s }", strings.Join(parts, ", "))
	}
	fmt.Fprintf(w, "\n) {\n")

	for _, widget := range pm.Widgets {
		renderWidget(w, widget, 1)
	}

	fmt.Fprintf(w, "}")
}

func renderWidget(w io.Writer, node *types.WidgetNode, depth int) {
	if node == nil {
		return
	}
	indent := strings.Repeat("  ", depth)

	switch node.Kind {
	case types.WidgetContainer, types.WidgetScrollView:
		kw := "container"
		if node.Kind == types.WidgetScrollView {
			kw = "scrollview"
		}
		fmt.Fprintf(w, "%s%s %s", indent, kw, node.Name)
		renderAppearanceInline(w, node)
		renderVisibility(w, node)
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutGrid:
		fmt.Fprintf(w, "%slayoutgrid %s {\n", indent, node.Name)
		for _, r := range node.Children {
			renderWidget(w, r, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutRow:
		fmt.Fprintf(w, "%srow %s {\n", indent, node.Name)
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutCol:
		cw := node.ColWidth
		fmt.Fprintf(w, "%scolumn %s", indent, node.Name)
		if cw.Desktop > 0 || cw.Tablet > 0 || cw.Phone > 0 {
			fmt.Fprintf(w, " (")
			sep := ""
			if cw.Desktop > 0 {
				fmt.Fprintf(w, "%sDesktopWidth: %d", sep, cw.Desktop)
				sep = ", "
			}
			if cw.Tablet > 0 {
				fmt.Fprintf(w, "%sTabletWidth: %d", sep, cw.Tablet)
				sep = ", "
			}
			if cw.Phone > 0 {
				fmt.Fprintf(w, "%sPhoneWidth: %d", sep, cw.Phone)
			}
			fmt.Fprintf(w, ")")
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetGroupBox:
		fmt.Fprintf(w, "%sgroupbox %s", indent, node.Name)
		props := []string{}
		if node.Caption != "" {
			props = append(props, fmt.Sprintf("caption: '%s'", escapeMDLString(node.Caption)))
		}
		if node.GroupBox != nil && node.GroupBox.Collapsible != "" && node.GroupBox.Collapsible != "No" {
			props = append(props, fmt.Sprintf("collapsible: %s", node.GroupBox.Collapsible))
		}
		if len(props) > 0 {
			fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetTabContainer:
		fmt.Fprintf(w, "%stabcontainer %s {\n", indent, node.Name)
		for _, tp := range node.Children {
			renderWidget(w, tp, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetTabPage:
		caption := ""
		if node.Caption != "" {
			caption = fmt.Sprintf(" (caption: '%s')", escapeMDLString(node.Caption))
		}
		fmt.Fprintf(w, "%stab %s%s {\n", indent, node.Name, caption)
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetDataView:
		fmt.Fprintf(w, "%sdataview %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			renderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetDataGrid:
		fmt.Fprintf(w, "%sdatagrid %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		if node.DataGrid != nil {
			for _, col := range node.DataGrid.Columns {
				renderDataGridColumn(w, col, depth+1)
			}
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetButton:
		fmt.Fprintf(w, "%sbutton %s", indent, node.Name)
		props := []string{}
		if node.Caption != "" {
			props = append(props, fmt.Sprintf("caption: '%s'", escapeMDLString(node.Caption)))
		}
		if node.OnClick != "" {
			props = append(props, fmt.Sprintf("action: call microflow %s", node.OnClick))
		}
		if node.ButtonStyle != "" && node.ButtonStyle != "Default" {
			props = append(props, fmt.Sprintf("style: %s", node.ButtonStyle))
		}
		if len(props) > 0 {
			fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetTextBox:
		fmt.Fprintf(w, "%stextbox %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetTextArea:
		fmt.Fprintf(w, "%stextarea %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetDatePicker:
		fmt.Fprintf(w, "%sdatepicker %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetLabel:
		fmt.Fprintf(w, "%slabel %s", indent, node.Name)
		if node.Caption != "" {
			fmt.Fprintf(w, " (caption: '%s')", escapeMDLString(node.Caption))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetText, types.WidgetTitle:
		kw := "text"
		if node.Kind == types.WidgetTitle {
			kw = "title"
		}
		fmt.Fprintf(w, "%s%s %s", indent, kw, node.Name)
		if node.Content != "" {
			fmt.Fprintf(w, " (content: '%s')", escapeMDLString(node.Content))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetSnippet:
		name := ""
		if node.Snippet != nil {
			name = node.Snippet.SnippetName
		}
		fmt.Fprintf(w, "%ssnippet %s (ref: %s)\n", indent, node.Name, name)

	case types.WidgetGallery:
		fmt.Fprintf(w, "%sgallery %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		if node.Gallery != nil {
			for _, c := range node.Gallery.ContentWidgets {
				renderWidget(w, c, depth+1)
			}
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetUnknown:
		widgetID := ""
		if node.Unknown != nil {
			widgetID = node.Unknown.WidgetID
		}
		fmt.Fprintf(w, "%s-- unsupported widget: %s (%s)\n", indent, node.Name, widgetID)

	default:
		fmt.Fprintf(w, "%s-- unhandled kind: %s %s\n", indent, node.Kind, node.Name)
	}
}

func renderDataSource(ds *types.DataSourceDef) string {
	switch ds.Kind {
	case types.DataSourceDatabase:
		s := "database " + ds.Entity
		if ds.XPathConstraint != "" {
			s += fmt.Sprintf(" where '%s'", ds.XPathConstraint)
		}
		return s
	case types.DataSourceMicroflow:
		return "call microflow " + ds.Reference
	case types.DataSourceNanoflow:
		return "call nanoflow " + ds.Reference
	case types.DataSourceParameter:
		return "parameter " + ds.Reference
	case types.DataSourceSelection:
		return "selection " + ds.Reference
	}
	return ""
}

func renderDataGridColumn(w io.Writer, col types.ColumnDef, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%scolumn %s", indent, col.Name)
	props := []string{}
	if col.Attribute != "" {
		props = append(props, fmt.Sprintf("Attribute: %s", col.Attribute))
	}
	if col.Caption != "" {
		props = append(props, fmt.Sprintf("Caption: '%s'", escapeMDLString(col.Caption)))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
	fmt.Fprintf(w, "\n")
}

func renderInputProps(w io.Writer, node *types.WidgetNode) {
	props := []string{}
	if node.EntityAttr != "" {
		props = append(props, fmt.Sprintf("Attribute: %s", node.EntityAttr))
	}
	if node.Editable != "" && node.Editable != "Always" {
		props = append(props, fmt.Sprintf("editable: %s", node.Editable))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
}

func renderAppearanceInline(w io.Writer, node *types.WidgetNode) {
	props := []string{}
	if node.Class != "" {
		props = append(props, fmt.Sprintf("class: '%s'", node.Class))
	}
	if node.Style != "" {
		props = append(props, fmt.Sprintf("style: '%s'", node.Style))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
}

func renderVisibility(w io.Writer, node *types.WidgetNode) {
	if node.VisibleIf != "" {
		fmt.Fprintf(w, " visible if '%s'", node.VisibleIf)
	}
}

func escapeMDLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./mdl/executor/...
```

Expected: no errors (the new file doesn't yet replace existing describe logic).

- [ ] **Step 3: Commit**

```bash
git add mdl/executor/cmd_pages_model_to_mdl.go
git commit -m "feat(executor): add pageModelToMDL renderer (PageModel→MDL text)"
```

---

## Task 6: Wire Read Path + Delete Old Describe Files

**Files:**
- Modify: `mdl/executor/cmd_pages_describe.go`
- Delete: `mdl/executor/cmd_pages_describe_parse.go`
- Delete: `mdl/executor/cmd_pages_describe_output.go`
- Delete: `mdl/executor/cmd_pages_describe_pluggable.go`

- [ ] **Step 1: Replace describePage() in cmd_pages_describe.go**

Find the `describePage()` function (line ~21) and replace its body to use `GetPageModel()` + `pageModelToMDL()`:

```go
func describePage(ctx *ExecContext, name ast.QualifiedName) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	var foundPage *genPg.Page
	var foundContainerID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			foundPage = p.Elem
			foundContainerID = model.ID(p.ContainerID)
			break
		}
	}
	if foundPage == nil {
		return mdlerrors.NewNotFound("page", name.String())
	}

	pageID := model.ID(foundPage.ID())
	modID := h.FindModuleID(foundContainerID)
	modName := h.GetModuleName(modID)

	pm, err := ctx.Backend.GetPageModel(pageID)
	if err != nil {
		return mdlerrors.NewBackend("get page model", err)
	}

	// Populate metadata from the gen-typed Page object (layout name, folder, params).
	pm.ModuleName = modName
	pm.Name = foundPage.Name()
	pm.Title = pickPageTitleGen(foundPage)
	pm.Layout = resolvePageLayoutName(ctx, foundPage)
	pm.Folder = resolvePageFolder(ctx, foundContainerID)
	pm.Params = resolvePageParams(ctx, foundPage)

	if doc := foundPage.Documentation(); doc != "" {
		lines := strings.Split(doc, "\n")
		fmt.Fprint(ctx.Output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(ctx.Output, " * %s\n", line)
		}
		fmt.Fprint(ctx.Output, " */\n")
	}

	pageModelToMDL(ctx.Output, pm, modName, foundPage.Name())
	fmt.Fprintln(ctx.Output)
	return nil
}
```

Add helper functions that extract metadata from gen-typed Page (these replace lookups that were previously done inline with raw BSON):

```go
// resolvePageLayoutName returns the qualified layout name for the page.
func resolvePageLayoutName(ctx *ExecContext, page *genPg.Page) string {
	// Try gen accessor first
	if lqn := page.LayoutQualifiedName(); lqn != "" {
		return lqn
	}
	// Fallback: raw BSON FormCall/LayoutCall
	rawData, _ := ctx.Backend.GetRawUnit(model.ID(page.ID()))
	if rawData == nil {
		return ""
	}
	formCall, _ := rawData["FormCall"].(map[string]any)
	if formCall == nil {
		formCall, _ = rawData["LayoutCall"].(map[string]any)
	}
	if formCall == nil {
		return ""
	}
	if lqn, ok := formCall["LayoutQualifiedName"].(string); ok {
		return lqn
	}
	return ""
}

// resolvePageFolder returns the folder path for a page given its container ID.
func resolvePageFolder(ctx *ExecContext, containerID model.ID) string {
	h, err := getHierarchy(ctx)
	if err != nil {
		return ""
	}
	return h.GetFolderPath(containerID)
}

// resolvePageParams extracts page parameters from the gen-typed Page object.
func resolvePageParams(ctx *ExecContext, page *genPg.Page) []types.PageParam {
	var params []types.PageParam
	for _, p := range page.ParametersItems() {
		params = append(params, types.PageParam{
			Name:       p.Name(),
			EntityName: p.EntityQualifiedName(),
		})
	}
	return params
}
```

- [ ] **Step 2: Remove imports of deleted files from describe.go**

Remove any imports that are only used by the now-deleted code:
- `"go.mongodb.org/mongo-driver/v2/bson"` (if no longer used directly)
- The `rawWidget`, `rawDataSource`, `rawDataGridColumn` type references

The remaining imports should be:
```go
import (
    "fmt"
    "strings"

    "github.com/mendixlabs/mxcli/mdl/ast"
    mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
    "github.com/mendixlabs/mxcli/mdl/types"
    "github.com/mendixlabs/mxcli/model"
    genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)
```

- [ ] **Step 3: Delete the old describe files**

```bash
git rm mdl/executor/cmd_pages_describe_parse.go \
       mdl/executor/cmd_pages_describe_output.go \
       mdl/executor/cmd_pages_describe_pluggable.go
```

- [ ] **Step 4: Delete associated test files for deleted code**

```bash
git rm mdl/executor/cmd_pages_describe_container_test.go \
       mdl/executor/cmd_pages_describe_pluggable_test.go 2>/dev/null || true
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./mdl/executor/...
```

Fix any remaining compilation errors from references to deleted types/functions.

- [ ] **Step 6: Run roundtrip tests**

```bash
go test -tags integration -run TestRoundtrip_PageModel ./mdl/executor/ -v 2>&1 | tail -40
```

Expected: tests now PASS (describe returns widget content, stability holds).

- [ ] **Step 7: Run describe_sanity test**

```bash
go test -tags integration -run TestDescribeSanity ./mdl/executor/ -v 2>&1 | tail -20
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add mdl/executor/cmd_pages_describe.go
git commit -m "feat(executor): wire describe to GetPageModel+pageModelToMDL; delete rawWidget layer"
```

---

## Task 7: MprBackend — WritePageModel (PageModel → BSON)

**Files:**
- Modify: `mdl/backend/mpr/page_model.go` (add toBSON + WritePageModel)

- [ ] **Step 1: Add WritePageModel and toBSON to page_model.go**

Append to `mdl/backend/mpr/page_model.go`:

```go
// ---------------------------------------------------------------------------
// WritePageModel / WriteSnippetModel
// ---------------------------------------------------------------------------

func (b *MprBackend) WritePageModel(id model.ID, pm *types.PageModel) error {
	return b.writeUnitWidgets(id, pm)
}

func (b *MprBackend) WriteSnippetModel(id model.ID, pm *types.PageModel) error {
	return b.writeUnitWidgets(id, pm)
}

// writeUnitWidgets loads the existing unit BSON, replaces the widget tree
// inside the LayoutCall/FormCall Arguments with the serialised PageModel
// widgets, and writes back.
func (b *MprBackend) writeUnitWidgets(id model.ID, pm *types.PageModel) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("WritePageModel: backend not open for writing")
	}

	raw, err := b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return fmt.Errorf("load unit: %w", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Build widget tree BSON from PageModel
	widgetsBSON := widgetsToBSON(pm.Widgets)

	// Inject into LayoutCall.Arguments[0].Widget.Widgets (gen-encoded path)
	// or FormCall.Arguments[0].Widgets (legacy path).
	callKey := "LayoutCall"
	if dGetDoc(doc, "LayoutCall") == nil {
		callKey = "FormCall"
	}
	callDoc := dGetDoc(doc, callKey)
	if callDoc == nil {
		return fmt.Errorf("writeUnitWidgets: no %s in unit %s", callKey, id)
	}

	args := dGetArrayElements(dGet(callDoc, "Arguments"))
	if len(args) == 0 {
		return fmt.Errorf("writeUnitWidgets: no Arguments in %s", callKey)
	}

	// First argument holds the widget slot
	arg0, ok := args[0].(bson.D)
	if !ok {
		return fmt.Errorf("writeUnitWidgets: first argument is not a bson.D")
	}

	// Gen-encoded: slot has a Widget (DivContainer wrapper) whose Widgets we replace
	if wrapper := dGetDoc(arg0, "Widget"); wrapper != nil {
		dSet(wrapper, "Widgets", bsonVersionedArray(widgetsBSON))
	} else {
		// Legacy: slot has a Widgets array directly
		dSet(arg0, "Widgets", bsonVersionedArray(widgetsBSON))
	}

	out, err := bson.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return b.msdkWriter.UpdateRawUnit(string(id), out)
}

// bsonVersionedArray wraps a []bson.D in Mendix's versioned-array format
// [int32(1), elem0, elem1, ...].
func bsonVersionedArray(docs []bson.D) bson.A {
	arr := bson.A{int32(1)}
	for _, d := range docs {
		arr = append(arr, d)
	}
	return arr
}

// ---------------------------------------------------------------------------
// PageModel → BSON (widgetsToBSON + widgetToBSON)
// ---------------------------------------------------------------------------

// widgetsToBSON converts a slice of WidgetNodes to []bson.D.
func widgetsToBSON(nodes []*types.WidgetNode) []bson.D {
	var docs []bson.D
	for _, n := range nodes {
		if d := widgetToBSON(n); d != nil {
			docs = append(docs, d)
		}
	}
	return docs
}

// widgetToBSON converts a single WidgetNode to a BSON document.
// This is the inverse of widgetNodeFromBSON.
func widgetToBSON(node *types.WidgetNode) bson.D {
	if node == nil {
		return nil
	}

	typeName := kindToBSONType(node.Kind)
	if typeName == "" {
		return nil // unknown kind — skip
	}

	doc := bson.D{
		{Key: "$Type", Value: typeName},
		{Key: "Name", Value: node.Name},
	}

	// Appearance
	app := bson.D{
		{Key: "$Type", Value: "Forms$Appearance"},
	}
	if node.Class != "" {
		app = append(app, bson.E{Key: "CSSClasses", Value: node.Class})
	}
	if node.Style != "" {
		app = append(app, bson.E{Key: "Style", Value: node.Style})
	}
	doc = append(doc, bson.E{Key: "Appearance", Value: app})

	// Children / kind-specific fields
	switch node.Kind {
	case types.WidgetContainer, types.WidgetScrollView:
		doc = append(doc, bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetLayoutGrid:
		rows := widgetsToBSON(node.Children)
		doc = append(doc, bson.E{Key: "Rows", Value: bsonVersionedArray(rows)})

	case types.WidgetLayoutRow:
		cols := widgetsToBSON(node.Children)
		doc = append(doc, bson.E{Key: "Columns", Value: bsonVersionedArray(cols)})

	case types.WidgetLayoutCol:
		doc = append(doc,
			bson.E{Key: "DesktopWeight", Value: int32(node.ColWidth.Desktop)},
			bson.E{Key: "TabletWeight", Value: int32(node.ColWidth.Tablet)},
			bson.E{Key: "PhoneWeight", Value: int32(node.ColWidth.Phone)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)

	case types.WidgetGroupBox:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)
		if node.GroupBox != nil {
			if node.GroupBox.Collapsible != "" {
				doc = append(doc, bson.E{Key: "Collapsible", Value: node.GroupBox.Collapsible})
			}
			if node.GroupBox.HeaderMode != "" {
				doc = append(doc, bson.E{Key: "HeaderMode", Value: node.GroupBox.HeaderMode})
			}
		}

	case types.WidgetTabContainer:
		doc = append(doc, bson.E{Key: "TabPages", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetTabPage:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))},
		)

	case types.WidgetDataView:
		if node.DataSource != nil {
			doc = append(doc, bson.E{Key: "DataSource", Value: dataSourceToBSON(node.DataSource)})
		}
		doc = append(doc, bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))})

	case types.WidgetButton:
		doc = append(doc,
			bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)},
			bson.E{Key: "ButtonStyle", Value: node.ButtonStyle},
		)

	case types.WidgetLabel:
		doc = append(doc, bson.E{Key: "Caption", Value: simpleTextBSON(node.Caption)})

	case types.WidgetText, types.WidgetTitle:
		doc = append(doc, bson.E{Key: "Content", Value: simpleTextBSON(node.Content)})

	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		if node.EntityAttr != "" {
			doc = append(doc, bson.E{Key: "AttributeRef", Value: bson.D{
				{Key: "$Type", Value: "Forms$AttributeRef"},
				{Key: "AttributeQualifiedName", Value: node.EntityAttr},
			}})
		}
		if node.Editable != "" {
			doc = append(doc, bson.E{Key: "Editable", Value: node.Editable})
		}

	case types.WidgetSnippet:
		if node.Snippet != nil {
			doc = append(doc, bson.E{Key: "Snippet", Value: bson.D{
				{Key: "$Type", Value: "Forms$SnippetRef"},
				{Key: "SnippetQualifiedName", Value: node.Snippet.SnippetName},
			}})
		}
	}

	return doc
}

// kindToBSONType maps WidgetKind → canonical BSON $Type (Pages$ namespace).
func kindToBSONType(kind types.WidgetKind) string {
	switch kind {
	case types.WidgetContainer:
		return "Pages$DivContainer"
	case types.WidgetScrollView:
		return "Pages$ScrollContainer"
	case types.WidgetGroupBox:
		return "Pages$GroupBox"
	case types.WidgetLayoutGrid:
		return "Pages$LayoutGrid"
	case types.WidgetLayoutRow:
		return "Pages$LayoutGridRow"
	case types.WidgetLayoutCol:
		return "Pages$LayoutGridColumn"
	case types.WidgetTabContainer:
		return "Pages$TabControl"
	case types.WidgetTabPage:
		return "Pages$TabPage"
	case types.WidgetDataView:
		return "Pages$DataView"
	case types.WidgetListView:
		return "Pages$ListView"
	case types.WidgetButton:
		return "Pages$ActionButton"
	case types.WidgetLabel:
		return "Pages$Label"
	case types.WidgetText:
		return "Pages$Text"
	case types.WidgetDynamicText:
		return "Pages$DynamicText"
	case types.WidgetTitle:
		return "Pages$Title"
	case types.WidgetTextBox:
		return "Pages$TextBox"
	case types.WidgetTextArea:
		return "Pages$TextArea"
	case types.WidgetDatePicker:
		return "Pages$DatePicker"
	case types.WidgetRadioButtons:
		return "Pages$RadioButtons"
	case types.WidgetCheckBox:
		return "Pages$CheckBox"
	case types.WidgetNavList:
		return "Pages$NavigationList"
	case types.WidgetSnippet:
		return "Pages$SnippetCallWidget"
	case types.WidgetDataGrid, types.WidgetGallery, types.WidgetComboBox,
		types.WidgetImage, types.WidgetUnknown:
		return "CustomWidgets$CustomWidget"
	}
	return ""
}

// simpleTextBSON creates a minimal Text BSON doc with a single en_US translation.
func simpleTextBSON(text string) bson.D {
	tr := bson.D{
		{Key: "$Type", Value: "Texts$Translation"},
		{Key: "LanguageCode", Value: "en_US"},
		{Key: "Text", Value: text},
	}
	return bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Translations", Value: bson.A{int32(1), tr}},
	}
}

// dataSourceToBSON converts a DataSourceDef to its BSON representation.
func dataSourceToBSON(ds *types.DataSourceDef) bson.D {
	switch ds.Kind {
	case types.DataSourceDatabase:
		doc := bson.D{
			{Key: "$Type", Value: "Pages$XPathSource"},
			{Key: "EntityQualifiedName", Value: ds.Entity},
		}
		if ds.XPathConstraint != "" {
			doc = append(doc, bson.E{Key: "XPathConstraint", Value: ds.XPathConstraint})
		}
		return doc
	case types.DataSourceMicroflow:
		return bson.D{
			{Key: "$Type", Value: "Pages$MicroflowSource"},
			{Key: "MicroflowQualifiedName", Value: ds.Reference},
		}
	case types.DataSourceNanoflow:
		return bson.D{
			{Key: "$Type", Value: "Pages$NanoflowSource"},
			{Key: "NanoflowQualifiedName", Value: ds.Reference},
		}
	case types.DataSourceParameter:
		return bson.D{
			{Key: "$Type", Value: "Pages$ContextSource"},
			{Key: "ParameterName", Value: ds.Reference},
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./mdl/backend/mpr/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add mdl/backend/mpr/page_model.go
git commit -m "feat(mpr): implement WritePageModel — PageModel→BSON injection"
```

---

## Task 8: Executor — pageASTToModel + Wire Write Path

**Files:**
- Create: `mdl/executor/cmd_pages_ast_to_model.go`
- Modify: `mdl/executor/cmd_pages_create_v3.go`

- [ ] **Step 1: Create pageASTToModel**

`mdl/executor/cmd_pages_ast_to_model.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// pageASTToModel converts a CreatePageStmtV3 AST to a types.PageModel.
// All name resolution (microflows, entities, snippets) is done here using
// the ExecContext; the backend receives only the typed PageModel.
func pageASTToModel(ctx *ExecContext, s *ast.CreatePageStmtV3) (*types.PageModel, error) {
	pm := &types.PageModel{
		ModuleName: s.Name.Module,
		Name:       s.Name.Name,
		Title:      s.Title,
		Layout:     s.Layout.String(),
		Folder:     s.Folder,
	}

	for _, p := range s.Parameters {
		pm.Params = append(pm.Params, types.PageParam{
			Name:       p.Name,
			EntityName: p.Type.String(),
		})
	}

	for _, w := range s.Widgets {
		node, err := astWidgetToNode(ctx, w, s.Name.Module)
		if err != nil {
			return nil, fmt.Errorf("widget %s: %w", w.Name, err)
		}
		if node != nil {
			pm.Widgets = append(pm.Widgets, node)
		}
	}

	return pm, nil
}

// astWidgetToNode converts a single AST widget to a WidgetNode.
func astWidgetToNode(ctx *ExecContext, w *ast.WidgetV3, moduleName string) (*types.WidgetNode, error) {
	if w == nil {
		return nil, nil
	}

	kind := astWidgetKind(w.Type)
	node := &types.WidgetNode{
		Kind: kind,
		Name: w.Name,
	}

	// Common appearance
	node.Class = w.Class
	node.Style = w.Style

	// Common conditional visibility
	if w.VisibleIf != "" {
		node.VisibleIf = w.VisibleIf
	}

	// Convert children recursively
	for _, child := range w.Children {
		childNode, err := astWidgetToNode(ctx, child, moduleName)
		if err != nil {
			return nil, err
		}
		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	// Kind-specific properties
	switch kind {
	case types.WidgetContainer, types.WidgetScrollView:
		// children already set above

	case types.WidgetLayoutGrid:
		// rows are in children

	case types.WidgetLayoutRow:
		// columns are in children

	case types.WidgetLayoutCol:
		node.ColWidth = types.ColWidthDef{
			Desktop: w.DesktopWidth,
			Tablet:  w.TabletWidth,
			Phone:   w.PhoneWidth,
		}

	case types.WidgetGroupBox:
		node.Caption = w.Caption
		if w.GroupBox != nil {
			node.GroupBox = &types.GroupBoxProps{
				Collapsible: w.GroupBox.Collapsible,
				HeaderMode:  w.GroupBox.HeaderMode,
			}
		}

	case types.WidgetTabContainer:
		// tab pages are in children

	case types.WidgetTabPage:
		node.Caption = w.Caption

	case types.WidgetDataView:
		node.DataSource = astDataSourceToModel(w.DataSource)
		if w.DataSource != nil {
			node.EntityCtx = astDataSourceEntity(ctx, w.DataSource)
		}

	case types.WidgetListView:
		node.DataSource = astDataSourceToModel(w.DataSource)

	case types.WidgetDataGrid:
		node.DataSource = astDataSourceToModel(w.DataSource)
		dgp := &types.DataGridProps{}
		for _, col := range w.DataGridColumns {
			dgp.Columns = append(dgp.Columns, types.ColumnDef{
				Name:      col.Name,
				Attribute: col.Attribute,
				Caption:   col.Caption,
			})
		}
		node.DataGrid = dgp

	case types.WidgetButton:
		node.Caption = w.Caption
		node.ButtonStyle = w.ButtonStyle
		node.OnClick = w.OnClick

	case types.WidgetLabel:
		node.Caption = w.Caption

	case types.WidgetText, types.WidgetTitle:
		node.Content = w.Content

	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		node.EntityAttr = w.Attribute
		node.Editable = w.Editable
		node.EditableIf = w.EditableIf
		node.ShowLabel = w.ShowLabel
		node.LabelPos = w.LabelPosition

	case types.WidgetSnippet:
		node.Snippet = &types.SnippetProps{SnippetName: w.SnippetRef}

	case types.WidgetGallery:
		node.DataSource = astDataSourceToModel(w.DataSource)
		gp := &types.GalleryProps{
			DesktopColumns: w.GalleryDesktopColumns,
			TabletColumns:  w.GalleryTabletColumns,
			PhoneColumns:   w.GalleryPhoneColumns,
			Selection:      w.GallerySelection,
		}
		for _, child := range w.Children {
			childNode, _ := astWidgetToNode(ctx, child, moduleName)
			if childNode != nil {
				gp.ContentWidgets = append(gp.ContentWidgets, childNode)
			}
		}
		node.Gallery = gp
		node.Children = nil // gallery body goes into Gallery.ContentWidgets

	case types.WidgetComboBox:
		node.DataSource = astDataSourceToModel(w.DataSource)
		node.EntityAttr = w.Attribute

	case types.WidgetImage:
		node.Image = &types.ImageProps{
			URL:         w.ImageURL,
			AltText:     w.AltText,
			Width:       w.ImageWidth,
			Height:      w.ImageHeight,
			WidthUnit:   w.WidthUnit,
			HeightUnit:  w.HeightUnit,
			DisplayAs:   w.DisplayAs,
			Responsive:  w.Responsive,
			ImageType:   w.ImageType,
			OnClickType: w.OnClickType,
		}
	}

	return node, nil
}

// astWidgetKind maps the AST widget type string to WidgetKind.
func astWidgetKind(astType string) types.WidgetKind {
	switch astType {
	case "container":
		return types.WidgetContainer
	case "scrollview":
		return types.WidgetScrollView
	case "groupbox":
		return types.WidgetGroupBox
	case "layoutgrid":
		return types.WidgetLayoutGrid
	case "row":
		return types.WidgetLayoutRow
	case "column":
		return types.WidgetLayoutCol
	case "tabcontainer":
		return types.WidgetTabContainer
	case "tab", "tabpage":
		return types.WidgetTabPage
	case "dataview":
		return types.WidgetDataView
	case "listview":
		return types.WidgetListView
	case "datagrid":
		return types.WidgetDataGrid
	case "gallery":
		return types.WidgetGallery
	case "button":
		return types.WidgetButton
	case "textbox":
		return types.WidgetTextBox
	case "textarea":
		return types.WidgetTextArea
	case "datepicker":
		return types.WidgetDatePicker
	case "radiobuttons":
		return types.WidgetRadioButtons
	case "checkbox":
		return types.WidgetCheckBox
	case "label":
		return types.WidgetLabel
	case "text":
		return types.WidgetText
	case "title":
		return types.WidgetTitle
	case "snippet":
		return types.WidgetSnippet
	case "combobox":
		return types.WidgetComboBox
	case "image":
		return types.WidgetImage
	case "navigationlist":
		return types.WidgetNavList
	}
	return types.WidgetUnknown
}

// astDataSourceToModel converts an AST data source node to DataSourceDef.
func astDataSourceToModel(ds *ast.DataSourceV3) *types.DataSourceDef {
	if ds == nil {
		return nil
	}
	def := &types.DataSourceDef{}
	switch ds.Kind {
	case "database":
		def.Kind = types.DataSourceDatabase
		def.Entity = ds.Reference
		def.XPathConstraint = ds.XPathConstraint
	case "microflow":
		def.Kind = types.DataSourceMicroflow
		def.Reference = ds.Reference
	case "nanoflow":
		def.Kind = types.DataSourceNanoflow
		def.Reference = ds.Reference
	case "parameter":
		def.Kind = types.DataSourceParameter
		def.Reference = ds.Reference
	case "selection":
		def.Kind = types.DataSourceSelection
		def.Reference = ds.Reference
	}
	return def
}

// astDataSourceEntity returns the entity name from a datasource (for dataview entity context).
func astDataSourceEntity(ctx *ExecContext, ds *ast.DataSourceV3) string {
	if ds == nil {
		return ""
	}
	if ds.Kind == "database" {
		return ds.Reference
	}
	return ""
}
```

- [ ] **Step 2: Wire the write path in cmd_pages_create_v3.go**

After the existing `pb.buildPageV3(s)` call, add a call to `WritePageModel`. The key change is that after creating the gen page (for metadata), we also write the widget tree via the new path.

Find the section after `genPage, err := pb.buildPageV3(s)` succeeds and add:

```go
	// Write widget tree via PageModel IR (roundtrip-stable path)
	pm, err := pageASTToModel(ctx, s)
	if err != nil {
		return mdlerrors.NewBackend("build page model", err)
	}
	_ = pm // used in next step after page is created/updated
```

Then after the page is created/updated (after `CreatePageGen` or `UpdatePageGen` call), add:

```go
	// Inject widget tree from PageModel into the newly written unit.
	newPageID := model.ID(genPage.ID())
	if err := ctx.Backend.WritePageModel(newPageID, pm); err != nil {
		return mdlerrors.NewBackend("write page model", err)
	}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./mdl/executor/...
```

Fix any compilation errors (adjust AST field names to match `ast.WidgetV3` actual struct fields — check `mdl/ast/page_v3.go` for the real field names).

- [ ] **Step 4: Run full roundtrip tests**

```bash
go test -tags integration -run TestRoundtrip_PageModel ./mdl/executor/ -v 2>&1 | tail -40
```

Expected: All pass including stability (second describe == first describe).

- [ ] **Step 5: Run existing page roundtrip tests**

```bash
go test -tags integration -run TestRoundtripPage ./mdl/executor/ -v 2>&1 | tail -20
```

Expected: All pass (no regression in existing tests).

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_pages_ast_to_model.go mdl/executor/cmd_pages_create_v3.go
git commit -m "feat(executor): wire write path through pageASTToModel+WritePageModel"
```

---

## Task 9: Delete Old Builder Files + Update Golden Snapshots

**Files:**
- Delete: `mdl/executor/cmd_pages_builder.go`
- Delete: `mdl/executor/cmd_pages_builder_input.go`
- Delete: `mdl/executor/cmd_pages_builder_input_filters.go`
- Update: `testdata/helpdesk-golden-*/describe-snapshot.mdl`

- [ ] **Step 1: Check builder files for remaining usages**

```bash
grep -rn "pageBuilder\b\|pb\.\|buildPageV3\b\|applyFormWidgetDefaults\b" \
  mdl/executor/cmd_pages_create_v3.go | head -20
```

Only delete builder files when `cmd_pages_create_v3.go` no longer references `pageBuilder` struct or its methods. If still referenced, the write path is not yet fully migrated — complete that migration first.

- [ ] **Step 2: Delete builder files once fully migrated**

```bash
git rm mdl/executor/cmd_pages_builder.go \
       mdl/executor/cmd_pages_builder_input.go \
       mdl/executor/cmd_pages_builder_input_filters.go
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 4: Update golden snapshots**

```bash
make update-helpdesk-golden
```

Expected: `testdata/helpdesk-golden-*/describe-snapshot.mdl` now contains non-empty page bodies.

- [ ] **Step 5: Verify golden snapshot page bodies are non-empty**

```bash
grep -A 5 "create or modify page" testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl | head -40
```

Expected: each page block has widget content between `{` and `}`.

- [ ] **Step 6: Run golden regression tests**

```bash
go test -tags integration -run TestHelpdeskGolden_DescribeSnapshot ./internal/goldenfs/ -v 2>&1 | tail -20
```

Expected: PASS.

- [ ] **Step 7: Run full test suite**

```bash
make test
```

Expected: no new failures.

- [ ] **Step 8: Final commit**

```bash
git add testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl \
        testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl \
        testdata/helpdesk-golden-11.10.0/describe-snapshot.mdl \
        testdata/helpdesk-clean-11.10.0/describe-snapshot.mdl
git commit -m "chore: update golden describe-snapshots with non-empty page bodies

Page widget trees now survive the BSON→PageModel→MDL roundtrip.
Snapshots updated to reflect correct describe output post-refactor."
```

---

## Self-Review Notes

**Spec coverage check:**
- Phase 1 (types) → Task 1 ✓
- Phase 2 (backend interface + mock) → Task 2 ✓
- Phase 3 (TDD red tests) → Task 3 ✓
- Phase 4 (read path) → Tasks 4 + 5 + 6 ✓
- Phase 5 (write path) → Tasks 7 + 8 ✓
- Phase 6 (cleanup + golden) → Task 9 ✓
- `assertPageBodiesNonEmpty` golden guard → Task 3 ✓
- compile-time interface guard → Task 2 (mock) + Task 4 (mpr) ✓

**Potential issues to watch during implementation:**
1. `ast.WidgetV3` field names in Task 8 may differ from what's shown — check `mdl/ast/page_v3.go` before writing `astWidgetToNode()`.
2. `dGetArrayElements` in `page_model.go` strips the int32 prefix — confirm this handles both `bson.A` and `[]any` formats.
3. `resolvePageLayoutName` in Task 6 uses `page.LayoutQualifiedName()` — verify this gen accessor exists in `genPg.Page`, else use only the raw BSON fallback.
4. The `WritePageModel` in Task 7 assumes the page unit already exists (created by `CreatePageGen`) — do not call `WritePageModel` before the unit is created.
