// SPDX-License-Identifier: Apache-2.0

// Command expr-hints-md generates the markdown reference for expression
// checker hint codes from the canonical hint registry. The output is
// committed to docs/06-mdl-reference/expr-hints.md and refreshed via
// `make expr-hints-md`.
package main

import (
	"fmt"
	"os"
)

func main() {
	out := "docs/06-mdl-reference/expr-hints.md"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := Generate(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", out)
}
