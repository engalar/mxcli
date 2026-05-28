// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func TestRunUpgradeWithEnv_ConcurrentLock(t *testing.T) {
	// Both goroutines call upgradeWithLock on the same Env.
	// Exactly one should succeed (return nil), the other should return an error.
	e := &Env{HomeDir: t.TempDir(), HTTPClient: nil}
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}

	results := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = e.acquireUpgradeLock()
			if results[i] == nil {
				// Hold the lock briefly to force contention
				time.Sleep(10 * time.Millisecond)
				e.releaseUpgradeLock()
			}
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 lock acquisition, got %d (errors: %v, %v)", successes, results[0], results[1])
	}
}
