// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
)

// ComponentConfig describes a managed binary component (daemon or local runner).
type ComponentConfig struct {
	Name           string // human-readable: "daemon" or "local"
	BinPath        string // installed binary path
	BakPath        string // backup binary path (for rollback)
	VersionPath    string // version file path
	BakVersionPath string // backup version file path
	TagPrefix      string // release tag prefix: "daemon-v" or "local-v"
	AssetName      string // binary name without platform suffix: "mxcli-daemon" or "mxcli-local"
}

// daemonComponentConfig returns the ComponentConfig for mxcli-daemon.
func (e *Env) daemonComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:           "daemon",
		BinPath:        e.daemonBinaryPath(),
		BakPath:        e.daemonBakPath(),
		VersionPath:    e.daemonVersionPath(),
		BakVersionPath: e.daemonVersionBakPath(),
		TagPrefix:      "daemon-v",
		AssetName:      "mxcli-daemon",
	}
}

// localComponentConfig returns the ComponentConfig for mxcli-local.
func (e *Env) localComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:           "local",
		BinPath:        e.localBinaryPath(),
		BakPath:        e.localBinaryBakPath(),
		VersionPath:    e.localVersionPath(),
		BakVersionPath: e.localVersionPath() + ".bak",
		TagPrefix:      "local-v",
		AssetName:      "mxcli-local",
	}
}

// upgradeComponent downloads and installs the latest release for a component.
// It fetches the latest tag matching cfg.TagPrefix, downloads the platform archive,
// verifies the SHA256 checksum, backs up the current binary, and installs the new one.
func (e *Env) upgradeComponent(cfg ComponentConfig) error {
	if err := os.MkdirAll(binaryDir(cfg.BinPath), 0700); err != nil {
		return fmt.Errorf("create %s dir: %w", cfg.Name, err)
	}

	tag, err := e.fetchLatestTagWithPrefix(cfg.TagPrefix)
	if err != nil {
		return fmt.Errorf("fetch latest %s tag: %w", cfg.Name, err)
	}

	current := readVersionFile(cfg.VersionPath)
	if current == tag {
		fmt.Printf("mxcli %s is already at %s — nothing to do.\n", cfg.Name, tag)
		return nil
	}
	fmt.Printf("Upgrading %s %s → %s\n", cfg.Name, current, tag)

	tmpDest := cfg.BinPath + ".new"
	defer os.Remove(tmpDest)

	if err := e.downloadLocalVersionForPlatform(tag, tmpDest, runtime.GOOS, runtime.GOARCH, cfg.AssetName); err != nil {
		return fmt.Errorf("download %s %s: %w", cfg.Name, tag, err)
	}

	// Backup current binary before replacing.
	if _, err := os.Stat(cfg.BinPath); err == nil {
		os.Rename(cfg.VersionPath, cfg.BakVersionPath)
		if err := os.Rename(cfg.BinPath, cfg.BakPath); err != nil {
			return fmt.Errorf("backup current %s: %w", cfg.Name, err)
		}
	}

	if err := os.Rename(tmpDest, cfg.BinPath); err != nil {
		return fmt.Errorf("install %s: %w", cfg.Name, err)
	}
	os.WriteFile(cfg.VersionPath, []byte(tag), 0644)
	fmt.Printf("✅ Upgraded %s to %s\n", cfg.Name, tag)
	return nil
}

// rollbackComponent restores the backup binary for a component.
func (e *Env) rollbackComponent(cfg ComponentConfig) error {
	if _, err := os.Stat(cfg.BakPath); err != nil {
		return fmt.Errorf("no backup available for %s", cfg.Name)
	}

	bakVer := readVersionFile(cfg.BakVersionPath)
	curVer := readVersionFile(cfg.VersionPath)
	fmt.Printf("Rolling back %s %s → %s\n", cfg.Name, curVer, bakVer)

	// Swap: current ↔ backup
	tmpBin := cfg.BinPath + ".rb-tmp"
	tmpVer := cfg.VersionPath + ".rb-tmp"
	os.Rename(cfg.BinPath, tmpBin)
	os.Rename(cfg.VersionPath, tmpVer)
	os.Rename(cfg.BakPath, cfg.BinPath)
	os.Rename(cfg.BakVersionPath, cfg.VersionPath)
	os.Rename(tmpBin, cfg.BakPath)
	os.Rename(tmpVer, cfg.BakVersionPath)

	fmt.Printf("✅ Rolled back %s to %s\n", cfg.Name, bakVer)
	return nil
}

// binaryDir returns the parent directory of a binary path.
func binaryDir(binPath string) string {
	for i := len(binPath) - 1; i >= 0; i-- {
		if binPath[i] == '/' || binPath[i] == '\\' {
			return binPath[:i]
		}
	}
	return "."
}
