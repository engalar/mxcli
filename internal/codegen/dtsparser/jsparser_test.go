package dtsparser

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseJsDomainModels(t *testing.T) {
	jsPath := findMendixModelSDKGenDir(t) + "/domainmodels.js"
	meta, err := ParseJsFile(jsPath)
	if err != nil {
		t.Fatalf("ParseJsFile: %v", err)
	}

	t.Logf("Namespace: %s", meta.Namespace)
	t.Logf("Total classes: %d", len(meta.Classes))

	// Build index
	idx := map[string]*JsClass{}
	for i := range meta.Classes {
		idx[meta.Classes[i].Name] = &meta.Classes[i]
	}

	// ──────────── Entity ────────────
	t.Run("Entity", func(t *testing.T) {
		entity := idx["Entity"]
		if entity == nil {
			t.Fatal("Entity not found")
		}
		t.Logf("StructureTypeName: %s", entity.StructureTypeName)
		t.Logf("BaseClass: %s", entity.BaseClass)
		t.Logf("StructureKind: %s", entity.StructureKind)
		t.Logf("IsAbstract: %v", entity.IsAbstract)
		t.Logf("Properties (%d):", len(entity.Properties))

		for _, p := range entity.Properties {
			t.Logf("  %-25s Kind=%-20s PrimType=%-8s Default=%-30s Target=%-30s Req=%v",
				p.Name, p.Kind, p.PrimitiveType, truncate(p.DefaultValue, 30), p.TargetType, p.IsRequired)
		}

		// Verify property kinds
		propIdx := map[string]*JsProp{}
		for i := range entity.Properties {
			propIdx[entity.Properties[i].Name] = &entity.Properties[i]
		}

		checks := []struct {
			name     string
			kind     PropKind
			primType PrimitiveType
			target   string
		}{
			{"name", PKPrimitive, PTString, ""},
			{"dataStorageGuid", PKPrimitive, PTGuid, ""},
			{"location", PKPrimitive, PTPoint, ""},
			{"documentation", PKPrimitive, PTString, ""},
			{"generalization", PKPart, "", ""},
			{"attributes", PKPartList, "", ""},
			{"validationRules", PKPartList, "", ""},
			{"eventHandlers", PKPartList, "", ""},
			{"indexes", PKPartList, "", ""},
			{"accessRules", PKPartList, "", ""},
			{"image", PKByNameRef, "", "Images$Image"},
			{"imageData", PKPrimitive, PTBlob, ""},
			{"isRemote", PKPrimitive, PTBoolean, ""},
			{"source", PKPart, "", ""},
			{"capabilities", PKPart, "", ""},
			{"exportLevel", PKEnum, "", ""},
		}

		for _, c := range checks {
			p := propIdx[c.name]
			if p == nil {
				t.Errorf("property %q not found", c.name)
				continue
			}
			if p.Kind != c.kind {
				t.Errorf("%s: Kind = %s, want %s", c.name, p.Kind, c.kind)
			}
			if c.primType != "" && p.PrimitiveType != c.primType {
				t.Errorf("%s: PrimitiveType = %s, want %s", c.name, p.PrimitiveType, c.primType)
			}
			if c.target != "" && p.TargetType != c.target {
				t.Errorf("%s: TargetType = %q, want %q", c.name, p.TargetType, c.target)
			}
		}

		// Verify StructureTypeName
		if entity.StructureTypeName != "DomainModels$Entity" {
			t.Errorf("STN = %q, want DomainModels$Entity", entity.StructureTypeName)
		}

		// Verify StructureKind
		if entity.StructureKind != SKElement {
			t.Errorf("StructureKind = %s, want Element", entity.StructureKind)
		}

		// Verify not abstract
		if entity.IsAbstract {
			t.Error("Entity should not be abstract")
		}
	})

	// ──────────── Entity defaults ────────────
	t.Run("EntityDefaults", func(t *testing.T) {
		entity := idx["Entity"]
		if entity == nil {
			t.Fatal("Entity not found")
		}
		t.Logf("Defaults (%d):", len(entity.Defaults))
		for _, d := range entity.Defaults {
			t.Logf("  %-25s VersionGated=%v  Expr=%s", d.Property, d.IsVersionGated, truncate(d.DefaultExpr, 60))
		}

		// Check some specific defaults
		foundGeneralization := false
		foundCapabilities := false
		foundExportLevel := false
		for _, d := range entity.Defaults {
			if d.Property == "generalization" {
				foundGeneralization = true
				if d.IsVersionGated {
					t.Error("generalization default should NOT be version-gated")
				}
			}
			if d.Property == "capabilities" {
				foundCapabilities = true
				if !d.IsVersionGated {
					t.Error("capabilities default SHOULD be version-gated")
				}
			}
			if d.Property == "exportLevel" {
				foundExportLevel = true
				if !d.IsVersionGated {
					t.Error("exportLevel default SHOULD be version-gated")
				}
			}
		}
		if !foundGeneralization {
			t.Error("generalization default not found")
		}
		if !foundCapabilities {
			t.Error("capabilities default not found")
		}
		if !foundExportLevel {
			t.Error("exportLevel default not found")
		}
	})

	// ──────────── Entity versionInfo ────────────
	t.Run("EntityVersionInfo", func(t *testing.T) {
		entity := idx["Entity"]
		if entity == nil || entity.VersionInfo == nil {
			t.Fatal("Entity or its VersionInfo not found")
		}
		vi := entity.VersionInfo
		t.Logf("StructureKind: %s", vi.StructureKind)
		t.Logf("Property version info (%d):", len(vi.PropertyInfos))
		for name, pvi := range vi.PropertyInfos {
			t.Logf("  %-25s Intro=%-8s Del=%-8s Public=%v Req=%v",
				name, pvi.Introduced, pvi.Deleted, pvi.PublicNow, pvi.RequiredNow)
		}

		// Check specific properties
		imgData := vi.PropertyInfos["imageData"]
		if imgData == nil {
			t.Error("imageData not in versionInfo")
		} else if imgData.Introduced != "9.17.0" {
			t.Errorf("imageData.Introduced = %q, want 9.17.0", imgData.Introduced)
		}

		isRemote := vi.PropertyInfos["isRemote"]
		if isRemote == nil {
			t.Error("isRemote not in versionInfo")
		} else {
			if isRemote.Introduced != "7.17.0" {
				t.Errorf("isRemote.Introduced = %q, want 7.17.0", isRemote.Introduced)
			}
			if isRemote.Deleted != "8.10.0" {
				t.Errorf("isRemote.Deleted = %q, want 8.10.0", isRemote.Deleted)
			}
		}
	})

	// ──────────── Abstract classes ────────────
	t.Run("AbstractClasses", func(t *testing.T) {
		abstracts := []string{}
		for _, c := range meta.Classes {
			if c.IsAbstract {
				abstracts = append(abstracts, c.Name)
			}
		}
		t.Logf("Abstract classes (%d): %v", len(abstracts), abstracts)

		// Verify some specific ones
		expected := map[string]bool{
			"AssociationBase":    true,
			"GeneralizationBase": true,
			"AttributeType":      true,
			"ValueType":          true,
			"EntitySource":       true,
			"MemberRef":          true,
		}
		for name := range expected {
			cls := idx[name]
			if cls == nil {
				t.Errorf("class %s not found", name)
				continue
			}
			if !cls.IsAbstract {
				t.Errorf("%s should be abstract", name)
			}
		}

		// Verify concrete classes are NOT abstract
		concrete := []string{"Entity", "Attribute", "Association", "DomainModel"}
		for _, name := range concrete {
			cls := idx[name]
			if cls == nil {
				t.Errorf("class %s not found", name)
				continue
			}
			if cls.IsAbstract {
				t.Errorf("%s should NOT be abstract", name)
			}
		}
	})

	// ──────────── DomainModel as ModelUnit ────────────
	t.Run("DomainModelIsModelUnit", func(t *testing.T) {
		dm := idx["DomainModel"]
		if dm == nil {
			t.Fatal("DomainModel not found")
		}
		if dm.StructureKind != SKModelUnit {
			t.Errorf("DomainModel.StructureKind = %s, want ModelUnit", dm.StructureKind)
		}
		t.Logf("DomainModel: Kind=%s, STN=%s, Props=%d", dm.StructureKind, dm.StructureTypeName, len(dm.Properties))
		for _, p := range dm.Properties {
			t.Logf("  %-25s Kind=%-15s", p.Name, p.Kind)
		}
	})

	// ──────────── ByNameRef cross-domain ────────────
	t.Run("CrossDomainRefs", func(t *testing.T) {
		// AccessRule.moduleRoles should be ByNameRefList targeting Security$ModuleRole
		ar := idx["AccessRule"]
		if ar == nil {
			t.Fatal("AccessRule not found")
		}
		for _, p := range ar.Properties {
			if p.Name == "moduleRoles" {
				if p.Kind != PKByNameRefList {
					t.Errorf("moduleRoles.Kind = %s, want ByNameRefList", p.Kind)
				}
				if p.TargetType != "Security$ModuleRole" {
					t.Errorf("moduleRoles.TargetType = %q, want Security$ModuleRole", p.TargetType)
				}
				t.Logf("AccessRule.moduleRoles: Kind=%s Target=%s", p.Kind, p.TargetType)
			}
		}
	})

	// ──────────── Association ByIdRef ────────────
	t.Run("ByIdRefDetection", func(t *testing.T) {
		assoc := idx["Association"]
		if assoc == nil {
			t.Fatal("Association not found")
		}
		for _, p := range assoc.Properties {
			if p.Name == "child" {
				if p.Kind != PKByIdRef {
					t.Errorf("child.Kind = %s, want ByIdRef", p.Kind)
				}
				t.Logf("Association.child: Kind=%s", p.Kind)
			}
		}
	})

	// ──────────── Part with IsRequired ────────────
	t.Run("PartIsRequired", func(t *testing.T) {
		entity := idx["Entity"]
		if entity == nil {
			t.Fatal("Entity not found")
		}
		for _, p := range entity.Properties {
			if p.Name == "generalization" {
				if p.Kind != PKPart {
					t.Errorf("generalization.Kind = %s, want Part", p.Kind)
				}
				if !p.IsRequired {
					t.Error("generalization.IsRequired should be true")
				}
				t.Logf("Entity.generalization: Kind=%s Required=%v", p.Kind, p.IsRequired)
			}
		}
	})

	// ──────────── Enums ────────────
	t.Run("Enums", func(t *testing.T) {
		t.Logf("Enums (%d):", len(meta.Enums))
		for _, e := range meta.Enums {
			vals := make([]string, len(e.Values))
			for i, v := range e.Values {
				vals[i] = v.Name
			}
			t.Logf("  %s: %v", e.Name, vals)
		}

		if len(meta.Enums) != 12 {
			t.Errorf("expected 12 enums, got %d", len(meta.Enums))
		}

		// Check specific enums exist
		enumIdx := map[string]*JsEnum{}
		for i := range meta.Enums {
			enumIdx[meta.Enums[i].Name] = &meta.Enums[i]
		}

		am := enumIdx["ActionMoment"]
		if am == nil {
			t.Fatal("ActionMoment enum not found")
		}
		if len(am.Values) != 2 {
			t.Errorf("ActionMoment: expected 2 values, got %d", len(am.Values))
		}

		an := enumIdx["AssociationNavigability"]
		if an == nil {
			t.Fatal("AssociationNavigability enum not found")
		}
		if len(an.Values) != 3 {
			t.Errorf("AssociationNavigability: expected 3 values, got %d", len(an.Values))
		}
	})
}

