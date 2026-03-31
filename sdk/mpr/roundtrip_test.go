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

// roundtripPage: baseline BSON → parse → serialize → NDSL compare.
func roundtripPage(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	page, err := r.parsePage("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parsePage failed: %v", err)
	}

	serialized, err := w.serializePage(page)
	if err != nil {
		t.Fatalf("serializePage failed: %v", err)
	}

	baselineNDSL := toNDSL(t, baselineBytes)
	roundtripNDSL := toNDSL(t, serialized)

	if baselineNDSL != roundtripNDSL {
		t.Errorf("roundtrip NDSL mismatch for page %q\n--- baseline ---\n%s\n--- roundtrip ---\n%s\n--- diff ---\n%s",
			page.Name, baselineNDSL, roundtripNDSL, ndslDiff(baselineNDSL, roundtripNDSL))
	}
}

// roundtripMicroflow: baseline BSON → parse → serialize → NDSL compare.
func roundtripMicroflow(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	mf, err := r.parseMicroflow("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseMicroflow failed: %v", err)
	}

	serialized, err := w.serializeMicroflow(mf)
	if err != nil {
		t.Fatalf("serializeMicroflow failed: %v", err)
	}

	baselineNDSL := toNDSL(t, baselineBytes)
	roundtripNDSL := toNDSL(t, serialized)

	if baselineNDSL != roundtripNDSL {
		t.Errorf("roundtrip NDSL mismatch for microflow %q\n--- baseline ---\n%s\n--- roundtrip ---\n%s\n--- diff ---\n%s",
			mf.Name, baselineNDSL, roundtripNDSL, ndslDiff(baselineNDSL, roundtripNDSL))
	}
}

// roundtripSnippet: baseline BSON → parse → serialize → NDSL compare.
func roundtripSnippet(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	snippet, err := r.parseSnippet("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseSnippet failed: %v", err)
	}

	serialized, err := w.serializeSnippet(snippet)
	if err != nil {
		t.Fatalf("serializeSnippet failed: %v", err)
	}

	baselineNDSL := toNDSL(t, baselineBytes)
	roundtripNDSL := toNDSL(t, serialized)

	if baselineNDSL != roundtripNDSL {
		t.Errorf("roundtrip NDSL mismatch for snippet %q\n--- baseline ---\n%s\n--- roundtrip ---\n%s\n--- diff ---\n%s",
			snippet.Name, baselineNDSL, roundtripNDSL, ndslDiff(baselineNDSL, roundtripNDSL))
	}
}

// roundtripEnumeration: baseline BSON → parse → serialize → NDSL compare.
func roundtripEnumeration(t *testing.T, baselineBytes []byte) {
	t.Helper()
	r := testReader()
	w := testWriter()

	enum, err := r.parseEnumeration("test-unit-id", "test-container-id", baselineBytes)
	if err != nil {
		t.Fatalf("parseEnumeration failed: %v", err)
	}

	serialized, err := w.serializeEnumeration(enum)
	if err != nil {
		t.Fatalf("serializeEnumeration failed: %v", err)
	}

	baselineNDSL := toNDSL(t, baselineBytes)
	roundtripNDSL := toNDSL(t, serialized)

	if baselineNDSL != roundtripNDSL {
		t.Errorf("roundtrip NDSL mismatch for enumeration %q\n--- baseline ---\n%s\n--- roundtrip ---\n%s\n--- diff ---\n%s",
			enum.Name, baselineNDSL, roundtripNDSL, ndslDiff(baselineNDSL, roundtripNDSL))
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
