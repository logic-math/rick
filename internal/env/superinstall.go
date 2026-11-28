package env

import (
	"os"

	superpkg "github.com/sunquan/rick/internal/super"
)

// rickHomeDotRick 返回 rick 状态根（~/.rick）——部署目标。与
// internal/super rickRootFromAgentEnv 同一解析语义（RICK_PI_AGENT_DIR 注入
// 剥 pi/agent 后缀 / HOME 推导）；本包已有 sourcetree.go 用 os.UserHomeDir
// 的先例，但为与 super 侧 ③ 回退严格同根，这里直接复用 super 包导出的
// 安装目标推导（见 super.SuperInstallTargetRickDir）。
//
// 注：本函数语义由 deploySuperRuntime 内部使用；导出面保持最小
// （superinstall 仅 init-pi 流程消费）。
func deploySuperRuntime() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// cwd 在 rick clone 内（上溯到 marker 所在根）才部署——非 clone cwd
	// 绝不动既有副本（install.sh / 手动 clone 内跑 init-pi 是唯一合法源）。
	root := rickRepoRootFrom(cwd)
	if root == "" {
		return nil
	}
	// 目标根：~/.rick（与 super 侧 resolveRepoRoot ③ 回退同一解析，
	// 测试经 RICK_PI_AGENT_DIR/HOME 注入同根隔离）。
	dst := superpkg.SuperInstallTargetRickDir()
	if dst == "" {
		return nil
	}
	return superpkg.DeploySuperRuntimeFromSrc(root, dst)
}
