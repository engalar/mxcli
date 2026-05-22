// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
)

const sampleJSON = `
{"Action":"run","Package":"github.com/mendixlabs/mxcli/mdl/visitor","Test":"TestBuild_XPath","Time":"2026-05-22T10:00:00Z"}
{"Action":"pass","Package":"github.com/mendixlabs/mxcli/mdl/visitor","Test":"TestBuild_XPath","Elapsed":0.12,"Time":"2026-05-22T10:00:00.12Z"}
{"Action":"run","Package":"github.com/mendixlabs/mxcli/mdl/executor","Test":"TestShowEntities_Mock","Time":"2026-05-22T10:00:00Z"}
{"Action":"fail","Package":"github.com/mendixlabs/mxcli/mdl/executor","Test":"TestShowEntities_Mock","Elapsed":0.05,"Output":"--- FAIL: TestShowEntities_Mock\n","Time":"2026-05-22T10:00:00.05Z"}
{"Action":"pass","Package":"github.com/mendixlabs/mxcli/mdl/visitor","Elapsed":1.2,"Time":"2026-05-22T10:00:01Z"}
{"Action":"fail","Package":"github.com/mendixlabs/mxcli/mdl/executor","Elapsed":0.8,"Time":"2026-05-22T10:00:01Z"}
`

func TestParseTestResults(t *testing.T) {
	results, err := parseTestResults(strings.NewReader(sampleJSON))
	if err != nil {
		t.Fatalf("parseTestResults: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected non-empty results")
	}

	// visitor package should have 1 pass
	var visitorPass, executorFail int
	for _, r := range results {
		if strings.Contains(r.Package, "visitor") && r.Action == "pass" && r.Test != "" {
			visitorPass++
		}
		if strings.Contains(r.Package, "executor") && r.Action == "fail" && r.Test != "" {
			executorFail++
		}
	}
	if visitorPass != 1 {
		t.Errorf("expected 1 visitor pass, got %d", visitorPass)
	}
	if executorFail != 1 {
		t.Errorf("expected 1 executor fail, got %d", executorFail)
	}
}
