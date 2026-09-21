package handler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

// TestWebPidPathHonorsStateDirEnv 验证 pid 路径跟随显式状态目录：
// $RICK_STATE_DIR 优先于 $HOME/.rick（两处 pid 路径此前都硬编码 .rick —— 这是
// 调研指出的第二处，dev 实例会与生产抢同一个 web.pid）。
func TestWebPidPathHonorsStateDirEnv(t *testing.T) {
	home := t.TempDir()
	state := filepath.Join(t.TempDir(), "dev-state")
	t.Setenv("HOME", home)
	t.Setenv("RICK_STATE_DIR", "")

	if got, want := webPidPath(), filepath.Join(home, ".rick", "web.pid"); got != want {
		t.Fatalf("默认 pid 路径 = %q, want %q", got, want)
	}

	t.Setenv("RICK_STATE_DIR", state)
	if got, want := webPidPath(), filepath.Join(state, "web.pid"); got != want {
		t.Fatalf("RICK_STATE_DIR pid 路径 = %q, want %q", got, want)
	}
}

// TestWebUsesExplicitPidPath 验证 cmd 组合根注入的 PidPath 生效：pid 文件落在
// 注入目录（而非 HOME），写入本进程 pid，且退出时被清理。
func TestWebUsesExplicitPidPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RICK_STATE_DIR", "")
	stateDir := filepath.Join(t.TempDir(), "state")
	pidPath := filepath.Join(stateDir, "web.pid")

	var sawPid string
	err := Web(WebOptions{Port: 0, Token: "x", StateDir: stateDir, PidPath: pidPath},
		func(ctx context.Context, opts WebOptions, token string, cfg *config.Config) error {
			data, err := os.ReadFile(pidPath)
			if err != nil {
				return err
			}
			sawPid = strings.TrimSpace(string(data))
			if opts.PidPath != pidPath || opts.StateDir != stateDir {
				return errors.New("opts 未透传 StateDir/PidPath")
			}
			return nil
		})
	if err != nil {
		t.Fatalf("Web() 失败: %v", err)
	}
	if sawPid != strconv.Itoa(os.Getpid()) {
		t.Fatalf("pid 文件内容 = %q, want %d", sawPid, os.Getpid())
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("退出后 pid 文件应被清理，stat err=%v", err)
	}
	// HOME 下不得留下第二条 pid 文件
	homePid := filepath.Join(os.Getenv("HOME"), ".rick", "web.pid")
	if _, err := os.Stat(homePid); !os.IsNotExist(err) {
		t.Fatalf("不应在 HOME 下写 pid（隔离失败）: %v", err)
	}
}

// TestWebDefaultPidPathStillHomeDerived 验证不注入 PidPath 时仍是 HOME 派生
// （兼容性红线：生产就是默认路径在跑）。
func TestWebDefaultPidPathStillHomeDerived(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RICK_STATE_DIR", "")

	var pidSeen bool
	err := Web(WebOptions{Port: 0, Token: "x"}, func(ctx context.Context, opts WebOptions, token string, cfg *config.Config) error {
		data, err := os.ReadFile(filepath.Join(home, ".rick", "web.pid"))
		if err != nil {
			return err
		}
		pidSeen = strings.TrimSpace(string(data)) == strconv.Itoa(os.Getpid())
		return nil
	})
	if err != nil {
		t.Fatalf("Web() 失败: %v", err)
	}
	if !pidSeen {
		t.Fatal("默认 pid 未落在 $HOME/.rick/web.pid（行为回归）")
	}
}

// TestWebExplicitPidPathStillEnforcesSingleton 验证换了 pid 路径后 singleton
// 语义不变：注入目录里存在活 pid → ErrWebAlreadyRunning，且不进入 serve。
func TestWebExplicitPidPathStillEnforcesSingleton(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "web.pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}

	serveCalled := false
	err := Web(WebOptions{Port: 0, PidPath: filepath.Join(stateDir, "web.pid")},
		func(ctx context.Context, opts WebOptions, token string, cfg *config.Config) error {
			serveCalled = true
			return nil
		})
	if !errors.Is(err, ErrWebAlreadyRunning) {
		t.Fatalf("want ErrWebAlreadyRunning, got %v", err)
	}
	if serveCalled {
		t.Fatal("singleton 失败时不应进入 serve")
	}
}
