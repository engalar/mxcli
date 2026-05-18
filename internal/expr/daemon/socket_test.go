// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/stretchr/testify/assert"
)

func TestSocketPath_Deterministic(t *testing.T) {
	p1 := daemon.SocketPath("/a/b/project.mpr")
	p2 := daemon.SocketPath("/a/b/project.mpr")
	assert.Equal(t, p1, p2, "同一 MPR 路径应得到同一 socket 路径")
}

func TestSocketPath_DifferentMPR(t *testing.T) {
	p1 := daemon.SocketPath("/a/App.mpr")
	p2 := daemon.SocketPath("/b/App.mpr")
	assert.NotEqual(t, p1, p2, "不同 MPR 路径应得到不同 socket 路径")
}

func TestSocketPath_InDaemonDir(t *testing.T) {
	p := daemon.SocketPath("/some/path/project.mpr")
	assert.Contains(t, p, ".mxcli", "socket 应在 .mxcli 目录下")
	assert.Contains(t, p, ".sock", "应为 .sock 后缀")
	assert.True(t, filepath.IsAbs(p), "应为绝对路径")
}

func TestIsAlive_NoSocket(t *testing.T) {
	alive := daemon.IsAlive("/tmp/nonexistent_test_9999.sock")
	assert.False(t, alive, "不存在的 socket 应返回 false")
}
