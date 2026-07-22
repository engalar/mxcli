// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// newPageBuilderWithNanoflowStub returns a pageBuilder primed with a
// createdNanoflows entry so buildDataSourceV3 can resolve the nanoflow
// without hitting the backend.
func newPageBuilderWithNanoflowStub(qualifiedName string) *pageBuilder {
	mb := &mock.MockBackend{}
	mb.SerializeGenElemToOpaqueFunc = func(elem element.Element) backend.OpaqueWidget {
		enc := codec.Encoder{}
		raw, err := enc.Encode(elem)
		if err != nil {
			return nil
		}
		return bson.Raw(raw)
	}
	return &pageBuilder{
		execCache: &executorCache{
			sessionTracker: sessionTracker{
				createdNanoflows: map[string]*createdNanoflowInfo{
					qualifiedName: {ID: model.ID("00000000-0000-0000-0000-000000000003"), Name: "NF_Search", ModuleName: "HD"},
				},
			},
		},
		paramEntityNames: map[string]string{"$Search": "HD.TicketSearch"},
		widgetBackend:    mb,
	}
}

// buildTestWidgetV3WithDesignProp creates a WidgetV3 with a single design property.
func buildTestWidgetV3WithDesignProp(key, value string) *ast.WidgetV3 {
	w := &ast.WidgetV3{Properties: map[string]interface{}{}}
	w.Properties["DesignProperties"] = []ast.DesignPropertyEntryV3{{Key: key, Value: value}}
	return w
}

// TestApplyWidgetAppearanceGenDesignPropertiesWrappedInDesignPropertyValue
// verifies that design properties are serialised as:
//
//	Appearance.DesignProperties[] → DesignPropertyValue → Value: OptionDesignPropertyValue
//
// NOT as:
//
//	Appearance.DesignProperties[] → OptionDesignPropertyValue  (missing wrapper)
//
// The missing wrapper causes Studio Pro to crash with:
//
//	"OptionDesignPropertyValue does not contain a constructor with a
//	 parameter of type Appearance"
func TestApplyWidgetAppearanceGenDesignPropertiesWrappedInDesignPropertyValue(t *testing.T) {
	w := buildTestWidgetV3WithDesignProp("spacing", "large")
	widget := genPg.NewDivContainer()

	applyWidgetAppearanceGen(widget, w, nil)

	app, ok := widget.Appearance().(*genPg.Appearance)
	if !ok || app == nil {
		t.Fatal("Appearance must be set on widget after applyWidgetAppearanceGen")
	}

	items := app.DesignPropertiesItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 DesignProperties item, got %d", len(items))
	}

	// The item MUST be a DesignPropertyValue wrapper, not OptionDesignPropertyValue directly.
	dpv, ok := items[0].(*genPg.DesignPropertyValue)
	if !ok {
		t.Fatalf("DesignProperties[0] = %T, want *DesignPropertyValue — "+
			"OptionDesignPropertyValue must be wrapped in DesignPropertyValue, "+
			"otherwise Studio Pro crashes with: 'OptionDesignPropertyValue does not "+
			"contain a constructor with a parameter of type Appearance'", items[0])
	}

	if dpv.Key() != "spacing" {
		t.Errorf("DesignPropertyValue.Key = %q, want %q", dpv.Key(), "spacing")
	}

	val, ok := dpv.Value().(*genPg.OptionDesignPropertyValue)
	if !ok {
		t.Fatalf("DesignPropertyValue.Value = %T, want *OptionDesignPropertyValue", dpv.Value())
	}
	if val.Option() != "large" {
		t.Errorf("OptionDesignPropertyValue.Option = %q, want %q", val.Option(), "large")
	}
}

