//go:build debug

package bson

import (
	"fmt"
	"strings"
)

// CoverageStatus represents field coverage state.
type CoverageStatus int

const (
	// Covered means the field value was found in the MDL text.
	Covered CoverageStatus = iota
	// Uncovered means the field has a non-default value but was not found in the MDL text.
	Uncovered
	// DefaultValue means the field is empty, null, or false (intentionally omitted from MDL).
	DefaultValue
	// Unknown means coverage cannot be determined automatically.
	Unknown
)

func (s CoverageStatus) String() string {
	switch s {
	case Covered:
		return "covered"
	case Uncovered:
		return "UNCOVERED"
	case DefaultValue:
		return "default"
	case Unknown:
		return "unknown"
	default:
		return "?"
	}
}

// RawUnit decouples discover logic from the mpr package.
type RawUnit struct {
	QualifiedName string
	BsonType      string
	Fields        map[string]any
}

// FieldCoverage holds coverage info for one field.
type FieldCoverage struct {
	StorageName string
	GoFieldName string
	GoType      string
	Category    FieldCategory
	Status      CoverageStatus
	SampleValue string // short string representation of sample value from BSON
}

// TypeCoverage holds coverage for one $Type.
type TypeCoverage struct {
	BsonType      string
	InstanceCount int
	Fields        []FieldCoverage
	UnknownFields []string // fields found in BSON but not in TypeRegistry
}

// SemanticCoverage returns the count of covered and total semantic fields.
func (tc *TypeCoverage) SemanticCoverage() (covered, total int) {
	for _, f := range tc.Fields {
		if f.Category == Semantic {
			total++
			if f.Status == Covered {
				covered++
			}
		}
	}
	return covered, total
}

// DiscoverResult holds complete discover output.
type DiscoverResult struct {
	Types []TypeCoverage
}

// checkFieldCoverage determines if a BSON field value appears in MDL text.
func checkFieldCoverage(storageName string, bsonValue any, mdlText string) CoverageStatus {
	switch v := bsonValue.(type) {
	case string:
		if v == "" {
			return DefaultValue
		}
		if strings.Contains(mdlText, v) {
			return Covered
		}
		return Uncovered
	case bool:
		if !v {
			return DefaultValue
		}
		if strings.Contains(mdlText, "true") {
			return Covered
		}
		return Uncovered
	case nil:
		return DefaultValue
	case map[string]any:
		return checkNestedCoverage(v, mdlText)
	default:
		return Unknown
	}
}

// checkNestedCoverage checks if any leaf string in a nested object appears in MDL text.
func checkNestedCoverage(obj map[string]any, mdlText string) CoverageStatus {
	for _, v := range obj {
		switch val := v.(type) {
		case string:
			if val != "" && strings.Contains(mdlText, val) {
				return Covered
			}
		case map[string]any:
			if checkNestedCoverage(val, mdlText) == Covered {
				return Covered
			}
		}
	}
	return Uncovered
}

// sampleValueString returns a short string representation of a BSON value for display.
func sampleValueString(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case string:
		if val == "" {
			return `""`
		}
		if len(val) > 40 {
			return fmt.Sprintf("%q", val[:40]+"...")
		}
		return fmt.Sprintf("%q", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case map[string]any:
		if t, ok := val["$Type"]; ok {
			return fmt.Sprintf("{$Type: %v}", t)
		}
		return fmt.Sprintf("{%d fields}", len(val))
	case []any:
		return fmt.Sprintf("[%d elements]", len(val))
	default:
		s := fmt.Sprintf("%v", val)
		if len(s) > 40 {
			return s[:40] + "..."
		}
		return s
	}
}

// RunDiscover analyzes field coverage for BSON objects against MDL text.
// mdlText is optional -- if empty, all non-default fields are marked Uncovered.
func RunDiscover(rawUnits []RawUnit, mdlText string) *DiscoverResult {
	// Group raw units by $Type.
	typeGroups := make(map[string][]RawUnit)
	var typeOrder []string
	for _, ru := range rawUnits {
		if _, seen := typeGroups[ru.BsonType]; !seen {
			typeOrder = append(typeOrder, ru.BsonType)
		}
		typeGroups[ru.BsonType] = append(typeGroups[ru.BsonType], ru)
	}

	var result DiscoverResult
	for _, bsonType := range typeOrder {
		units := typeGroups[bsonType]
		tc := analyzeTypeCoverage(bsonType, units, mdlText)
		result.Types = append(result.Types, tc)
	}
	return &result
}

