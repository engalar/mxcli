package codec

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// ---------------------------------------------------------------------------
// BSONDocBuilder — fluent API for constructing BSON documents without
// exposing bson.D/bson.E/bson.A types to engine/ callers.
// ---------------------------------------------------------------------------

// BSONDocBuilder builds a BSON document (bson.D) without requiring the
// caller to import go.mongodb.org/mongo-driver/bson.
type BSONDocBuilder struct {
	doc bson.D
}

// NewBSONDoc creates a new empty BSON document builder.
func NewBSONDoc() *BSONDocBuilder {
	return &BSONDocBuilder{}
}

// Set adds or replaces a field with any value.
func (b *BSONDocBuilder) Set(key string, value any) *BSONDocBuilder {
	b.doc = append(b.doc, bson.E{Key: key, Value: value})
	return b
}

// SetIf conditionally adds a field only when the condition is true.
func (b *BSONDocBuilder) SetIf(cond bool, key string, value any) *BSONDocBuilder {
	if cond {
		b.doc = append(b.doc, bson.E{Key: key, Value: value})
	}
	return b
}

// SetIfNotEmpty adds a string field only when the value is non-empty.
func (b *BSONDocBuilder) SetIfNotEmpty(key, value string) *BSONDocBuilder {
	if value != "" {
		b.doc = append(b.doc, bson.E{Key: key, Value: value})
	}
	return b
}

// SetChild adds a nested document (another builder) as a field value.
func (b *BSONDocBuilder) SetChild(key string, child *BSONDocBuilder) *BSONDocBuilder {
	b.doc = append(b.doc, bson.E{Key: key, Value: child.doc})
	return b
}

// SetArray adds a versioned array (prefixed with int32(3)) of sub-documents.
func (b *BSONDocBuilder) SetArray(key string, items []*BSONDocBuilder) *BSONDocBuilder {
	arr := bson.A{int32(3)}
	for _, item := range items {
		arr = append(arr, item.doc)
	}
	b.doc = append(b.doc, bson.E{Key: key, Value: arr})
	return b
}

// SetStringArray adds an unversioned array of strings.
func (b *BSONDocBuilder) SetStringArray(key string, items []string) *BSONDocBuilder {
	arr := bson.A{}
	for _, s := range items {
		arr = append(arr, s)
	}
	b.doc = append(b.doc, bson.E{Key: key, Value: arr})
	return b
}

// Marshal serializes the document to BSON bytes.
func (b *BSONDocBuilder) Marshal() ([]byte, error) {
	return bson.Marshal(b.doc)
}

// ---------------------------------------------------------------------------
// PatchMultiBSONFields — batch set/replace for ALTER operations on []byte.
// ---------------------------------------------------------------------------

// PatchMultiBSONFields sets or replaces multiple fields in a BSON document.
// This is more efficient than chaining PatchBSONField calls because it
// only unmarshals/marshals once.
func PatchMultiBSONFields(data []byte, fields map[string]any) ([]byte, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	for key, val := range fields {
		found := false
		for i, e := range doc {
			if e.Key == key {
				doc[i].Value = val
				found = true
				break
			}
		}
		if !found {
			doc = append(doc, bson.E{Key: key, Value: val})
		}
	}
	return bson.Marshal(doc)
}

// ---------------------------------------------------------------------------
// BSON tree-walking helpers — for styling.go and widgets.go recursive
// traversal of widget trees without exposing bson types to engine/.
// ---------------------------------------------------------------------------

// BSONWalkDocFunc is called for each bson.D node in a tree traversal.
// The function receives the document and can modify it in place.
// Return true to stop traversal (early exit).
type BSONWalkDocFunc func(doc []byte) (replacement []byte, stop bool, err error)

// WalkBSONTree recursively walks a BSON document tree, calling fn for each
// embedded document node. This is used for widget tree traversal.
// Returns the modified document bytes.
func WalkBSONTree(data []byte, fn func(doc bson.D) (bson.D, bool)) ([]byte, bool, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", err)
	}

	doc, stopped := walkDoc(doc, fn)

	out, err := bson.Marshal(doc)
	if err != nil {
		return nil, false, fmt.Errorf("marshal: %w", err)
	}
	return out, stopped, nil
}

