package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"

	_ "modernc.org/sqlite"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type valueDiff struct {
	path  string
	field string
	old   string
	new   string
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s before.mpr after.mpr", os.Args[0])
	}
	before := extractWidget(os.Args[1], "dgItems")
	after := extractWidget(os.Args[2], "dgItems")
	if before == nil || after == nil {
		log.Fatal("dgItems not found")
	}
	diffs := compareValues("", before, after)
	if len(diffs) == 0 {
		fmt.Println("NO DIFFERENCES")
		return
	}
	fmt.Printf("Found %d differences:\n\n", len(diffs))
	for _, d := range diffs {
		fmt.Printf("[%s] %s:\n  BEFORE: %s\n  AFTER:  %s\n", d.path, d.field, d.old, d.new)
	}
}

func extractWidget(path string, name string) map[string]any {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT Contents FROM Unit WHERE length(Contents) > 10000")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var contents []byte
		if err := rows.Scan(&contents); err != nil {
			continue
		}
		var doc bson.D
		if err := bson.Unmarshal(contents, &doc); err != nil {
			continue
		}
		if r := findWidget(doc, name); r != nil {
			return r
		}
	}
	return nil
}

func findWidget(doc bson.D, name string) map[string]any {
	if hasWidgetID(doc) {
		var n string
		for _, elem := range doc {
			if elem.Key == "Name" {
				if s, ok := elem.Value.(string); ok {
					n = s
				}
			}
		}
		if n == name {
			if s, ok := simplify(doc).(map[string]any); ok {
				return s
			}
		}
	}
	for _, elem := range doc {
		if sub, ok := elem.Value.(bson.D); ok {
			if r := findWidget(sub, name); r != nil {
				return r
			}
		}
		if arr, ok := elem.Value.(bson.A); ok {
			for _, item := range arr {
				if sub, ok := item.(bson.D); ok {
					if r := findWidget(sub, name); r != nil {
						return r
					}
				}
			}
		}
	}
	return nil
}

func hasWidgetID(doc bson.D) bool {
	for _, elem := range doc {
		if elem.Key == "Type" {
			if sub, ok := elem.Value.(bson.D); ok {
				for _, se := range sub {
					if se.Key == "WidgetId" {
						return true
					}
				}
			}
		}
	}
	return false
}

func simplify(v any) any {
	switch val := v.(type) {
	case bson.D:
		m := make(map[string]any)
		for _, elem := range val {
			if elem.Key == "$ID" {
				continue
			}
			m[elem.Key] = simplify(elem.Value)
		}
		return m
	case bson.A:
		var result []any
		for _, item := range val {
			if s := simplify(item); s != nil {
				if _, ok := s.(map[string]any); ok {
					result = append(result, s)
				} else if _, ok := s.([]any); ok {
					result = append(result, s)
				}
			}
		}
		return result
	case bson.Binary:
		return "<BINARY>"
	case []byte:
		if len(val) == 16 {
			return "<BINARY>"
		}
		return val
	default:
		return val
	}
}

func compareValues(path string, old, new any) []valueDiff {
	var diffs []valueDiff
	oldStr := fmt.Sprintf("%v", old)
	newStr := fmt.Sprintf("%v", new)
	if oldStr == newStr {
		return nil
	}
	oldMap, oldIsMap := old.(map[string]any)
	newMap, newIsMap := new.(map[string]any)
	if oldIsMap || newIsMap {
		allKeys := make(map[string]bool)
		if oldIsMap { for k := range oldMap { allKeys[k] = true } }
		if newIsMap { for k := range newMap { allKeys[k] = true } }
		keys := sortedKeys(allKeys)
		for _, k := range keys {
			var ov, nv any
			oldExists, newExists := false, false
			if oldIsMap { if v, ok := oldMap[k]; ok { ov, oldExists = v, true } }
			if newIsMap { if v, ok := newMap[k]; ok { nv, newExists = v, true } }
			subPath := path + "." + k
			if !oldExists && newExists {
				diffs = append(diffs, valueDiff{path: subPath, field: "(added)", old: "<MISSING>", new: fmt.Sprintf("%v", simplify(nv))})
			} else if oldExists && !newExists {
				diffs = append(diffs, valueDiff{path: subPath, field: "(removed)", old: fmt.Sprintf("%v", simplify(ov)), new: "<MISSING>"})
			} else {
				diffs = append(diffs, compareValues(subPath, ov, nv)...)
			}
		}
		return diffs
	}
	o := fmt.Sprintf("%v", old)
	n := fmt.Sprintf("%v", new)
	if o != n {
		diffs = append(diffs, valueDiff{path: path, field: "(value)", old: o, new: n})
	}
	return diffs
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