// TestApplyWidgetAppearanceGenToggleOnWrappedInDesignPropertyValue verifies that
// toggle-on design properties (value="on") are also wrapped in DesignPropertyValue.
func TestApplyWidgetAppearanceGenToggleOnWrappedInDesignPropertyValue(t *testing.T) {
	w := buildTestWidgetV3WithDesignProp("bold", "on")
	widget := genPg.NewDivContainer()
	applyWidgetAppearanceGen(widget, w, nil)

	app, ok := widget.Appearance().(*genPg.Appearance)
	if !ok || app == nil {
		t.Fatal("Appearance must be set")
	}
	items := app.DesignPropertiesItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	dpv, ok := items[0].(*genPg.DesignPropertyValue)
	if !ok {
		t.Fatalf("DesignProperties[0] = %T, want *DesignPropertyValue (toggle-on must also be wrapped)", items[0])
	}
	if dpv.Key() != "bold" {
		t.Errorf("Key = %q, want bold", dpv.Key())
	}
	if _, ok := dpv.Value().(*genPg.ToggleDesignPropertyValue); !ok {
		t.Fatalf("DesignPropertyValue.Value = %T, want *ToggleDesignPropertyValue", dpv.Value())
	}
}

// TestBuildDynamicTextV3_EmptyContentUsesSafeDefault verifies CE0720 fix:
// when a DYNAMICTEXT widget has an empty content string, buildDynamicTextV3
// must NOT produce a template with text "{1}" and zero parameters — that
// combination causes Studio Pro to raise CE0720. A single space " " is the
// safe fallback (documented in CLAUDE.md).
func TestBuildDynamicTextV3_EmptyContentUsesSafeDefault(t *testing.T) {
	pb := &pageBuilder{
		widgetScope:      map[string]model.ID{},
		paramEntityNames: map[string]string{},
	}
	w := &ast.WidgetV3{
		Type:       "dynamictext",
		Name:       "txt1",
		Properties: map[string]interface{}{
			// no Content property → GetContent() returns ""
		},
	}
	result, err := pb.buildDynamicTextV3(w)
	if err != nil {
		t.Fatalf("buildDynamicTextV3 failed: %v", err)
	}
	dt, ok := result.(*genPg.DynamicText)
	if !ok {
		t.Fatalf("result = %T, want *DynamicText", result)
	}
	ct, ok := dt.Content().(*genPg.ClientTemplate)
	if !ok {
		t.Fatalf("Content = %T, want *ClientTemplate", dt.Content())
	}
	// CE0720 fix: template text must NOT be "{1}" when there are no parameters.
	if tmpl, ok := ct.Template().(*genTexts.Text); ok {
		for _, item := range tmpl.TranslationsItems() {
			if tr, ok := item.(*genTexts.Translation); ok {
				if tr.Text() == "{1}" && len(ct.ParametersItems()) == 0 {
					t.Fatalf("CE0720: template text is {1} but there are 0 parameters — Studio Pro will raise CE0720")
				}
			}
		}
	}
	// Parameters list must be empty (no auto-generated params for empty content).
	if len(ct.ParametersItems()) != 0 {
		t.Fatalf("expected 0 parameters for empty content, got %d", len(ct.ParametersItems()))
	}
}

// newPageBuilderWithMicroflowStub returns a pageBuilder primed so that
// resolveMicroflow finds qualifiedName via execCache.createdMicroflows,
// avoiding any backend round-trip. widgetBackend is wired with a codec-based
// SerializeGenElemToOpaque so genClientActionToBsonD can round-trip through gen.
func newPageBuilderWithMicroflowStub(qualifiedName string) *pageBuilder {
	mb := &mock.MockBackend{}
	mb.SerializeGenElemToOpaqueFunc = func(elem element.Element) backend.OpaqueWidget {
		enc := codec.Encoder{}
		raw, err := enc.Encode(elem)
		if err != nil {
			return nil
		}
		return bson.Raw(raw)
	}
	return &pageBuilder{
		execCache: &executorCache{
			sessionTracker: sessionTracker{
				createdMicroflows: map[string]*createdMicroflowInfo{
					qualifiedName: {ID: model.ID("00000000-0000-0000-0000-000000000001"), Name: "SomeMF", ModuleName: "MyMod"},
				},
			},
		},
		widgetBackend: mb,
	}
}

