package main

import (
	"fmt"
	"os"

	"github.com/sunquan/rick/internal/cmd"
)

func main() {
	// 单一版本源（human 裁决 2026-11-28「就一个 rick 版本」）：rick 版本 =
	// 构建指纹 BuildID（ldflags 注入：release --version 8.1.12 / dev 自动
	// sha7-时间戳 / 本地构建 "dev"）。旧 VERSION 源码常量（6.0.0）已删除
	// ——它从未随发版更新，UI 上与 build_id 形成双版本误导。
	rootCmd := cmd.NewRootCmd(cmd.FuncVersion())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
