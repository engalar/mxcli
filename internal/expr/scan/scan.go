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
	SlotPath string // maps to exprcheck.SlotResolver key, e.g. "IfStmt.Condition"
	UnitPath string // relative .mxunit path from mprcontents/
}

// GetRaw returns the raw expression (satisfies repair package interface).
func (r ExprRecord) GetRaw() string { return r.Raw }

// Options controls scan behaviour.
type Options struct {
	// FilterType is an optional substring filter on UnitType (case-insensitive).
	FilterType string
}

// exprSlots maps "$Type.Field" → SlotPath (for exprcheck.SlotResolver).
// SlotPath values align with mdl/exprcheck staticExpectations keys.
var exprSlots = map[string]string{
	"Microflows$ExpressionSplitCondition.Expression":     "IfStmt.Condition",
	"Microflows$WhileLoopCondition.WhileExpression":      "WhileStmt.Condition",
	"Microflows$EndEvent.ReturnValue":                    "ReturnStmt.Value",
	"Microflows$CreateVariableAction.InitialValue":       "DeclareStmt.InitialValue",
	"Microflows$ChangeVariableAction.Value":              "MfSetStmt.Value",
	"Microflows$ChangeActionItem.Value":                  "ChangeItem.Value",
	"Microflows$DatabaseRetrieveSource.XpathConstraint":  "RetrieveStmt.XPath",
	"Microflows$MicroflowCallParameterMapping.Argument":  "CallArgument.Value",
	"Microflows$NanoflowCallParameterMapping.Argument":   "CallArgument.Value",
	"Microflows$BasicCodeActionParameterValue.Argument":  "CallArgument.Value",
	"Microflows$TemplateParameter.Expression":            "TemplateParam.Value",
	"Microflows$CustomRange.LimitExpression":             "RetrieveStmt.LimitExpr",
	"Forms$ConditionalVisibilitySettings.Expression":     "IfStmt.Condition",
	"Forms$WidgetValidation.Expression":                  "IfStmt.Condition",
	"Forms$MicroflowParameterMapping.Expression":         "CallArgument.Value",
	"Forms$ClientTemplateParameter.Expression":           "TemplateParam.Value",
	"Forms$PageParameterMapping.Argument":                "CallArgument.Value",
	"DomainModels$AccessRule.XPathConstraint":            "AccessRule.XPath",
	"Workflows$MicroflowCallParameterMapping.Expression": "CallArgument.Value",
	"Workflows$SingleUserTaskActivity.DueDate":           "RetrieveStmt.LimitExpr",
	"CustomWidgets$CustomWidgetXPathSource.XPathConstraint": "AccessRule.XPath",
	"CustomWidgets$WidgetValue.Expression":               "ChangeItem.Value",
}

// exprFields maps $Type → field names containing Mendix expressions.
var exprFields map[string][]string

func init() {
	exprFields = make(map[string][]string)
	for key := range exprSlots {
		dot := strings.LastIndex(key, ".")
		if dot < 0 {
			continue
		}
		typ, field := key[:dot], key[dot+1:]
		exprFields[typ] = append(exprFields[typ], field)
	}
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

func slotPathOf(unitType, field string) string {
	return exprSlots[unitType+"."+field]
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
					*out = append(*out, ExprRecord{
						UnitID:   uid,
						Project:  project,
						UnitType: unitType,
						Field:    field,
						Raw:      raw,
						Category: categoryOf(unitType),
						SlotPath: slotPathOf(unitType, field),
						UnitPath: relPath,
					})
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
