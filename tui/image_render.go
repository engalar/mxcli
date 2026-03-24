package tui

import (
	"encoding/base64"
	"os"
	"strings"
)

// detectImageProtocol checks which inline image protocol the terminal supports.
// Returns "kitty", "iterm2", or "" if none detected.
func detectImageProtocol() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if os.Getenv("TERM") == "xterm-kitty" {
		return "kitty"
	}
	prog := os.Getenv("TERM_PROGRAM")
	if prog == "iTerm.app" || prog == "WezTerm" {
		return "iterm2"
	}
	return ""
}

// renderImageKitty renders an image file using the Kitty graphics protocol.
// Uses t=f (transmit file path) so the terminal reads the file directly.
func renderImageKitty(path string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(path))
	return "\x1b_Ga=T,f=32,t=f,q=2;" + encoded + "\x1b\\"
}

// renderImageIterm2 renders an image file using the iTerm2 inline image protocol.
// Reads the file and embeds the base64-encoded content.
func renderImageIterm2(path string) string {
	imageBytes, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	return "\x1b]1337;File=inline=1;width=auto:" + encoded + "\a"
}

// renderImages renders a list of image file paths using the detected terminal protocol.
// Returns empty string if the terminal does not support inline images.
func renderImages(paths []string) string {
	protocol := detectImageProtocol()
	if protocol == "" || len(paths) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, p := range paths {
		switch protocol {
		case "kitty":
			sb.WriteString(renderImageKitty(p))
		case "iterm2":
			sb.WriteString(renderImageIterm2(p))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// extractImagePaths parses DESCRIBE IMAGE COLLECTION output and extracts
// file paths from lines matching: IMAGE "name" FROM FILE '/path/to/file'
func extractImagePaths(output string) []string {
	var paths []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, "FROM FILE '")
		if idx == -1 {
			continue
		}
		rest := line[idx+len("FROM FILE '"):]
		// Strip trailing quote and optional comma/semicolon
		end := strings.Index(rest, "'")
		if end == -1 {
			continue
		}
		paths = append(paths, rest[:end])
	}
	return paths
}
