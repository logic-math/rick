// Package web 是 rick 四层架构第一层「WEB-UI 入口」的 Go 落地（rick-spec：
// CLI / TUI / WEB-UI 三入口之一）。它承载 rick web 中心服务的实现：
// 工作区注册表、会话注册表、SSE 事件总线、HTTP API 与静态资源服务。
//
// web 状态是机器级（single-instance-per-machine）的，与工作区无关：
//
//	<stateDir>/web.json          工作区注册表（用户添加的含 .rick 的目录清单）
//	<stateDir>/web/sessions.json 会话注册表（跨重启的会话元数据）
//	<stateDir>/web/              前端可写覆盖层（自迭代的 src/ 与 dist/，见 env.DeployWebScaffold）
//	<stateDir>/web.pid           singleton 判活文件（老机制，保留）
//	<stateDir>/web.lock          singleton 强约束（flock；见 statedir.go）
//
// 状态目录 <stateDir> 默认 $HOME/.rick，可由 --state-dir / RICK_STATE_DIR 显式
// 重定向（dev 实例隔离用）；刻意不跟随 RICK_PI_AGENT_DIR——后者是 pi 子进程配置
// 的隔离开关（runtime.AgentDir）。测试用 t.Setenv("HOME", t.TempDir()) 或
// SetStateDir(t.TempDir()) 隔离。
package web

import (
	"path/filepath"
)

// 路径解析：全部经 StateDir()（见 statedir.go）——它优先返回启动时显式钉住的
// 状态目录（--state-dir / RICK_STATE_DIR），未钉住时回退 $HOME/.rick。
// 因此默认行为与改造前**逐字节一致**（生产就是默认路径在跑），而 dev 实例可以
// 把整套机器级状态重定向到隔离目录。

// WebStateDir returns the machine-level web state directory (<stateDir>/web),
// which holds the writable frontend overlay (src/, dist/) and the session
// registry. Tests isolate it via HOME or SetStateDir.
func WebStateDir() string {
	return filepath.Join(StateDir(), "web")
}

// WebConfigPath returns the workspace registry file path (<stateDir>/web.json).
func WebConfigPath() string {
	return filepath.Join(StateDir(), "web.json")
}

// SessionsPath returns the session registry file path
// (~/.rick/web/sessions.json).
func SessionsPath() string {
	return filepath.Join(WebStateDir(), "sessions.json")
}

// ArchivedPath returns the job archive registry file path
// (~/.rick/web/archived.json). It holds the web-layer soft-archive state
// (per-workspace archived job ids) — rick job files are never touched.
func ArchivedPath() string {
	return filepath.Join(WebStateDir(), "archived.json")
}

// JobNamesPath returns the job display-name file path
// (~/.rick/web/job-names.json) — user-assigned job「任务名」别名（展示层数据，
// 不碰工作区里的 rick job 文件）。
func JobNamesPath() string {
	return filepath.Join(WebStateDir(), "job-names.json")
}

// PidPath returns the singleton liveness file path (<stateDir>/web.pid).
func PidPath() string {
	return filepath.Join(StateDir(), "web.pid")
}
