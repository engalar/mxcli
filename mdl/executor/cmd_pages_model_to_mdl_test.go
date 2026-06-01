// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// TestPageModelToMDL_Header verifies the header line includes title and layout.
func TestPageModelToMDL_Header(t *testing.T) {
	pm := &types.PageModel{
		Title:  "Container Test",
		Layout: "Atlas_Core.Atlas_Default",
	}
	var buf bytes.Buffer
	pageModelToMDL(&buf, pm, "Mod", "MyPage")
	out := buf.String()
	if !strings.Contains(out, "create or modify page Mod.MyPage (") {
		t.Errorf("missing header: %s", out)
	}
	if !strings.Contains(out, "title: 'Container Test'") {
		t.Errorf("missing title: %s", out)
	}
	if !strings.Contains(out, "layout: Atlas_Core.Atlas_Default") {
		t.Errorf("missing layout: %s", out)
	}
}

// TestPageModelToMDL_Container_With_Button verifies a container holding one
// button renders the keyword 'container', the button keyword, the caption,
// and the indentation depth.
func TestPageModelToMDL_Container_With_Button(t *testing.T) {
	pm := &types.PageModel{
		Title: "T",
		Widgets: []*types.WidgetNode{
			{
				Kind: types.WidgetContainer,
				Name: "mainBox",
				Children: []*types.WidgetNode{
					{Kind: types.WidgetButton, Name: "btn", Caption: "Click Me",
						OnClick: "Mod.ACT_Noop"},
				},
			},
		},
	}
	var buf bytes.Buffer
	pageModelToMDL(&buf, pm, "Mod", "P")
	out := buf.String()
	if !strings.Contains(out, "container mainBox {") {
		t.Errorf("missing container line: %s", out)
	}
	if !strings.Contains(out, "    actionbutton btn (") {
		t.Errorf("expected 4-space indented actionbutton under container; got: %s", out)
	}
	if !strings.Contains(out, "caption: 'Click Me'") {
		t.Errorf("missing button caption: %s", out)
	}
	if !strings.Contains(out, "action: microflow Mod.ACT_Noop") {
		t.Errorf("missing button action: %s", out)
	}
}

// TestPageModelToMDL_EscapeQuotes verifies that single quotes inside strings
// are doubled (MDL escape convention).
func TestPageModelToMDL_EscapeQuotes(t *testing.T) {
	pm := &types.PageModel{
		Title: "it's complicated",
		Widgets: []*types.WidgetNode{
			{Kind: types.WidgetLabel, Name: "lbl", Caption: "don't stop"},
		},
	}
	var buf bytes.Buffer
	pageModelToMDL(&buf, pm, "M", "P")
	out := buf.String()
	if !strings.Contains(out, "title: 'it''s complicated'") {
		t.Errorf("title quote not doubled: %s", out)
	}
	if !strings.Contains(out, "Content: 'don''t stop'") {
		t.Errorf("Content quote not doubled (Label now renders as statictext Content): %s", out)
	}
}
