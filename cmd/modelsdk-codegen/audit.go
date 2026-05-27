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
func auditAliases(paths []string, registeredTypes map[string]bool, regFields map[string]map[string]bool) {
	// Collect $Type counts AND field sets in one pass.
	typeFields := map[string]map[string]bool{}
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

	// Find gaps
	type gap struct {
		TypeName   string
		Count      int
		Candidate  string
		Method     string
		Confidence string
	}
	var gaps []gap
	for t, fields := range typeFields {
		if !registeredTypes[t] {
			candidate, method, confidence := findCandidateWithFields(t, fields, registeredTypes, regFields)
			gaps = append(gaps, gap{t, len(fields), candidate, method, confidence})
		}
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].Count > gaps[j].Count })

	fmt.Printf("\nScanned %d unique $Type values from %d source(s)\n", len(typeFields), len(paths))
	fmt.Printf("Registered in codegen: %d\n", len(registeredTypes))
	fmt.Printf("Gaps: %d\n\n", len(gaps))

	if len(gaps) == 0 {
		fmt.Println("No gaps found — all BSON $Type values have registered factories.")
		return
	}

	var actionable, lowConf, orphan []gap
	for _, g := range gaps {
		switch {
		case g.Candidate == "":
			orphan = append(orphan, g)
		case g.Confidence == "low":
			lowConf = append(lowConf, g)
		default:
			actionable = append(actionable, g)
		}
	}

	if len(actionable) > 0 {
		fmt.Println("=== ALIAS CANDIDATES (add to storage_aliases in supplements.json) ===")
		fmt.Println()
		for _, g := range actionable {
			fmt.Printf("  %6dx  %-50s → %s  (%s)\n", g.Count, g.TypeName, g.Candidate, g.Method)
		}
		fmt.Println()
		fmt.Println("Suggested supplements.json entries:")
		fmt.Println()
		for _, g := range actionable {
			fmt.Printf("\t\"%s\": \"%s\",\n", g.Candidate, g.TypeName)
		}
		fmt.Println()
	}

	if len(lowConf) > 0 {
		fmt.Println("=== LOW-CONFIDENCE MATCHES (field Jaccard < 0.5 — review manually) ===")
		fmt.Println()
		for _, g := range lowConf {
			fmt.Printf("  %6dx  %-50s → %s  (%s)\n", g.Count, g.TypeName, g.Candidate, g.Method)
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

// findCandidateWithFields applies heuristics to find a registered type that
// likely corresponds to the given BSON $Type, then validates the match with
// field fingerprint similarity (Jaccard). Returns (candidate, method, confidence)
// where confidence is "low" when Jaccard < 0.5.
//
// When no name heuristic produces a confident match, falls back to scanning
// all registered types for the best field-fingerprint match ("field-match").
func findCandidateWithFields(
	bsonType string,
	bsonFields map[string]bool,
	registered map[string]bool,
	regFields map[string]map[string]bool,
) (candidate, method, confidence string) {
	ns, cls := splitType(bsonType)

	// Collect name-heuristic candidates (ordered by specificity).
	type nameHit struct{ cand, method string }
	var hits []nameHit

	// 1. Impl suffix strip
	if strings.HasSuffix(cls, "Impl") {
		base := cls[:len(cls)-4]
		if tryMatch(ns, base, registered) {
			hits = append(hits, nameHit{ns + "$" + base, "strip-Impl"})
		}
		for _, altNs := range collectNamespaces(registered) {
			if altNs != ns && tryMatch(altNs, base, registered) {
				hits = append(hits, nameHit{altNs + "$" + base, "strip-Impl+ns"})
			}
		}
	}

	// 2. Add Impl suffix
	if tryMatch(ns, cls+"Impl", registered) {
		hits = append(hits, nameHit{ns + "$" + cls + "Impl", "add-Impl"})
	}

	// 3. Marker suffix strip
	if strings.HasSuffix(cls, "Marker") {
		base := cls[:len(cls)-6]
		if tryMatch(ns, base, registered) {
			hits = append(hits, nameHit{ns + "$" + base, "strip-Marker"})
		}
	}

	// 4. Namespace swap (same class name, different namespace)
	for _, altNs := range collectNamespaces(registered) {
		if altNs != ns && registered[altNs+"$"+cls] {
			hits = append(hits, nameHit{altNs + "$" + cls, "ns-swap"})
		}
	}

	// 5. Prefix strip/add
	for _, prefix := range []string{"Export", "Import", "Published", "Consumed"} {
		if strings.HasPrefix(cls, prefix) {
			stripped := cls[len(prefix):]
			if tryMatch(ns, stripped, registered) {
				hits = append(hits, nameHit{ns + "$" + stripped, "strip-" + prefix})
			}
		}
		if tryMatch(ns, prefix+cls, registered) {
			hits = append(hits, nameHit{ns + "$" + prefix + cls, "add-" + prefix})
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
			hits = append(hits, nameHit{ns + "$" + swapped, "Form↔Page"})
		}
		if registered["Forms$"+swapped] {
			hits = append(hits, nameHit{"Forms$" + swapped, "Form↔Page+ns"})
		}
	}

	// Validate each name-heuristic hit with field Jaccard.
	const threshold = 0.5
	bestScore := -1.0
	bestHit := nameHit{}
	for _, h := range hits {
		score := jaccardSimilarity(bsonFields, regFields[h.cand])
		if score > bestScore {
			bestScore = score
			bestHit = h
		}
	}
	// Run field-fingerprint scan over all registered types.
	// This runs whether or not a name heuristic matched, so a strong field-match
	// can override a low-confidence name hit.
	var fbest string
	fbScore := -1.0
	if len(bsonFields) > 0 {
		for t := range registered {
			s := jaccardSimilarity(bsonFields, regFields[t])
			if s > fbScore {
				fbScore = s
				fbest = t
			}
		}
	}

	if bestScore >= threshold {
		// Name heuristic matched with sufficient field similarity.
		return bestHit.cand, bestHit.method, "ok"
	}
	if fbScore >= 0.6 && fbScore > bestScore {
		// Field-match beats the name heuristic candidate.
		return fbest, "field-match", "ok"
	}
	if bestScore >= 0 {
		// Name heuristic fired but field similarity is low and no better field-match found.
		return bestHit.cand, bestHit.method, "low"
	}

	return "", "", ""
}

// jaccardSimilarity returns |A∩B| / |A∪B|, or 0 for empty sets.
func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	var inter int
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// collectRegisteredFields scans generated types.go files for property key names
// per registered $Type. Returns map[storageTypeName]set[bsonKey].
func collectRegisteredFields(genBase string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	entries, err := os.ReadDir(genBase)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(genBase, e.Name(), "types.go"))
		if err != nil {
			continue
		}
		parseGenFields(string(data), result)
	}
	return result
}