// analyzeTypeCoverage builds coverage for a single $Type across all its instances.
func analyzeTypeCoverage(bsonType string, units []RawUnit, mdlText string) TypeCoverage {
	tc := TypeCoverage{
		BsonType:      bsonType,
		InstanceCount: len(units),
	}

	// Collect all field names and one sample value across all instances.
	fieldSamples := make(map[string]any)
	for _, u := range units {
		for fieldName, fieldValue := range u.Fields {
			if _, exists := fieldSamples[fieldName]; !exists {
				fieldSamples[fieldName] = fieldValue
			} else if fieldValue != nil {
				// Prefer non-nil sample values.
				existing := fieldSamples[fieldName]
				if existing == nil {
					fieldSamples[fieldName] = fieldValue
				}
			}
		}
	}

	// Look up metadata from TypeRegistry.
	meta := GetFieldMeta(bsonType)
	if meta == nil {
		// Type not in registry -- report all fields as unknown category.
		for fieldName, sample := range fieldSamples {
			category := classifyField(fieldName)
			tc.Fields = append(tc.Fields, FieldCoverage{
				StorageName: fieldName,
				GoFieldName: "",
				GoType:      "",
				Category:    category,
				Status:      checkFieldCoverage(fieldName, sample, mdlText),
				SampleValue: sampleValueString(sample),
			})
		}
		return tc
	}

	// Build a lookup from storage name to meta.
	metaByStorage := make(map[string]PropertyMeta, len(meta))
	for _, m := range meta {
		metaByStorage[m.StorageName] = m
	}

	// Process fields from metadata (preserves struct field order).
	for _, m := range meta {
		sample, inBson := fieldSamples[m.StorageName]
		status := DefaultValue
		if inBson {
			status = checkFieldCoverage(m.StorageName, sample, mdlText)
		}
		tc.Fields = append(tc.Fields, FieldCoverage{
			StorageName: m.StorageName,
			GoFieldName: m.GoFieldName,
			GoType:      m.GoType,
			Category:    m.Category,
			Status:      status,
			SampleValue: sampleValueString(sample),
		})
	}

	// Find fields in BSON but not in metadata.
	for fieldName := range fieldSamples {
		if _, inMeta := metaByStorage[fieldName]; !inMeta {
			tc.UnknownFields = append(tc.UnknownFields, fieldName)
		}
	}

	return tc
}

// FormatResult formats a DiscoverResult for terminal output.
func FormatResult(dr *DiscoverResult) string {
	var sb strings.Builder
	for i, tc := range dr.Types {
		if i > 0 {
			sb.WriteString("\n")
		}
		formatTypeCoverage(&sb, &tc)
	}
	return sb.String()
}

func formatTypeCoverage(sb *strings.Builder, tc *TypeCoverage) {
	sb.WriteString(fmt.Sprintf("%s (%d objects scanned)\n", tc.BsonType, tc.InstanceCount))

	// Separate fields by category.
	var semanticFields []FieldCoverage
	var structuralNames []string
	var layoutNames []string

	for _, f := range tc.Fields {
		switch f.Category {
		case Semantic:
			semanticFields = append(semanticFields, f)
		case Structural:
			structuralNames = append(structuralNames, f.StorageName)
		case Layout:
			layoutNames = append(layoutNames, f.StorageName)
		}
	}

	// Print semantic fields.
	for _, f := range semanticFields {
		marker := "\u2717" // ✗
		if f.Status == Covered {
			marker = "\u2713" // ✓
		}
		detail := f.Status.String()
		if f.Status == Uncovered {
			detail = fmt.Sprintf("UNCOVERED (%s, ex: %s)", typeLabel(f.GoType), f.SampleValue)
		}
		sb.WriteString(fmt.Sprintf("  %s %-30s %s\n", marker, f.StorageName, detail))
	}

	// Print structural/layout summary.
	nonSemanticCount := len(structuralNames) + len(layoutNames)
	if nonSemanticCount > 0 {
		allNames := append(structuralNames, layoutNames...)
		if len(allNames) <= 5 {
			sb.WriteString(fmt.Sprintf("  - %s    structural/layout (%d fields)\n",
				strings.Join(allNames, ", "), nonSemanticCount))
		} else {
			preview := strings.Join(allNames[:3], ", ")
			sb.WriteString(fmt.Sprintf("  - %s...    structural/layout (%d fields)\n",
				preview, nonSemanticCount))
		}
	}

	// Print unknown fields.
	if len(tc.UnknownFields) > 0 {
		sb.WriteString(fmt.Sprintf("  ? unknown-to-schema: %s\n", strings.Join(tc.UnknownFields, ", ")))
	}

	// Coverage summary.
	covered, total := tc.SemanticCoverage()
	pct := 0
	if total > 0 {
		pct = covered * 100 / total
	}
	sb.WriteString(fmt.Sprintf("\n  Coverage: %d/%d semantic fields (%d%%)\n", covered, total, pct))
}

// typeLabel returns a short label for a Go type string.
func typeLabel(goType string) string {
	if goType == "" {
		return "any"
	}
	// Strip package prefix for readability.
	if idx := strings.LastIndex(goType, "."); idx >= 0 {
		return goType[idx+1:]
	}
	return goType
}
