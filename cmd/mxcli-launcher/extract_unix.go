// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

// extractTarZst extracts expectedName from a .tar.zst archive into destPath.
// Used on Linux and macOS where daemon archives are compressed with zstd.
func extractTarZst(r io.Reader, destPath, expectedName string) error {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != expectedName {
			continue
		}
		tmp := destPath + ".tmp"
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		f.Close()
		return os.Rename(tmp, destPath)
	}
	return fmt.Errorf("no file named %q found in archive", expectedName)
}
