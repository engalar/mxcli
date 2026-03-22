package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var allCommands = []struct {
	name string
	desc string
}{
	{"callers", "show callers of selected element"},
	{"callees", "show callees of selected element"},
	{"context", "show context of selected element"},
	{"impact", "show impact of selected element"},
	{"refs", "show references to selected element"},
	{"diagram", "open diagram in browser"},
	{"search", "full-text search: search <keyword>"},
	{"run", "run MDL file: run <file.mdl>"},
	{"check", "check MDL syntax: check <file.mdl>"},
}

// CmdBar is the bottom command input bar activated by ":".
type CmdBar struct {
	input             textinput.Model
	visible           bool
	candidates        []string
	selectedCandidate int
}

func NewCmdBar() CmdBar {
	ti := textinput.New()
	ti.Placeholder = "command (callers, callees, context, impact, refs, diagram, search, run, check)"
	ti.Prompt = ": "
	ti.CharLimit = 200
	return CmdBar{input: ti}
}

func (c *CmdBar) Show() {
	c.visible = true
	c.input.SetValue("")
	c.input.Focus()
	c.candidates = nil
	c.selectedCandidate = 0
}

func (c *CmdBar) Hide() {
	c.visible = false
	c.input.Blur()
	c.candidates = nil
	c.selectedCandidate = 0
}

func (c CmdBar) IsVisible() bool { return c.visible }

// Command returns the current input value split into verb + args.
func (c CmdBar) Command() (verb string, rest string) {
	parts := strings.SplitN(strings.TrimSpace(c.input.Value()), " ", 2)
	if len(parts) == 0 {
		return "", ""
	}
	verb = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest = parts[1]
	}
	return verb, rest
}

func (c *CmdBar) filterCandidates() {
	verb, _ := c.Command()
	if verb == "" {
		c.candidates = nil
		return
	}
	var result []string
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd.name, verb) || strings.Contains(cmd.name, verb) {
			result = append(result, cmd.name)
		}
	}
	// If exact match, no need to show candidates
	if len(result) == 1 && result[0] == verb {
		c.candidates = nil
		return
	}
	c.candidates = result
	if c.selectedCandidate >= len(c.candidates) {
		c.selectedCandidate = 0
	}
}

// ApplyCompletion applies the selected candidate if one is highlighted and not yet complete.
// Returns true if completion was applied (caller should not submit yet).
func (c *CmdBar) ApplyCompletion() bool {
	if len(c.candidates) > 0 && c.selectedCandidate < len(c.candidates) {
		verb, _ := c.Command()
		candidate := c.candidates[c.selectedCandidate]
		if verb != candidate {
			c.input.SetValue(candidate + " ")
			c.input.CursorEnd()
			c.candidates = nil
			c.selectedCandidate = 0
			return true
		}
	}
	return false
}

func (c CmdBar) Update(msg tea.Msg) (CmdBar, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab":
			if len(c.candidates) > 0 {
				chosen := c.candidates[c.selectedCandidate]
				c.input.SetValue(chosen + " ")
				c.input.CursorEnd()
				c.candidates = nil
				c.selectedCandidate = 0
			}
			return c, nil
		case "up":
			if len(c.candidates) > 0 {
				c.selectedCandidate--
				if c.selectedCandidate < 0 {
					c.selectedCandidate = len(c.candidates) - 1
				}
			}
			return c, nil
		case "down":
			if len(c.candidates) > 0 {
				c.selectedCandidate++
				if c.selectedCandidate >= len(c.candidates) {
					c.selectedCandidate = 0
				}
			}
			return c, nil
		}
	}
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	c.filterCandidates()
	return c, cmd
}

func (c CmdBar) View() string {
	if !c.visible {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" :callers  :callees  :context  :impact  :refs  :diagram  :search  :run  :check")
	}

	inputLine := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("214")).
		Render(c.input.View())

	if len(c.candidates) == 0 {
		return inputLine
	}

	// Show up to 5 candidates
	maxShow := min(5, len(c.candidates))

	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)

	var lines []string
	lines = append(lines, inputLine)
	for i := 0; i < maxShow; i++ {
		name := c.candidates[i]
		// Find description
		desc := ""
		for _, cmd := range allCommands {
			if cmd.name == name {
				desc = cmd.desc
				break
			}
		}
		entry := "  " + name
		if desc != "" {
			entry += "  " + desc
		}
		if i == c.selectedCandidate {
			lines = append(lines, highlightStyle.Render(entry))
		} else {
			lines = append(lines, normalStyle.Render(entry))
		}
	}
	return strings.Join(lines, "\n")
}
