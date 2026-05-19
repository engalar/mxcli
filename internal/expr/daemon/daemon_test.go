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

func TestDaemon_ServeAndPing(t *testing.T) {
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	d, err := daemon.New(corpusAMPR, 5*time.Minute)
	require.NoError(t, err)

	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	// 等待 socket 出现 (最多 2s)
	sockPath := daemon.SocketPath(corpusAMPR)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(sockPath) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, daemon.IsAlive(sockPath), "socket should be alive after Serve")

	// 用空 MprPath 触发 ping 路径
	conn, err := net.Dial("unix", sockPath)
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
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")

	d, err := daemon.New(corpusAMPR, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	sockPath := daemon.SocketPath(corpusAMPR)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(sockPath) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, daemon.IsAlive(sockPath))

	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, json.NewEncoder(conn).Encode(daemon.ValidateRequest{
		MprPath: corpusAMPR,
	}))
	var resp daemon.ValidateResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))

	assert.Empty(t, resp.Error, "no daemon-side error expected")
	// corpus-a 项目期望至少有少量 SEM 命中（不强求精确数字）
	// 仅验证 daemon pipeline 跑通；results 可能为 0 也可能 >0。
	assert.NotNil(t, resp.Results)
}

func TestDaemon_SocketDirExists(t *testing.T) {
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	d, err := daemon.New(corpusAMPR, 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	sockPath := daemon.SocketPath(corpusAMPR)
	_, err = os.Stat(filepath.Dir(sockPath))
	assert.NoError(t, err, "socket parent dir must exist after New()")
}

func TestDaemon_NewWithSocket_HonoursOverride(t *testing.T) {
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
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	d, err := daemon.NewWithSocket(corpusAMPR, "", 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	assert.Equal(t, daemon.SocketPath(corpusAMPR), d.SocketPath(),
		"empty socketPath should fall back to SocketPath(mprPath)")
}

func TestDaemon_NewWithSocket_ServeBindsCustom(t *testing.T) {
	corpusAMPR := testutil.FindMPR(t, "CORPUS_A_MPR", "testdata/corpus-a/app.mpr")
	custom := filepath.Join(t.TempDir(), "bind.sock")
	d, err := daemon.NewWithSocket(corpusAMPR, custom, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(custom) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, daemon.IsAlive(custom),
		"Serve() must bind the override socket path")
	assert.False(t, daemon.IsAlive(daemon.SocketPath(corpusAMPR)),
		"default SocketPath must NOT be bound when override is given")
}
