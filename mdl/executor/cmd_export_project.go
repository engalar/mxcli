// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/model"
)

// ExportOptions controls the behaviour of ExportProject.
type ExportOptions struct {
	Module   string
	DryRun   bool
	Progress func(line string)
}

// documentFilePath returns the output file path for a single document.
func documentFilePath(outputDir, moduleName, folderPath, qname string) string {
	if folderPath != "" {
		return filepath.Join(outputDir, moduleName, filepath.FromSlash(folderPath), qname+".mdl")
	}
	return filepath.Join(outputDir, moduleName, qname+".mdl")
}

func classifyModules(mods []*model.Module) (regular, marketplace []*model.Module) {
	for _, m := range mods {
		if m.FromAppStore {
			marketplace = append(marketplace, m)
		} else {
			regular = append(regular, m)
		}
	}
	return
}

func marketplaceFileContent(mods []*model.Module) string {
	var sb strings.Builder
	sb.WriteString("-- Marketplace modules detected in this project.\n")
	sb.WriteString("-- Reinstall these before running mxcli import.\n")
	sb.WriteString("--\n")
	for _, m := range mods {
		version := m.AppStoreVersion
		if version == "" {
			version = "unknown"
		}
		fmt.Fprintf(&sb, "-- Module: %-30s (version: %s)\n", m.Name, version)
	}
	return sb.String()
}