// parseGenFields extracts (typeName → fieldSet) from a single types.go file.
// It tracks the current type by watching o.SetTypeName("Ns$Type") lines,
// then collects property.NewXxx[...]("FieldName", ...) field names.
func parseGenFields(src string, out map[string]map[string]bool) {
	const setTypePrefix = `o.SetTypeName("`
	const newPropMarker = `property.New`
	var currentType string
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, setTypePrefix) {
			rest := trimmed[len(setTypePrefix):]
			if end := strings.IndexByte(rest, '"'); end > 0 {
				currentType = rest[:end]
				if out[currentType] == nil {
					out[currentType] = map[string]bool{}
				}
			}
			continue
		}

		if currentType == "" || !strings.Contains(trimmed, newPropMarker) {
			continue
		}
		// Extract first string argument: property.NewXxx[...]("FieldName", ...)
		// Find the opening paren of the type parameter list: NewXxx[
		idx := strings.Index(trimmed, newPropMarker)
		rest := trimmed[idx:]
		// Skip to closing '](' which precedes the first argument.
		bracket := strings.Index(rest, "](")
		if bracket < 0 {
			continue
		}
		rest = rest[bracket+2:] // rest starts at first argument
		if len(rest) == 0 || rest[0] != '"' {
			continue
		}
		rest = rest[1:]
		if end := strings.IndexByte(rest, '"'); end > 0 {
			out[currentType][rest[:end]] = true
		}
	}
}

// fieldDiff holds the result of comparing gen-expected vs BSON-actual field sets.
type fieldDiff struct {
	genOnly  map[string]bool // in gen but not in BSON (possible wrong key name)
	bsonOnly map[string]bool // in BSON but not in gen (possible missing or renamed field)
}

