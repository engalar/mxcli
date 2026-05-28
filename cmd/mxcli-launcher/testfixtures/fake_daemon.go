// SPDX-License-Identifier: Apache-2.0

// Package testfixtures provides test helpers for launcher install/update tests.
package testfixtures

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"

	"github.com/klauspost/compress/zstd"
)

// DaemonPayload holds a fake daemon archive and its SHA256 checksum.
type DaemonPayload struct {
	// AssetName is the filename the fake server uses, e.g. "mxcli-daemon-linux-amd64.tar.zst"
	AssetName string
	// Archive is the raw bytes of the tar.zst or .exe.zip archive.
	Archive []byte
	// Checksum is the correct SHA256 hex digest of Archive.
	Checksum string
}

// BuildDaemonPayload creates a fake daemon archive for the current platform.
// Windows → .exe.zip containing "mxcli-daemon.exe".
// Linux/Darwin → .tar.zst containing "mxcli-daemon".
func BuildDaemonPayload(content []byte) (*DaemonPayload, error) {
	return BuildDaemonPayloadForPlatform(runtime.GOOS, runtime.GOARCH, content)
}

// BuildDaemonPayloadForPlatform creates a fake daemon archive for an arbitrary platform.
// This allows Linux CI to test the Windows download path without a real Windows machine.
func BuildDaemonPayloadForPlatform(goos, goarch string, content []byte) (*DaemonPayload, error) {
	if goos == "windows" {
		return buildWindowsPayload(goos, goarch, content)
	}
	return buildUnixPayload(goos, goarch, content)
}

func buildWindowsPayload(goos, goarch string, content []byte) (*DaemonPayload, error) {
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s.exe.zip", goos, goarch)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("mxcli-daemon.exe")
	if err != nil {
		return nil, fmt.Errorf("zip create: %w", err)
	}
	if _, err := f.Write(content); err != nil {
		return nil, fmt.Errorf("zip write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zip close: %w", err)
	}

	archiveBytes := buf.Bytes()
	h := sha256.Sum256(archiveBytes)
	return &DaemonPayload{
		AssetName: assetName,
		Archive:   archiveBytes,
		Checksum:  hex.EncodeToString(h[:]),
	}, nil
}

func buildUnixPayload(goos, goarch string, content []byte) (*DaemonPayload, error) {
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s.tar.zst", goos, goarch)

	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("zstd writer: %w", err)
	}
	tw := tar.NewWriter(zw)
	hdr := &tar.Header{
		Name:     "mxcli-daemon",
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	tw.Close()
	zw.Close()

	archiveBytes := buf.Bytes()
	h := sha256.Sum256(archiveBytes)
	return &DaemonPayload{
		AssetName: assetName,
		Archive:   archiveBytes,
		Checksum:  hex.EncodeToString(h[:]),
	}, nil
}

// CorruptChecksum returns a deliberately wrong SHA256 hex string (all zeros).
func CorruptChecksum() string {
	return "0000000000000000000000000000000000000000000000000000000000000000"
}
