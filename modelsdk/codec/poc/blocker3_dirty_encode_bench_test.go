// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// BenchmarkBlocker3_FullEncode measures the cost of encoding a microflow
// where the top-level Name property is touched (forcing the encoder to
// rebuild the doc). For a clean microflow with no dirty bits, the encoder
// short-circuits to raw passthrough, so we mark Name dirty to exercise the
// real per-property loop.
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 3.
func BenchmarkBlocker3_FullEncode(b *testing.B) {
	mf := loadFixtureMicroflow(b)
	mf.SetName(mf.Name())

	enc := &codec.Encoder{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := enc.Encode(mf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBlocker3_IncrementalEncode measures the cost of encoding a
// microflow where only the top-level Name scalar is dirty. With the
// encoder's selective-rebuild logic, clean child elements should pass
// through their raw bytes unchanged.
//
// If incremental encoding works as designed, this benchmark should be
// noticeably faster than BenchmarkBlocker3_FullEncode on the same
// fixture. The two benchmarks share the same fixture and dirty bit
// (SetName), so the only thing they're really separating is repeated
// invocation pattern — a meaningful gap requires the encoder to truly
// short-circuit clean subtrees.
func BenchmarkBlocker3_IncrementalEncode(b *testing.B) {
	mf := loadFixtureMicroflow(b)
	mf.SetName("ChangedNameOnly")

	enc := &codec.Encoder{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := enc.Encode(mf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBlocker3_CleanPassthrough measures the cost when nothing is
// dirty — encoder should hit the raw passthrough fast path (return raw
// bytes unchanged). This is the lower bound for incremental encoding.
func BenchmarkBlocker3_CleanPassthrough(b *testing.B) {
	mf := loadFixtureMicroflow(b)
	// no SetName / dirty marking — encode the freshly-decoded element.

	enc := &codec.Encoder{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := enc.Encode(mf); err != nil {
			b.Fatal(err)
		}
	}
}

// loadFixtureMicroflow opens the canonical fixture, picks the largest
// Microflows$Microflow unit by raw byte length, decodes it, and returns
// the *genMf.Microflow.
func loadFixtureMicroflow(b *testing.B) *genMf.Microflow {
	b.Helper()

	_, r := openTestMPRForBench(b)
	refs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		b.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		b.Fatal("no Microflows$Microflow units in fixture")
	}

	pick := refs[0]
	for _, ref := range refs[1:] {
		if len(ref.Contents) > len(pick.Contents) {
			pick = ref
		}
	}
	b.Logf("fixture microflow: id=%s len=%d bytes (selected from %d candidates)",
		pick.ID, len(pick.Contents), len(refs))

	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(pick.Contents))
	if err != nil {
		b.Fatalf("Decode: %v", err)
	}
	mf, ok := elem.(*genMf.Microflow)
	if !ok {
		b.Fatalf("Decode produced %T, want *genMf.Microflow", elem)
	}
	return mf
}
