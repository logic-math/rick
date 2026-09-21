package web

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveStateDirPrecedence 验证状态目录解析阶梯：--state-dir flag >
// RICK_STATE_DIR env > $HOME/.rick，且返回绝对路径并把目录建出来。
// 全部在 t.TempDir 的假 HOME 下进行（绝不触碰真实 ~/.rick）。
func TestResolveStateDirPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(StateDirEnvVar, "")
	t.Cleanup(func() { SetStateDir("") })

	// ① 都不给 → $HOME/.rick（兼容性红线：默认行为不变）
	got, err := ResolveStateDir("")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	if want := filepath.Join(home, ".rick"); got != want {
		t.Fatalf("default = %q, want %q", got, want)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Fatalf("default state dir not created: %v", err)
	}

	// ② 只有 env → env 生效
	envDir := filepath.Join(t.TempDir(), "env-state")
	t.Setenv(StateDirEnvVar, envDir)
	got, err = ResolveStateDir("")
	if err != nil {
		t.Fatalf("resolve env: %v", err)
	}
	if got != envDir {
		t.Fatalf("env = %q, want %q", got, envDir)
	}

	// ③ flag 优先于 env
	flagDir := filepath.Join(t.TempDir(), "flag-state")
	got, err = ResolveStateDir(flagDir)
	if err != nil {
		t.Fatalf("resolve flag: %v", err)
	}
	if got != flagDir {
		t.Fatalf("flag = %q, want %q (flag must beat env)", got, flagDir)
	}

	// ④ 相对路径 → 绝对化
	rel := "rel-state-" + filepath.Base(t.TempDir())
	got, err = ResolveStateDir(rel)
	if err != nil {
		t.Fatalf("resolve relative: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("relative path not made absolute: %q", got)
	}
}

// TestStateDirPinsAllPaths 验证 SetStateDir 后**所有**机器级路径 helper 都跟随
// 新目录（web.json / web/ / sessions / archived / job-names / web.pid），
// 且 SetStateDir("") 复位回默认——这是「dev 与生产状态完全隔离」的基础。
func TestStateDirPinsAllPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(StateDirEnvVar, "")
	t.Cleanup(func() { SetStateDir("") })

	def := StateDir()
	if def != filepath.Join(home, ".rick") {
		t.Fatalf("default StateDir = %q", def)
	}

	pinned := filepath.Join(t.TempDir(), "dev-state")
	SetStateDir(pinned)
	cases := map[string]string{
		"WebStateDir":   WebStateDir(),
		"WebConfigPath": WebConfigPath(),
		"SessionsPath":  SessionsPath(),
		"ArchivedPath":  ArchivedPath(),
		"JobNamesPath":  JobNamesPath(),
		"PidPath":       PidPath(),
	}
	want := map[string]string{
		"WebStateDir":   filepath.Join(pinned, "web"),
		"WebConfigPath": filepath.Join(pinned, "web.json"),
		"SessionsPath":  filepath.Join(pinned, "web", "sessions.json"),
		"ArchivedPath":  filepath.Join(pinned, "web", "archived.json"),
		"JobNamesPath":  filepath.Join(pinned, "web", "job-names.json"),
		"PidPath":       filepath.Join(pinned, "web.pid"),
	}
	for name, got := range cases {
		if got != want[name] {
			t.Errorf("%s = %q, want %q", name, got, want[name])
		}
		if strings.HasPrefix(got, home) {
			t.Errorf("%s still under HOME (%s) — 隔离失败", name, got)
		}
	}

	SetStateDir("")
	if StateDir() != def {
		t.Fatalf("SetStateDir(\"\") 未复位：%q", StateDir())
	}
}

