package dtsparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAllDomains(t *testing.T) {
	genDir := findMendixModelSDKGenDir(t)

	// Collect all enums across modules
	allEnums := collectCrossModuleEnums(genDir)
	t.Logf("Cross-module enums collected: %d", len(allEnums))

	entries, err := os.ReadDir(genDir)
	if err != nil {
		t.Fatalf("cannot read gen dir: %v", err)
	}

	totalClasses := 0
	totalEnums := 0
	totalProps := 0
	totalStructTypeNames := 0
	kindCounts := map[PropertyKind]int{}
	domainSummary := []string{}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".d.ts") {
			continue
		}
		domain := strings.TrimSuffix(entry.Name(), ".d.ts")
		if domain == "base-model" || domain == "all-model-classes" {
			continue // meta files, not domain files
		}

		dtsData, err := os.ReadFile(filepath.Join(genDir, entry.Name()))
		if err != nil {
			t.Errorf("cannot read %s: %v", entry.Name(), err)
			continue
		}

		classes, enums := parseDtsFileWithEnums(string(dtsData), allEnums)

		// Also parse .js for structureTypeNames
		jsFile := strings.TrimSuffix(entry.Name(), ".d.ts") + ".js"
		jsData, _ := os.ReadFile(filepath.Join(genDir, jsFile))
		stns := parseStructureTypeNames(string(jsData))

		domainClasses := len(classes)
		domainEnums := len(enums)
		domainProps := 0
		for _, c := range classes {
			domainProps += len(c.Properties)
			for _, p := range c.Properties {
				kindCounts[p.Kind]++
			}
		}

		totalClasses += domainClasses
		totalEnums += domainEnums
		totalProps += domainProps
		totalStructTypeNames += len(stns)

		domainSummary = append(domainSummary,
			fmt.Sprintf("%-30s classes=%3d enums=%2d props=%4d $Types=%3d",
				domain, domainClasses, domainEnums, domainProps, len(stns)))
	}

	t.Log("=== Per-Domain Summary ===")
	for _, s := range domainSummary {
		t.Log(s)
	}

	t.Log("=== Totals ===")
	t.Logf("Domains:             %d", len(domainSummary))
	t.Logf("Total classes:       %d", totalClasses)
	t.Logf("Total enums:         %d", totalEnums)
	t.Logf("Total properties:    %d", totalProps)
	t.Logf("Total $Type names:   %d", totalStructTypeNames)

	t.Log("=== Property Kind Distribution ===")
	for k, v := range kindCounts {
		t.Logf("  %-10s: %4d (%.1f%%)", k, v, float64(v)/float64(totalProps)*100)
	}

	// Classification rate
	unknownPct := float64(kindCounts[KindUnknown]) / float64(totalProps) * 100
	classRate := 100 - unknownPct
	t.Logf("Classification rate: %.1f%%", classRate)

	// Assertions
	if len(domainSummary) < 40 {
		t.Errorf("expected at least 40 domains, got %d", len(domainSummary))
	}
	if totalClasses < 200 {
		t.Errorf("expected at least 200 classes, got %d", totalClasses)
	}
	if classRate < 80 {
		t.Errorf("classification rate %.1f%% too low, expected >= 80%%", classRate)
	}
}
