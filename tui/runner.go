package tui

import (
	"bytes"
	"os/exec"
	"strings"
)

// runMxcli runs a mxcli subcommand and returns stdout.
// mxcliPath is the path to the mxcli binary (os.Args[0] works for self).
func runMxcli(mxcliPath string, args ...string) (string, error) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.Command(mxcliPath, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		combined := strings.TrimSpace(buf.String() + "\n" + errBuf.String())
		return combined, err
	}
	return buf.String(), nil
}

// openBrowser opens a URL or file path in the system browser.
func openBrowser(target string) error {
	// Try xdg-open (Linux), open (macOS), start (Windows)
	for _, bin := range []string{"xdg-open", "open", "start"} {
		if binPath, err := exec.LookPath(bin); err == nil {
			return exec.Command(binPath, target).Start()
		}
	}
	return nil // silently ignore if no browser found
}
