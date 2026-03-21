//go:build debug

package bson

// FieldCategory classifies BSON fields by their role in serialization.
type FieldCategory int

const (
	// Semantic fields carry business meaning (e.g., Name, Caption, Expression).
	Semantic FieldCategory = iota
	// Structural fields are BSON framing (e.g., $ID, $Type, PersistentId).
	Structural
	// Layout fields control visual positioning (e.g., RelativeMiddlePoint, Size).
	Layout
)

func (c FieldCategory) String() string {
	switch c {
	case Semantic:
		return "Semantic"
	case Structural:
		return "Structural"
	case Layout:
		return "Layout"
	default:
		return "Unknown"
	}
}

var structuralFields = map[string]bool{
	"$ID":          true,
	"$Type":        true,
	"PersistentId": true,
}

var layoutFields = map[string]bool{
	"RelativeMiddlePoint": true,
	"Size":                true,
}

// classifyField returns the FieldCategory for a given BSON storage name.
func classifyField(storageName string) FieldCategory {
	if structuralFields[storageName] {
		return Structural
	}
	if layoutFields[storageName] {
		return Layout
	}
	return Semantic
}
