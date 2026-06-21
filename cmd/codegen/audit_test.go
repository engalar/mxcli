package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func makeGenDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// ── collectRegisteredFields ───────────────────────────────────────────────────

func TestCollectRegisteredFields_extractsFieldNamesPerType(t *testing.T) {
	genDir := makeGenDir(t, map[string]string{
		"pages/types.go": `package pages
func initTabContainer() *TabContainer {
	o := &TabContainer{}
	o.SetTypeName("Forms$TabControl")
	o.name = property.NewPrimitive[string]("Name", property.DecodeString)
	o.name.Bind(&o.Base, 0)
	o.tabPages = property.NewPartList[element.Element]("TabPages")
	o.tabPages.Bind(&o.Base, 1)
	o.defaultPage = property.NewByIdRef[element.Element]("DefaultPage")
	o.defaultPage.Bind(&o.Base, 2)
	return o
}
`,
	})

	fields := collectRegisteredFields(genDir)

	got, ok := fields["Forms$TabControl"]
	if !ok {
		t.Fatal("Forms$TabControl not found in result")
	}
	for _, want := range []string{"Name", "TabPages", "DefaultPage"} {
		if !got[want] {
			t.Errorf("missing field %q in Forms$TabControl, got %v", want, got)
		}
	}
}

func TestCollectRegisteredFields_multipleTypesInOneFile(t *testing.T) {
	genDir := makeGenDir(t, map[string]string{
		"microflows/types.go": `package microflows
func initFoo() *Foo {
	o := &Foo{}
	o.SetTypeName("Microflows$Foo")
	o.x = property.NewPrimitive[string]("X", property.DecodeString)
	o.x.Bind(&o.Base, 0)
	return o
}
func initBar() *Bar {
	o := &Bar{}
	o.SetTypeName("Microflows$Bar")
	o.y = property.NewPart[element.Element]("Y")
	o.y.Bind(&o.Base, 0)
	o.z = property.NewByNameRef[element.Element]("Z", "SomeTarget")
	o.z.Bind(&o.Base, 1)
	return o
}
`,
	})

	fields := collectRegisteredFields(genDir)

	if !fields["Microflows$Foo"]["X"] {
		t.Error("Foo: missing field X")
	}
	if !fields["Microflows$Bar"]["Y"] || !fields["Microflows$Bar"]["Z"] {
		t.Errorf("Bar: missing Y or Z, got %v", fields["Microflows$Bar"])
	}
}

func TestCollectRegisteredFields_storageAliasTypeNameUsed(t *testing.T) {
	// When SetTypeName uses a storage name different from the Go struct name,
	// the map key must be the storage name (what appears in BSON $Type).
	genDir := makeGenDir(t, map[string]string{
		"mappings/types.go": `package mappings
func initMappingMicroflowParameter() *MappingMicroflowParameter {
	o := &MappingMicroflowParameter{}
	o.SetTypeName("Mappings$MappingMicroflowParameter")
	o.parameter = property.NewByNameRef[element.Element]("Parameter", "Microflows$MicroflowParameter")
	o.parameter.Bind(&o.Base, 0)
	o.levelOfParent = property.NewPrimitive[int32]("LevelOfParent", property.DecodeInt32)
	o.levelOfParent.Bind(&o.Base, 1)
	o.jsonValueElementPath = property.NewPrimitive[string]("JsonValueElementPath", property.DecodeString)
	o.jsonValueElementPath.Bind(&o.Base, 2)
	o.xmlValueElementPath = property.NewPrimitive[string]("XmlValueElementPath", property.DecodeString)
	o.xmlValueElementPath.Bind(&o.Base, 3)
	return o
}
`,
	})

	fields := collectRegisteredFields(genDir)

	got := fields["Mappings$MappingMicroflowParameter"]
	for _, want := range []string{"Parameter", "LevelOfParent", "JsonValueElementPath", "XmlValueElementPath"} {
		if !got[want] {
			t.Errorf("missing field %q, got %v", want, got)
		}
	}
}

// ── jaccardSimilarity ────────────────────────────────────────────────────────

func TestJaccardSimilarity_identicalSets(t *testing.T) {
	a := map[string]bool{"A": true, "B": true, "C": true}
	if got := jaccardSimilarity(a, a); got != 1.0 {
		t.Errorf("identical sets: want 1.0, got %f", got)
	}
}

func TestJaccardSimilarity_disjointSets(t *testing.T) {
	a := map[string]bool{"A": true}
	b := map[string]bool{"B": true}
	if got := jaccardSimilarity(a, b); got != 0.0 {
		t.Errorf("disjoint sets: want 0.0, got %f", got)
	}
}

func TestJaccardSimilarity_partialOverlap(t *testing.T) {
	// {A,B} ∩ {B,C} = {B}  union={A,B,C}  → 1/3 ≈ 0.333
	a := map[string]bool{"A": true, "B": true}
	b := map[string]bool{"B": true, "C": true}
	got := jaccardSimilarity(a, b)
	if got < 0.33 || got > 0.34 {
		t.Errorf("want ~0.333, got %f", got)
	}
}

