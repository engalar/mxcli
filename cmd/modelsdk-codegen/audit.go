package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	_ "modernc.org/sqlite"
)

// auditAliases scans MPR files for all $Type values, compares against the
// codegen registry, and reports gaps with heuristic alias suggestions.
//
// Usage: modelsdk-codegen -audit <mpr-or-dir> [<mpr-or-dir> ...]
//
// For each gap, applies these heuristics to find a candidate match:
//  1. Namespace swap (ExportXmlAction$ → Microflows$)
//  2. Impl/Marker suffix strip/add
//  3. Prefix strip (ExportObjectMappingElement → ObjectMappingElement)
//  4. BSON field fingerprint matching against registered types
func auditAliases(paths []string, registeredTypes map[string]bool) {
	// Collect all $Type values from BSON
	bsonTypes := map[string]int{}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", p, err)
			continue
		}
		if info.IsDir() {
			scanDir(p, bsonTypes)
		} else {
			scanSQLite(p, bsonTypes)
		}
	}

	// Find gaps
	type gap struct {
		TypeName  string
		Count     int
		Candidate string
		Method    string
	}
	var gaps []gap
	for t, c := range bsonTypes {
		if !registeredTypes[t] {
			candidate, method := findCandidate(t, registeredTypes)
			gaps = append(gaps, gap{t, c, candidate, method})
		}
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].Count > gaps[j].Count })

	// Report
	fmt.Printf("\nScanned %d unique $Type values from %d source(s)\n", len(bsonTypes), len(paths))
	fmt.Printf("Registered in codegen: %d\n", len(registeredTypes))
	fmt.Printf("Gaps: %d\n\n", len(gaps))

	if len(gaps) == 0 {
		fmt.Println("No gaps found — all BSON $Type values have registered factories.")
		return
	}

	// Separate into actionable (has candidate) and orphan (no candidate)
	var actionable, orphan []gap
	for _, g := range gaps {
		if g.Candidate != "" {
			actionable = append(actionable, g)
		} else {
			orphan = append(orphan, g)
		}
	}

	if len(actionable) > 0 {
		fmt.Println("=== ALIAS CANDIDATES (add to storage_aliases in supplements.json) ===")
		fmt.Println()
		for _, g := range actionable {
			fmt.Printf("  %6dx  %-50s → %s  (%s)\n", g.Count, g.TypeName, g.Candidate, g.Method)
		}
		fmt.Println()
		fmt.Println("Suggested Go map entries:")
		fmt.Println()
		for _, g := range actionable {
			fmt.Printf("\t\"%s\": \"%s\",\n", g.Candidate, g.TypeName)
		}
		fmt.Println()
	}

	if len(orphan) > 0 {
		fmt.Println("=== ORPHAN TYPES (no SDK match, will round-trip as element.Base) ===")
		fmt.Println()
		for _, g := range orphan {
			fmt.Printf("  %6dx  %s\n", g.Count, g.TypeName)
		}
		fmt.Println()
	}
}

// findCandidate applies heuristics to find a registered type that likely
// corresponds to the given BSON $Type.
func findCandidate(bsonType string, registered map[string]bool) (candidate, method string) {
	ns, cls := splitType(bsonType)

	// 1. Impl suffix: FooImpl → Foo
	if strings.HasSuffix(cls, "Impl") {
		base := cls[:len(cls)-4]
		if tryMatch(ns, base, registered) {
			return ns + "$" + base, "strip-Impl"
		}
		// Try other namespaces
		for _, altNs := range collectNamespaces(registered) {
			if tryMatch(altNs, base, registered) {
				return altNs + "$" + base, "strip-Impl+ns"
			}
		}
	}

	// 2. Add Impl suffix: Foo → FooImpl
	if tryMatch(ns, cls+"Impl", registered) {
		return ns + "$" + cls + "Impl", "add-Impl"
	}

	// 3. Marker suffix: FooMarker → Foo or Foo → FooMarker
	if strings.HasSuffix(cls, "Marker") {
		base := cls[:len(cls)-6]
		if tryMatch(ns, base, registered) {
			return ns + "$" + base, "strip-Marker"
		}
	}

	// 4. Namespace swap: check all registered namespaces
	for _, altNs := range collectNamespaces(registered) {
		if altNs == ns {
			continue
		}
		if registered[altNs+"$"+cls] {
			return altNs + "$" + cls, "ns-swap"
		}
	}

	// 5. Prefix strip: ExportObjectMappingElement → ObjectMappingElement
	//    Try removing common prefixes (Export, Import, Published, Consumed)
	for _, prefix := range []string{"Export", "Import", "Published", "Consumed"} {
		if strings.HasPrefix(cls, prefix) {
			stripped := cls[len(prefix):]
			if tryMatch(ns, stripped, registered) {
				return ns + "$" + stripped, "strip-" + prefix
			}
		}
		// Or add prefix
		prefixed := prefix + cls
		if tryMatch(ns, prefixed, registered) {
			return ns + "$" + prefixed, "add-" + prefix
		}
	}

	// 6. Form/Page swap
	swapped := cls
	if strings.Contains(cls, "Form") {
		swapped = strings.ReplaceAll(cls, "Form", "Page")
	} else if strings.Contains(cls, "Page") {
		swapped = strings.ReplaceAll(cls, "Page", "Form")
	}
	if swapped != cls {
		if tryMatch(ns, swapped, registered) {
			return ns + "$" + swapped, "Form↔Page"
		}
		// Also try Forms namespace
		if registered["Forms$"+swapped] {
			return "Forms$" + swapped, "Form↔Page+ns"
		}
	}

	return "", ""
}

