package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// writeClipboard writes text to the system clipboard.
// Tries OSC 52 terminal escape first (works over SSH/tmux), then falls back
// to native clipboard tools (pbcopy, wl-copy, xclip, xsel).
func writeClipboard(text string) error {
	// OSC 52 works in kitty, alacritty, iTerm2, xterm, and over SSH.
	// Try it first — it's the most reliable in headless/SSH environments.
	if err := writeClipboardOSC52(text); err == nil {
		return nil
	}
	return writeClipboardNative(text)
}

// writeClipboardOSC52 uses the OSC 52 terminal escape sequence.
// Format: ESC ] 52 ; c ; <base64> BEL
// Works in most modern terminals and survives SSH tunnels.
func writeClipboardOSC52(text string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	// When inside tmux, the escape must be wrapped in a tmux passthrough.
	if os.Getenv("TMUX") != "" {
		seq := fmt.Sprintf("\x1bPtmux;\x1b\x1b]52;c;%s\x07\x1b\\", encoded)
		_, err := fmt.Fprint(os.Stderr, seq)
		return err
	}

	seq := fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
	_, err := fmt.Fprint(os.Stderr, seq)
	return err
}

func writeClipboardNative(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if isCommandAvailable("wl-copy") {
			cmd = exec.Command("wl-copy")
		} else if isCommandAvailable("xclip") {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if isCommandAvailable("xsel") {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool available")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
