// SPDX-License-Identifier: Apache-2.0
package main

import "testing"

func TestClassifyPackageTest(t *testing.T) {
	cases := []struct {
		pkg      string
		testFile string
		wantID   string
		wantName string
	}{
		{"github.com/mendixlabs/mxcli/mdl/visitor", "visitor_test.go", "L1L2", "L1/L2 Parser+Visitor"},
		{"github.com/mendixlabs/mxcli/mdl/visitor", "visitor_microflow_test.go", "L1L2", "L1/L2 Parser+Visitor"},
		{"github.com/mendixlabs/mxcli/mdl/executor", "cmd_entities_mock_test.go", "L3", "L3 Executor Mock"},
		{"github.com/mendixlabs/mxcli/mdl/executor", "cmd_microflows_show_gen_test.go", "L4", "L4 Executor Gen"},
		{"github.com/mendixlabs/mxcli/mdl/backend/mpr", "convert_roundtrip_test.go", "L5", "L5 Decode"},
		{"github.com/mendixlabs/mxcli/mdl/executor", "roundtrip_microflow_test.go", "L6a", "L6a Roundtrip"},
		{"github.com/mendixlabs/mxcli/mdl/executor", "describe_sanity_test.go", "L6b", "L6b Describe Sanity"},
		{"github.com/mendixlabs/mxcli/modelsdk/codec", "codec_bench_test.go", "bench", "Benchmark"},
		{"github.com/mendixlabs/mxcli/modelsdk/codec", "codec_test.go", "other", "Other"},
	}
	for _, c := range cases {
		t.Run(c.testFile, func(t *testing.T) {
			gotID, gotName := classifyTest(c.pkg, c.testFile)
			if gotID != c.wantID {
				t.Errorf("classifyTest(%q, %q) id = %q, want %q", c.pkg, c.testFile, gotID, c.wantID)
			}
			if gotName != c.wantName {
				t.Errorf("classifyTest(%q, %q) name = %q, want %q", c.pkg, c.testFile, gotName, c.wantName)
			}
		})
	}
}