// TestGenClientActionToBsonD_MicroflowClosePage verifies that a microflow action
// with ClosePage=true emits ClosePage:true in BSON via the gen codec path.
// Regression guard for the `action: microflow M.F (...) close_page` syntax.
func TestGenClientActionToBsonD_MicroflowClosePage(t *testing.T) {
	pb := newPageBuilderWithMicroflowStub("MyMod.SomeMF")
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: true,
	}
	got, err := pb.genClientActionToBsonD(action)
	if err != nil {
		t.Fatalf("genClientActionToBsonD: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && v {
				return // pass
			}
			t.Errorf("ClosePage = %v, want true", kv.Value)
			return
		}
	}
	t.Error("ClosePage key not found in BSON for microflow action with ClosePage=true")
}

// TestGenClientActionToBsonD_MicroflowNoClosePage verifies that a microflow
// action without close_page does not emit ClosePage:true (Mendix defaults to false
// when the field is absent, so we do not inject it for the false case).
func TestGenClientActionToBsonD_MicroflowNoClosePage(t *testing.T) {
	pb := newPageBuilderWithMicroflowStub("MyMod.SomeMF")
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: false,
	}
	got, err := pb.genClientActionToBsonD(action)
	if err != nil {
		t.Fatalf("genClientActionToBsonD: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && v {
				t.Errorf("ClosePage = true in BSON for action without close_page")
			}
			return
		}
	}
	// ClosePage absent is also correct — Mendix treats absent as false.
}

