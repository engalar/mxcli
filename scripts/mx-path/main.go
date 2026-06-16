// scripts/mx-path/main.go
// 通过 docker.CachedMxPath() 解析本机缓存的 mx 二进制路径。
// 用法：go run ./scripts/mx-path/main.go <version>
// 成功时向 stdout 输出绝对路径（无换行），失败时 exit 1/2。
package main

import (
	"fmt"
	"os"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/mx-path/main.go <version>")
		os.Exit(2)
	}
	version := os.Args[1]
	path := docker.CachedMxPath(version)
	if path == "" {
		fmt.Fprintf(os.Stderr,
			"ERROR: mx %s not cached\n  Run: go run ./cmd/mxcli setup mxbuild --version %s\n",
			version, version)
		os.Exit(1)
	}
	fmt.Print(path)
}
