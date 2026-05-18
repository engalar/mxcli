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

// SocketPath 根据 MPR 文件绝对路径和当前二进制版本生成唯一的 Unix socket 路径。
//
// 文件名格式：{mprHash[0:4]}-{binHash[0:2]}.sock
//   - mprHash 前缀标识 MPR，方便清理同一项目的旧版本 socket
//   - binHash 后缀随二进制重新编译而变化，使旧 daemon 自动失效
func SocketPath(mprPath string) string {
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		abs = mprPath
	}

	mprHash := sha256.Sum256([]byte(abs))
	binHash := sha256.Sum256([]byte(binaryMtime()))

	name := fmt.Sprintf("%x-%x.sock", mprHash[:4], binHash[:2])

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	return filepath.Join(dir, name)
}

// mprSocketPrefix 返回给定 MPR 路径的 socket 文件名前缀（不含二进制版本部分）。
// 用于识别同一 MPR 的所有历史 socket 文件。
func mprSocketPrefix(mprPath string) string {
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		abs = mprPath
	}
	hash := sha256.Sum256([]byte(abs))
	return fmt.Sprintf("%x-", hash[:4])
}

// binaryMtime 返回当前可执行文件的修改时间（纳秒字符串）。
// 读取失败时返回空字符串（退化为仅基于 MPR 路径的哈希）。
func binaryMtime() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	info, err := os.Stat(exe)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", info.ModTime().UnixNano())
}

// CleanupStaleSocketsForMPR 在启动新 daemon 前清理磁盘上的旧 socket 文件。
//
// 策略：
//   - 同 MPR 的旧版本 socket（前缀相同、后缀不同）：无论是否存活都删除。
//     删除 socket 文件等效于"优雅停止"旧 daemon——它无法再接受新连接，
//     idle watcher 触发后自动退出。
//   - 不属于当前 MPR 的 socket：仅在已死亡（无进程监听）时删除，
//     避免干扰其他 MPR 的 daemon。
func CleanupStaleSocketsForMPR(mprPath string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := mprSocketPrefix(mprPath)
	current := filepath.Base(SocketPath(mprPath))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sock") {
			continue
		}
		sockPath := filepath.Join(dir, name)
		if name == current {
			continue // 当前版本，保留
		}
		if strings.HasPrefix(name, prefix) {
			// 同 MPR 旧版本：直接删除（停止旧 daemon）
			_ = os.Remove(sockPath)
		} else if !IsAlive(sockPath) {
			// 其他 MPR 的死亡 socket：清理孤儿文件
			_ = os.Remove(sockPath)
		}
	}
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
