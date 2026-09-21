package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// 本文件把「web 状态目录」从隐含的 HOME 派生变成**显式契约**，并用跨进程
// file lock 把 singleton 从「pid 文件尽力而为」升级为强约束。
//
// 背景（调研 research-L4 §1/§4 实测）：
//   - 机器级 web 状态（web.json / web/sessions.json / archived.json /
//     job-names.json / web.pid / 前端覆盖层）此前 100% 由 os.UserHomeDir()
//     派生，**没有任何 flag 或环境变量可以重定向**；
//   - 生产实例的 `~/.rick/web.pid` 实测竟然不存在 → 旧的 singleton 门禁形同
//     虚设：同 HOME 起第二个实例会被放行，而它的 ReconcileOnStart 会把生产
//     active 会话改写成 error 落盘。
//
// 因此这里提供两件事：
//  1. StateDir（= ~/.rick 这一层）的显式解析：--state-dir flag > RICK_STATE_DIR
//     env > $HOME/.rick。**默认（都不给）行为与改造前逐字节一致**——生产就是
//     默认路径在跑，兼容性是红线。
//  2. AcquireStateLock：对 <stateDir>/web.lock 取 flock(LOCK_EX|LOCK_NB)，让
//     「同一状态目录只能有一个实例」成为内核级强约束（不再依赖 pid 文件的
//     TOCTOU/可删除性）。

// StateDirEnvVar 显式指定状态目录的环境变量（与 --state-dir 等价，flag 优先）。
const StateDirEnvVar = "RICK_STATE_DIR"

// ProdStateDirEnvVar 指向**生产**状态目录，供 dev 实例做「拒绝注册生产工作区」
// 的隔离守卫（dev 实例的 HOME 已被切换到 dev home，故无法再从 HOME 推断生产
// 位置；未设置时回退到 /etc/passwd 的真实家目录）。
const ProdStateDirEnvVar = "RICK_PROD_STATE_DIR"

var (
	stateDirMu       sync.RWMutex
	stateDirResolved string

	addGuardMu sync.RWMutex
	addGuard   func(absPath string) error
)

// SetStateDir pins the process-wide state directory (called once at startup by
// the composition root). Empty resets to the HOME-derived default — 既有测试
// 与默认启动路径完全不受影响。
func SetStateDir(dir string) {
	stateDirMu.Lock()
	defer stateDirMu.Unlock()
	stateDirResolved = strings.TrimSpace(dir)
}

// StateDir returns the resolved web state directory（= 旧实现里的 ~/.rick）。
// 未显式设置时回退 HOME 派生 → **默认行为与改造前一致**。
func StateDir() string {
	stateDirMu.RLock()
	pinned := stateDirResolved
	stateDirMu.RUnlock()
	if pinned != "" {
		return pinned
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick")
}

// ResolveStateDir resolves the state directory from the flag/env/HOME ladder
// and makes sure it exists. Priority: flag > $RICK_STATE_DIR > $HOME/.rick.
// A leading "~" is expanded (shells usually do this, but agents pass raw
// strings). The returned path is absolute.
func ResolveStateDir(flagValue string) (string, error) {
	candidate := strings.TrimSpace(flagValue)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv(StateDirEnvVar))
	}
	if candidate == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve state dir: %w", err)
		}
		candidate = filepath.Join(home, ".rick")
	} else if candidate == "~" || strings.HasPrefix(candidate, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve state dir %q: %w", candidate, err)
		}
		candidate = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(candidate, "~"), "/"))
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve state dir %q: %w", candidate, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create state dir %s: %w", abs, err)
	}
	return abs, nil
}

// IsDevStateDir reports whether dir is anything other than the HOME-derived
// default (i.e. the operator explicitly isolated this instance). dev 语义只在
// 显式指定时启用，默认实例永不进入 dev 分支。
func IsDevStateDir(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	def, err := filepath.Abs(filepath.Join(home, ".rick"))
	if err != nil {
		return false
	}
	got, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return filepath.Clean(got) != filepath.Clean(def)
}

