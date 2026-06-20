// SPDX-License-Identifier: Apache-2.0

// mdlrun is a minimal MDL runner for development. It executes MDL commands
// or script files directly against an MPR file without the daemon/socket layer.
//
// Usage:
//
//	go run ./cmd/mdlrun -p app.mpr -c "show entities"
//	go run ./cmd/mdlrun -p app.mpr script.mdl
//	go run ./cmd/mdlrun -p app.mpr file1.mdl file2.mdl file3.mdl
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

func main() {
	p := flag.String("p", "", "path to .mpr file (required)")
	c := flag.String("c", "", "MDL command string to execute")
	stopOnError := flag.Bool("stop-on-error", false, "stop executing files after the first failure")
	flag.Parse()

	exec := executor.Build().
		Out(os.Stdout).
		ProgressOut(os.Stderr).
		WithFactory(func() executor.BackendIface { return mprbackend.New() }).
		Create()
	defer exec.Close()

	if *p != "" {
		if err := runMDL(exec, fmt.Sprintf("CONNECT LOCAL '%s'", *p)); err != nil {
			fmt.Fprintf(os.Stderr, "connect: %v\n", err)
			os.Exit(1)
		}
	}

	if *c != "" {
		if err := runMDL(exec, *c); err != nil {
			os.Exit(1)
		}
		return
	}

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	runFiles(exec, files, *stopOnError)
}

// fileResult holds the outcome of executing a single MDL file.
type fileResult struct {
	file    string
	stmts   int
	elapsed time.Duration
	err     error
}

func runFiles(exec *executor.Executor, files []string, stopOnError bool) {
	results := make([]fileResult, 0, len(files))

	for _, f := range files {
		fmt.Fprintf(os.Stderr, "\n--- %s ---\n", f)

		content, err := os.ReadFile(f)
		start := time.Now()
		if err != nil {
			elapsed := time.Since(start)
			fmt.Fprintf(os.Stderr, "  ✗ failed after %.2fs: %v\n", elapsed.Seconds(), err)
			results = append(results, fileResult{file: f, elapsed: elapsed, err: err})
			if stopOnError {
				break
			}
			continue
		}

		stmts, err := runMDLCounted(exec, string(content))
		elapsed := time.Since(start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ failed after %.2fs\n", elapsed.Seconds())
			results = append(results, fileResult{file: f, stmts: stmts, elapsed: elapsed, err: err})
			if stopOnError {
				break
			}
			continue
		}

		fmt.Fprintf(os.Stderr, "  ✓ %d statement(s), %.2fs\n", stmts, elapsed.Seconds())
		results = append(results, fileResult{file: f, stmts: stmts, elapsed: elapsed})
	}

	printSummary(results, len(files))

	for _, r := range results {
		if r.err != nil {
			os.Exit(1)
		}
	}
}

func printSummary(results []fileResult, total int) {
	succeeded := 0
	failed := 0
	totalStmts := 0
	var totalElapsed time.Duration

	for _, r := range results {
		totalStmts += r.stmts
		totalElapsed += r.elapsed
		if r.err != nil {
			failed++
		} else {
			succeeded++
		}
	}

	skipped := total - len(results)
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Summary: %d/%d files succeeded, %d failed, %d skipped, %d statement(s), %.2fs total ===\n",
			succeeded, total, failed, skipped, totalStmts, totalElapsed.Seconds())
	} else {
		fmt.Fprintf(os.Stderr, "\n=== Summary: %d/%d files succeeded, %d failed, %d statement(s), %.2fs total ===\n",
			succeeded, total, failed, totalStmts, totalElapsed.Seconds())
	}
}

func runMDL(exec *executor.Executor, mdl string) error {
	_, err := runMDLCounted(exec, mdl)
	return err
}

// runMDLCounted parses and executes MDL, returning the statement count and any error.
func runMDLCounted(exec *executor.Executor, mdl string) (int, error) {
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", e)
		}
		return 0, fmt.Errorf("parse failed with %d error(s)", len(errs))
	}
	stmts := len(prog.Statements)
	err := exec.ExecuteProgram(prog)
	if errors.Is(err, executor.ErrExit) {
		return stmts, nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return stmts, err
}
