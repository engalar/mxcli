// SPDX-License-Identifier: Apache-2.0

// mdlrun is a minimal MDL runner for development. It executes MDL commands
// or script files directly against an MPR file without the daemon/socket layer.
//
// Usage:
//
//	go run ./cmd/mdlrun -p app.mpr -c "show entities"
//	go run ./cmd/mdlrun -p app.mpr script.mdl
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

func main() {
	p := flag.String("p", "", "path to .mpr file (required)")
	c := flag.String("c", "", "MDL command string to execute")
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

	if f := flag.Arg(0); f != "" {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", f, err)
			os.Exit(1)
		}
		if err := runMDL(exec, string(content)); err != nil {
			os.Exit(1)
		}
		return
	}

	flag.Usage()
	os.Exit(1)
}

func runMDL(exec *executor.Executor, mdl string) error {
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", e)
		}
		return fmt.Errorf("parse failed with %d error(s)", len(errs))
	}
	err := exec.ExecuteProgram(prog)
	if errors.Is(err, executor.ErrExit) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return err
}