// AcquireStateLock takes an exclusive, non-blocking flock on
// <stateDir>/web.lock and returns an idempotent release func. A second
// instance on the same state directory fails immediately with a Chinese
// explanation plus the best-effort holder pid — this is the strong singleton
// contract (pid files can be deleted; kernel locks cannot).
func AcquireStateLock(stateDir string) (func(), error) {
	if strings.TrimSpace(stateDir) == "" {
		return func() {}, nil // 无状态目录（极端环境）→ 退回旧行为，不阻断启动
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir %s: %w", stateDir, err)
	}
	lockPath := filepath.Join(stateDir, "web.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open state lock %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder := readLockHolder(lockPath)
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf(
				"状态目录 %s 已被另一个 rick web 实例占用（web.lock 锁持有者 pid=%s）——请先停掉它，或换一个 --state-dir",
				stateDir, holder)
		}
		return nil, fmt.Errorf("acquire state lock %s: %w", lockPath, err)
	}
	// 写入本进程 pid（纯诊断用途；锁的权威性来自 flock 而非文件内容）。
	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		})
	}, nil
}

// readLockHolder best-effort reads the pid recorded in the lock file for the
// "who holds it" hint ("" when unreadable).
func readLockHolder(lockPath string) string {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return "unknown"
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return "unknown"
	}
	return s
}

// SetWorkspaceAddGuard installs a process-wide guard invoked by
// WorkspaceRegistry.Add with the resolved absolute path. nil removes it
// (default: no guard → 行为与旧版完全一致). The dev instance uses it to
// refuse registering workspaces that the production registry already owns.
func SetWorkspaceAddGuard(fn func(absPath string) error) {
	addGuardMu.Lock()
	defer addGuardMu.Unlock()
	addGuard = fn
}

// workspaceAddGuard returns the current guard (nil when unset).
func workspaceAddGuard() func(absPath string) error {
	addGuardMu.RLock()
	defer addGuardMu.RUnlock()
	return addGuard
}

// ProductionStateDir locates the *production* state directory from a dev
// instance: $RICK_PROD_STATE_DIR wins; otherwise the real (passwd) home is
// used when it differs from the current $HOME (dev instances run with a
// swapped HOME). Returns "" when it cannot be determined.
func ProductionStateDir() string {
	if v := strings.TrimSpace(os.Getenv(ProdStateDirEnvVar)); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs
		}
	}
	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if filepath.Clean(u.HomeDir) == filepath.Clean(home) {
		return "" // 当前就是生产 HOME，不需要守卫
	}
	return filepath.Join(u.HomeDir, ".rick")
}

// ProdWorkspaceGuard builds the dev-instance guard: reject any workspace path
// that the production registry (read-only) already has registered. This stops
// a dev session from editing production sources and writing production
// workspaces' .rick/ directories.
func ProdWorkspaceGuard(prodStateDir string) func(absPath string) error {
	prodPaths := map[string]bool{}
	if strings.TrimSpace(prodStateDir) != "" {
		prodPaths = readRegistryWorkspacePaths(filepath.Join(prodStateDir, "web.json"))
	}
	return func(absPath string) error {
		clean := filepath.Clean(absPath)
		if prodPaths[clean] {
			return newValidationError("invalid_workspace",
				"%s 已被生产实例注册（dev 实例拒绝共享生产工作区：dev 会话会写坏生产的 .rick/ 与任务状态）——请用 dev 专属工作区，或改用生产 UI",
				clean)
		}
		return nil
	}
}

// readRegistryWorkspacePaths reads a workspace registry file and returns its
// paths (empty on any error — a missing/unreadable prod registry must never
// block the dev instance from working).
func readRegistryWorkspacePaths(path string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var file struct {
		Workspaces []struct {
			Path string `json:"path"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return out
	}
	for _, w := range file.Workspaces {
		if strings.TrimSpace(w.Path) == "" {
			continue
		}
		out[filepath.Clean(w.Path)] = true
	}
	return out
}
