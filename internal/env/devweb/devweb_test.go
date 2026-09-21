package devweb

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// newLayout 构造一个完全落在 t.TempDir() 里的 Layout（不碰真实 dev 树/dev HOME）。
func newLayout(t *testing.T) Layout {
	t.Helper()
	root := t.TempDir()
	prodRepo := filepath.Join(root, "prod", "rick")
	home := filepath.Join(root, "dev-home")
	for _, d := range []string{prodRepo, home, filepath.Join(prodRepo, "web", "dist"), filepath.Join(home, ".rick", "pi", "agent")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return Layout{
		Tree:       filepath.Join(root, "dev-tree"),
		Home:       home,
		AgentDir:   filepath.Join(home, ".rick", "pi", "agent"),
		ProdRepo:   prodRepo,
		ProdHome:   filepath.Join(root, "real-home"),
		Port:       18414,
		Token:      "devtok-test",
		GoCache:    filepath.Join(root, "gocache"),
		GoModCache: filepath.Join(root, "gomod"),
		NpmCache:   filepath.Join(root, "npm"),
	}
}

// TestDefaultLayoutPathsAndEnv 覆盖默认布局推导与环境变量覆盖（含端口校验）。
func TestDefaultLayoutPathsAndEnv(t *testing.T) {
	root := t.TempDir()
	prodRepo := filepath.Join(root, "base", "nest", "rick")
	if err := os.MkdirAll(filepath.Join(prodRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 默认：`<prodRepo 的祖父目录>/rick-dev{,-home}`
	t.Setenv(EnvTree, "")
	t.Setenv(EnvHome, "")
	t.Setenv(EnvPort, "")
	t.Setenv("GOCACHE", filepath.Join(root, "gc"))
	t.Setenv("GOMODCACHE", filepath.Join(root, "gm"))
	t.Setenv("npm_config_cache", filepath.Join(root, "npm-cache"))

	l, err := DefaultLayout(prodRepo)
	if err != nil {
		t.Fatalf("DefaultLayout: %v", err)
	}
	wantTree := filepath.Join(root, "base", "rick-dev")
	if l.Tree != wantTree {
		t.Fatalf("Tree = %q, want %q", l.Tree, wantTree)
	}
	if l.Home != filepath.Join(root, "base", "rick-dev-home") {
		t.Fatalf("Home = %q", l.Home)
	}
	if l.AgentDir != filepath.Join(l.Home, ".rick", "pi", "agent") {
		t.Fatalf("AgentDir = %q", l.AgentDir)
	}
	if l.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", l.Port, DefaultPort)
	}
	if l.GoCache != filepath.Join(root, "gc") || l.GoModCache != filepath.Join(root, "gm") || l.NpmCache != filepath.Join(root, "npm-cache") {
		t.Fatalf("缓存未按显式值解析: %+v", l)
	}
	if !strings.HasPrefix(l.Token, "devtok-") || len(l.Token) != len("devtok-")+8 {
		t.Fatalf("生成 token 形态错误: %q", l.Token)
	}
	// 派生路径
	if l.StateDir() != filepath.Join(l.Home, ".rick") ||
		l.OverlayDist() != filepath.Join(l.Home, ".rick", "web", "dist") ||
		l.PidPath() != filepath.Join(l.Home, ".rick", "web.pid") ||
		l.BinDir() != filepath.Join(l.Home, "bin") {
		t.Fatalf("派生路径不符: %+v", l)
	}

	// 环境覆盖：Tree/Home/Port
	t.Setenv(EnvTree, filepath.Join(root, "custom-tree"))
	t.Setenv(EnvHome, filepath.Join(root, "custom-home"))
	t.Setenv(EnvPort, "19999")
	l2, err := DefaultLayout(prodRepo)
	if err != nil {
		t.Fatalf("DefaultLayout(override): %v", err)
	}
	if l2.Tree != filepath.Join(root, "custom-tree") || l2.Home != filepath.Join(root, "custom-home") || l2.Port != 19999 {
		t.Fatalf("env 覆盖未生效: %+v", l2)
	}

	// 非法端口必须报错（而不是静默用默认值）
	t.Setenv(EnvPort, "not-a-port")
	if _, err := DefaultLayout(prodRepo); err == nil {
		t.Fatal("非法 RICK_DEV_PORT 应报错")
	}
	t.Setenv(EnvPort, "70000")
	if _, err := DefaultLayout(prodRepo); err == nil {
		t.Fatal("越界端口应报错")
	}
}

// TestEnsureTokenPersistsAndDiffersFromProd 验证 token 生成-持久化-复用，
// 且 dev token 与生产 token 不共用（不同文件、不同值）。
func TestEnsureTokenPersistsAndDiffersFromProd(t *testing.T) {
	l := newLayout(t)
	first, err := EnsureToken(l)
	if err != nil {
		t.Fatalf("EnsureToken: %v", err)
	}
	fi, err := os.Stat(filepath.Join(l.Home, "DEV_TOKEN"))
	if err != nil {
		t.Fatalf("DEV_TOKEN 未落盘: %v", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("DEV_TOKEN 权限 = %v, want 0600", fi.Mode().Perm())
	}
	second, err := EnsureToken(l)
	if err != nil {
		t.Fatalf("EnsureToken(2): %v", err)
	}
	if first != second {
		t.Fatalf("重复调用应复用同一 token: %q vs %q", first, second)
	}
	// 用户手工指定 token 时必须被尊重（不被覆盖）
	if err := os.WriteFile(filepath.Join(l.Home, "DEV_TOKEN"), []byte("devtok-mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	third, err := EnsureToken(l)
	if err != nil {
		t.Fatal(err)
	}
	if third != "devtok-mine" {
		t.Fatalf("应复用已有 token，得到 %q", third)
	}
}

// TestServerEnvWhitelist 验证启动环境是白名单：必须有隔离三件套（HOME /
// RICK_STATE_DIR / RICK_PI_AGENT_DIR）+ 构建缓存，且不泄漏生产变量。
func TestServerEnvWhitelist(t *testing.T) {
	l := newLayout(t)
	t.Setenv("PI_SESSION_ID", "leak-me")
	t.Setenv("RICK_PROD_STATE_DIR", "")
	env := l.ServerEnv()
	joined := strings.Join(env, "\n")
	for _, want := range []string{
		"HOME=" + l.Home,
		"RICK_STATE_DIR=" + l.StateDir(),
		"RICK_PI_AGENT_DIR=" + l.AgentDir,
		"RICK_PROD_STATE_DIR=" + filepath.Join(l.ProdHome, ".rick"),
		"GOCACHE=" + l.GoCache,
		"GOMODCACHE=" + l.GoModCache,
		"npm_config_cache=" + l.NpmCache,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("ServerEnv 缺少 %q\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "PI_SESSION_ID=") {
		t.Error("ServerEnv 泄漏了调用者的 PI_SESSION_ID（应白名单，不继承整个 environ）")
	}
}

// TestOwnedByDevRefusesForeignProcess 是**防误杀**的核心测试：
// 自己起的、带 dev 标记的子进程 → 归属成立；不带标记的进程（本测试进程）→ 拒绝。
func TestOwnedByDevRefusesForeignProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("依赖 /proc")
	}
	l := newLayout(t)

	// ① 带 dev 标记的进程（真的 spawn 一个 sleep，环境里只有我们的白名单）
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep binary: %v", err)
	}
	cmd := exec.Command(sleep, "30")
	cmd.Env = l.ServerEnv()
	cmd.Dir = l.Tree
	if err := os.MkdirAll(l.Tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	// 等 /proc/<pid>/environ 稳定
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !ownedByDev(cmd.Process.Pid, l) {
		time.Sleep(20 * time.Millisecond)
	}
	if !ownedByDev(cmd.Process.Pid, l) {
		t.Fatalf("带 dev 标记的进程未被识别（pid %d）", cmd.Process.Pid)
	}

	// ② 本测试进程没有 dev 标记 → 必须拒绝（否则 Terminate 会杀错人）
	if ownedByDev(os.Getpid(), l) {
		t.Fatal("无 dev 标记的进程被判为归属（误杀风险！）")
	}

	// ③ StopOld 只杀归属成立的进程：dev 进程被杀；外来进程既不在候选也不被杀
	foreign := exec.Command(sleep, "30")
	foreign.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir()}
	if err := foreign.Start(); err != nil {
		t.Fatalf("start foreign sleep: %v", err)
	}
	t.Cleanup(func() { _ = foreign.Process.Kill(); _, _ = foreign.Process.Wait() })

	stopped, skipped := StopOld(l)
	if !containsInt(stopped, cmd.Process.Pid) {
		t.Fatalf("StopOld 未停掉 dev 进程（stopped=%v skipped=%v）", stopped, skipped)
	}
	if containsInt(stopped, foreign.Process.Pid) || containsInt(skipped, foreign.Process.Pid) {
		t.Fatalf("StopOld 不该把无 dev 标记的进程当候选（stopped=%v skipped=%v）", stopped, skipped)
	}
	if !alive(foreign.Process.Pid) {
		t.Fatal("外来进程不应被终止")
	}

	// ④ 危险场景：pid 文件里写着一个**外来**进程 → 必须拒杀并上报 skipped
	if err := os.MkdirAll(l.StateDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.PidPath(), []byte(itoa(foreign.Process.Pid)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stopped2, skipped2 := StopOld(l)
	if containsInt(stopped2, foreign.Process.Pid) {
		t.Fatalf("pid 文件指向外来进程时仍然杀了它（误杀！）: stopped=%v", stopped2)
	}
	if !containsInt(skipped2, foreign.Process.Pid) {
		t.Fatalf("拒杀的外来进程应上报 skipped（可见性）: skipped=%v", skipped2)
	}
	if !alive(foreign.Process.Pid) {
		t.Fatal("外来进程必须仍然存活")
	}
}

// TestStopOldClearsStalePidFileAndHandlesMissingFile 验证陈旧 pid 文件不会
// 让下一次 up 误判（pid 文件内容指向一个不存在的 pid → 直接清理）。
func TestStopOldClearsStalePidFileAndHandlesMissingFile(t *testing.T) {
	l := newLayout(t)
	if err := os.MkdirAll(l.StateDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.PidPath(), []byte("999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stopped, skipped := StopOld(l)
	if len(stopped) != 0 || len(skipped) != 0 {
		t.Fatalf("陈旧 pid 文件不该产生 stopped/skipped: %v %v", stopped, skipped)
	}
	if _, err := os.Stat(l.PidPath()); !os.IsNotExist(err) {
		t.Fatal("陈旧 pid 文件应被清理")
	}
	// 无 pid 文件也必须安全返回
	if stopped, skipped := StopOld(l); len(stopped) != 0 || len(skipped) != 0 {
		t.Fatalf("无 pid 文件时应静默返回: %v %v", stopped, skipped)
	}
}

// TestBuildFailureReturnsStderrTail 验证构建失败时回传 stderr 尾部（AI 会话
// 靠它定位构建错误），并且不污染 PATH（fake 脚本恢复系统 PATH）。
func TestBuildFailureReturnsStderrTail(t *testing.T) {
	l := newLayout(t)
	// dev 树里放最小可构建骨架（只需 web/package.json 存在即通过前置检查）
	if err := os.MkdirAll(filepath.Join(l.Tree, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.Tree, "web", "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 注入假的 npm：打印可识别错误到 stderr 并非零退出
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "npm")
	script := "#!/bin/sh\nPATH=/usr/bin:/bin\necho 'FAKE-NPM-BOOM-12345' 1>&2\nexit 7\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := Build(l, "", "")
	if err == nil {
		t.Fatal("npm 失败时 Build 应返回错误")
	}
	var se *StageError
	if !asStageError(err, &se) || se.Stage != "build" {
		t.Fatalf("错误应带 stage=build: %v", err)
	}
	if !strings.Contains(err.Error(), "FAKE-NPM-BOOM-12345") {
		t.Fatalf("错误里应包含 npm stderr 尾部: %v", err)
	}
}

// TestBuildIDRoundTripAndLatestBin 验证「文件名即指纹」：BuildIDFromBin 能反解
// LatestBin 取到最新构建。
func TestBuildIDRoundTripAndLatestBin(t *testing.T) {
	l := newLayout(t)
	if err := os.MkdirAll(l.BinDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"abc1234-260101010101", "abc1234-260101020202"} {
		if err := os.WriteFile(filepath.Join(l.BinDir(), DevBinPrefix+id), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := LatestBin(l)
	if err != nil {
		t.Fatalf("LatestBin: %v", err)
	}
	if got := BuildIDFromBin(latest); got != "abc1234-260101020202" {
		t.Fatalf("BuildIDFromBin = %q, 期望最新的指纹", got)
	}
	if !strings.Contains(filepath.Base(latest), "260101020202") {
		t.Fatalf("LatestBin 未取到最新: %s", latest)
	}

	// 空 bin 目录 → 明确报错（而不是起一个不存在的二进制）
	empty := newLayout(t)
	if _, err := LatestBin(empty); err == nil {
		t.Fatal("无构建时应报错")
	}
}

// TestUpRejectsUnusableBinary 验证 Up 对不可用二进制的早失败（stage=start）。
func TestUpRejectsUnusableBinary(t *testing.T) {
	l := newLayout(t)
	_, err := Up(l, filepath.Join(l.Home, "does-not-exist"))
	if err == nil {
		t.Fatal("不存在的二进制应失败")
	}
	var se *StageError
	if !asStageError(err, &se) || se.Stage != "start" {
		t.Fatalf("应报 stage=start: %v", err)
	}
	// 目录也不是可执行文件
	if _, err := Up(l, l.Home); err == nil {
		t.Fatal("目录不应被当作二进制")
	}
}

// TestWriteEnvFileAndState 验证布局快照与运行态文件的字段完整性（人/AI 读它）。
func TestWriteEnvFileAndState(t *testing.T) {
	l := newLayout(t)
	if err := l.WriteEnvFile(); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}
	body, err := os.ReadFile(l.EnvPath())
	if err != nil {
		t.Fatalf("读 dev.env: %v", err)
	}
	for _, want := range []string{"RICK_DEV_TREE=" + l.Tree, "RICK_DEV_HOME=" + l.Home,
		"RICK_DEV_AGENT_DIR=" + l.AgentDir, "RICK_DEV_PORT=" + itoa(l.Port), "RICK_DEV_TOKEN=" + l.Token} {
		if !strings.Contains(string(body), want) {
			t.Errorf("dev.env 缺少 %q\n%s", want, body)
		}
	}

	res := UpResult{Bin: filepath.Join(l.BinDir(), DevBinPrefix+"abc1234-260101010101"),
		BuildID: "abc1234-260101010101", PID: 4242, Port: l.Port, HealthMS: 137}
	if err := l.writeState(res); err != nil {
		t.Fatalf("writeState: %v", err)
	}
	got, err := l.readState()
	if err != nil {
		t.Fatalf("readState: %v", err)
	}
	if got.BuildID != res.BuildID || got.PID != 4242 || got.HealthMS != 137 || got.Bin != res.Bin {
		t.Fatalf("状态文件往返不一致: %+v", got)
	}
	if err := l.clearState(); err != nil {
		t.Fatalf("clearState: %v", err)
	}
	if _, err := l.readState(); err == nil {
		t.Fatal("clearState 后不该还能读到状态")
	}
}

// TestInitIsIdempotentOnSeededAgent 验证 pi 沙盒种子「已存在不覆盖」与
// overlay 软链幂等。
func TestInitIsIdempotentOnSeededAgent(t *testing.T) {
	l := newLayout(t)
	// 生产侧种子
	prodAgent := filepath.Join(l.ProdHome, ".rick", "pi", "agent")
	if err := os.MkdirAll(prodAgent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prodAgent, "auth.json"), []byte(`{"prod":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureAgentSeed(l); err != nil {
		t.Fatalf("ensureAgentSeed: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(l.AgentDir, "auth.json")); string(data) != `{"prod":true}` {
		t.Fatalf("种子未拷贝: %s", data)
	}
	// dev 侧改动后再次 init → 不覆盖
	if err := os.WriteFile(filepath.Join(l.AgentDir, "auth.json"), []byte(`{"dev":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureAgentSeed(l); err != nil {
		t.Fatalf("ensureAgentSeed(2): %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(l.AgentDir, "auth.json")); string(data) != `{"dev":true}` {
		t.Fatalf("已存在的 dev 配置被覆盖: %s", data)
	}

	// overlay 软链幂等（dist/index.html 必须存在）
	dist := filepath.Join(l.Tree, "web", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureOverlay(l); err != nil {
		t.Fatalf("ensureOverlay: %v", err)
	}
	if err := ensureOverlay(l); err != nil {
		t.Fatalf("ensureOverlay(2) 应幂等: %v", err)
	}
	if target, err := os.Readlink(l.OverlayDist()); err != nil || target != dist {
		t.Fatalf("overlay 软链指向错误: %q %v", target, err)
	}
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// asStageError 是 errors.As 的小包装（避免在测试文件里再引 errors 包名冲突）。
func asStageError(err error, target **StageError) bool {
	for err != nil {
		if se, ok := err.(*StageError); ok {
			*target = se
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
