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

	verify(t, described)

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
create or modify microflow `+testModule+`.ACT_Noop () returns Nothing { return; }
create or modify page `+page+` (
  title: 'Container Test',
  layout: Atlas_Core.Atlas_Default
) {
  container mainBox (class: 'spacing-outer') {
    actionbutton btn (caption: 'Click Me', action: microflow `+testModule+`.ACT_Noop)
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "container") {
			t.Errorf("expected 'container' in describe output, got:\n%s", described)
		}
		if !strings.Contains(described, "actionbutton") {
			t.Errorf("expected 'actionbutton' in describe output, got:\n%s", described)
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
  dataview dv (DataSource: $Item) {
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
    tabpage tab1 (caption: 'First Tab') {
      statictext lbl (Content: 'Hello')
    }
    tabpage tab2 (caption: 'Second Tab') {
      statictext lbl2 (Content: 'World')
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
    statictext lbl (Content: 'Content')
  }
};`, page, func(t *testing.T, described string) {
		t.Helper()
		if !strings.Contains(described, "groupbox") {
			t.Errorf("expected 'groupbox' in describe output")
		}
		// Legacy describe maps "YesInitiallyExpanded" → "YesExpanded"; IR
		// renderer keeps the literal storage value. Either form is acceptable.
		if !strings.Contains(described, "YesInitiallyExpanded") &&
			!strings.Contains(described, "YesExpanded") {
			t.Errorf("expected collapsible setting in describe output, got:\n%s", described)
		}
	})
}
