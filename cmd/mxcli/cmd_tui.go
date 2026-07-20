// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

Flags:
  -c, --continue   Restore previous session (tab, navigation, preview mode)

Example:
  mxcli tui -p app.mpr
  mxcli tui -c
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		continueSession, _ := cmd.Flags().GetBool("continue")
		mxcliPath := resolveMxcliPath()

		// Try to restore session when -c flag is set
		var session *tui.TUISession
		if continueSession {
			loaded, err := tui.LoadSession()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load session: %v\n", err)
			} else if loaded != nil {
				session = loaded
				// Use project path from session if not explicitly provided
				if projectPath == "" && len(session.Tabs) > 0 {
					projectPath = session.Tabs[0].ProjectPath
				}
			}
		}

		if projectPath == "" {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Fprintln(os.Stderr, "Error: --project (-p) is required when stdin is not an interactive terminal")
				fmt.Fprintln(os.Stderr, "\nExample: mxcli tui -p app.mpr")
				return fmt.Errorf("--project (-p) is required")
			}
			picker := tui.NewPickerModel()
			p := tea.NewProgram(picker, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				return fmt.Errorf("TUI picker: %w", err)
			}
			m := result.(tui.PickerModel)
			if m.Chosen() == "" {
				return nil
			}
			projectPath = m.Chosen()
		}

		// Verify project file exists
		if _, err := os.Stat(projectPath); err != nil {
			return fmt.Errorf("project file not found: %s", projectPath)
		}

		tui.SaveHistory(projectPath)

		m := tui.NewApp(mxcliPath, projectPath)
		if session != nil {
			m.SetPendingSession(session)
		}

		// Set agent auto-proceed BEFORE tea.NewProgram so the model copy has the value
		agentSocket, _ := cmd.Flags().GetString("agent-socket")
		agentAutoProceed, _ := cmd.Flags().GetBool("agent-auto-proceed")
		if agentSocket != "" {
			m.SetAgentAutoProceed(agentAutoProceed)
		}

		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		m.StartWatcher(p)

		if agentSocket != "" {
			if err := m.StartAgentListener(p, agentSocket, agentAutoProceed); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: agent listener failed: %v\n", err)
			}
			defer m.CloseAgentListener()
		}

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI runtime: %w", err)
		}
		return nil
	},
}

func init() {
	tuiCmd.Flags().BoolP("continue", "c", false, "Restore previous TUI session")
	tuiCmd.Flags().String("agent-socket", "", "Unix socket path for agent communication (e.g. /tmp/mxcli-agent.sock)")
	tuiCmd.Flags().Bool("agent-auto-proceed", false, "Skip human confirmation for agent operations")
}

// resolveMxcliPath returns the binary path the TUI should use for spawning
// subcommands (project-tree, describe, exec, etc.).
func resolveMxcliPath() string {
	p, _ := os.Executable()
	return p
}
