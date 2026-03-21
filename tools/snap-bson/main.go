// SPDX-License-Identifier: Apache-2.0

// snap-bson extracts raw BSON units from an MPR file as JSON fixtures.
//
// Usage:
//
//	go run ./tools/snap-bson -mpr /path/to/app.mpr -type workflow
//	go run ./tools/snap-bson -mpr /path/to/app.mpr -type workflow -out sdk/mpr/testdata/workflows/
//	go run ./tools/snap-bson -mpr /path/to/app.mpr -type workflow -name "Module.WorkflowName"
//
// Supported types: workflow, microflow, nanoflow, page, snippet, layout, enumeration
//
// Output format: one JSON file per unit, named <ModuleName>.<UnitName>.json
// The JSON is the BSON document converted to readable JSON (arrays include int markers).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mpr "github.com/mendixlabs/mxcli/sdk/mpr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	mprPath := flag.String("mpr", "", "Path to .mpr file (required)")
	unitType := flag.String("type", "workflow", "Unit type: workflow, microflow, page, snippet, enumeration, nanoflow, layout")
	outDir := flag.String("out", "", "Output directory for fixtures (default: print to stdout)")
	unitName := flag.String("name", "", "Filter by qualified name (e.g. MyModule.MyWorkflow)")
	format := flag.String("format", "json", "Output format: json (human-readable) or bson (raw bytes for test fixtures)")
	flag.Parse()

	if *mprPath == "" {
		fmt.Fprintln(os.Stderr, "error: -mpr is required")
		flag.Usage()
		os.Exit(1)
	}

	r, err := mpr.Open(*mprPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening MPR: %v\n", err)
		os.Exit(1)
	}
	defer r.Close()

	units, err := r.ListRawUnits(*unitType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing units: %v\n", err)
		os.Exit(1)
	}

	if len(units) == 0 {
		fmt.Fprintf(os.Stderr, "no %s units found in %s\n", *unitType, *mprPath)
		os.Exit(0)
	}

	// Filter by name if specified
	if *unitName != "" {
		var filtered []*mpr.RawUnitInfo
		for _, u := range units {
			if strings.EqualFold(u.QualifiedName, *unitName) {
				filtered = append(filtered, u)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "no unit named %q found\n", *unitName)
			os.Exit(1)
		}
		units = filtered
	}

	if *outDir == "" {
		// List mode: print summary to stdout
		fmt.Printf("Found %d %s unit(s) in %s:\n\n", len(units), *unitType, filepath.Base(*mprPath))
		for _, u := range units {
			fmt.Printf("  %-50s  id=%s\n", u.QualifiedName, u.ID)
		}
		fmt.Printf("\nTo extract BSON fixtures (for tests):\n")
		fmt.Printf("  go run ./tools/snap-bson -mpr %s -type %s -format bson -out sdk/mpr/testdata/%ss/\n",
			*mprPath, *unitType, *unitType)
		fmt.Printf("\nTo extract JSON (human-readable debug):\n")
		fmt.Printf("  go run ./tools/snap-bson -mpr %s -type %s -format json -out /tmp/%ss/\n",
			*mprPath, *unitType, *unitType)
		return
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	for _, u := range units {
		safeName := strings.ReplaceAll(u.QualifiedName, "/", "_")

		if *format == "bson" {
			// Raw BSON bytes — for use as test fixtures via os.ReadFile + bson.Unmarshal
			outPath := filepath.Join(*outDir, safeName+".bson")
			if err := os.WriteFile(outPath, u.Contents, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
				continue
			}
			fmt.Printf("wrote: %s\n", outPath)
		} else {
			// Human-readable JSON — for debugging and inspection
			jsonBytes, err := bsonToJSON(u.Contents)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not convert %s to JSON: %v\n", u.QualifiedName, err)
				continue
			}
			outPath := filepath.Join(*outDir, safeName+".json")
			if err := os.WriteFile(outPath, jsonBytes, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
				continue
			}
			fmt.Printf("wrote: %s\n", outPath)
		}
	}
}

// bsonToJSON converts raw BSON bytes to indented JSON.
// BSON-specific types (Binary, ObjectID) are converted to string representations.
func bsonToJSON(contents []byte) ([]byte, error) {
	var raw bson.D
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}

	// Convert bson.D to a JSON-serializable map preserving key order
	ordered := bsonDToOrdered(raw)
	return json.MarshalIndent(ordered, "", "  ")
}

// orderedMap preserves key insertion order for JSON output.
type orderedMap struct {
	keys []string
	vals map[string]any
}

func (o *orderedMap) set(k string, v any) {
	if _, exists := o.vals[k]; !exists {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *orderedMap) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("{")
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteString(",")
		}
		keyBytes, _ := json.Marshal(k)
		buf.Write(keyBytes)
		buf.WriteString(":")
		valBytes, err := json.Marshal(o.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteString("}")
	return []byte(buf.String()), nil
}

func newOrderedMap() *orderedMap {
	return &orderedMap{vals: make(map[string]any)}
}

// bsonDToOrdered recursively converts bson.D to ordered JSON-serializable structures.
func bsonDToOrdered(d bson.D) any {
	m := newOrderedMap()
	for _, e := range d {
		m.set(e.Key, bsonValueToJSON(e.Value))
	}
	return m
}

// bsonValueToJSON converts a BSON value to a JSON-compatible Go value.
func bsonValueToJSON(v any) any {
	switch val := v.(type) {
	case bson.D:
		return bsonDToOrdered(val)
	case bson.A:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = bsonValueToJSON(item)
		}
		return result
	case primitive.Binary:
		// Represent as hex string with type prefix for readability
		return fmt.Sprintf("Binary(%x)", val.Data)
	default:
		return val
	}
}