// TestAcquireStateLockExclusive 验证 flock 强约束：同目录第二次获取失败（含
// 中文提示与 web.lock 字样），release 后可再次获取，且 release 幂等。
func TestAcquireStateLockExclusive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")

	release, err := AcquireStateLock(dir)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "web.lock")); err != nil {
		t.Fatalf("web.lock 未创建: %v", err)
	}

	_, err = AcquireStateLock(dir)
	if err == nil {
		t.Fatal("同目录第二次获取竟然成功（flock 未生效）")
	}
	msg := err.Error()
	if !strings.Contains(msg, "web.lock") || !strings.Contains(msg, "占用") {
		t.Fatalf("错误提示缺少锁/占用说明: %q", msg)
	}

	release()
	release() // 幂等：重复 release 不得 panic

	release2, err := AcquireStateLock(dir)
	if err != nil {
		t.Fatalf("release 后重新获取失败: %v", err)
	}
	release2()

	// 不同目录互不影响
	other := filepath.Join(t.TempDir(), "state2")
	r1, err := AcquireStateLock(dir)
	if err != nil {
		t.Fatalf("re-acquire dir: %v", err)
	}
	defer r1()
	r2, err := AcquireStateLock(other)
	if err != nil {
		t.Fatalf("独立目录应可同时持锁: %v", err)
	}
	defer r2()
}

// TestAcquireStateLockEmptyDir 验证无状态目录（极端环境）时优雅降级：
// 返回 no-op release 且不报错（不阻断启动，回到旧行为）。
func TestAcquireStateLockEmptyDir(t *testing.T) {
	release, err := AcquireStateLock("")
	if err != nil {
		t.Fatalf("empty state dir must not fail: %v", err)
	}
	release()
}

// TestIsDevStateDir 验证 dev 语义判定：默认目录 = 非 dev；任何显式目录 = dev。
func TestIsDevStateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if IsDevStateDir(filepath.Join(home, ".rick")) {
		t.Fatal("默认状态目录不应判为 dev")
	}
	if IsDevStateDir(filepath.Join(home, ".rick") + "/") {
		t.Fatal("带尾斜杠的同一目录不应判为 dev")
	}
	if !IsDevStateDir(filepath.Join(t.TempDir(), "dev-state")) {
		t.Fatal("显式目录应判为 dev")
	}
	if IsDevStateDir("") {
		t.Fatal("空路径不应判为 dev")
	}
}

// TestProdWorkspaceGuard 验证 dev 隔离守卫：生产注册表里已存在的工作区路径
// 被拒绝，其余放行；生产注册表缺失/损坏时不得阻断（守卫降级为放行）。
func TestProdWorkspaceGuard(t *testing.T) {
	prodState := t.TempDir()
	victim := t.TempDir()
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(prodState, "web.json"), []byte(
		`{"version":1,"workspaces":[{"id":"aaaabbbb","path":"`+victim+`","name":"prod-ws","added_at":"2026-01-01T00:00:00Z"}]}`), 0644); err != nil {
		t.Fatalf("write prod registry: %v", err)
	}

	guard := ProdWorkspaceGuard(prodState)
	err := guard(victim)
	if err == nil {
		t.Fatal("生产已注册的工作区路径竟然被放行")
	}
	if !strings.Contains(err.Error(), "生产") {
		t.Fatalf("守卫错误提示应说明原因: %q", err)
	}
	if verr, ok := err.(*ValidationError); !ok || verr.Code != "invalid_workspace" {
		t.Fatalf("守卫错误应为 invalid_workspace ValidationError（HTTP 400），实为 %T", err)
	}
	if err := guard(other); err != nil {
		t.Fatalf("非生产工作区应放行: %v", err)
	}
	// 路径规范差异（尾斜杠/相对形式）也要判等
	if err := guard(victim + "/"); err == nil {
		t.Fatal("尾斜杠形式应被同一守卫拦截")
	}

	// 生产注册表不存在 → 守卫降级为放行（不能让 dev 起不来）
	empty := ProdWorkspaceGuard(filepath.Join(t.TempDir(), "nope"))
	if err := empty(victim); err != nil {
		t.Fatalf("生产注册表缺失时不应拦截: %v", err)
	}
	// 显式空生产目录 → 放行
	if err := ProdWorkspaceGuard("")(victim); err != nil {
		t.Fatalf("空生产目录不应拦截: %v", err)
	}
}

