// web.go 是 `rick web` 的 handler 编排：singleton 判活、pid 文件、token
// 解析与服务生命周期驱动（task14）。
//
// 组装根说明：web server 的实际装配（internal/web 的 registries/supervisor/
// hub/session manager/NewServer）由 cmd 层注入的 WebServeFunc 完成——
// internal/web 依赖 internal/handler（sessions.go 消费 DoingIn/DreamIn/
// DoingEvent 等导出），handler 反向 import internal/web 会构成循环；
// rick-spec 例外三（组合根 DIP 越级豁免）早已确立 cmd 的 RunE 是唯一组装
// 根（cmd/doing.go 的 NewPiRuntime 同款）。Web() 负责所有 handler 域的
// 编排逻辑（pid/token/config/信号/优雅退出），serve 细节经注入解耦。
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/sunquan/rick/internal/config"
)

// ErrWebAlreadyRunning is returned when another rick web instance holds the
// pid file and is alive (singleton contract: one server per machine).
var ErrWebAlreadyRunning = errors.New("rick web is already running")

// WebOptions carries the flag values resolved by the cmd layer.
type WebOptions struct {
	// Port is the listen port (cmd layer applies the 6137 default).
	Port int
	// Listen is the bind host (cmd layer applies the 127.0.0.1 default;
	// --listen 0.0.0.0 is the deliberate network-open escape hatch).
	Listen string
	// Token overrides the config web_token when non-empty.
	Token string
	// Verbose enables [INFO] logging.
	Verbose bool
	// Version is the rick binary version (→ /api/config rick_version).
	Version string
}

// WebServeFunc assembles and serves the web server for one run. The cmd
// composition root injects the real implementation (internal/web assembly);
// tests inject fakes. ctx is the signal context; token is the resolved
// auth token (never empty unless auth is intentionally disabled — an empty
// config token resolves to a generated one before this point).
type WebServeFunc func(ctx context.Context, opts WebOptions, token string, cfg *config.Config) error

// Web runs the rick web server: singleton check → pid file → config →
// token resolution → signal context → serve (injected). The pid file is
// removed on exit (defer); interrupt/SIGTERM both trigger graceful shutdown
// (the injected serve honors ctx cancellation).
func Web(opts WebOptions, serve WebServeFunc) error {
	if serve == nil {
		return fmt.Errorf("web: serve function is nil (composition root must inject one)")
	}

	// ① Singleton: pid file + liveness probe (design-tree J2.7).
	pidPath := webPidPath()
	if err := checkSingleton(pidPath); err != nil {
		return err
	}
	if pidPath != "" {
		if err := os.MkdirAll(filepath.Dir(pidPath), 0755); err != nil {
			return fmt.Errorf("create pid dir: %w", err)
		}
		if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			return fmt.Errorf("write pid file %s: %w", pidPath, err)
		}
		defer os.Remove(pidPath)
	}

	// ② Config (LoadConfig tolerates a missing file: creates defaults —
	// `rick web --token x` must work without a pre-existing config.json).
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// ③ Token resolution: flag > config > auto-generate (written back +
	// printed exactly once — the auto-generation run).
	token, generated, err := resolveWebToken(opts.Token, cfg)
	if err != nil {
		return err
	}
	if generated {
		fmt.Printf("[rick-web] 已自动生成 web token 并写入 ~/.rick/config.json：\n")
		fmt.Printf("[rick-web]   %s\n", token)
		fmt.Printf("[rick-web] 后续启动不再打印（用 --token 可覆盖；清空 config 的 web_token 可重新生成）\n")
	}

	// ④ Signal context (interrupt + SIGTERM → graceful shutdown).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ⑤ Serve (injected assembly). EADDRINUSE gets the extra hint.
	if err := serve(ctx, opts, token, cfg); err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			return fmt.Errorf("%w（端口被占用，可能已有实例在跑或端口冲突——换 --port 或检查占用进程）", err)
		}
		return err
	}
	return nil
}

// webPidPath resolves the singleton pid file path (~/.rick/web.pid). It
// mirrors internal/web.PidPath without importing internal/web (import
// cycle: web → handler). HOME-based, same isolation contract.
func webPidPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.rick/web.pid"
}

// checkSingleton enforces one-server-per-machine: a pid file whose pid is
// alive → ErrWebAlreadyRunning; a stale file (dead pid) is removed and the
// caller proceeds.
func checkSingleton(pidPath string) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no prior instance
		}
		return fmt.Errorf("read pid file %s: %w", pidPath, err)
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		// Unparseable pid file: treat as stale, remove, continue.
		_ = os.Remove(pidPath)
		return nil
	}
	// Probe liveness: signal 0 (no delivery, permission/existence check).
	if err := syscall.Kill(pid, 0); err == nil {
		return fmt.Errorf("%w：PID %d（先停旧实例，或删除 %s 后重试）",
			ErrWebAlreadyRunning, pid, pidPath)
	}
	// Dead pid: stale file, remove and proceed.
	_ = os.Remove(pidPath)
	return nil
}

// resolveWebToken implements flag > config > auto-generate. The generated
// token is written back to the config file (so it persists) and flagged so
// the caller prints it exactly once.
func resolveWebToken(flagToken string, cfg *config.Config) (token string, generated bool, err error) {
	if flagToken != "" {
		return flagToken, false, nil
	}
	if cfg.WebToken != "" {
		return cfg.WebToken, false, nil
	}
	// Auto-generate: 16 random bytes → 32 hex chars.
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", false, fmt.Errorf("generate web token: %w", err)
	}
	token = hex.EncodeToString(buf)
	cfg.WebToken = token
	if err := config.SaveConfig(cfg); err != nil {
		return "", false, fmt.Errorf("persist generated web token: %w", err)
	}
	return token, true, nil
}
