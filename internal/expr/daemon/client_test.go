// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_DefaultsSocketPath(t *testing.T) {
	t.Parallel()
	c := daemon.NewClient(daemon.ClientOptions{MprPath: "/tmp/foo/App.mpr"})
	assert.NotEmpty(t, c.SocketPath())
	assert.Equal(t, daemon.SocketPath("/tmp/foo/App.mpr"), c.SocketPath())
}

func TestNewClient_ExplicitSocketPath(t *testing.T) {
	t.Parallel()
	c := daemon.NewClient(daemon.ClientOptions{
		MprPath:    "/tmp/foo/App.mpr",
		SocketPath: "/tmp/custom_42.sock",
	})
	assert.Equal(t, "/tmp/custom_42.sock", c.SocketPath())
}

func TestClient_PingAndValidate(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	// Use a unique socket so this test can run in parallel with other Serve() tests.
	sock := filepath.Join(t.TempDir(), "c.sock")
	d, err := daemon.NewWithSocket(corpusAMPR, sock, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })
	waitForAlive(t, sock)

	c := daemon.NewClient(daemon.ClientOptions{MprPath: corpusAMPR, SocketPath: sock})

	// Ping
	pr, err := c.Ping()
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.True(t, pr.OK)
	assert.Equal(t, corpusAMPR, pr.MprPath)
	assert.Greater(t, pr.EntityCount, 0)

	// Validate
	vr, err := c.Validate(daemon.ValidateRequest{MprPath: corpusAMPR})
	require.NoError(t, err)
	require.NotNil(t, vr)
	assert.Empty(t, vr.Error)
	assert.NotEmpty(t, vr.IndexAge)
}

func TestClient_StartIfNeeded_NoOpWhenAlive(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	sock := filepath.Join(t.TempDir(), "s.sock")
	d, err := daemon.NewWithSocket(corpusAMPR, sock, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })
	waitForAlive(t, sock)

	c := daemon.NewClient(daemon.ClientOptions{MprPath: corpusAMPR, SocketPath: sock})
	// daemon 已活：StartIfNeeded 必须立即返回 nil，不尝试 exec.Command
	require.NoError(t, c.StartIfNeeded())
}
