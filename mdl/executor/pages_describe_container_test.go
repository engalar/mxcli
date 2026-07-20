// SPDX-License-Identifier: Apache-2.0

// Tests for issue #219: parseRawWidget missed ScrollContainer / TabControl
// children because they live under CenterRegion.Widgets and TabPages[].Widgets
// respectively, not under the top-level Widgets array that every other
// container uses.

package executor

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRawWidget_ScrollContainerRecursesIntoCenterRegion(t *testing.T) {
	ctx, _ := newMockCtx(t)

	raw := map[string]any{
		"$Type": "Pages$ScrollContainer",
		"Name":  "Scroll1",
		"CenterRegion": map[string]any{
			"Widgets": []any{
				map[string]any{"$Type": "Pages$TextBox", "Name": "InnerText"},
			},
		},
	}

	got := parseRawWidget(ctx, raw)
	if len(got) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(got))
	}
	sc := got[0]
	if sc.Type != "Pages$ScrollContainer" || sc.Name != "Scroll1" {
		t.Errorf("outer widget: type=%q name=%q", sc.Type, sc.Name)
	}
	if len(sc.Children) != 1 {
		t.Fatalf("expected 1 child under ScrollContainer, got %d", len(sc.Children))
	}
	if sc.Children[0].Name != "InnerText" {
		t.Errorf("child name: got %q, want InnerText", sc.Children[0].Name)
	}
}

func TestParseRawWidget_ScrollContainerFallsBackToWidgets(t *testing.T) {
	// Older/legacy BSON shape where children lived directly under Widgets.
	// parseRawWidget must still recurse so existing projects don't regress.
	ctx, _ := newMockCtx(t)

	raw := map[string]any{
		"$Type": "Forms$ScrollContainer",
		"Name":  "LegacyScroll",
		"Widgets": []any{
			map[string]any{"$Type": "Forms$TextBox", "Name": "LegacyText"},
		},
	}

	got := parseRawWidget(ctx, raw)
	if len(got) != 1 || len(got[0].Children) != 1 {
		t.Fatalf("expected 1 widget with 1 child, got %+v", got)
	}
	if got[0].Children[0].Name != "LegacyText" {
		t.Errorf("child name: got %q, want LegacyText", got[0].Children[0].Name)
	}
}

func TestParseRawWidget_TabControlPreservesTabPages(t *testing.T) {
	ctx, _ := newMockCtx(t)

	raw := map[string]any{
		"$Type": "Pages$TabControl",
		"Name":  "Tabs1",
		"TabPages": []any{
			map[string]any{
				"Name": "GeneralTab",
				"Widgets": []any{
					map[string]any{"$Type": "Pages$TextBox", "Name": "GeneralField"},
				},
			},
			map[string]any{
				"Name": "DetailsTab",
				"Widgets": []any{
					map[string]any{"$Type": "Pages$TextBox", "Name": "DetailsField"},
					map[string]any{"$Type": "Pages$TextBox", "Name": "DetailsNote"},
				},
			},
		},
	}

	got := parseRawWidget(ctx, raw)
	if len(got) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(got))
	}
	tc := got[0]
	if tc.Type != "Pages$TabControl" || tc.Name != "Tabs1" {
		t.Errorf("outer widget: type=%q name=%q", tc.Type, tc.Name)
	}
	if len(tc.Children) != 2 {
		t.Fatalf("expected 2 TabPage children, got %d", len(tc.Children))
	}

	for i, expectedName := range []string{"GeneralTab", "DetailsTab"} {
		if tc.Children[i].Type != "Pages$TabPage" {
			t.Errorf("tab %d type: got %q, want Pages$TabPage", i, tc.Children[i].Type)
		}
		if tc.Children[i].Name != expectedName {
			t.Errorf("tab %d name: got %q, want %q", i, tc.Children[i].Name, expectedName)
		}
	}

	if len(tc.Children[0].Children) != 1 || tc.Children[0].Children[0].Name != "GeneralField" {
		t.Errorf("GeneralTab children: %+v", tc.Children[0].Children)
	}
	if len(tc.Children[1].Children) != 2 {
		t.Fatalf("DetailsTab expected 2 children, got %d", len(tc.Children[1].Children))
	}
}

func TestOutputWidgetMDLV3_TabControlEmitsTabPageStructure(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{
		Output: &buf,
	}

	tab := rawWidget{
		Type: "Pages$TabControl",
		Name: "Tabs1",
		Children: []rawWidget{
			{
				Type:       "Pages$TabPage",
				Name:       "GeneralTab",
				TabCaption: "General",
				Children: []rawWidget{
					{Type: "Pages$TextBox", Name: "GeneralField"},
				},
			},
		},
	}
	outputWidgetMDLV3(ctx, tab, 0)

	out := buf.String()
	for _, want := range []string{
		"tabcontainer Tabs1",
		"tabpage GeneralTab",
		"caption: 'General'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestOutputWidgetMDLV3_ScrollContainerEmitsHeader(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{
		Output: &buf,
	}

	sc := rawWidget{
		Type: "Pages$ScrollContainer",
		Name: "Scroll1",
		Children: []rawWidget{
			{Type: "Pages$TextBox", Name: "InnerText"},
		},
	}
	outputWidgetMDLV3(ctx, sc, 0)

	out := buf.String()
	if !strings.Contains(out, "scrollcontainer Scroll1") {
		t.Errorf("expected 'scrollcontainer Scroll1' in output, got:\n%s", out)
	}
}
