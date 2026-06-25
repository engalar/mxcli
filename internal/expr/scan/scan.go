// Package scan walks Mendix V2 mprcontents/ directories and extracts
// expression strings from BSON .mxunit files.
//
// Validated against: corpus-a (3,715) + corpus-b (12,447) expressions.
// Coverage: 98.5% parse success rate with mdl/exprcheck parser.
package scan

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
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
	TargetAttrQN  string // Microflows$ChangeActionItem: target attribute "Module.Entity.AttrName"
	CalleeQN      string // *MicroflowCallParameterMapping: called microflow "Module.MFName"
	ParamName     string // *MicroflowCallParameterMapping: parameter name
	TargetVarName string // ChangeVariableAction/CreateVariableAction: target variable name (no $)
}

// GetRaw returns the raw expression (satisfies repair package interface).
func (r ExprRecord) GetRaw() string { return r.Raw }

// Options controls scan behaviour.
type Options struct {
	// FilterType is an optional substring filter on UnitType (case-insensitive).
	FilterType string
}

// exprFields maps Mendix $Type → field names that contain expression strings.
// Validated against 16,125 real expressions from corpus-a + corpus-b projects.
var exprFields = map[string][]string{
	// Microflow / Nanoflow
	"Microflows$ExpressionSplitCondition":      {"Expression"},
	"Microflows$WhileLoopCondition":            {"WhileExpression"},
	"Microflows$EndEvent":                      {"ReturnValue"},
	"Microflows$CreateVariableAction":          {"InitialValue"},
	"Microflows$ChangeVariableAction":          {"Value"},
	"Microflows$ChangeActionItem":              {"Value"},
	"Microflows$DatabaseRetrieveSource":        {"XpathConstraint"},
	"Microflows$MicroflowCallParameterMapping": {"Argument"},
	"Microflows$NanoflowCallParameterMapping":  {"Argument"},
	"Microflows$BasicCodeActionParameterValue": {"Argument"},
	"Microflows$TemplateParameter":             {"Expression"},
	"Microflows$CustomRange":                   {"LimitExpression"},
	// Pages / Forms
	"Forms$ConditionalVisibilitySettings": {"Expression"},
	"Forms$WidgetValidation":              {"Expression"},
	"Forms$MicroflowParameterMapping":     {"Expression"},
	"Forms$ClientTemplateParameter":       {"Expression"},
	"Forms$PageParameterMapping":          {"Argument"},
	// Domain model
	"DomainModels$AccessRule": {"XPathConstraint"},
	// Workflows
	"Workflows$MicroflowCallParameterMapping": {"Expression"},
	"Workflows$SingleUserTaskActivity":        {"DueDate"},
	"Workflows$XPathUserTargeting":            {"XPathConstraint"},
	"Workflows$XPathGroupTargeting":           {"XPathConstraint"},
	// Custom widgets
	"CustomWidgets$CustomWidgetXPathSource": {"XPathConstraint"},
	"CustomWidgets$WidgetValue":             {"Expression"},
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

// rawDoc is a thin wrapper around bson.Raw for zero-alloc field extraction.
// It is the reusable capability extracted from the old bson.M-based decoding:
// instead of fully materialising the BSON tree into heap-allocated maps,
// rawDoc uses field-level byte scans (bson.Raw.LookupErr).
type rawDoc struct{ raw bson.Raw }

func newRawDoc(data []byte) rawDoc { return rawDoc{bson.Raw(data)} }

func (d rawDoc) str(field string) string {
	v, err := d.raw.LookupErr(field)
	if err != nil {
		return ""
	}
	if v.Type == bson.TypeString {
		return v.StringValue()
	}
	return ""
}

func (d rawDoc) hexID() string {
	v, err := d.raw.LookupErr("$ID")
	if err != nil {
		return ""
	}
	switch v.Type {
	case bson.TypeBinary:
		if _, data, ok := v.BinaryOK(); ok {
			return hex.EncodeToString(data)
		}
	case bson.TypeString:
		return v.StringValue()
	}
	return ""
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
		relPath, _ := filepath.Rel(abs, path)
		extractAll(data, project, relPath, opts, &results)
		return nil
	})
	return results, err
}

