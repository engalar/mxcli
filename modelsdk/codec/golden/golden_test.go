// SPDX-License-Identifier: Apache-2.0

package golden

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
)

func TestGoldenBSON(t *testing.T) {
	for _, entry := range Registry() {
		t.Run(entry.Name, func(t *testing.T) {
			obj := entry.Builder()
			enc := &codec.Encoder{}
			got, err := enc.Encode(obj)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			skipFields := entry.SkipFields
			if skipFields == nil {
				skipFields = []string{}
			}

			diffs := CompareBSON(got, entry.BSON, skipFields)
			if len(diffs) == 0 {
				return
			}

			// Group diffs by kind for readable output.
			missing := DiffsByKind(diffs, DiffMissing)
			extra := DiffsByKind(diffs, DiffExtra)
			order := DiffsByKind(diffs, DiffOrder)
			value := DiffsByKind(diffs, DiffValue)
			marker := DiffsByKind(diffs, DiffMarker)
			dtype := DiffsByKind(diffs, DiffType)

			if len(dtype) > 0 {
				t.Errorf("--- TYPE MISMATCHES (%d) ---", len(dtype))
				for _, d := range dtype {
					t.Errorf("  %s", d)
				}
			}
			if len(missing) > 0 {
				t.Errorf("--- MISSING FIELDS (%d) ---", len(missing))
				for _, d := range missing {
					t.Errorf("  %s", d)
				}
			}
			if len(extra) > 0 {
				t.Errorf("--- EXTRA FIELDS (%d) ---", len(extra))
				for _, d := range extra {
					t.Errorf("  %s", d)
				}
			}
			if len(order) > 0 {
				t.Errorf("--- ORDER MISMATCHES (%d) ---", len(order))
				for _, d := range order {
					t.Errorf("  %s", d)
				}
			}
			if len(marker) > 0 {
				t.Errorf("--- MARKER MISMATCHES (%d) ---", len(marker))
				for _, d := range marker {
					t.Errorf("  %s", d)
				}
			}
			if len(value) > 0 {
				t.Errorf("--- VALUE MISMATCHES (%d) ---", len(value))
				for _, d := range value {
					t.Errorf("  %s", d)
				}
			}
			t.Logf("  Encoded %d bytes vs golden %d bytes", len(got), len(entry.BSON))
			t.Logf("  Diffs: %d missing, %d extra, %d order, %d value, %d marker, %d type",
				len(missing), len(extra), len(order), len(value), len(marker), len(dtype))
		})
	}
}

func TestGoldenRegistry(t *testing.T) {
	entries := Registry()
	if len(entries) == 0 {
		t.Fatal("no golden entries registered")
	}
	for _, e := range entries {
		t.Logf("Registered golden: %s (%d bytes from %s)", e.Name, len(e.BSON), e.Source)
	}
}

// BenchmarkGoldenBSON measures BSON encoding performance against the golden.
func BenchmarkGoldenBSON(b *testing.B) {
	for _, entry := range Registry() {
		b.Run(entry.Name, func(b *testing.B) {
			b.ReportAllocs()
			enc := &codec.Encoder{}
			for i := 0; i < b.N; i++ {
				obj := entry.Builder()
				_, err := enc.Encode(obj)
				if err != nil {
					b.Fatalf("Encode: %v", err)
				}
			}
		})
	}
}

// FormatGoldenDiff is a helper that returns a compact diff summary string.
func FormatGoldenDiff(diffs []Diff) string {
	if len(diffs) == 0 {
		return "OK"
	}
	var counts = map[DiffKind]int{}
	for _, d := range diffs {
		counts[d.Kind]++
	}
	return fmt.Sprintf("diffs: %v", counts)
}
