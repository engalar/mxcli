// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bsondebug "github.com/mendixlabs/mxcli/cmd/mxcli/bson"
	"go.mongodb.org/mongo-driver/bson"
)

// testReader creates a minimal Reader for roundtrip tests (no database needed).
func testReader() *Reader {
	return &Reader{version: MPRVersionV1}
}

// testWriter creates a minimal Writer for roundtrip tests (no database needed).
func testWriter() *Writer {
	return &Writer{reader: testReader()}
}

// toNDSL unmarshals raw BSON bytes and renders as Normalized DSL text.
func toNDSL(t *testing.T, data []byte) string {
	t.Helper()
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to unmarshal BSON: %v", err)
	}
	return bsondebug.Render(doc, 0)
}

// roundtripPage: baseline → parse → serialize → parse → serialize → compare two serializations.
// Verifies serialization idempotency. Original baseline is preserved as ground truth.
func roundtripPage(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	// First pass: baseline → parse → serialize
	page1, err := r.parsePage("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parsePage (pass 1) failed: %v", err)
	}
	serialized1, err := w.serializePage(page1)
	if err != nil {
		t.Fatalf("serializePage (pass 1) failed: %v", err)
	}

	// Second pass: serialized → parse → serialize
	page2, err := r.parsePage("test-unit-id", "test-container-id", serialized1)
	if err != nil {
		t.Fatalf("parsePage (pass 2) failed: %v", err)
	}
	serialized2, err := w.serializePage(page2)
	if err != nil {
		t.Fatalf("serializePage (pass 2) failed: %v", err)
	}

	ndsl1 := toNDSL(t, serialized1)
	ndsl2 := toNDSL(t, serialized2)

	if ndsl1 != ndsl2 {
		t.Errorf("serialization not idempotent for page %q\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s\n--- diff ---\n%s",
			page1.Name, ndsl1, ndsl2, ndslDiff(ndsl1, ndsl2))
	}
}

// roundtripMicroflow: baseline → parse → serialize → parse → serialize → compare two serializations.
func roundtripMicroflow(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	// First pass
	mf1, err := r.parseMicroflow("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseMicroflow (pass 1) failed: %v", err)
	}
	serialized1, err := w.serializeMicroflow(mf1)
	if err != nil {
		t.Fatalf("serializeMicroflow (pass 1) failed: %v", err)
	}

	// Second pass
	mf2, err := r.parseMicroflow("test-unit-id", "test-container-id", serialized1)
	if err != nil {
		t.Fatalf("parseMicroflow (pass 2) failed: %v", err)
	}
	serialized2, err := w.serializeMicroflow(mf2)
	if err != nil {
		t.Fatalf("serializeMicroflow (pass 2) failed: %v", err)
	}

	ndsl1 := toNDSL(t, serialized1)
	ndsl2 := toNDSL(t, serialized2)

	if ndsl1 != ndsl2 {
		t.Errorf("serialization not idempotent for microflow %q\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s\n--- diff ---\n%s",
			mf1.Name, ndsl1, ndsl2, ndslDiff(ndsl1, ndsl2))
	}
}

// roundtripSnippet: double roundtrip idempotency test.
func roundtripSnippet(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	snippet1, err := r.parseSnippet("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseSnippet (pass 1) failed: %v", err)
	}
	serialized1, err := w.serializeSnippet(snippet1)
	if err != nil {
		t.Fatalf("serializeSnippet (pass 1) failed: %v", err)
	}

	snippet2, err := r.parseSnippet("test-unit-id", "test-container-id", serialized1)
	if err != nil {
		t.Fatalf("parseSnippet (pass 2) failed: %v", err)
	}
	serialized2, err := w.serializeSnippet(snippet2)
	if err != nil {
		t.Fatalf("serializeSnippet (pass 2) failed: %v", err)
	}

	ndsl1 := toNDSL(t, serialized1)
	ndsl2 := toNDSL(t, serialized2)

	if ndsl1 != ndsl2 {
		t.Errorf("serialization not idempotent for snippet %q\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s\n--- diff ---\n%s",
			snippet1.Name, ndsl1, ndsl2, ndslDiff(ndsl1, ndsl2))
	}
}

// roundtripEnumeration: double roundtrip idempotency test.
func roundtripEnumeration(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	enum1, err := r.parseEnumeration("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseEnumeration (pass 1) failed: %v", err)
	}
	serialized1, err := w.serializeEnumeration(enum1)
	if err != nil {
		t.Fatalf("serializeEnumeration (pass 1) failed: %v", err)
	}

	enum2, err := r.parseEnumeration("test-unit-id", "test-container-id", serialized1)
	if err != nil {
		t.Fatalf("parseEnumeration (pass 2) failed: %v", err)
	}
	serialized2, err := w.serializeEnumeration(enum2)
	if err != nil {
		t.Fatalf("serializeEnumeration (pass 2) failed: %v", err)
	}

	ndsl1 := toNDSL(t, serialized1)
	ndsl2 := toNDSL(t, serialized2)

	if ndsl1 != ndsl2 {
		t.Errorf("serialization not idempotent for enumeration %q\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s\n--- diff ---\n%s",
			enum1.Name, ndsl1, ndsl2, ndslDiff(ndsl1, ndsl2))
	}
}

// TestRoundtrip_Pages runs roundtrip tests on all page baselines in testdata/.
func TestRoundtrip_Pages(t *testing.T) {
	runRoundtripDir(t, "testdata/pages", roundtripPage)
}

// TestRoundtrip_Microflows runs roundtrip tests on all microflow baselines.
func TestRoundtrip_Microflows(t *testing.T) {
	runRoundtripDir(t, "testdata/microflows", roundtripMicroflow)
}

// TestRoundtrip_Snippets runs roundtrip tests on all snippet baselines.
func TestRoundtrip_Snippets(t *testing.T) {
	runRoundtripDir(t, "testdata/snippets", roundtripSnippet)
}

// TestRoundtrip_Enumerations runs roundtrip tests on all enumeration baselines.
func TestRoundtrip_Enumerations(t *testing.T) {
	runRoundtripDir(t, "testdata/enumerations", roundtripEnumeration)
}

// runRoundtripDir loads all .mxunit files from a directory and runs the given roundtrip function.
func runRoundtripDir(t *testing.T, dir string, fn func(*testing.T, []byte)) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("no baseline directory: %s", dir)
			return
		}
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mxunit") {
			continue
		}
		count++
		name := strings.TrimSuffix(entry.Name(), ".mxunit")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatalf("failed to read baseline: %v", err)
			}
			fn(t, data)
		})
	}
	if count == 0 {
		t.Skipf("no .mxunit baselines in %s", dir)
	}
}

// ndslDiff returns a simple line-by-line diff of two NDSL strings.
func ndslDiff(a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	var diffs []string
	maxLen := len(linesA)
	if len(linesB) > maxLen {
		maxLen = len(linesB)
	}

	for i := 0; i < maxLen; i++ {
		la, lb := "", ""
		if i < len(linesA) {
			la = linesA[i]
		}
		if i < len(linesB) {
			lb = linesB[i]
		}
		if la != lb {
			diffs = append(diffs, "- "+la)
			diffs = append(diffs, "+ "+lb)
		}
	}
	return strings.Join(diffs, "\n")
}
