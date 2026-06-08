package bson

import (
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Render converts a bson.D document to Normalized DSL text using ":" field separators.
// indent is the base indentation level (0 for top-level).
func Render(doc bson.D, indent int) string {
	var sb strings.Builder
	renderDoc(&sb, doc, indent, ": ")
	return strings.TrimRight(sb.String(), "\n")
}

// RenderForDiff converts a bson.D document to NDSL text using " =" field separators,
// suitable for use in unified diff output.
func RenderForDiff(doc bson.D, indent int) string {
	var sb strings.Builder
	renderDoc(&sb, doc, indent, " = ")
	return strings.TrimRight(sb.String(), "\n")
}

func renderDoc(sb *strings.Builder, doc bson.D, indent int, sep string) {
	pad := strings.Repeat("  ", indent)

	typeName := ""
	for _, e := range doc {
		if e.Key == "$Type" {
			typeName, _ = e.Value.(string)
			break
		}
	}
	if typeName != "" {
		sb.WriteString(pad + typeName + "\n")
	}

	renderFields(sb, doc, indent+1, sep)
}

// renderFields renders only the non-structural fields of a doc, sorted alphabetically.
func renderFields(sb *strings.Builder, doc bson.D, indent int, sep string) {
	type field struct {
		key string
		val any
	}
	var fields []field
	for _, e := range doc {
		if e.Key == "$ID" || e.Key == "$Type" {
			continue
		}
		fields = append(fields, field{e.Key, e.Value})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].key < fields[j].key
	})

	for _, f := range fields {
		renderField(sb, f.key, f.val, indent, sep)
	}
}

func renderField(sb *strings.Builder, key string, val any, indent int, sep string) {
	pad := strings.Repeat("  ", indent)

	switch v := val.(type) {
	case nil:
		fmt.Fprintf(sb, "%s%s%snull\n", pad, key, sep)

	case bson.Binary:
		fmt.Fprintf(sb, "%s%s%s<uuid>\n", pad, key, sep)

	case bson.D:
		typeName := ""
		for _, e := range v {
			if e.Key == "$Type" {
				typeName, _ = e.Value.(string)
				break
			}
		}
		if typeName != "" {
			fmt.Fprintf(sb, "%s%s%s%s\n", pad, key, sep, typeName)
		} else {
			fmt.Fprintf(sb, "%s%s%s\n", pad, key, sep)
		}
		renderFields(sb, v, indent+1, sep)

	case bson.A:
		renderArray(sb, key, v, indent, sep)

	case string:
		fmt.Fprintf(sb, "%s%s%s%q\n", pad, key, sep, v)

	case bool:
		fmt.Fprintf(sb, "%s%s%s%v\n", pad, key, sep, v)

	default:
		fmt.Fprintf(sb, "%s%s%s%v\n", pad, key, sep, v)
	}
}

func renderArray(sb *strings.Builder, key string, arr bson.A, indent int, sep string) {
	pad := strings.Repeat("  ", indent)

	markerStr := ""
	startIdx := 0
	if len(arr) > 0 {
		if marker, ok := arr[0].(int32); ok {
			markerStr = fmt.Sprintf(" [marker=%d]", marker)
			startIdx = 1
		}
	}

	elements := arr[startIdx:]
	if len(elements) == 0 {
		fmt.Fprintf(sb, "%s%s%s%s[]\n", pad, key, markerStr, sep)
		return
	}

	fmt.Fprintf(sb, "%s%s%s:\n", pad, key, markerStr)
	for _, elem := range elements {
		renderArrayElement(sb, elem, indent+1, sep)
	}
}

func renderArrayElement(sb *strings.Builder, elem any, indent int, sep string) {
	pad := strings.Repeat("  ", indent)

	switch v := elem.(type) {
	case bson.D:
		typeName := ""
		for _, e := range v {
			if e.Key == "$Type" {
				typeName, _ = e.Value.(string)
				break
			}
		}
		if typeName != "" {
			fmt.Fprintf(sb, "%s- %s\n", pad, typeName)
		} else {
			fmt.Fprintf(sb, "%s-\n", pad)
		}
		renderFields(sb, v, indent+2, sep)

	case string:
		fmt.Fprintf(sb, "%s- %q\n", pad, v)

	default:
		fmt.Fprintf(sb, "%s- %v\n", pad, elem)
	}
}
