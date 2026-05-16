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
	source := generateJavaSourceGen(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
	filePath := filepath.Join(dir, actionName+".java")
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
