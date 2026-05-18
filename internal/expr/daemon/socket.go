// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// SocketPath 根据 MPR 文件绝对路径生成唯一的 Unix socket 文件路径。
func SocketPath(mprPath string) string {
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		abs = mprPath
	}
	hash := sha256.Sum256([]byte(abs))
	name := fmt.Sprintf("%x.sock", hash[:4])

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	return filepath.Join(dir, name)
}

// IsAlive 检查给定 socket 路径的 daemon 是否在运行。
func IsAlive(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// EnsureDaemonDir 确保 socket 目录存在。
func EnsureDaemonDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	return os.MkdirAll(dir, 0o700)
}

// ListRunning 扫描 daemon 目录，返回所有活跃 daemon 的状态。
func ListRunning() ([]PingResponse, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []PingResponse
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sock") {
			continue
		}
		sockPath := filepath.Join(dir, e.Name())
		if !IsAlive(sockPath) {
			continue
		}
		conn, err := net.Dial("unix", sockPath)
		if err != nil {
			continue
		}
		_ = json.NewEncoder(conn).Encode(ValidateRequest{})
		var resp PingResponse
		if err := json.NewDecoder(conn).Decode(&resp); err == nil {
			results = append(results, resp)
		}
		_ = conn.Close()
	}
	return results, nil
}