func walkDoc(doc bson.D, fn func(bson.D) (bson.D, bool)) (bson.D, bool) {
	doc, stop := fn(doc)
	if stop {
		return doc, true
	}
	for i, e := range doc {
		switch v := e.Value.(type) {
		case bson.D:
			v, stop = walkDoc(v, fn)
			doc[i].Value = v
			if stop {
				return doc, true
			}
		case bson.A:
			v, stop = walkArray(v, fn)
			doc[i].Value = v
			if stop {
				return doc, true
			}
		}
	}
	return doc, false
}

func walkArray(arr bson.A, fn func(bson.D) (bson.D, bool)) (bson.A, bool) {
	for i, item := range arr {
		if d, ok := item.(bson.D); ok {
			d, stop := walkDoc(d, fn)
			arr[i] = d
			if stop {
				return arr, true
			}
		}
	}
	return arr, false
}

// BSONDocGetString returns the string value for a key in a BSON document,
// or empty string if not found.
func BSONDocGetString(doc bson.D, key string) string {
	for _, e := range doc {
		if e.Key == key {
			if s, ok := e.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}

// BSONDocGetDoc returns a nested bson.D for a key, or nil if not found.
func BSONDocGetDoc(doc bson.D, key string) bson.D {
	for _, e := range doc {
		if e.Key == key {
			if d, ok := e.Value.(bson.D); ok {
				return d
			}
		}
	}
	return nil
}

// NewBSONEmptyDoc returns an empty bson.D for use where callers need
// an empty document value without importing bson.
func NewBSONEmptyDoc() bson.D {
	return bson.D{}
}

// UnmarshalBSONDoc unmarshals raw BSON bytes into a bson.D.
func UnmarshalBSONDoc(data []byte) (bson.D, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// MarshalBSONDoc marshals a bson.D into raw BSON bytes.
func MarshalBSONDoc(doc bson.D) ([]byte, error) {
	return bson.Marshal(doc)
}

// ---------------------------------------------------------------------------
// Styling-specific helpers
// ---------------------------------------------------------------------------

// ApplyStylingToWidget finds a widget by name in a BSON document tree
// and applies styling changes (Class, Style, or DesignProperties).
// Returns (modifiedDoc, found).
func ApplyStylingToWidget(data []byte, widgetName string, changes map[string]string, clearDesignProps bool) ([]byte, bool, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf("unmarshal: %w", err)
	}

	found := applyStyling(&doc, widgetName, changes, clearDesignProps)
	if !found {
		return data, false, nil
	}

	out, err := bson.Marshal(doc)
	if err != nil {
		return nil, false, fmt.Errorf("marshal: %w", err)
	}
	return out, true, nil
}

func applyStyling(doc *bson.D, widgetName string, changes map[string]string, clearDesignProps bool) bool {
	name := ""
	for _, e := range *doc {
		if e.Key == "Name" {
			name, _ = e.Value.(string)
			break
		}
	}
	if name == widgetName {
		for prop, val := range changes {
			switch prop {
			case "Class", "Style":
				*doc = SetBSONDocField(*doc, prop, val)
			default:
				// Design property
				applyDesignProp(doc, prop, val)
			}
		}
		if clearDesignProps {
			*doc = SetBSONDocField(*doc, "DesignProperties", bson.D{})
		}
		return true
	}

	for i, e := range *doc {
		switch v := e.Value.(type) {
		case bson.D:
			if applyStyling(&v, widgetName, changes, clearDesignProps) {
				(*doc)[i].Value = v
				return true
			}
		case bson.A:
			if applyStylingArr(&v, widgetName, changes, clearDesignProps) {
				(*doc)[i].Value = v
				return true
			}
		}
	}
	return false
}

func applyStylingArr(arr *bson.A, widgetName string, changes map[string]string, clearDesignProps bool) bool {
	for i, item := range *arr {
		if v, ok := item.(bson.D); ok {
			if applyStyling(&v, widgetName, changes, clearDesignProps) {
				(*arr)[i] = v
				return true
			}
		}
	}
	return false
}

func applyDesignProp(doc *bson.D, prop, val string) {
	for i, e := range *doc {
		if e.Key == "DesignProperties" {
			if dp, ok := e.Value.(bson.D); ok {
				dp = SetBSONDocField(dp, prop, val)
				(*doc)[i].Value = dp
				return
			}
		}
	}
	*doc = append(*doc, bson.E{Key: "DesignProperties", Value: bson.D{
		{Key: prop, Value: val},
	}})
}

// CollectWidgetStyling recursively collects styling info from a BSON document tree.
// Returns rows of [widgetName, class, style].
func CollectWidgetStyling(data []byte, filterWidget string) ([][]string, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	var rows [][]string
	collectStyling(&doc, filterWidget, &rows)
	return rows, nil
}

func collectStyling(doc *bson.D, filterWidget string, rows *[][]string) {
	widgetName := ""
	class := ""
	style := ""

	for _, e := range *doc {
		switch e.Key {
		case "Name":
			if s, ok := e.Value.(string); ok {
				widgetName = s
			}
		case "Class":
			if s, ok := e.Value.(string); ok {
				class = s
			}
		case "Style":
			if s, ok := e.Value.(string); ok {
				style = s
			}
		}
	}

	if widgetName != "" && (class != "" || style != "") {
		if filterWidget == "" || widgetName == filterWidget {
			*rows = append(*rows, []string{widgetName, class, style})
		}
	}

	for _, e := range *doc {
		switch v := e.Value.(type) {
		case bson.D:
			collectStyling(&v, filterWidget, rows)
		case bson.A:
			for _, item := range v {
				if d, ok := item.(bson.D); ok {
					collectStyling(&d, filterWidget, rows)
				}
			}
		}
	}
}

// FindWidgetByName searches for a widget with the given name in a BSON doc tree.
// Returns a textual description of the widget subtree, or empty string if not found.
func FindWidgetByName(data []byte, name string) (bson.D, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return findWidget(&doc, name), nil
}

func findWidget(doc *bson.D, name string) bson.D {
	for _, e := range *doc {
		if e.Key == "Name" {
			if s, ok := e.Value.(string); ok && s == name {
				return *doc
			}
		}
	}
	for _, e := range *doc {
		switch v := e.Value.(type) {
		case bson.D:
			if result := findWidget(&v, name); result != nil {
				return result
			}
		case bson.A:
			for _, item := range v {
				if d, ok := item.(bson.D); ok {
					if result := findWidget(&d, name); result != nil {
						return result
					}
				}
			}
		}
	}
	return nil
}

// DescribeWidgetTree produces a textual representation of a widget subtree.
func DescribeWidgetTree(doc bson.D, indent int) string {
	var result string
	result = describeTree(&doc, indent)
	return result
}

func describeTree(doc *bson.D, indent int) string {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	widgetType := ""
	widgetName := ""
	for _, e := range *doc {
		if e.Key == "$Type" {
			widgetType, _ = e.Value.(string)
		}
		if e.Key == "Name" {
			widgetName, _ = e.Value.(string)
		}
	}
	result := ""
	if widgetType != "" {
		result += fmt.Sprintf("%s%s", prefix, widgetType)
		if widgetName != "" {
			result += fmt.Sprintf(" (%s)", widgetName)
		}
		result += "\n"
	}

	for _, e := range *doc {
		switch v := e.Value.(type) {
		case bson.D:
			hasType := false
			for _, f := range v {
				if f.Key == "$Type" {
					hasType = true
					break
				}
			}
			if hasType {
				result += describeTree(&v, indent+1)
			}
		case bson.A:
			for _, item := range v {
				if d, ok := item.(bson.D); ok {
					hasType := false
					for _, f := range d {
						if f.Key == "$Type" {
							hasType = true
							break
						}
					}
					if hasType {
						result += describeTree(&d, indent+1)
					}
				}
			}
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Widget update helpers
// ---------------------------------------------------------------------------

// WidgetFilter describes a filter condition for matching widgets.
type WidgetFilter struct {
	Field    string
	Operator string // "=" or "LIKE"
	Value    string
}

// WidgetAssignment describes a property to set on matched widgets.
type WidgetAssignment struct {
	PropertyPath string
	Value        any
}

// UpdateWidgets recursively traverses a BSON document looking for widget
// nodes that match the filters, applying assignments to matching ones.
// Returns (modifiedData, matchCount, error).
func UpdateWidgets(data []byte, filters []WidgetFilter, assignments []WidgetAssignment, dryRun bool) ([]byte, int, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, 0, fmt.Errorf("unmarshal: %w", err)
	}

	count := updateWidgetsDoc(&doc, filters, assignments, dryRun)

	if count > 0 && !dryRun {
		out, err := bson.Marshal(doc)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal: %w", err)
		}
		return out, count, nil
	}
	return data, count, nil
}

func updateWidgetsDoc(doc *bson.D, filters []WidgetFilter, assignments []WidgetAssignment, dryRun bool) int {
	count := 0
	for i, e := range *doc {
		switch v := e.Value.(type) {
		case bson.D:
			if matchesFilters(v, filters) {
				if !dryRun {
					for _, a := range assignments {
						v = SetBSONDocField(v, a.PropertyPath, a.Value)
					}
					(*doc)[i].Value = v
				}
				count++
			}
			count += updateWidgetsDoc(&v, filters, assignments, dryRun)
			(*doc)[i].Value = v
		case bson.A:
			count += updateWidgetsArr(&v, filters, assignments, dryRun)
			(*doc)[i].Value = v
		}
	}
	return count
}

func updateWidgetsArr(arr *bson.A, filters []WidgetFilter, assignments []WidgetAssignment, dryRun bool) int {
	count := 0
	for i, item := range *arr {
		switch v := item.(type) {
		case bson.D:
			if matchesFilters(v, filters) {
				if !dryRun {
					for _, a := range assignments {
						v = SetBSONDocField(v, a.PropertyPath, a.Value)
					}
					(*arr)[i] = v
				}
				count++
			}
			count += updateWidgetsDoc(&v, filters, assignments, dryRun)
			(*arr)[i] = v
		case bson.A:
			count += updateWidgetsArr(&v, filters, assignments, dryRun)
			(*arr)[i] = v
		}
	}
	return count
}

func matchesFilters(doc bson.D, filters []WidgetFilter) bool {
	if len(filters) == 0 {
		return false
	}
	for _, f := range filters {
		val := ""
		for _, e := range doc {
			if e.Key == f.Field || (f.Field == "WidgetType" && e.Key == "$Type") {
				val, _ = e.Value.(string)
				break
			}
		}
		if f.Operator == "=" && val != f.Value {
			return false
		}
		if f.Operator == "LIKE" {
			if len(val) == 0 || !containsStr(val, f.Value) {
				return false
			}
		}
	}
	return true
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// ALTER helpers for versioned arrays
// ---------------------------------------------------------------------------

// AppendToVersionedArray appends a builder's document to a versioned array
// field in a BSON document. If the array doesn't exist, creates it with
// the version marker int32(3).
func AppendToVersionedArray(data []byte, arrayKey string, item *BSONDocBuilder) ([]byte, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	found := false
	for i, e := range doc {
		if e.Key == arrayKey {
			if arr, ok := e.Value.(bson.A); ok {
				arr = append(arr, item.doc)
				doc[i].Value = arr
				found = true
			}
			break
		}
	}
	if !found {
		doc = append(doc, bson.E{Key: arrayKey, Value: bson.A{int32(3), item.doc}})
	}
	return bson.Marshal(doc)
}

// RemoveFromVersionedArrayByName removes an element from a versioned array
// where the element has a "Name" field matching the given name.
func RemoveFromVersionedArrayByName(data []byte, arrayKey, name string) ([]byte, error) {
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	for i, e := range doc {
		if e.Key == arrayKey {
			if arr, ok := e.Value.(bson.A); ok {
				newArr := bson.A{int32(3)}
				for _, item := range arr[1:] { // skip version marker
					if resDoc, ok := item.(bson.D); ok {
						nameMatch := false
						for _, field := range resDoc {
							if field.Key == "Name" {
								if s, ok := field.Value.(string); ok && s == name {
									nameMatch = true
								}
							}
						}
						if !nameMatch {
							newArr = append(newArr, item)
						}
					}
				}
				doc[i].Value = newArr
			}
			break
		}
	}
	return bson.Marshal(doc)
}
