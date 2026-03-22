//go:build debug

package bson

import (
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Render converts a bson.D document to Normalized DSL text.
// indent is the base indentation level (0 for top-level).
func Render(doc bson.D, indent int) string {
	var sb strings.Builder
	renderDoc(&sb, doc, indent)
	return strings.TrimRight(sb.String(), "\n")
}

func renderDoc(sb *strings.Builder, doc bson.D, indent int) {
	pad := strings.Repeat("  ", indent)

	// Extract $Type for header
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

	// Collect non-structural fields, sort alphabetically
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
		renderField(sb, f.key, f.val, indent+1)
	}
}

func renderField(sb *strings.Builder, key string, val any, indent int) {
	pad := strings.Repeat("  ", indent)

	switch v := val.(type) {
	case nil:
		fmt.Fprintf(sb, "%s%s: null\n", pad, key)

	case primitive.Binary:
		fmt.Fprintf(sb, "%s%s: <uuid>\n", pad, key)

	case bson.D:
		typeName := ""
		for _, e := range v {
			if e.Key == "$Type" {
				typeName, _ = e.Value.(string)
				break
			}
		}
		if typeName != "" {
			fmt.Fprintf(sb, "%s%s: %s\n", pad, key, typeName)
			renderDoc(sb, v, indent+1)
		} else {
			fmt.Fprintf(sb, "%s%s:\n", pad, key)
			renderDoc(sb, v, indent+1)
		}

	case bson.A:
		renderArray(sb, key, v, indent)

	case string:
		fmt.Fprintf(sb, "%s%s: %q\n", pad, key, v)

	case bool:
		fmt.Fprintf(sb, "%s%s: %v\n", pad, key, v)

	default:
		fmt.Fprintf(sb, "%s%s: %v\n", pad, key, v)
	}
}

func renderArray(sb *strings.Builder, key string, arr bson.A, indent int) {
	pad := strings.Repeat("  ", indent)

	// Check for array marker (first element is int32)
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
		fmt.Fprintf(sb, "%s%s%s: []\n", pad, key, markerStr)
		return
	}

	fmt.Fprintf(sb, "%s%s%s:\n", pad, key, markerStr)
	for _, elem := range elements {
		renderArrayElement(sb, elem, indent+1)
	}
}

func renderArrayElement(sb *strings.Builder, elem any, indent int) {
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
			type field struct {
				key string
				val any
			}
			var fields []field
			for _, e := range v {
				if e.Key == "$ID" || e.Key == "$Type" {
					continue
				}
				fields = append(fields, field{e.Key, e.Value})
			}
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].key < fields[j].key
			})
			for _, f := range fields {
				renderField(sb, f.key, f.val, indent+2)
			}
		} else {
			fmt.Fprintf(sb, "%s-\n", pad)
			renderDoc(sb, v, indent+2)
		}

	case string:
		fmt.Fprintf(sb, "%s- %q\n", pad, v)

	default:
		fmt.Fprintf(sb, "%s- %v\n", pad, elem)
	}
}
