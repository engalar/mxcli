// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed changelog.md
var changelogContent string

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show release notes",
	Long:  "Display the mxcli changelog. Use --latest to show only the most recent version.",
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, _ := cmd.Flags().GetBool("latest")
		if latest {
			fmt.Print(extractLatestVersion(changelogContent))
		} else {
			fmt.Print(changelogContent)
		}
		return nil
	},
}

// extractLatestVersion returns the first version section from the changelog.
func extractLatestVersion(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inVersion := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## [") {
			if inVersion {
				break // hit the next version, stop
			}
			inVersion = true
		}
		if inVersion {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return content
	}
	return strings.Join(result, "\n") + "\n"
}

func init() {
	changelogCmd.Flags().BoolP("latest", "l", false, "Show only the most recent version")
}
