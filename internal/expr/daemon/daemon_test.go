// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForAlive polls the socket until it is alive or a 6-second deadline
// expires. 6 s covers the corpus-a index build (~5 s) plus headroom.
func waitForAlive(t *testing.T, sockPath string) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(sockPath) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, daemon.IsAlive(sockPath), "daemon not alive within 6 s at %s", sockPath)
}

// newTestDaemon starts a daemon bound to a unique socket in t.TempDir()
// so tests that call Serve() can run in parallel without socket conflicts.
func newTestDaemon(t *testing.T, mprPath string) *daemon.Daemon {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "d.sock")
	d, err := daemon.NewWithSocket(mprPath, sock, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })
	return d
}

func TestDaemon_ServeAndPing(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	d := newTestDaemon(t, corpusAMPR)
	waitForAlive(t, d.SocketPath())

	// 用空 MprPath 触发 ping 路径
	conn, err := net.Dial("unix", d.SocketPath())
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, json.NewEncoder(conn).Encode(daemon.ValidateRequest{}))
	var resp daemon.PingResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))

	assert.True(t, resp.OK)
	assert.Equal(t, corpusAMPR, resp.MprPath)
	assert.Greater(t, resp.EntityCount, 0, "EntityCount > 0")
	assert.Greater(t, resp.EnumCount, 0, "EnumCount > 0")
}

func TestDaemon_ValidateReturnsResults(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	d := newTestDaemon(t, corpusAMPR)
	waitForAlive(t, d.SocketPath())

	conn, err := net.Dial("unix", d.SocketPath())
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, json.NewEncoder(conn).Encode(daemon.ValidateRequest{
		MprPath: corpusAMPR,
	}))
	var resp daemon.ValidateResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))

	assert.Empty(t, resp.Error, "no daemon-side error expected")
	assert.NotNil(t, resp.Results)
}

func TestDaemon_SocketDirExists(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	d, err := daemon.New(corpusAMPR, 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	sockPath := daemon.SocketPath(corpusAMPR)
	_, err = os.Stat(filepath.Dir(sockPath))
	assert.NoError(t, err, "socket parent dir must exist after New()")
}

func TestDaemon_NewWithSocket_HonoursOverride(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	custom := filepath.Join(t.TempDir(), "custom.sock")
	d, err := daemon.NewWithSocket(corpusAMPR, custom, 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	assert.Equal(t, custom, d.SocketPath(), "SocketPath() must reflect the override")
	assert.NotEqual(t, daemon.SocketPath(corpusAMPR), d.SocketPath(),
		"override should differ from default SocketPath(mprPath)")
}

func TestDaemon_NewWithSocket_EmptyFallsBackToDefault(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	d, err := daemon.NewWithSocket(corpusAMPR, "", 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	assert.Equal(t, daemon.SocketPath(corpusAMPR), d.SocketPath(),
		"empty socketPath should fall back to SocketPath(mprPath)")
}

func TestDaemon_NewWithSocket_ServeBindsCustom(t *testing.T) {
	t.Parallel()
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	d := newTestDaemon(t, corpusAMPR)
	waitForAlive(t, d.SocketPath())

	assert.True(t, daemon.IsAlive(d.SocketPath()),
		"Serve() must bind the override socket path")
	assert.False(t, daemon.IsAlive(daemon.SocketPath(corpusAMPR)),
		"default SocketPath must NOT be bound when override is given")
}
