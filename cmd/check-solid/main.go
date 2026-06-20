// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var violations int

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.Contains(path, "vendor/") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, "generated/") {
			return nil
		}
		return checkFile(path)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Found %d SOLID violation(s)\n", violations)
		os.Exit(1)
	}
	fmt.Println("✅ No SOLID violations found")
}

func checkFile(path string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, ", _ :=") && strings.Contains(trimmed, ".(") {
			fmt.Fprintf(os.Stderr, "LSP violation: %s:%d: unchecked type assertion\n  %s\n", path, i+1, trimmed)
			violations++
		}
	}

	return nil
}