func splitType(t string) (ns, cls string) {
	if idx := strings.IndexByte(t, '$'); idx > 0 {
		return t[:idx], t[idx+1:]
	}
	return "", t
}

func tryMatch(ns, cls string, registered map[string]bool) bool {
	return registered[ns+"$"+cls]
}

func collectNamespaces(registered map[string]bool) []string {
	nsSet := map[string]bool{}
	for t := range registered {
		if ns, _ := splitType(t); ns != "" {
			nsSet[ns] = true
		}
	}
	nsList := make([]string, 0, len(nsSet))
	for ns := range nsSet {
		nsList = append(nsList, ns)
	}
	sort.Strings(nsList)
	return nsList
}

// ── audit-keys: ByIdRef BSON key mismatch detection ──

// byIdRefEntry describes a single ByIdRef property declaration found in generated code.
type byIdRefEntry struct {
	TypeName string // e.g. "Microflows$SequenceFlow"
	BSONKey  string // key passed to NewByIdRef (e.g. "Origin" or "OriginPointer")
}

// collectByIdRefKeys scans generated types.go files for NewByIdRef[...]("Key") calls.
func collectByIdRefKeys(genBase string) []byIdRefEntry {
	var entries []byIdRefEntry
	dirs, _ := os.ReadDir(genBase)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		typesFile := filepath.Join(genBase, d.Name(), "types.go")
		data, err := os.ReadFile(typesFile)
		if err != nil {
			continue
		}
		content := string(data)
		// Find each init function and its type name, then find NewByIdRef calls inside
		var currentType string
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			const prefix = `o.SetTypeName("`
			if strings.HasPrefix(trimmed, prefix) {
				rest := trimmed[len(prefix):]
				end := strings.IndexByte(rest, '"')
				if end > 0 {
					currentType = rest[:end]
				}
			}
			if strings.Contains(trimmed, "property.NewByIdRef[") && currentType != "" {
				// Extract the BSON key from: property.NewByIdRef[element.Element]("Key")
				// Find the last ("..." pattern which is the constructor argument
				const marker = `](`
				idx := strings.Index(trimmed, marker)
				if idx < 0 {
					continue
				}
				rest := trimmed[idx+len(marker):]
				if len(rest) < 3 || rest[0] != '"' {
					continue
				}
				end := strings.IndexByte(rest[1:], '"')
				if end < 0 {
					continue
				}
				bsonKey := rest[1 : 1+end]
				entries = append(entries, byIdRefEntry{TypeName: currentType, BSONKey: bsonKey})
			}
		}
	}
	return entries
}

