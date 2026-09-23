package cmd

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/web"
)

// webPidFile resolves the singleton pid file under an isolated HOME.
func webPidFile(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, ".rick", "web.pid")
}

func TestWebSingletonRejectsLiveInstance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Occupy the pid file with OUR pid (a live process — the test itself).
	pidFile := webPidFile(t)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}

	serveCalled := false
	err := handler.Web(handler.WebOptions{Port: 0}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		serveCalled = true
		return nil
	})
	if !errors.Is(err, handler.ErrWebAlreadyRunning) {
		t.Fatalf("want ErrWebAlreadyRunning, got %v", err)
	}
	if serveCalled {
		t.Fatal("serve must not run when singleton check fails")
	}
}

func TestWebSingletonClearsStalePidFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Stale pid file: a pid that is definitely not alive (but parseable).
	pidFile := webPidFile(t)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatal(err)
	}
	// 2^31-1 area: extremely unlikely to be a live process; if it were,
	// kill(pid,0) permission semantics still treat it as alive and the test
	// would flake — use a pid from a finished child process instead.
	dead := spawnDeadPid(t)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(dead)), 0644); err != nil {
		t.Fatal(err)
	}

	err := handler.Web(handler.WebOptions{Port: 0, Token: "x"}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		return nil // immediate clean exit
	})
	if err != nil {
		t.Fatalf("stale pid file should be cleared and Web proceed: %v", err)
	}
	// pid file written by this run and removed on exit (defer).
	if _, statErr := os.Stat(pidFile); !os.IsNotExist(statErr) {
		t.Errorf("pid file should be removed after clean exit, stat err=%v", statErr)
	}
}

// spawnDeadPid forks a child that exits immediately and returns its pid.
func spawnDeadPid(t *testing.T) int {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	type pidMsg struct{ pid int }
	done := make(chan pidMsg, 1)
	go func() {
		buf := make([]byte, 32)
		n, _ := r.Read(buf)
		p, _ := strconv.Atoi(strings.TrimSpace(string(buf[:n])))
		done <- pidMsg{p}
	}()
	proc, err := os.StartProcess("/bin/true", []string{"/bin/true"}, &os.ProcAttr{Files: []*os.File{nil, w, w}})
	if err != nil {
		w.Close()
		t.Skipf("cannot spawn dead pid helper: %v", err)
	}
	w.Close()
	state, err := proc.Wait()
	if err != nil || !state.Success() {
		t.Skipf("dead pid helper did not exit cleanly: %v %v", state, err)
	}
	// The pid never gets written by /bin/true; use proc.Pid directly.
	msg := <-done
	_ = msg // pipe read may race; use proc.Pid
	deadPid := proc.Pid
	if deadPid <= 0 {
		t.Skip("invalid dead pid")
	}
	return deadPid
}

func TestWebTokenAutoGenerateWrittenBack(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotToken string
	err := handler.Web(handler.WebOptions{Port: 0}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}

	// Token delivered to serve...
	if len(gotToken) != 32 {
		t.Fatalf("auto-generated token should be 32 hex chars, got %q", gotToken)
	}
	// ...and persisted to config.json (written back).
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.WebToken != gotToken {
		t.Fatalf("config web_token %q != served token %q", cfg.WebToken, gotToken)
	}
}

func TestWebTokenFlagOverridesConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Seed a config token; the flag must win.
	seed, err := config.LoadConfig() // creates default config file
	if err != nil {
		t.Fatal(err)
	}
	seed.WebToken = "config-token"
	if err := config.SaveConfig(seed); err != nil {
		t.Fatal(err)
	}

	var gotToken string
	err = handler.Web(handler.WebOptions{Token: "flag-token"}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}
	if gotToken != "flag-token" {
		t.Fatalf("flag token should override config: got %q", gotToken)
	}
	// Config must remain untouched (flag does not write back).
	cfg, _ := config.LoadConfig()
	if cfg.WebToken != "config-token" {
		t.Fatalf("config token must not be overwritten by flag run: %q", cfg.WebToken)
	}
}

func TestWebTokenConfigUsedWithoutFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	seed, err := config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	seed.WebToken = "from-config"
	if err := config.SaveConfig(seed); err != nil {
		t.Fatal(err)
	}

	var gotToken string
	err = handler.Web(handler.WebOptions{}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}
	if gotToken != "from-config" {
		t.Fatalf("config token should be used: got %q", gotToken)
	}
}

func TestNewWebCmdHelpSurface(t *testing.T) {
	cmd := NewWebCmd("test-version")
	if cmd == nil {
		t.Fatal("NewWebCmd returned nil")
	}
	// Subcommands present.
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"customize", "reset"} {
		if !subs[want] {
			t.Errorf("web cmd missing subcommand %q", want)
		}
	}
	// Flags surface: --port / --listen / --token.
	for _, want := range []string{"port", "listen", "token"} {
		if cmd.Flags().Lookup(want) == nil {
			t.Errorf("web cmd missing --%s flag", want)
		}
	}
}