// TestDataViewSelectionDatasource_StoresListenTargetSource verifies that
// datasource: selection widgetName on a DataView is stored as Forms$ListenTargetSource
// in BSON, enabling CE3637-free cross-widget selection listening.
func TestDataViewSelectionDatasource_StoresListenTargetSource(t *testing.T) {
	pb := newPageBuilderWithMicroflowStub("ignored")
	pb.widgetScope = map[string]model.ID{}
	pb.paramEntityNames = map[string]string{}
	// Register a fake gallery widget so the selection datasource can resolve it
	pb.widgetScope["artGallery"] = model.ID("00000000-0000-0000-0000-000000000002")
	pb.paramEntityNames["artGallery"] = "KB.Article"

	ds := &ast.DataSourceV3{
		Type:      "selection",
		Reference: "artGallery",
	}

	elem, entityName, err := pb.buildDataSourceV3(ds)
	if err != nil {
		t.Fatalf("buildDataSourceV3 selection: %v", err)
	}
	if elem == nil {
		t.Fatal("buildDataSourceV3 selection: returned nil element")
	}
	if entityName != "KB.Article" {
		t.Errorf("entityName = %q, want KB.Article", entityName)
	}

	// Serialise to BSON via the codec path (same as when page is written to MPR)
	enc := codec.Encoder{}
	raw, err := enc.Encode(elem)
	if err != nil {
		t.Fatalf("encode ListenTargetSource: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var gotType, gotTarget string
	for _, kv := range doc {
		switch kv.Key {
		case "$Type":
			if s, ok := kv.Value.(string); ok {
				gotType = s
			}
		case "ListenTarget":
			if s, ok := kv.Value.(string); ok {
				gotTarget = s
			}
		}
	}
	if gotType != "Forms$ListenTargetSource" {
		t.Errorf("$Type = %q, want Forms$ListenTargetSource", gotType)
	}
	if gotTarget != "artGallery" {
		t.Errorf("ListenTarget = %q, want artGallery", gotTarget)
	}
}

// TestApplyWidgetAppearanceGen_SetsDefaultAppearanceWhenNoStyling verifies that
// a widget which supports SetAppearance always gets a Forms$Appearance even when
// no class, style, or designproperties are specified in MDL.
// Studio Pro 11.6.6 requires Forms$Appearance on every widget — without it the
// widget is invisible (see newDefaultAppearance comment in cmd_pages_builder_input.go).
func TestApplyWidgetAppearanceGen_SetsDefaultAppearanceWhenNoStyling(t *testing.T) {
	for _, tc := range []struct {
		name   string
		widget func() element.Element
	}{
		{"DataView", func() element.Element { return genPg.NewDataView() }},
		{"DivContainer", func() element.Element { return genPg.NewDivContainer() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &ast.WidgetV3{Properties: map[string]interface{}{}}
			widget := tc.widget()

			applyWidgetAppearanceGen(widget, w, nil)

			type appearanceGetter interface{ Appearance() element.Element }
			ag, ok := widget.(appearanceGetter)
			if !ok {
				t.Fatalf("%s does not implement Appearance()", tc.name)
			}
			if ag.Appearance() == nil {
				t.Errorf("%s: Appearance must not be nil even when no class/style/designprops specified", tc.name)
			}
			app, ok := ag.Appearance().(*genPg.Appearance)
			if !ok {
				t.Fatalf("%s: Appearance = %T, want *genPg.Appearance", tc.name, ag.Appearance())
			}
			if app.Class() != "" {
				t.Errorf("%s: default Appearance.Class must be empty, got %q", tc.name, app.Class())
			}
		})
	}
}

// TestApplyWidgetAppearanceGen_PreservesExistingAppearance verifies that
// a widget which already has Appearance set (e.g. by applyFormWidgetDefaults)
// does not have its Appearance replaced when no styling is specified in MDL.
func TestApplyWidgetAppearanceGen_PreservesExistingAppearance(t *testing.T) {
	tb := genPg.NewTextBox()
	existing := newDefaultAppearance()
	tb.SetAppearance(existing)

	w := &ast.WidgetV3{Properties: map[string]interface{}{}}
	applyWidgetAppearanceGen(tb, w, nil)

	if tb.Appearance() != existing {
		t.Error("pre-existing Appearance must not be replaced when no class/style/designprops specified")
	}
}

// TestNanoflowDatasourceResolvesViaCache verifies that buildDataSourceV3
// resolves a nanoflow datasource reference using execCache.createdNanoflows,
// enabling session-local resolution without a backend round-trip.
func TestNanoflowDatasourceResolvesViaCache(t *testing.T) {
	pb := newPageBuilderWithNanoflowStub("HD.NF_TicketSearch_Apply")
	pb.widgetScope = map[string]model.ID{}

	ds := &ast.DataSourceV3{
		Type:      "nanoflow",
		Reference: "HD.NF_TicketSearch_Apply",
		Args:      []ast.FlowArgV3{{Name: "Search", Value: "$Search"}},
	}

	elem, _, err := pb.buildDataSourceV3(ds)
	if err != nil {
		t.Fatalf("buildDataSourceV3 nanoflow: %v", err)
	}

	ns, ok := elem.(*genPg.NanoflowSource)
	if !ok {
		t.Fatalf("buildDataSourceV3 nanoflow: got %T, want *NanoflowSource", elem)
	}
	if ns.NanoflowQualifiedName() != "HD.NF_TicketSearch_Apply" {
		t.Errorf("NanoflowQualifiedName = %q", ns.NanoflowQualifiedName())
	}
}

// TestBuildTabPageV3_CaptionIsTextsText verifies that a tabpage caption is stored
// as Texts$Text (not Forms$ClientTemplate). Using ClientTemplate causes:
//
//	StorageLoadException: "ClientTemplate cannot be converted to Text"
//
// Root cause: genSimpleLabel wraps in ClientTemplate; genSimpleText gives Texts$Text.
// Same bug as page.Title (fixed separately with a comment in buildPageV3).
func TestBuildTabPageV3_CaptionIsTextsText(t *testing.T) {
	pb := &pageBuilder{
		execCache:        &executorCache{},
		paramEntityNames: map[string]string{},
		widgetScope:      map[string]model.ID{},
	}

	w := &ast.WidgetV3{
		Type:       "tabpage",
		Name:       "tabActive",
		Properties: map[string]interface{}{"caption": "Active Accounts"},
		Children:   []*ast.WidgetV3{},
	}

	elem, err := pb.buildTabPageV3(w)
	if err != nil {
		t.Fatalf("buildTabPageV3: %v", err)
	}

	tp, ok := elem.(*genPg.TabPage)
	if !ok {
		t.Fatalf("got %T, want *genPg.TabPage", elem)
	}

	cap := tp.Caption()
	if cap == nil {
		t.Fatal("TabPage Caption is nil — expected Texts$Text")
	}
	// Must be *genTexts.Text (Texts$Text), NOT *genPg.ClientTemplate (Forms$ClientTemplate).
	if _, isText := cap.(*genTexts.Text); !isText {
		t.Errorf("TabPage Caption is %T — must be *genTexts.Text, not ClientTemplate", cap)
	}
}

// ─────────────────────────────────────────────────────────────
// Bug B: resolveAttributePath with empty entityContext
// ─────────────────────────────────────────────────────────────

func TestResolveAttributePath_EmptyContextReturnsEmpty(t *testing.T) {
	pb := &pageBuilder{}
	got := pb.resolveAttributePath("DataAttr")
	if got != "" {
		t.Fatalf("resolveAttributePath with empty entityContext: got %q, want empty string", got)
	}
}

func TestResolveAttributePath_WithContextQualifies(t *testing.T) {
	pb := &pageBuilder{entityContext: "Module.Entity"}
	got := pb.resolveAttributePath("DataAttr")
	want := "Module.Entity.DataAttr"
	if got != want {
		t.Fatalf("resolveAttributePath: got %q, want %q", got, want)
	}
}

func TestResolveAttributePath_AlreadyQualifiedPassthrough(t *testing.T) {
	pb := &pageBuilder{}
	got := pb.resolveAttributePath("Module.Entity.DataAttr")
	want := "Module.Entity.DataAttr"
	if got != want {
		t.Fatalf("resolveAttributePath: got %q, want %q", got, want)
	}
}

// ─────────────────────────────────────────────────────────────
// Bug C: getMicroflowReturnEntityName — two scenarios:
//   a) Cache hit with empty ReturnEntityName (session-created microflow
//      with bare return entity) — must fall through to MicroflowReturnType
//   b) No cache, repo path only — also must use MicroflowReturnType
// ─────────────────────────────────────────────────────────────

func TestGetMicroflowReturnEntityName_CacheEmptyFallsThrough(t *testing.T) {
	// Reproduces the real bug: cache has entry with ReturnEntityName==""
	// (because bare return entity wasn't resolved at creation time).
	// The function must fall through to MicroflowReturnType() and return
	// the qualified entity name, NOT return "" immediately.
	mf := genMf.NewMicroflow()
	mf.SetID(element.ID("mf-001"))
	mf.SetName("ACT_GetOrders")

	objType := genDT.NewObjectType()
	objType.SetEntityQualifiedName("MyModule.Order")
	mf.SetMicroflowReturnType(objType)

	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc:          func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}

	pb := &pageBuilder{
		microflowsRepo: mfRepo,
		execCache: &executorCache{
			sessionTracker: sessionTracker{
				createdMicroflows: map[string]*createdMicroflowInfo{
					"MyModule.ACT_GetOrders": {
						ReturnEntityName: "", // ← bare return, not resolved
					},
				},
			},
		},
	}
	pb.execCache.hierarchy = h

	got := pb.getMicroflowReturnEntityName("MyModule.ACT_GetOrders")
	want := "MyModule.Order"
	if got != want {
		t.Fatalf("getMicroflowReturnEntityName with empty cache ReturnEntityName: got %q, want %q", got, want)
	}
}

func TestGetMicroflowReturnEntityName_MicroflowReturnTypeFallback(t *testing.T) {
	mf := genMf.NewMicroflow()
	mf.SetID(element.ID("mf-001"))
	mf.SetName("ACT_GetOrders")

	objType := genDT.NewObjectType()
	objType.SetEntityQualifiedName("MyModule.Order")
	mf.SetMicroflowReturnType(objType)

	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc:          func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}

	pb := &pageBuilder{
		microflowsRepo: mfRepo,
		execCache:      &executorCache{},
	}
	pb.execCache.hierarchy = h

	got := pb.getMicroflowReturnEntityName("MyModule.ACT_GetOrders")
	want := "MyModule.Order"
	if got != want {
		t.Fatalf("getMicroflowReturnEntityName: got %q, want %q", got, want)
	}
}

