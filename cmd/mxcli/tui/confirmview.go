// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// ConfirmView is a lightweight yes/no confirmation dialog.
type ConfirmView struct {
	title       string
	message     string
	dropCmd     string
	mxcliPath   string
	projectPath string
}

// NewConfirmView creates a confirmation dialog that executes dropCmd on 'y'.
func NewConfirmView(title, message, dropCmd, mxcliPath, projectPath string) *ConfirmView {
	return &ConfirmView{
		title:       title,
		message:     message,
		dropCmd:     dropCmd,
		mxcliPath:   mxcliPath,
		projectPath: projectPath,
	}
}

func (v *ConfirmView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			mxcliPath := v.mxcliPath
			projectPath := v.projectPath
			dropCmd := v.dropCmd
			return v, func() tea.Msg {
				out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", dropCmd)
				return execShowResultMsg{Content: out, Success: err == nil}
			}
		default:
			// Any other key cancels
			return v, func() tea.Msg { return PopViewMsg{} }
		}
	}
	return v, nil
}

func (v *ConfirmView) Render(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(kernel.CheckErrorStyle.GetForeground())

	var b strings.Builder
	b.WriteString(titleStyle.Render(v.title))
	b.WriteString("\n\n")
	b.WriteString(v.message)
	b.WriteString("\n\n")
	b.WriteString(kernel.HintLabelStyle.Render("Press y to confirm, any other key to cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(kernel.CheckErrorStyle.GetForeground()).
		Padding(1, 2).
		Width(min(70, width-4))

	box := boxStyle.Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (v *ConfirmView) Hints() []Hint {
	return []Hint{
		{Key: "y", Label: "confirm"},
		{Key: "any", Label: "cancel"},
	}
}

func (v *ConfirmView) StatusInfo() StatusInfo {
	return StatusInfo{Mode: "Confirm"}
}

func (v *ConfirmView) Mode() ViewMode { return ModeConfirm }

// buildDropCmd returns the MDL DROP command for a given node type and qualified name.
// Returns empty string for unsupported types.
func buildDropCmd(nodeType, qname string) string {
	switch strings.ToLower(nodeType) {
	case "entity":
		return "DROP ENTITY " + qname
	case "association":
		return "DROP ASSOCIATION " + qname
	case "enumeration":
		return "DROP ENUMERATION " + qname
	case "constant":
		return "DROP CONSTANT " + qname
	case "microflow":
		return "DROP MICROFLOW " + qname
	case "nanoflow":
		return "DROP NANOFLOW " + qname
	case "layout":
		return "DROP LAYOUT " + qname
	case "page":
		return "DROP PAGE " + qname
	case "snippet":
		return "DROP SNIPPET " + qname
	case "module":
		// Module name is just the first part before the dot
		parts := strings.SplitN(qname, ".", 2)
		return "DROP MODULE " + parts[0]
	case "workflow":
		return "DROP WORKFLOW " + qname
	case "imagecollection":
		return "DROP IMAGE COLLECTION " + qname
	case "javaaction":
		return "DROP JAVA ACTION " + qname
	case "javascriptaction":
		return "DROP JAVASCRIPT ACTION " + qname
	default:
		return ""
	}
}

// buildDeleteMessage returns a user-friendly confirmation message.
func buildDeleteMessage(nodeType, qname string) string {
	return fmt.Sprintf("  %s  %s\n\n  Command: %s",
		strings.ToUpper(nodeType), qname, buildDropCmd(nodeType, qname))
}
