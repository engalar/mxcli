// SPDX-License-Identifier: Apache-2.0

// Package testfixtures provides test helpers for launcher install/update tests.
package testfixtures

import (
	"archive/tar"
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
	// Archive is the raw bytes of the tar.zst archive.
	Archive []byte
	// Checksum is the correct SHA256 hex digest of Archive.
	Checksum string
}

// BuildDaemonPayload creates a minimal tar.zst containing a fake daemon binary
// for the current platform. The binary content is the provided content bytes.
func BuildDaemonPayload(content []byte) (*DaemonPayload, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	binaryName := "mxcli-daemon"
	if goos == "windows" {
		binaryName = "mxcli-daemon.exe"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s.tar.zst", goos, goarch)

	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("zstd writer: %w", err)
	}
	tw := tar.NewWriter(zw)
	hdr := &tar.Header{
		Name:     binaryName,
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
	checksum := hex.EncodeToString(h[:])

	return &DaemonPayload{
		AssetName: assetName,
		Archive:   archiveBytes,
		Checksum:  checksum,
	}, nil
}

// CorruptChecksum returns a deliberately wrong SHA256 hex string (all zeros).
func CorruptChecksum() string {
	return "0000000000000000000000000000000000000000000000000000000000000000"
}
