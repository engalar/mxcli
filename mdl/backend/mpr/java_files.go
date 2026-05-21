// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// javaSourceDir returns the javasource/actions directory for the given module.
func (b *MprBackend) javaSourceDir(moduleName string) string {
	projectRoot := filepath.Dir(b.path)
	return filepath.Join(projectRoot, "javasource", strings.ToLower(moduleName), "actions")
}

// writeJavaSourceFileViaPathGen renders + writes the .java file for a
// gen-typed JavaAction. Stage 3.3.2.E1: previously bridged to
// sdkmpr.GenerateJavaSource via gen→sdk shims; now generates the Java
// source directly from gen types via generateJavaSourceGen, eliminating
// the sdk/javaactions and sdk/mpr dependencies from this file.
func (b *MprBackend) writeJavaSourceFileViaPathGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error {
	dir := b.javaSourceDir(moduleName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create javasource directory: %w", err)
	}
	filePath := filepath.Join(dir, actionName+".java")

	// If the file already exists, update sections in-place (merge imports,
	// replace code/extra). Generate a fresh skeleton only for new files.
	if _, statErr := os.Stat(filePath); statErr == nil {
		return b.updateJavaSourceSections(moduleName, actionName, javaCode, extraImports, extraCode)
	}

	source := generateJavaSourceGen(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
	if err := os.WriteFile(filePath, []byte(source), 0644); err != nil {
		return fmt.Errorf("write Java source file: %w", err)
	}
	return nil
}

func (b *MprBackend) deleteJavaSourceFileViaPath(moduleName, actionName string) error {
	filePath := filepath.Join(b.javaSourceDir(moduleName), actionName+".java")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete Java source file: %w", err)
	}
	return nil
}

func (b *MprBackend) renameJavaSourceFileViaPath(moduleName, oldName, newName string) error {
	dir := b.javaSourceDir(moduleName)
	oldPath := filepath.Join(dir, oldName+".java")
	newPath := filepath.Join(dir, newName+".java")
	if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rename Java source file: %w", err)
	}
	return nil
}

func (b *MprBackend) readJavaSourceFileViaPath(moduleName, actionName string) (string, error) {
	filePath := filepath.Join(b.javaSourceDir(moduleName), actionName+".java")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read Java source file: %w", err)
	}
	return string(content), nil
}

// updateJavaSourceSections updates an existing .java file in-place:
//   - userCode: replaces text between // BEGIN USER CODE / // END USER CODE (skipped if "")
//   - newImports: merged into the import block (idempotent — duplicates skipped)
//   - extraCode: replaces text between // BEGIN EXTRA CODE / // END EXTRA CODE (skipped if "")
func (b *MprBackend) updateJavaSourceSections(moduleName, actionName, userCode string, newImports []string, extraCode string) error {
	filePath := filepath.Join(b.javaSourceDir(moduleName), actionName+".java")
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read java source for update: %w", err)
	}
	content := string(raw)

	if len(newImports) > 0 {
		content = mergeJavaImports(content, newImports)
	}
	if userCode != "" {
		content, err = replaceJavaSection(content, "// BEGIN USER CODE", "// END USER CODE", userCode)
		if err != nil {
			return fmt.Errorf("replace user code section: %w", err)
		}
	}
	if extraCode != "" {
		content, err = replaceJavaSection(content, "// BEGIN EXTRA CODE", "// END EXTRA CODE", extraCode)
		if err != nil {
			return fmt.Errorf("replace extra code section: %w", err)
		}
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// mergeJavaImports adds each line from newImports into src's import block,
// skipping lines already present. New imports are inserted after the last
// existing import or package line.
func mergeJavaImports(src string, newImports []string) string {
	existing := make(map[string]bool)
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "import ") {
			existing[t] = true
		}
	}
	var toAdd []string
	for _, imp := range newImports {
		imp = strings.TrimSpace(imp)
		if imp != "" && !existing[imp] {
			toAdd = append(toAdd, imp)
		}
	}
	if len(toAdd) == 0 {
		return src
	}
	lines := strings.Split(src, "\n")
	insertAt := -1
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "package ") {
			insertAt = i
		}
	}
	if insertAt < 0 {
		return src
	}
	var result []string
	result = append(result, lines[:insertAt+1]...)
	result = append(result, toAdd...)
	result = append(result, lines[insertAt+1:]...)
	return strings.Join(result, "\n")
}

// replaceJavaSection replaces content between beginMarker and endMarker
// (exclusive) with newContent indented to match the endMarker's indentation.
func replaceJavaSection(src, beginMarker, endMarker, newContent string) (string, error) {
	beginIdx := strings.Index(src, beginMarker)
	endIdx := strings.Index(src, endMarker)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return src, fmt.Errorf("markers %q / %q not found", beginMarker, endMarker)
	}

	// Detect indentation from the line containing endMarker
	indent := ""
	lineStart := strings.LastIndex(src[:endIdx], "\n")
	if lineStart >= 0 {
		rest := src[lineStart+1 : endIdx]
		for _, ch := range rest {
			if ch == '\t' || ch == ' ' {
				indent += string(ch)
			} else {
				break
			}
		}
	}

	var middle strings.Builder
	middle.WriteString("\n")
	for _, line := range strings.Split(newContent, "\n") {
		if strings.TrimSpace(line) == "" {
			middle.WriteString("\n")
		} else {
			middle.WriteString(indent)
			middle.WriteString(line)
			middle.WriteString("\n")
		}
	}
	middle.WriteString(indent)

	before := src[:beginIdx+len(beginMarker)]
	after := src[endIdx:]
	return before + middle.String() + after, nil
}
