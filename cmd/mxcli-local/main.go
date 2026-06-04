// cmd/mxcli-local/main.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {
	root := &cobra.Command{
		Use:          "mxcli-local",
		Short:        "Build and run Mendix apps without Docker",
		Version:      Version,
		SilenceUsage: true,
	}
	root.AddCommand(buildCmd(), runCmd(), reloadCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
