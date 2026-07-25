package rules

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
)

// mpr018Reader implements linter.LintReader for MPR018 tests with fake raw unit data.
type mpr018Reader struct {
	linter.LintReader
	rawUnits map[model.ID]map[string]any
}

func (r *mpr018Reader) GetRawUnit(id model.ID) (map[string]any, error) {
	if r.rawUnits == nil {
		return nil, nil
	}
	return r.rawUnits[id], nil
}

func (r *mpr018Reader) ListAllUnitIDs() ([]string, error) {
	if r.rawUnits == nil {
		return nil, nil
	}
	ids := make([]string, 0, len(r.rawUnits))
	for id := range r.rawUnits {
		ids = append(ids, string(id))
	}
	return ids, nil
}

// TestFindPluggableWidgets_CountMismatch verifies detection.
func TestFindPluggableWidgets_CountMismatch(t *testing.T) {
	pageMap := map[string]any{
		"FormCall": map[string]any{
			"Arguments": []any{
				int32(2),
				map[string]any{
					"Widgets": []any{
						int32(2),
						map[string]any{
							"$ID":   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							"$Type": "CustomWidgets$CustomWidget",
							"Name":  "testDG",
							"Type": map[string]any{
								"WidgetId": "com.mendix.widget.web.datagrid.Datagrid",
								"ObjectType": map[string]any{
									"PropertyTypes": []any{
										int32(2),
										map[string]any{
											"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
											"PropertyKey": "datasource",
										},
										map[string]any{
											"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x11},
											"PropertyKey": "columns",
										},
										map[string]any{
											"$ID":         []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x12},
											"PropertyKey": "label",
										},
									},
								},
							},
							"Object": map[string]any{
								"Properties": []any{
									int32(2),
									map[string]any{
										"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10},
										"Value":       map[string]any{"DataSource": nil},
									},
									map[string]any{
										"TypePointer": []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x11},
										"Value":       map[string]any{"Objects": nil},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	widgets := findPluggableWidgets(pageMap)
	if len(widgets) != 1 {
		t.Fatalf("got %d widgets, want 1", len(widgets))
	}
	w := widgets[0]
	if w.TypeCount != 3 {
		t.Errorf("TypeCount = %d, want 3", w.TypeCount)
	}
	if w.ObjCount != 2 {
		t.Errorf("ObjCount = %d, want 2", w.ObjCount)
	}
	if w.Name != "testDG" {
		t.Errorf("Name = %q, want testDG", w.Name)
	}
}

// TestFindPluggableWidgets_CountMatch verifies widgets with matching
// counts are still returned by the finder (the count filter is in Check).
func TestFindPluggableWidgets_CountMatch(t *testing.T) {
	pageMap := map[string]any{
		"FormCall": map[string]any{
			"Arguments": []any{
				int32(2),
				map[string]any{
					"Widgets": []any{
						int32(2),
						map[string]any{
							"$ID":   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
							"$Type": "CustomWidgets$CustomWidget",
							"Name":  "okWidget",
							"Type": map[string]any{
								"WidgetId": "test.widget.Ok",
								"ObjectType": map[string]any{
									"PropertyTypes": []any{int32(2), map[string]any{"PropertyKey": "p1"}},
								},
							},
							"Object": map[string]any{
								"Properties": []any{int32(2), map[string]any{"Value": map[string]any{}}},
							},
						},
					},
				},
			},
		},
	}
	// The finder returns all pluggable widgets regardless of count match
	widgets := findPluggableWidgets(pageMap)
	if len(widgets) != 1 {
		t.Fatalf("got %d widgets, want 1 (finder returns all pluggable widgets)", len(widgets))
	}
	if widgets[0].TypeCount != widgets[0].ObjCount {
		t.Errorf("TypeCount=%d != ObjCount=%d (should match)", widgets[0].TypeCount, widgets[0].ObjCount)
	}
}

// TestCheckMPR018_SkipsMatching verifies the rule Check reports only mismatched widgets.
func TestCheckMPR018_SkipsMatching(t *testing.T) {
	pageID := model.ID("page-1")
	g := &mock.MockProjectGraph{
		PagesFunc: func(module string) []graphcatalog.PageNode {
			return []graphcatalog.PageNode{
				{ID: string(pageID), Name: "MyPage", QualifiedName: "Mod.MyPage", Module: "Mod"},
			}
		},
		SnippetsFunc: func(module string) []graphcatalog.SnippetNode {
			return nil
		},
	}
	reader := &mpr018Reader{
		rawUnits: map[model.ID]map[string]any{
			pageID: {
				"$Type": "Forms$Page",
				"FormCall": map[string]any{
					"Arguments": []any{
						int32(2),
						map[string]any{
							"Widgets": []any{
								int32(2),
								map[string]any{
									"$ID":   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									"$Type": "CustomWidgets$CustomWidget",
									"Name":  "ok",
									"Type": map[string]any{
										"WidgetId": "test.widget",
										"ObjectType": map[string]any{
											"PropertyTypes": []any{int32(2), map[string]any{"PropertyKey": "p1"}},
										},
									},
									"Object": map[string]any{
										"Properties": []any{int32(2), map[string]any{"Value": map[string]any{}}},
									},
								},
								map[string]any{
									"$ID":   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
									"$Type": "CustomWidgets$CustomWidget",
									"Name":  "bad",
									"Type": map[string]any{
										"WidgetId": "test.widget.Bad",
										"ObjectType": map[string]any{
											"PropertyTypes": []any{int32(2), map[string]any{"PropertyKey": "a"}, map[string]any{"PropertyKey": "b"}},
										},
									},
									"Object": map[string]any{
										"Properties": []any{int32(2), map[string]any{"Value": map[string]any{}}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := newGraphContext(g, reader)
	rule := NewWidgetDefinitionRule()
	violations := rule.Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	if !strings.Contains(violations[0].Message, "bad") {
		t.Errorf("violation message = %q, should mention 'bad'", violations[0].Message)
	}
	extra := violations[0].Extra.(map[string]any)
	if extra["unitID"] != string(pageID) {
		t.Errorf("extra.unitID = %v, want %s", extra["unitID"], string(pageID))
	}
}
