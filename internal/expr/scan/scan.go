// Package scan walks Mendix V2 mprcontents/ directories and extracts
// expression strings from BSON .mxunit files.
//
// Validated against: macnica (3,715) + Mx2026AIDay (12,447) expressions.
// Coverage: 98.5% parse success rate with mdl/exprcheck parser.
package scan

import (
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExprRecord is one expression extracted from a BSON unit file.
type ExprRecord struct {
	UnitID   string // hex UUID from $ID field
	Project  string // parent directory name of mprcontents/
	UnitType string // e.g. "Microflows$ExpressionSplitCondition"
	Field    string // e.g. "Expression"
	Raw      string // the expression string, trimmed
	Category string // microflow | page | domain | workflow | widget
	UnitPath string // relative .mxunit path from mprcontents/
	// SlotPath was removed: the parse package detects XPath vs regular expressions
	// by content (starts with "[" but not "[%"), which is more robust than a mapping table.

	// Type-checking context — populated only for specific UnitTypes; empty otherwise.
	TargetAttrQN string // Microflows$ChangeActionItem: target attribute "Module.Entity.AttrName"
	CalleeQN     string // *MicroflowCallParameterMapping: called microflow "Module.MFName"
	ParamName    string // *MicroflowCallParameterMapping: parameter name
}

// GetRaw returns the raw expression (satisfies repair package interface).
func (r ExprRecord) GetRaw() string { return r.Raw }

// Options controls scan behaviour.
type Options struct {
	// FilterType is an optional substring filter on UnitType (case-insensitive).
	FilterType string
}

// exprFields maps Mendix $Type → field names that contain expression strings.
// Validated against 16,125 real expressions from macnica + Mx2026AIDay projects.
var exprFields = map[string][]string{
	// Microflow / Nanoflow
	"Microflows$ExpressionSplitCondition":    {"Expression"},
	"Microflows$WhileLoopCondition":           {"WhileExpression"},
	"Microflows$EndEvent":                     {"ReturnValue"},
	"Microflows$CreateVariableAction":         {"InitialValue"},
	"Microflows$ChangeVariableAction":         {"Value"},
	"Microflows$ChangeActionItem":             {"Value"},
	"Microflows$DatabaseRetrieveSource":       {"XpathConstraint"},
	"Microflows$MicroflowCallParameterMapping":{"Argument"},
	"Microflows$NanoflowCallParameterMapping": {"Argument"},
	"Microflows$BasicCodeActionParameterValue":{"Argument"},
	"Microflows$TemplateParameter":            {"Expression"},
	"Microflows$CustomRange":                  {"LimitExpression"},
	// Pages / Forms
	"Forms$ConditionalVisibilitySettings":    {"Expression"},
	"Forms$WidgetValidation":                 {"Expression"},
	"Forms$MicroflowParameterMapping":        {"Expression"},
	"Forms$ClientTemplateParameter":          {"Expression"},
	"Forms$PageParameterMapping":             {"Argument"},
	// Domain model
	"DomainModels$AccessRule":                {"XPathConstraint"},
	// Workflows
	"Workflows$MicroflowCallParameterMapping":{"Expression"},
	"Workflows$SingleUserTaskActivity":       {"DueDate"},
	// Custom widgets
	"CustomWidgets$CustomWidgetXPathSource":  {"XPathConstraint"},
	"CustomWidgets$WidgetValue":              {"Expression"},
}

var categoryMap = map[string]string{
	"Microflows$":    "microflow",
	"Forms$":         "page",
	"DomainModels$":  "domain",
	"Workflows$":     "workflow",
	"CustomWidgets$": "widget",
}

func categoryOf(unitType string) string {
	for prefix, cat := range categoryMap {
		if strings.HasPrefix(unitType, prefix) {
			return cat
		}
	}
	return "other"
}

// ScanMprcontents walks a mprcontents/ directory and returns all expression records.
func ScanMprcontents(mprcontentsPath string, opts Options) ([]ExprRecord, error) {
	abs, err := filepath.Abs(mprcontentsPath)
	if err != nil {
		return nil, err
	}
	project := filepath.Base(filepath.Dir(abs))
	var results []ExprRecord

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".mxunit") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files silently
		}
		var doc bson.M
		if bsonErr := bson.Unmarshal(data, &doc); bsonErr != nil {
			return nil // skip malformed BSON
		}
		relPath, _ := filepath.Rel(abs, path)
		scanObj(doc, project, relPath, opts, &results)
		return nil
	})
	return results, err
}

func scanObj(v interface{}, project, relPath string, opts Options, out *[]ExprRecord) {
	switch val := v.(type) {
	case bson.M:
		unitType, _ := val["$Type"].(string)
		if fields, ok := exprFields[unitType]; ok {
			if opts.FilterType == "" || strings.Contains(
				strings.ToLower(unitType), strings.ToLower(opts.FilterType)) {
				uid := extractID(val["$ID"])
				for _, field := range fields {
					raw, _ := val[field].(string)
					raw = strings.TrimSpace(raw)
					// Exclude empty values and URLs (not expressions)
					if raw == "" || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
						continue
					}
					rec := ExprRecord{
						UnitID:   uid,
						Project:  project,
						UnitType: unitType,
						Field:    field,
						Raw:      raw,
						Category: categoryOf(unitType),
						UnitPath: relPath,
					}
					// Populate type-checking context fields.
					switch unitType {
					case "Microflows$ChangeActionItem":
						rec.TargetAttrQN, _ = val["Attribute"].(string)
					case "Microflows$MicroflowCallParameterMapping",
						"Mappings$MicroflowCallParameterMappingImpl",
						"Workflows$MicroflowCallParameterMapping":
						rec.CalleeQN, _ = val["Microflow"].(string)
						rec.ParamName, _ = val["Parameter"].(string)
					}
					*out = append(*out, rec)
				}
			}
		}
		for _, child := range val {
			scanObj(child, project, relPath, opts, out)
		}
	case bson.A:
		for _, item := range val {
			scanObj(item, project, relPath, opts, out)
		}
	}
}

// MprContentsPath converts a .mpr file path to its sibling mprcontents/ directory.
// Returns the input path unchanged if it does not end in ".mpr".
func MprContentsPath(mprPath string) string {
	if !strings.HasSuffix(mprPath, ".mpr") {
		return mprPath
	}
	return filepath.Join(filepath.Dir(mprPath), "mprcontents")
}

func extractID(raw interface{}) string {
	switch v := raw.(type) {
	case primitive.Binary:
		return hex.EncodeToString(v.Data)
	case []byte:
		return hex.EncodeToString(v)
	case string:
		return v
	}
	return ""
}
