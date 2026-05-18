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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func TestDaemon_ServeAndPing(t *testing.T) {
	if _, err := os.Stat(macnicaMPR); err != nil {
		t.Skipf("MPR fixture not available: %v", err)
	}

	d, err := daemon.New(macnicaMPR, 5*time.Minute)
	require.NoError(t, err)

	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	// 等待 socket 出现 (最多 2s)
	sockPath := daemon.SocketPath(macnicaMPR)
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
	assert.Equal(t, macnicaMPR, resp.MprPath)
	assert.Greater(t, resp.EntityCount, 0, "EntityCount > 0")
	assert.Greater(t, resp.EnumCount, 0, "EnumCount > 0")
}

func TestDaemon_ValidateReturnsResults(t *testing.T) {
	if _, err := os.Stat(macnicaMPR); err != nil {
		t.Skipf("MPR fixture not available: %v", err)
	}

	d, err := daemon.New(macnicaMPR, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	sockPath := daemon.SocketPath(macnicaMPR)
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
		MprPath: macnicaMPR,
	}))
	var resp daemon.ValidateResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))

	assert.Empty(t, resp.Error, "no daemon-side error expected")
	// macnica 项目期望至少有少量 SEM 命中（不强求精确数字）
	// 仅验证 daemon pipeline 跑通；results 可能为 0 也可能 >0。
	assert.NotNil(t, resp.Results)
}

func TestDaemon_SocketDirExists(t *testing.T) {
	if _, err := os.Stat(macnicaMPR); err != nil {
		t.Skipf("MPR fixture not available: %v", err)
	}
	d, err := daemon.New(macnicaMPR, 5*time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Stop() })

	sockPath := daemon.SocketPath(macnicaMPR)
	_, err = os.Stat(filepath.Dir(sockPath))
	assert.NoError(t, err, "socket parent dir must exist after New()")
}