// TestDevModeInfo 验证 dev 形态判定（F1 修复的核心谓词）：**换 HOME 形态**也必须
// 判为 dev —— 这正是 `rick tools dev-web` 的实际启动形态，而旧判定
// IsDevStateDir（只看 resolved != $HOME/.rick）在这里恒为 false，导致守卫没装、
// dev 实测可以注册生产工作区（HTTP 201）并写坏它的 .rick/。
//
// 四种组合（用 RICK_PROD_STATE_DIR 固定「生产状态目录」，勿碰真实 ~/.rick）：
//
//	生产 HOME + 默认       → 非 dev
//	生产 HOME + --state-dir → dev（state-dir-only，homeSwapped=false）
//	换 HOME   + 默认       → dev（home-swapped，homeSwapped=true）← F1 的关键格
//	换 HOME   + --state-dir → dev（home-swapped）
func TestDevModeInfo(t *testing.T) {
	realHome, err := user.Current()
	if err != nil || realHome.HomeDir == "" {
		t.Skip("无法获取真实家目录（user.Current 不可用）")
	}
	fakeProdState := filepath.Join(t.TempDir(), "prod-state")
	if err := os.MkdirAll(fakeProdState, 0o755); err != nil {
		t.Fatalf("mkdir fake prod state: %v", err)
	}
	t.Setenv(ProdStateDirEnvVar, fakeProdState)

	devHome := t.TempDir()
	devState := filepath.Join(t.TempDir(), "dev-state")

	cases := []struct {
		name            string
		home            string
		resolved        string
		wantDev         bool
		wantHomeSwapped bool
	}{
		{"生产 HOME + 默认", realHome.HomeDir, fakeProdState, false, false},
		{"生产 HOME + --state-dir", realHome.HomeDir, devState, true, false},
		{"换 HOME + 默认", devHome, filepath.Join(devHome, ".rick"), true, true},
		{"换 HOME + --state-dir", devHome, devState, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", tc.home)
			isDev, prodState, homeSwapped := DevModeInfo(tc.resolved)
			if filepath.Clean(prodState) != filepath.Clean(fakeProdState) {
				t.Fatalf("prodState = %q, want %q", prodState, fakeProdState)
			}
			if isDev != tc.wantDev {
				t.Fatalf("isDev = %v, want %v (resolved=%s HOME=%s)", isDev, tc.wantDev, tc.resolved, tc.home)
			}
			if homeSwapped != tc.wantHomeSwapped {
				t.Fatalf("homeSwapped = %v, want %v (HOME=%s)", homeSwapped, tc.wantHomeSwapped, tc.home)
			}
			// F1 回归钉子：换 HOME 形态下**旧判定必须漏判**（否则这条用例没在守 F1）。
			if tc.name == "换 HOME + 默认" {
				if IsDevStateDir(tc.resolved) {
					t.Fatal("旧判定 IsDevStateDir 竟然也认为这是 dev —— 用例已失去 F1 回归意义")
				}
				if !isDev {
					t.Fatal("换 HOME 形态必须判为 dev（F1：否则隔离守卫不会安装）")
				}
			}
		})
	}

	// 空 resolved → 不是 dev（不装守卫），与旧行为一致
	if isDev, _, _ := DevModeInfo(""); isDev {
		t.Fatal("空状态目录不应判为 dev")
	}
	// 无 RICK_PROD_STATE_DIR 时回退真实用户家目录：换 HOME 的默认状态目录仍是 dev
	t.Setenv(ProdStateDirEnvVar, "")
	t.Setenv("HOME", devHome)
	if isDev, prodState, swapped := DevModeInfo(filepath.Join(devHome, ".rick")); !isDev || !swapped {
		t.Fatalf("回退 user.Current 路径：isDev=%v homeSwapped=%v prodState=%q", isDev, swapped, prodState)
	}
}