// TestResolveEntity_FromCache verifies that resolveEntity finds entities
// that were created in the same session via the createdEntities cache.
// Regression guard for "entity not found: HD.PasswordForm" error.
func TestResolveEntity_FromCache(t *testing.T) {
	mod := mkModule("HD")

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListDomainModelsGenFunc: func() ([]*genDm.DomainModel, error) {
			return nil, nil
		},
	}
	entityID := model.ID("ent-session-1")
	cache := &executorCache{
		sessionTracker: sessionTracker{
			createdEntities: map[string]*createdEntityInfo{
				"HD.PasswordForm": {
					ID:         entityID,
					Name:       "PasswordForm",
					ModuleName: "HD",
				},
			},
		},
	}
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	pb := &pageBuilder{
		domainModelReader: mb,
		execCache:         cache,
	}
	pb.execCache.hierarchy = h

	id, err := pb.resolveEntity(ast.QualifiedName{Module: "HD", Name: "PasswordForm"})
	if err != nil {
		t.Fatalf("resolveEntity returned error: %v", err)
	}
	if id != entityID {
		t.Fatalf("expected ID %q, got %q", entityID, id)
	}
}

// TestBuildClientActionV3_NanoflowParamQualifiedName is a regression guard for the
// page-actionbutton nanoflow parameter-mapping crash: a nanoflow action with a
// parameter must store a FULLY-QUALIFIED ParameterQualifiedName
// ("Module.NanoflowName.ParamName"), not a bare param name. A bare name leaves
// Mendix unable to resolve the Parameter reference and crashes `mx check` with
// "Parameter property ... null". (CE0115 arg matching is a separate, still-open
// issue — see TODO in cmd_pages_builder_v3.go.)
func TestBuildClientActionV3_NanoflowParamQualifiedName(t *testing.T) {
	pb := newPageBuilderWithNanoflowStub("MyMod.SomeNF")
	action := &ast.ActionV3{
		Type:   "nanoflow",
		Target: "MyMod.SomeNF",
		Args:   []ast.FlowArgV3{{Name: "Order", Value: "$currentObject"}},
	}
	elem, err := pb.buildClientActionV3(action)
	if err != nil {
		t.Fatalf("buildClientActionV3: %v", err)
	}
	nfAct, ok := elem.(*genPg.CallNanoflowClientAction)
	if !ok {
		t.Fatalf("expected *CallNanoflowClientAction, got %T", elem)
	}
	mappings := nfAct.ParameterMappingsItems()
	if len(mappings) != 1 {
		t.Fatalf("expected 1 parameter mapping, got %d", len(mappings))
	}
	nm, ok := mappings[0].(*genPg.NanoflowParameterMapping)
	if !ok {
		t.Fatalf("expected *NanoflowParameterMapping, got %T", mappings[0])
	}
	got := nm.ParameterQualifiedName()
	want := "MyMod.SomeNF.Order"
	if got != want {
		t.Errorf("ParameterQualifiedName = %q, want %q (bare name regresses to the mx check crash)", got, want)
	}
}

// TestBuildClientActionV3_NanoflowInvalidAttrPath verifies that attribute paths
// using a module prefix for system attributes (e.g. $currentObject/System.changedDate)
// are rejected as invalid syntax during action building.
func TestBuildClientActionV3_NanoflowInvalidAttrPath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{
			name:    "system attribute with module prefix",
			value:   "$currentObject/System.changedDate",
			wantErr: "System.changedDate",
		},
		{
			name:    "valid attribute path is accepted",
			value:   "$currentObject",
			wantErr: "",
		},
		{
			name:    "valid parameter variable is accepted",
			value:   "$Ticket/Subject",
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := newPageBuilderWithNanoflowStub("MyMod.SomeNF")
			pb.paramEntityNames = map[string]string{"Ticket": "HD.Ticket"}
			action := &ast.ActionV3{
				Type:   "nanoflow",
				Target: "MyMod.SomeNF",
				Args:   []ast.FlowArgV3{{Name: "Order", Value: tt.value}},
			}
			_, err := pb.buildClientActionV3(action)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want containing %q", err.Error(), tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
