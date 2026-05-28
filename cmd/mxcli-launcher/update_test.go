// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestShouldCheckUpdate_Missing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "last-check")
	if !shouldCheckUpdate(p) {
		t.Error("should check when file missing")
	}
}

func TestShouldCheckUpdate_TooRecent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "last-check")
	ts := time.Now().Add(-30 * time.Minute).Unix()
	os.WriteFile(p, []byte(strconv.FormatInt(ts, 10)), 0644)
	if shouldCheckUpdate(p) {
		t.Error("should not check within 1h")
	}
}

func TestShouldCheckUpdate_Expired(t *testing.T) {
	p := filepath.Join(t.TempDir(), "last-check")
	ts := time.Now().Add(-2 * time.Hour).Unix()
	os.WriteFile(p, []byte(strconv.FormatInt(ts, 10)), 0644)
	if !shouldCheckUpdate(p) {
		t.Error("should check after 1h")
	}
}

func TestFprintUpdateNotice_PrintsThenDeletes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "update-available")
	os.WriteFile(p, []byte("v1.2.3"), 0644)

	var stderr bytes.Buffer
	fprintUpdateNotice(&stderr, p)

	if !strings.Contains(stderr.String(), "v1.2.3") {
		t.Errorf("expected version in output, got: %q", stderr.String())
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("update-available file should be deleted after notice")
	}
}

func TestFprintUpdateNotice_NoFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "update-available")
	var stderr bytes.Buffer
	fprintUpdateNotice(&stderr, p)
	if stderr.Len() > 0 {
		t.Errorf("expected no output when file missing, got: %q", stderr.String())
	}
}
