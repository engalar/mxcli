// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/mendixlabs/mxcli/internal/expr/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_DefaultsSocketPath(t *testing.T) {
	c := daemon.NewClient(daemon.ClientOptions{MprPath: "/tmp/foo/App.mpr"})
	assert.NotEmpty(t, c.SocketPath())
	assert.Equal(t, daemon.SocketPath("/tmp/foo/App.mpr"), c.SocketPath())
}

func TestNewClient_ExplicitSocketPath(t *testing.T) {
	c := daemon.NewClient(daemon.ClientOptions{
		MprPath:    "/tmp/foo/App.mpr",
		SocketPath: "/tmp/custom_42.sock",
	})
	assert.Equal(t, "/tmp/custom_42.sock", c.SocketPath())
}

func TestClient_PingAndValidate(t *testing.T) {
	macnicaMPR := testutil.FindMPR(t, "MACNICA_MPR", "testdata/macnica/MacnicaApp.mpr")

	// 直接启动 in-process daemon（无需走 StartIfNeeded → exec.Command 子进程路径），
	// 这样 client.go 的 Ping/Validate 可以独立验证而不依赖 Task 8 的 CLI。
	d, err := daemon.New(macnicaMPR, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(d.SocketPath()) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, daemon.IsAlive(d.SocketPath()))

	c := daemon.NewClient(daemon.ClientOptions{MprPath: macnicaMPR})

	// Ping
	pr, err := c.Ping()
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.True(t, pr.OK)
	assert.Equal(t, macnicaMPR, pr.MprPath)
	assert.Greater(t, pr.EntityCount, 0)

	// Validate
	vr, err := c.Validate(daemon.ValidateRequest{MprPath: macnicaMPR})
	require.NoError(t, err)
	require.NotNil(t, vr)
	assert.Empty(t, vr.Error)
	assert.NotEmpty(t, vr.IndexAge)
}

func TestClient_StartIfNeeded_NoOpWhenAlive(t *testing.T) {
	macnicaMPR := testutil.FindMPR(t, "MACNICA_MPR", "testdata/macnica/MacnicaApp.mpr")

	d, err := daemon.New(macnicaMPR, 5*time.Minute)
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	t.Cleanup(func() { _ = d.Stop() })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if daemon.IsAlive(d.SocketPath()) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, daemon.IsAlive(d.SocketPath()))

	c := daemon.NewClient(daemon.ClientOptions{MprPath: macnicaMPR})
	// daemon 已活：StartIfNeeded 必须立即返回 nil，不尝试 exec.Command
	require.NoError(t, c.StartIfNeeded())
}
