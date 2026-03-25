// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for Mendix projects",
	Long: `Launch a ranger-style three-column TUI for browsing and operating on a Mendix project.

Navigation:
  h/←   move focus left       l/→/Enter  move focus right / open
  j/↓   move down             k/↑        move up
  Tab   cycle panel focus     /          search in current column
  :     open command bar      q          quit

Commands (via : bar):
  :check           check MDL syntax
  :run             run current MDL file
  :callers         show callers of selected element
  :callees         show callees of selected element
  :context         show context of selected element
  :impact          show impact of selected element
  :refs            show references to selected element
  :diagram         open diagram in browser
  :search <kw>     full-text search

Example:
  mxcli tui -p app.mpr
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		mxcliPath, _ := os.Executable()

		if projectPath == "" {
			picker := tui.NewPickerModel()
			p := tea.NewProgram(picker, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			m := result.(tui.PickerModel)
			if m.Chosen() == "" {
				return
			}
			projectPath = m.Chosen()
		}

		tui.SaveHistory(projectPath)

		m := tui.NewApp(mxcliPath, projectPath)
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		m.StartWatcher(p)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}
