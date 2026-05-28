// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"os"
	"time"
)

// Env holds injectable dependencies for launcher operations.
// Use DefaultEnv() in production; substitute fields in tests.
type Env struct {
	HomeDir     string
	HTTPClient  *http.Client
	upgradeLock *lockFile // non-nil while upgrade lock is held
}

// DefaultEnv returns an Env configured for production use.
func DefaultEnv() *Env {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return &Env{
		HomeDir:    home,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}
