// SPDX-License-Identifier: Apache-2.0

// Command exprgrammar-mine walks the microflows of an MPR file and
// emits a Go source file (generated/exprgrammar/mined.go) with the
// SlotExpectations, Functions, and Productions tables that the
// expression checker uses.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		mpr = flag.String("mpr", "", "path to .mpr file to mine (required)")
		out = flag.String("out", "generated/exprgrammar/mined.go", "output Go file")
	)
	flag.Parse()
	if *mpr == "" {
		fmt.Fprintln(os.Stderr, "exprgrammar-mine: --mpr is required")
		os.Exit(2)
	}
	if err := run(*mpr, *out); err != nil {
		fmt.Fprintln(os.Stderr, "exprgrammar-mine:", err)
		os.Exit(1)
	}
}

func run(mprPath, outPath string) error {
	_ = outPath
	m := NewMiner()
	if err := MineMPR(m, mprPath); err != nil {
		return err
	}
	fmt.Printf("mined %d expression records\n", len(m.Records))
	return nil
}
