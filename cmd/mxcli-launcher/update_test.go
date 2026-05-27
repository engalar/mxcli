// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strconv"
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