// TestConfigureDevIsolationSwappedHome 是 F1 的回归钉子：**换 HOME 形态**
// （`rick tools dev-web` 启动 dev 实例的实际形态：HOME=<dev-home>，状态目录仍是
// 默认的 $HOME/.rick）也必须安装工作区守卫。
//
// 修复前：开关用 IsDevStateDir(resolved) = (resolved != $HOME/.rick)，换 HOME
// 后该式恒为 false → 守卫不装 → 实测 `POST /api/workspaces` 注册生产工作区
// /workdir/.../BERT_KEETA 返回 HTTP 201，dev 会话可写坏生产 .rick/。
func TestConfigureDevIsolationSwappedHome(t *testing.T) {
	// 假生产状态目录（内含一个已注册的生产工作区）—— 不碰真实 ~/.rick
	prodState := t.TempDir()
	prodWS := filepath.Join(t.TempDir(), "prod-ws")
	if err := os.MkdirAll(filepath.Join(prodWS, ".rick"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"version":1,"workspaces":[{"id":"aaaabbbb","path":"` + prodWS + `","name":"prod","added_at":"2026-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(filepath.Join(prodState, "web.json"), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RICK_PROD_STATE_DIR", prodState)

	// 形态一：换 HOME（dev-web 的实际形态）→ 必须判为 dev 并装守卫
	devHome := t.TempDir()
	t.Setenv("HOME", devHome)
	t.Setenv("RICK_PI_AGENT_DIR", "") // 换 HOME 时 pi 沙盒随 HOME 隔离，不强制

	mode, err := configureDevIsolation(filepath.Join(devHome, ".rick"))
	if err != nil {
		t.Fatalf("换 HOME 形态不应报错（pi 沙盒已随 HOME 隔离）: %v", err)
	}
	if mode != "home-swapped" {
		t.Fatalf("mode = %q, want home-swapped（F1：换 HOME 形态必须识别为 dev）", mode)
	}
	// 守卫真的装上了：注册生产工作区 → 拒绝
	guarded, err := web.LoadWorkspaceRegistry(filepath.Join(t.TempDir(), "web.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = guarded.Add(prodWS, "should-be-rejected")
	if err == nil {
		t.Fatal("换 HOME 形态下注册生产工作区竟然成功 —— 守卫没装（F1 未修复）")
	}
	if !strings.Contains(err.Error(), "生产") {
		t.Fatalf("拒绝原因不像隔离守卫: %v", err)
	}
	// 非生产工作区照常放行
	freshWS := filepath.Join(t.TempDir(), "fresh-ws")
	if err := os.MkdirAll(filepath.Join(freshWS, ".rick"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := guarded.Add(freshWS, "ok"); err != nil {
		t.Fatalf("dev 专属工作区应放行: %v", err)
	}

	// 形态二：只换状态目录（HOME 未换）→ 必须强制显式 RICK_PI_AGENT_DIR
	t.Setenv("HOME", mustRealHome(t))
	mode, err = configureDevIsolation(filepath.Join(t.TempDir(), "dev-state"))
	if err == nil {
		t.Fatal("state-dir-only 形态未强制 RICK_PI_AGENT_DIR（应报错）")
	}
	if !strings.Contains(err.Error(), "RICK_PI_AGENT_DIR") {
		t.Fatalf("错误信息未点明 RICK_PI_AGENT_DIR: %v", err)
	}
	t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(t.TempDir(), "agent"))
	mode, err = configureDevIsolation(filepath.Join(t.TempDir(), "dev-state"))
	if err != nil {
		t.Fatalf("显式给出 agent dir 后应可启动: %v", err)
	}
	if mode != "state-dir-only" {
		t.Fatalf("mode = %q, want state-dir-only", mode)
	}

	// 形态三：生产实例（状态目录 == 生产状态目录）→ 不装守卫，注册不受限
	mode, err = configureDevIsolation(prodState)
	if err != nil {
		t.Fatalf("生产实例的 configureDevIsolation 不应报错: %v", err)
	}
	if mode != "" {
		t.Fatalf("mode = %q, want \"\"（生产实例）", mode)
	}
	plain, err := web.LoadWorkspaceRegistry(filepath.Join(t.TempDir(), "web.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := plain.Add(prodWS, "prod-side"); err != nil {
		t.Fatalf("生产实例侧不应被守卫拦截: %v", err)
	}
}

// mustRealHome returns the passwd user home (the tests above swap $HOME).
func mustRealHome(t *testing.T) string {
	t.Helper()
	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		t.Skip("无法获取真实家目录")
	}
	return u.HomeDir
}

// TestWebDaemonFlagRegisters 验证 --daemon/--log-file flag 已注册（简化部署的
// 核心入口：`rick web --daemon` 一条命令后台运行 + 日志默认 ~/.rick/web.log）。
func TestWebDaemonFlagRegisters(t *testing.T) {
	cmd := NewWebCmd("test")
	daemon := cmd.Flags().Lookup("daemon")
	if daemon == nil {
		t.Fatal("web 命令缺少 --daemon flag")
	}
	if daemon.Usage == "" {
		t.Fatal("--daemon 的 help 文本为空")
	}
	lf := cmd.Flags().Lookup("log-file")
	if lf == nil {
		t.Fatal("web 命令缺少 --log-file flag")
	}
}
