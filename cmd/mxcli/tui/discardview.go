package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// discardDoneMsg carries the result of git checkout for MPR discard.
type discardDoneMsg struct {
	Output  string
	Success bool
}

// DiscardConfirmView asks the user to confirm discarding MPR changes via git checkout.
type DiscardConfirmView struct {
	projectPath string
	mxcliPath   string
}

func NewDiscardConfirmView(projectPath, mxcliPath string) *DiscardConfirmView {
	return &DiscardConfirmView{
		projectPath: projectPath,
		mxcliPath:   mxcliPath,
	}
}

func (v *DiscardConfirmView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			projectPath := v.projectPath
			return v, func() tea.Msg {
				projectDir := filepath.Dir(projectPath)

				// Build git checkout args based on MPR version
				args := []string{"checkout", "HEAD", "--", filepath.Base(projectPath)}
				mprContents := filepath.Join(projectDir, "mprcontents")
				if info, err := os.Stat(mprContents); err == nil && info.IsDir() {
					args = append(args, "mprcontents/")
				}

				gitCmd := exec.Command("git", args...)
				gitCmd.Dir = projectDir
				out, err := gitCmd.CombinedOutput()
				if err != nil {
					return discardDoneMsg{
						Output:  fmt.Sprintf("git checkout failed: %s\n%s", err, strings.TrimSpace(string(out))),
						Success: false,
					}
				}
				return discardDoneMsg{
					Output:  "Changes discarded (git checkout HEAD -- " + strings.Join(args[3:], " ") + ")",
					Success: true,
				}
			}
		default:
			return v, func() tea.Msg { return PopViewMsg{} }
		}
	}
	return v, nil
}

func (v *DiscardConfirmView) Render(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(kernel.CheckErrorStyle.GetForeground())
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Discard Changes"))
	b.WriteString("\n\n")
	b.WriteString("  Restore project to last git commit state?\n\n")
	b.WriteString(warnStyle.Render("  git checkout HEAD -- mprcontents/ *.mpr"))
	b.WriteString("\n\n")
	b.WriteString("  This will discard all uncommitted modifications.\n\n")
	b.WriteString(kernel.HintLabelStyle.Render("Press y to confirm, any other key to cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(kernel.CheckErrorStyle.GetForeground()).
		Padding(1, 2).
		Width(min(70, width-4))

	box := boxStyle.Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (v *DiscardConfirmView) Hints() []Hint {
	return []Hint{
		{Key: "y", Label: "discard"},
		{Key: "any", Label: "cancel"},
	}
}

func (v *DiscardConfirmView) StatusInfo() StatusInfo {
	return StatusInfo{Mode: "Confirm"}
}

func (v *DiscardConfirmView) Mode() ViewMode { return ModeConfirm }
