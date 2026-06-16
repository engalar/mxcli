// scripts/mx-path/main.go
// 解析本机可执行的 mx 二进制路径，供 bash 脚本调用。
// 使用与 mxcli new 相同的解析逻辑：Windows/macOS 优先找 Studio Pro，
// Linux 或找不到时才查 CDN cache。
// 用法：go run ./scripts/mx-path/main.go <version>
// 成功时向 stdout 输出绝对路径（无换行），失败时 exit 1/2。
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/mx-path/main.go <version>")
		os.Exit(2)
	}
	version := os.Args[1]
	path, err := docker.ResolveMxForNewProject(version, io.Discard)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: mx %s not found: %v\n  On Windows: install Mendix Studio Pro %s\n  On Linux:   go run ./cmd/mxcli setup mxbuild --version %s\n",
			version, err, version, version)
		os.Exit(1)
	}
	fmt.Print(path)
}