// diffTypeFields computes the symmetric difference between gen and BSON field sets.
// System keys ($ID, $Type) in the BSON set are ignored.
func diffTypeFields(genKeys, bsonKeys map[string]bool) fieldDiff {
	d := fieldDiff{
		genOnly:  map[string]bool{},
		bsonOnly: map[string]bool{},
	}
	for k := range genKeys {
		if !bsonKeys[k] {
			d.genOnly[k] = true
		}
	}
	for k := range bsonKeys {
		if strings.HasPrefix(k, "$") {
			continue // skip $ID, $Type
		}
		if !genKeys[k] {
			d.bsonOnly[k] = true
		}
	}
	return d
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

// auditPropertyKeys performs a full field-key diff between gen-expected keys and
// actual BSON keys for every registered type seen in the corpus.
//
// For each type that appears in the corpus AND has registered gen fields:
//   - gen-only keys that have a BSON counterpart with "Pointer" suffix → suggest
//     property_key_override (existing behavior, now applied to ALL property types)
//   - gen-only keys with no BSON counterpart → needs-review (optional field or
//     version difference)
//   - bson-only keys with no gen counterpart → possible extra_property candidate
func auditPropertyKeys(paths []string, regFields map[string]map[string]bool) {
	// Collect BSON keys per $Type.
	typeFields := map[string]map[string]bool{}
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

	type keyMismatch struct {
		typeName   string
		genKey     string
		bsonKey    string // non-empty when a rename was found
		suggestion string
	}
	var overrides []keyMismatch
	var needsReview []keyMismatch
	var bsonOnly []keyMismatch
	checked, okCount := 0, 0

	// Sort type names for deterministic output.
	typeNames := make([]string, 0, len(regFields))
	for t := range regFields {
		typeNames = append(typeNames, t)
	}
	sort.Strings(typeNames)

	for _, typeName := range typeNames {
		genKeys := regFields[typeName]
		bsonKeys, seen := typeFields[typeName]
		if !seen {
			continue // type not in corpus, skip
		}
		diff := diffTypeFields(genKeys, bsonKeys)
		checked += len(genKeys)
		okCount += len(genKeys) - len(diff.genOnly)

		_, cls := splitType(typeName)

		for genKey := range diff.genOnly {
			// Check if BSON has a Pointer-suffix variant.
			if bsonKeys[genKey+"Pointer"] {
				overrides = append(overrides, keyMismatch{
					typeName:   typeName,
					genKey:     genKey,
					bsonKey:    genKey + "Pointer",
					suggestion: fmt.Sprintf("%q: %q,", cls+"."+lcFirst(genKey), genKey+"Pointer"),
				})
			} else {
				needsReview = append(needsReview, keyMismatch{typeName: typeName, genKey: genKey})
			}
		}
		for bsonKey := range diff.bsonOnly {
			bsonOnly = append(bsonOnly, keyMismatch{typeName: typeName, bsonKey: bsonKey})
		}
	}

	sort.Slice(overrides, func(i, j int) bool {
		return overrides[i].typeName < overrides[j].typeName || overrides[i].genKey < overrides[j].genKey
	})
	sort.Slice(needsReview, func(i, j int) bool { return needsReview[i].typeName < needsReview[j].typeName })
	sort.Slice(bsonOnly, func(i, j int) bool { return bsonOnly[i].typeName < bsonOnly[j].typeName })

	fmt.Printf("\nFull Property Key Audit\n")
	fmt.Printf("  Types checked: %d  Fields checked: %d  OK: %d\n", len(typeFields), checked, okCount)
	fmt.Printf("  Pointer-rename candidates: %d  Needs-review: %d  BSON-only: %d\n\n",
		len(overrides), len(needsReview), len(bsonOnly))

	if len(overrides) > 0 {
		fmt.Println("=== POINTER-RENAME (add to property_key_overrides in supplements.json) ===")
		fmt.Println()
		for _, m := range overrides {
			fmt.Printf("  %-50s  gen=%q → bson=%q\n", m.typeName, m.genKey, m.bsonKey)
			fmt.Printf("    %s\n", m.suggestion)
		}
		fmt.Println()
	}

	if len(needsReview) > 0 {
		fmt.Println("=== NEEDS REVIEW (gen key not in BSON — optional field or version diff) ===")
		fmt.Println()
		for _, m := range needsReview {
			fmt.Printf("  %-50s  gen=%q\n", m.typeName, m.genKey)
		}
		fmt.Println()
	}

	if len(bsonOnly) > 0 {
		fmt.Println("=== BSON-ONLY KEYS (not in gen — possible extra_property candidate) ===")
		fmt.Println()
		for _, m := range bsonOnly {
			fmt.Printf("  %-50s  bson=%q\n", m.typeName, m.bsonKey)
		}
		fmt.Println()
	}

	if len(overrides) == 0 && len(needsReview) == 0 && len(bsonOnly) == 0 {
		fmt.Println("No mismatches found.")
	}
}

// lcFirst lowercases the first byte of s (ASCII only).
func lcFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
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