// ── findCandidateWithFields ───────────────────────────────────────────────────

func TestFindCandidateWithFields_rejectsLowJaccardMatch(t *testing.T) {
	// BSON type has fields {JsonValueElementPath, LevelOfParent, Parameter, XmlValueElementPath}
	// Wrong candidate (Microflows$MicroflowCallParameterMapping) has {Parameter, Argument, ArgumentModel}
	// → Jaccard 0.17, should be rejected (below 0.5 threshold)
	registered := map[string]bool{
		"Microflows$MicroflowCallParameterMapping": true,
	}
	regFields := map[string]map[string]bool{
		"Microflows$MicroflowCallParameterMapping": {
			"Parameter": true, "Argument": true, "ArgumentModel": true,
		},
	}
	bsonFields := map[string]bool{
		"JsonValueElementPath": true, "LevelOfParent": true,
		"Parameter": true, "XmlValueElementPath": true,
	}

	candidate, _, confidence := findCandidateWithFields(
		"Mappings$MicroflowCallParameterMappingImpl", bsonFields, registered, regFields,
	)

	if confidence != "low" && candidate == "Microflows$MicroflowCallParameterMapping" {
		t.Errorf("should reject low-Jaccard candidate, got candidate=%q confidence=%q", candidate, confidence)
	}
}

func TestFindCandidateWithFields_findsCorrectCandidateByFieldMatch(t *testing.T) {
	// Correct candidate: Mappings$MappingMicroflowParameter, Jaccard 0.80
	registered := map[string]bool{
		"Microflows$MicroflowCallParameterMapping": true,
		"Mappings$MappingMicroflowParameter":       true,
	}
	regFields := map[string]map[string]bool{
		"Microflows$MicroflowCallParameterMapping": {
			"Parameter": true, "Argument": true, "ArgumentModel": true,
		},
		"Mappings$MappingMicroflowParameter": {
			"Parameter": true, "LevelOfParent": true,
			"ValueElementPath": true, "JsonValueElementPath": true, "XmlValueElementPath": true,
		},
	}
	bsonFields := map[string]bool{
		"JsonValueElementPath": true, "LevelOfParent": true,
		"Parameter": true, "XmlValueElementPath": true,
	}

	candidate, _, confidence := findCandidateWithFields(
		"Mappings$MicroflowCallParameterMappingImpl", bsonFields, registered, regFields,
	)

	if candidate != "Mappings$MappingMicroflowParameter" {
		t.Errorf("want Mappings$MappingMicroflowParameter, got %q (confidence=%q)", candidate, confidence)
	}
	if confidence == "low" {
		t.Errorf("should not be low confidence for Jaccard 0.80 match")
	}
}

func TestFindCandidateWithFields_highJaccardNameMatchPassesThrough(t *testing.T) {
	// strip-Impl with high field overlap → normal candidate
	registered := map[string]bool{"Foo$Bar": true}
	regFields := map[string]map[string]bool{
		"Foo$Bar": {"X": true, "Y": true, "Z": true},
	}
	bsonFields := map[string]bool{"X": true, "Y": true, "Z": true}

	candidate, _, confidence := findCandidateWithFields(
		"Foo$BarImpl", bsonFields, registered, regFields,
	)

	if candidate != "Foo$Bar" {
		t.Errorf("want Foo$Bar, got %q", candidate)
	}
	if confidence == "low" {
		t.Errorf("high Jaccard match should not be low confidence")
	}
}

// ── diffTypeFields ────────────────────────────────────────────────────────────

func TestDiffTypeFields_detectsGenOnlyKey(t *testing.T) {
	// gen has "EntityType", BSON has "EntityTypePointer" → gen-only key
	genKeys := map[string]bool{"AlternativeExposedName": true, "EntityType": true, "ExposedName": true}
	bsonKeys := map[string]bool{"AlternativeExposedName": true, "EntityTypePointer": true, "ExposedName": true}

	diff := diffTypeFields(genKeys, bsonKeys)

	if !diff.genOnly["EntityType"] {
		t.Errorf("EntityType should be gen-only, got genOnly=%v", diff.genOnly)
	}
	if !diff.bsonOnly["EntityTypePointer"] {
		t.Errorf("EntityTypePointer should be bson-only, got bsonOnly=%v", diff.bsonOnly)
	}
}

func TestDiffTypeFields_noMismatch(t *testing.T) {
	keys := map[string]bool{"Name": true, "Value": true}
	diff := diffTypeFields(keys, keys)
	if len(diff.genOnly) != 0 || len(diff.bsonOnly) != 0 {
		t.Errorf("identical key sets should produce empty diff, got %+v", diff)
	}
}

func TestDiffTypeFields_ignoresSystemKeys(t *testing.T) {
	// $ID and $Type are BSON system keys, should not appear in bsonOnly
	genKeys := map[string]bool{"Name": true}
	bsonKeys := map[string]bool{"Name": true, "$ID": true, "$Type": true}

	diff := diffTypeFields(genKeys, bsonKeys)

	if len(diff.bsonOnly) != 0 {
		t.Errorf("system keys should be ignored, got bsonOnly=%v", diff.bsonOnly)
	}
}
