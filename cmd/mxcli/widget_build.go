// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// widgetInfo holds the discovered metadata for one widget (from src/<Name>.xml).
type widgetInfo struct {
	Name        string // e.g. "MySlider"
	WidgetID    string // e.g. "com.acme.widget.MySlider.MySlider"
	DisplayName string // e.g. "My Slider" (from <name> element)
	XMLPath     string // absolute path to the XML file
}

// xmlWidgetRoot is a minimal struct for parsing just id and name from a widget XML.
type xmlWidgetRoot struct {
	XMLName     xml.Name `xml:"widget"`
	ID          string   `xml:"id,attr"`
	DisplayName string   `xml:"name"`
}

// discoverWidgets globs src/*.xml in projectDir and parses each to extract widgetID and name.
func discoverWidgets(projectDir string) ([]widgetInfo, error) {
	pattern := filepath.Join(projectDir, "src", "*.xml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var infos []widgetInfo
	for _, xmlPath := range matches {
		data, err := os.ReadFile(xmlPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", xmlPath, err)
		}
		var root xmlWidgetRoot
		if err := xml.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", xmlPath, err)
		}
		name := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
		infos = append(infos, widgetInfo{
			Name:        name,
			WidgetID:    root.ID,
			DisplayName: root.DisplayName,
			XMLPath:     xmlPath,
		})
	}
	return infos, nil
}

// validateWidgetInfo checks that a discovered widget has a valid ID format and non-empty name.
func validateWidgetInfo(info widgetInfo) error {
	parts := strings.Split(info.WidgetID, ".")
	if len(parts) < 4 {
		return fmt.Errorf("widget %q: widget ID must have at least 4 dot-separated segments (e.g. com.acme.widget.MyName), got %q", info.Name, info.WidgetID)
	}
	if info.DisplayName == "" {
		return fmt.Errorf("widget %q: <name> element is empty in XML", info.Name)
	}
	return nil
}

// detectToolchain returns "bun" or "npm" depending on what is available in PATH.
func detectToolchain() (string, error) {
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun", nil
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm", nil
	}
	return "", fmt.Errorf("bun not found, npm not found\n" +
		"  install bun: curl -fsSL https://bun.sh/install | bash\n" +
		"  install npm: https://nodejs.org/")
}

// installDeps runs bun install or npm install if node_modules/ is absent.
func installDeps(projectDir, tool string) error {
	if _, err := os.Stat(filepath.Join(projectDir, "node_modules")); err == nil {
		return nil
	}
	fmt.Printf("Installing dependencies (%s install)...\n", tool)
	cmd := exec.Command(tool, "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
