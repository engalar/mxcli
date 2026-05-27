// SPDX-License-Identifier: Apache-2.0

package pages

import "testing"

// TestDecodeMicroflowQNFromDataSource verifies the supplement helper reads
// the microflow qualified name via MicroflowSource → MicroflowSettings.
func TestDecodeMicroflowQNFromDataSource(t *testing.T) {
	const mf = "MyModule.DS_Items"
	ds := map[string]any{
		"$Type": "Forms$MicroflowSource",
		"MicroflowSettings": map[string]any{
			"$Type":     "Forms$MicroflowSettings",
			"Microflow": mf,
		},
	}

	got := DecodeMicroflowQNFromDataSource(ds)

	if got != mf {
		t.Errorf("DecodeMicroflowQNFromDataSource = %q, want %q", got, mf)
	}
}

// TestDecodeNanoflowQNFromDataSource verifies the supplement helper for
// NanoflowSource.
func TestDecodeNanoflowQNFromDataSource(t *testing.T) {
	const nf = "MyModule.NF_Items"
	ds := map[string]any{
		"$Type":    "Forms$NanoflowSource",
		"Nanoflow": nf,
	}

	got := DecodeNanoflowQNFromDataSource(ds)

	if got != nf {
		t.Errorf("DecodeNanoflowQNFromDataSource = %q, want %q", got, nf)
	}
}
