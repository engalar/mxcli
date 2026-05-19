// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// findProjectRoot finds the project root by looking for go.mod, searching upward.
func findProjectRoot() (string, error) {
	// First try: directory of the executable's source
	// More reliable: use git rev-parse if in a git repo
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	// Fallback: walk up from cwd looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// collectFiles walks the project root and collects source files with line counts.
func collectFiles(root string) []fileInfo {
	var result []fileInfo

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)

		// Skip excluded directories
		if d.IsDir() {
			base := d.Name()
			switch base {
			case "vendor", ".git", "generated", "libs", "reference", "node_modules", "out", ".vscode-test":
				return filepath.SkipDir
			}
			// Skip parser directories (*/parser/)
			if base == "parser" {
				return filepath.SkipDir
			}
			return nil
		}

		// Determine if this file should be included
		ext := filepath.Ext(path)
		include := false

		if ext == ".go" {
			include = true
		}

		if !include {
			return nil
		}

		// Count lines
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := bytes.Count(data, []byte{'\n'})

		result = append(result, fileInfo{lines: lines, path: rel})
		return nil
	})

	return result
}

// parseCoverage parses a Go coverage.out file into per-file coverage data.
func parseCoverage(file, modPath string) map[string]covData {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()

	prefix := modPath + "/"
	totalMap := map[string]int{}
	coveredMap := map[string]int{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		// Format: file:startLine.startCol,endLine.endCol numStmts count
		// Find the colon before line number
		colonIdx := -1
		for i := len(line) - 1; i >= 0; i-- {
			if line[i] == ':' {
				colonIdx = i
				break
			}
		}
		if colonIdx < 0 {
			continue
		}
		// Find the space after the range
		spaceIdx := strings.Index(line[colonIdx:], " ")
		if spaceIdx < 0 {
			continue
		}
		filePath := line[:colonIdx]
		rest := strings.TrimSpace(line[colonIdx+spaceIdx:])
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			continue
		}

		stmts, _ := strconv.Atoi(parts[0])
		count, _ := strconv.Atoi(parts[1])

		// Strip module prefix
		filePath = strings.TrimPrefix(filePath, prefix)

		totalMap[filePath] += stmts
		if count > 0 {
			coveredMap[filePath] += stmts
		}
	}

	result := map[string]covData{}
	for f, total := range totalMap {
		pct := 0
		if total > 0 {
			pct = coveredMap[f] * 100 / total
		}
		result[f] = covData{pct: pct, total: total, covered: coveredMap[f]}
	}
	return result
}

// collectCommits returns recent commit hashes, their subjects, and per-commit per-file change counts.
func collectCommits(root string) ([]string, map[string]string, map[string]map[string]int) {
	// Check if we're in a git repo
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return nil, nil, nil
	}

	// Get recent commit hashes and subjects
	out := runCmd(root, "git", "log", "--format=%h\t%s", "-10")
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil, nil
	}

	var commits []string
	commitSubjects := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		hash, subject, _ := strings.Cut(line, "\t")
		commits = append(commits, hash)
		commitSubjects[hash] = subject
	}
	commitData := map[string]map[string]int{}

	for _, hash := range commits {
		// Get diff stats for this commit
		cmd := exec.Command("git", "diff", "--numstat", "--no-renames", hash+"~1.."+hash)
		cmd.Dir = root
		diffOut, err := cmd.Output()
		if err != nil {
			// Initial commit or other error - skip
			continue
		}

		fileChanges := map[string]int{}
		scanner := bufio.NewScanner(bytes.NewReader(diffOut))
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Split(line, "\t")
			if len(fields) < 3 {
				continue
			}
			// Skip binary files
			if fields[0] == "-" {
				continue
			}
			added, _ := strconv.Atoi(fields[0])
			deleted, _ := strconv.Atoi(fields[1])
			filePath := fields[2]
			fileChanges[filePath] = added + deleted
		}
		commitData[hash] = fileChanges
	}

	return commits, commitSubjects, commitData
}

// computeDepths runs `go list` and computes dependency depth via iterative relaxation.
func computeDepths(root, modPath string) map[string]int {
	out := runCmd(root, "go", "list", "-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}", "./...")
	out = strings.TrimSpace(out)
	if out == "" {
		return map[string]int{".": 0}
	}

	prefix := modPath + "/"
	packages := map[string]bool{}
	deps := map[string][]string{}

	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := fields[0]
		pkg = strings.TrimPrefix(pkg, prefix)
		if pkg == modPath {
			pkg = "."
		}
		packages[pkg] = true

		for _, imp := range fields[1:] {
			if strings.HasPrefix(imp, modPath) {
				dep := strings.TrimPrefix(imp, prefix)
				if dep == modPath {
					dep = "."
				}
				deps[pkg] = append(deps[pkg], dep)
			}
		}
	}

	depth := map[string]int{}
	for p := range packages {
		depth[p] = 0
	}

	// Iterative relaxation
	for range 30 {
		changed := false
		for p := range packages {
			for _, d := range deps[p] {
				if _, ok := packages[d]; ok {
					if depth[d]+1 > depth[p] {
						depth[p] = depth[d] + 1
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	return depth
}

// runCmd runs a command in the given directory and returns stdout (trimmed).
func runCmd(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