// auditPropertyKeys scans MPR BSON for ByIdRef key mismatches.
// For each registered ByIdRef property, checks if the BSON key exists in actual
// documents. If not, checks if a "Pointer" suffix variant exists.
func auditPropertyKeys(paths []string, refs []byIdRefEntry) {
	// Collect BSON keys per $Type from all MPR sources.
	typeFields := map[string]map[string]bool{} // $Type -> set of BSON keys
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", p, err)
			continue
		}
		if info.IsDir() {
			walkDirForKeys(p, typeFields)
		} else {
			walkSQLiteForKeys(p, typeFields)
		}
	}

	// Compare
	type mismatch struct {
		TypeName   string
		CodegenKey string
		BSONKey    string
	}
	var mismatches []mismatch
	var ok, missing int

	for _, ref := range refs {
		fields, found := typeFields[ref.TypeName]
		if !found {
			continue // type not in any MPR, skip
		}
		if fields[ref.BSONKey] {
			ok++
			continue
		}
		// Try Pointer suffix
		pointerKey := ref.BSONKey + "Pointer"
		if fields[pointerKey] {
			mismatches = append(mismatches, mismatch{ref.TypeName, ref.BSONKey, pointerKey})
		} else {
			missing++
		}
	}

	fmt.Printf("\nByIdRef BSON Key Audit\n")
	fmt.Printf("  Checked: %d ByIdRef properties\n", ok+len(mismatches)+missing)
	fmt.Printf("  OK: %d\n", ok)
	fmt.Printf("  Mismatches: %d\n", len(mismatches))
	fmt.Printf("  No instances: %d\n\n", missing)

	if len(mismatches) == 0 {
		fmt.Println("No mismatches found.")
		return
	}

	fmt.Println("=== MISMATCHES (add to property_key_overrides in supplements.json) ===")
	fmt.Println()
	for _, m := range mismatches {
		_, cls := splitType(m.TypeName)
		fmt.Printf("  %-50s codegen=%q  bson=%q\n", m.TypeName, m.CodegenKey, m.BSONKey)
		// Suggest override entry
		fmt.Printf("    \"%s.???\":  \"%s\",\n\n", cls, m.BSONKey)
	}
}

func walkDirForKeys(dir string, typeFields map[string]map[string]bool) {
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".mxunit" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		var doc bson.D
		if bson.Unmarshal(data, &doc) == nil {
			walkDocKeys(doc, typeFields)
		}
		return nil
	})
}

func walkSQLiteForKeys(dbPath string, typeFields map[string]map[string]bool) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT Contents FROM Unit WHERE Contents IS NOT NULL AND LENGTH(Contents) > 0")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var contents []byte
		if err := rows.Scan(&contents); err != nil {
			continue
		}
		var doc bson.D
		if bson.Unmarshal(contents, &doc) == nil {
			walkDocKeys(doc, typeFields)
		}
	}
}

func walkDocKeys(doc bson.D, typeFields map[string]map[string]bool) {
	var typeName string
	for _, elem := range doc {
		if elem.Key == "$Type" {
			if s, ok := elem.Value.(string); ok {
				typeName = s
			}
		}
	}
	if typeName != "" {
		if typeFields[typeName] == nil {
			typeFields[typeName] = map[string]bool{}
		}
		for _, elem := range doc {
			typeFields[typeName][elem.Key] = true
		}
	}
	// Recurse
	for _, elem := range doc {
		switch v := elem.Value.(type) {
		case bson.D:
			walkDocKeys(v, typeFields)
		case bson.A:
			walkArrayKeys(v, typeFields)
		}
	}
}

func walkArrayKeys(arr bson.A, typeFields map[string]map[string]bool) {
	for _, item := range arr {
		switch v := item.(type) {
		case bson.D:
			walkDocKeys(v, typeFields)
		case bson.A:
			walkArrayKeys(v, typeFields)
		}
	}
}

// --- BSON scanning ---

func scanDir(dir string, types map[string]int) {
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".mxunit" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		var doc bson.D
		if bson.Unmarshal(data, &doc) == nil {
			walkDoc(doc, types)
		}
		return nil
	})
}

func scanSQLite(dbPath string, types map[string]int) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT Contents FROM Unit WHERE Contents IS NOT NULL AND LENGTH(Contents) > 0")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var contents []byte
		if err := rows.Scan(&contents); err != nil {
			continue
		}
		var doc bson.D
		if bson.Unmarshal(contents, &doc) == nil {
			walkDoc(doc, types)
		}
	}
}

func walkDoc(doc bson.D, types map[string]int) {
	for _, elem := range doc {
		if elem.Key == "$Type" {
			if s, ok := elem.Value.(string); ok {
				types[s]++
			}
		}
		switch v := elem.Value.(type) {
		case bson.D:
			walkDoc(v, types)
		case bson.A:
			walkArray(v, types)
		}
	}
}

func walkArray(arr bson.A, types map[string]int) {
	for _, item := range arr {
		switch v := item.(type) {
		case bson.D:
			walkDoc(v, types)
		case bson.A:
			walkArray(v, types)
		}
	}
}