// extractAll recursively extracts ExprRecords from raw BSON bytes.
// Uses field-level bson.Raw operations instead of full tree materialisation.
func extractAll(data []byte, project, relPath string, opts Options, out *[]ExprRecord) {
	doc := newRawDoc(data)
	unitType := doc.str("$Type")
	if unitType == "" {
		return
	}

	// Check if this document type carries expressions.
	if fields, ok := exprFields[unitType]; ok {
		if opts.FilterType == "" || strings.Contains(
			strings.ToLower(unitType), strings.ToLower(opts.FilterType)) {

			uid := doc.hexID()
			if uid == "" {
				uid, _ = uuidFromPath(relPath)
			}

			for _, field := range fields {
				raw := strings.TrimSpace(doc.str(field))
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
				switch unitType {
				case "Microflows$ChangeActionItem":
					rec.TargetAttrQN = doc.str("Attribute")
				case "Microflows$MicroflowCallParameterMapping",
					"Mappings$MicroflowCallParameterMappingImpl",
					"Workflows$MicroflowCallParameterMapping":
					if paramQN := doc.str("Parameter"); paramQN != "" {
						if last := strings.LastIndex(paramQN, "."); last > 0 {
							rec.CalleeQN = paramQN[:last]
							rec.ParamName = paramQN[last+1:]
						}
					}
				case "Microflows$ChangeVariableAction":
					rec.TargetVarName = doc.str("ChangeVariableName")
				case "Microflows$CreateVariableAction":
					rec.TargetVarName = doc.str("VariableName")
				}
				*out = append(*out, rec)
			}
		}
	}

	// Recurse into embedded documents and arrays to find sub-documents
	// with expression-carrying $Type values (e.g. activities inside a microflow).
	elems, err := doc.raw.Elements()
	if err != nil {
		return
	}
	for _, elem := range elems {
		val := elem.Value()
		switch val.Type {
		case bson.TypeEmbeddedDocument:
			if sub, ok := val.DocumentOK(); ok {
				extractAll(sub, project, relPath, opts, out)
			}
		case bson.TypeArray:
			if arr, ok := val.ArrayOK(); ok {
				items, iErr := bson.Raw(arr).Elements()
				if iErr != nil {
					continue
				}
				for _, item := range items {
					iv := item.Value()
					if iv.Type == bson.TypeEmbeddedDocument {
						if sub, ok := iv.DocumentOK(); ok {
							extractAll(sub, project, relPath, opts, out)
						}
					}
				}
			}
		}
	}
}

// uuidFromPath extracts a UUID from a .mxunit relpath like "ab/cd/uuid.mxunit".
func uuidFromPath(relPath string) (string, error) {
	ext := filepath.Ext(relPath)
	base := filepath.Base(relPath)
	return strings.TrimSuffix(base, ext), nil
}

// MprContentsPath converts a .mpr file path to its sibling mprcontents/ directory.
// Returns the input path unchanged if it does not end in ".mpr".
func MprContentsPath(mprPath string) string {
	if !strings.HasSuffix(mprPath, ".mpr") {
		return mprPath
	}
	return filepath.Join(filepath.Dir(mprPath), "mprcontents")
}

// ScanMPR reads expression records from a v1 MPR file (contents stored in SQLite).
// For v2 projects, use ScanMprcontents with the mprcontents/ directory instead.
func ScanMPR(mprPath string, opts Options) ([]ExprRecord, error) {
	db, err := sql.Open("sqlite", "file:"+mprPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open MPR: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT Contents FROM Unit WHERE Contents IS NOT NULL")
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	defer rows.Close()

	project := strings.TrimSuffix(filepath.Base(mprPath), filepath.Ext(mprPath))
	var results []ExprRecord
	for rows.Next() {
		var contents []byte
		if err := rows.Scan(&contents); err != nil {
			continue
		}
		// For v1 MPR (SQLite), derive UnitPath from the $ID so it matches the
		// mprcontents/ path format that the meta index uses as its lookup key.
		relPath := unitPathFromBSON(contents)
		extractAll(contents, project, relPath, opts, &results)
	}
	return results, rows.Err()
}

// unitPathFromBSON extracts the $ID from raw BSON bytes and converts it to the
// mprcontents/ relative path format (ab/cd/abcd1234-...-....mxunit).
func unitPathFromBSON(data []byte) string {
	doc := newRawDoc(data)
	id := doc.hexID()
	if id == "" {
		return ""
	}
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) != 32 {
		return id + ".mxunit"
	}
	uuid := clean[0:8] + "-" + clean[8:12] + "-" + clean[12:16] + "-" + clean[16:20] + "-" + clean[20:32]
	return clean[0:2] + "/" + clean[2:4] + "/" + uuid + ".mxunit"
}