func TestParseJsAllDomains(t *testing.T) {
	genDir := findMendixModelSDKGenDir(t)
	domains, err := ParseAllDomains(genDir)
	if err != nil {
		t.Fatalf("ParseAllDomains: %v", err)
	}

	totalClasses := 0
	totalProps := 0
	kindCounts := map[PropKind]int{}
	skCounts := map[StructureKind]int{}
	abstractCount := 0
	withVersionInfo := 0
	withDefaults := 0
	crossDomainRefs := 0

	domainSummary := []string{}

	for _, meta := range domains {
		dClasses := len(meta.Classes)
		dProps := 0
		dAbstract := 0
		dWithVI := 0
		dWithDef := 0
		dCrossRef := 0

		for _, c := range meta.Classes {
			dProps += len(c.Properties)
			for _, p := range c.Properties {
				kindCounts[p.Kind]++
				if p.TargetType != "" && strings.Contains(p.TargetType, "$") {
					dCrossRef++
				}
			}
			if c.IsAbstract {
				dAbstract++
			}
			if c.VersionInfo != nil {
				dWithVI++
				skCounts[c.StructureKind]++
			}
			if len(c.Defaults) > 0 {
				dWithDef++
			}
		}

		totalClasses += dClasses
		totalProps += dProps
		abstractCount += dAbstract
		withVersionInfo += dWithVI
		withDefaults += dWithDef
		crossDomainRefs += dCrossRef

		domainSummary = append(domainSummary,
			fmt.Sprintf("%-30s classes=%3d props=%4d abstract=%2d vinfo=%3d defaults=%3d xref=%3d",
				meta.Namespace, dClasses, dProps, dAbstract, dWithVI, dWithDef, dCrossRef))
	}

	t.Log("=== Per-Domain Summary ===")
	for _, s := range domainSummary {
		t.Log(s)
	}

	t.Log("\n=== Totals ===")
	t.Logf("Domains:            %d", len(domains))
	t.Logf("Total classes:      %d", totalClasses)
	t.Logf("Total properties:   %d", totalProps)
	t.Logf("Abstract classes:   %d", abstractCount)
	t.Logf("With VersionInfo:   %d", withVersionInfo)
	t.Logf("With defaults:      %d", withDefaults)
	t.Logf("Cross-domain refs:  %d", crossDomainRefs)

	t.Log("\n=== Property Kind Distribution ===")
	for k, v := range kindCounts {
		t.Logf("  %-20s: %4d (%.1f%%)", k, v, float64(v)/float64(totalProps)*100)
	}

	t.Log("\n=== Structure Kind Distribution ===")
	for k, v := range skCounts {
		t.Logf("  %-20s: %d", k, v)
	}

	// Unknown should be zero or near zero from .js parsing
	unknownPct := float64(kindCounts[PKUnknown]) / float64(totalProps) * 100
	t.Logf("\nUnknown rate: %.1f%% (%d/%d)", unknownPct, kindCounts[PKUnknown], totalProps)

	// Assertions
	if len(domains) < 40 {
		t.Errorf("expected >= 40 domains, got %d", len(domains))
	}
	if totalClasses < 1000 {
		t.Errorf("expected >= 1000 classes, got %d", totalClasses)
	}
	if unknownPct > 2.0 {
		t.Errorf("unknown rate %.1f%% too high for .js parsing", unknownPct)
	}
	if withVersionInfo < totalClasses/2 {
		t.Errorf("expected most classes to have versionInfo, got %d/%d", withVersionInfo, totalClasses)
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
