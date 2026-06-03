// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
)

// ComponentPayload generalises DaemonPayload to any binary component
// (mxcli-daemon, mxcli-local, mxcli launcher).
type ComponentPayload struct {
	AssetName string
	Archive   []byte
	Checksum  string
}

// BuildComponentPayload builds a fake archive for the given component and platform.
// component is the binary name without extension, e.g. "mxcli-local" or "mxcli-daemon".
// Windows → .exe.zip; Linux/Darwin → .tar.zst.
func BuildComponentPayload(component, goos, goarch string, content []byte) (*ComponentPayload, error) {
	dp, err := BuildDaemonPayloadForPlatformNamed(component, goos, goarch, content)
	if err != nil {
		return nil, err
	}
	return &ComponentPayload{
		AssetName: dp.AssetName,
		Archive:   dp.Archive,
		Checksum:  dp.Checksum,
	}, nil
}

// ToComponentPayload converts a DaemonPayload to ComponentPayload for compatibility.
func (d *DaemonPayload) ToComponentPayload() *ComponentPayload {
	return &ComponentPayload{
		AssetName: d.AssetName,
		Archive:   d.Archive,
		Checksum:  d.Checksum,
	}
}

// BuildLocalPayload is a convenience wrapper for mxcli-local payloads.
func BuildLocalPayload(goos, goarch string, content []byte) (*ComponentPayload, error) {
	return BuildComponentPayload("mxcli-local", goos, goarch, content)
}

// LocalAssetName returns the release asset name for mxcli-local.
func LocalAssetName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("mxcli-local-%s-%s.exe.zip", goos, goarch)
	}
	return fmt.Sprintf("mxcli-local-%s-%s.tar.zst", goos, goarch)
}
