package tui

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
)

// detectImageProtocol checks which inline image protocol the terminal supports.
// Priority: $MXCLI_IMAGE_PROTOCOL env var > chafa auto-detect > static env checks.
// Returns "kitty", "iterm2", "sixel", "chafa", or "" if none detected.
//
// SSH note: $WT_SESSION (Windows Terminal) is not forwarded over SSH.
// Set MXCLI_IMAGE_PROTOCOL=sixel in ~/.bashrc on the remote server to enable
// Sixel rendering when connected from Windows Terminal or other Sixel-capable clients.
func detectImageProtocol() string {
	// User-explicit override — works over SSH
	if p := os.Getenv("MXCLI_IMAGE_PROTOCOL"); p != "" {
		return p
	}

	// chafa auto-detects terminal capabilities (including DA1 query through SSH)
	if _, err := exec.LookPath("chafa"); err == nil {
		return "chafa"
	}

	// Local terminal detection (not forwarded over SSH)
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("TERM") == "xterm-kitty" {
		return "kitty"
	}
	// Windows Terminal — only reachable when running locally (not over SSH)
	if os.Getenv("WT_SESSION") != "" {
		return "sixel"
	}
	prog := os.Getenv("TERM_PROGRAM")
	if prog == "iTerm.app" {
		return "iterm2"
	}
	// WezTerm supports both iTerm2 and Sixel; prefer Sixel for broader compat
	if prog == "WezTerm" {
		return "sixel"
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

// renderImageChafa renders an image file using chafa, which auto-detects the
// best protocol for the current terminal (including Sixel via DA1 query).
func renderImageChafa(path string) string {
	out, err := exec.Command("chafa", "--format=symbols", path).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// renderImageSixel renders an image file using the Sixel protocol via img2sixel.
// Falls back to chafa --format=sixel if img2sixel is not available.
func renderImageSixel(path string) string {
	if p, err := exec.LookPath("img2sixel"); err == nil {
		out, err := exec.Command(p, path).Output()
		if err == nil {
			return string(out)
		}
	}
	out, err := exec.Command("chafa", "--format=sixel", path).Output()
	if err != nil {
		return ""
	}
	return string(out)
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
		case "sixel":
			sb.WriteString(renderImageSixel(p))
		case "chafa":
			sb.WriteString(renderImageChafa(p))
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
