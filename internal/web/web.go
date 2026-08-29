// Package web 是 rick 四层架构第一层「WEB-UI 入口」的 Go 落地（rick-spec：
// CLI / TUI / WEB-UI 三入口之一）。它承载 rick web 中心服务的实现：
// 工作区注册表、会话注册表、SSE 事件总线、HTTP API 与静态资源服务。
//
// web 状态是机器级（single-instance-per-machine）的，与工作区无关：
//
//	~/.rick/web.json          工作区注册表（用户添加的含 .rick 的目录清单）
//	~/.rick/web/sessions.json 会话注册表（跨重启的会话元数据）
//	~/.rick/web/              前端可写覆盖层（自迭代的 src/ 与 dist/，见 env.DeployWebScaffold）
//	~/.rick/web.pid           singleton 判活文件
//
// 注意：web 状态跟随 HOME（os.UserHomeDir()），刻意不跟随 RICK_PI_AGENT_DIR
// ——后者是 pi 子进程配置的隔离开关（runtime.AgentDir），web 状态与之无关；
// 测试用 t.Setenv("HOME", t.TempDir()) 隔离即可。
package web

import (
	"os"
	"path/filepath"
)

// WebStateDir returns the machine-level web state directory (~/.rick/web),
// which holds the writable frontend overlay (src/, dist/) and the session
// registry. Tests isolate it via HOME.
func WebStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "web")
}

// WebConfigPath returns the workspace registry file path (~/.rick/web.json).
func WebConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "web.json")
}

// SessionsPath returns the session registry file path
// (~/.rick/web/sessions.json).
func SessionsPath() string {
	return filepath.Join(WebStateDir(), "sessions.json")
}

// PidPath returns the singleton liveness file path (~/.rick/web.pid).
func PidPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "web.pid")
}
