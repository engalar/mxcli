// SPDX-License-Identifier: Apache-2.0

package main

// Stubs for T8 (update mechanism). These let T6 main.go build and run
// while T8 is in flight. T8's commit will replace this file with the
// real cmd/mxcli-launcher/update.go containing the actual implementations.

import "fmt"

func runUpgrade(args []string) int {
	fmt.Println("mxcli: upgrade not yet implemented (T8 pending)")
	return 1
}

func runRollback(args []string) int {
	fmt.Println("mxcli: rollback not yet implemented (T8 pending)")
	return 1
}

func backgroundVersionCheck() {
	// no-op until T8 lands
}
