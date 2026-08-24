package env

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// update-pi（职责 1+2 的更新侧）
// ---------------------------------------------------------------------------

// TestParseUpdateTarget 覆盖 CLI 位置参数 → 更新目标的全部映射。
func TestParseUpdateTarget(t *testing.T) {
	cases := []struct {
		arg        string
		wantTarget UpdateTarget
		wantOne    string
	}{
		{"", UpdateAll, ""},
		{"all", UpdateAll, ""},
		{"ALL", UpdateAll, ""},
		{"pi", UpdateSelf, ""},
		{"self", UpdateSelf, ""},
		{"PI", UpdateSelf, ""},
		{"extensions", UpdateExtensions, ""},
		{"ext", UpdateExtensions, ""},
		{"models", UpdateModels, ""},
		{"pi-subagents", UpdateOne, "pi-subagents"},
		{"npm:pi-web-access", UpdateOne, "npm:pi-web-access"},
	}
	for _, tc := range cases {
		gotTarget, gotOne := ParseUpdateTarget(tc.arg)
		if gotTarget != tc.wantTarget || gotOne != tc.wantOne {
			t.Errorf("ParseUpdateTarget(%q) = (%v, %q), want (%v, %q)",
				tc.arg, gotTarget, gotOne, tc.wantTarget, tc.wantOne)
		}
	}
}

// TestBuildUpdatePlan 验证更新计划：All 的顺序必须是 pi → extensions → models
// （先换 runtime 再更新其上安装的扩展）。
func TestBuildUpdatePlan(t *testing.T) {
	if got := buildUpdatePlan(UpdateAll, ""); len(got) != 3 ||
		got[0].kind != "pi" || got[1].kind != "extensions" || got[2].kind != "models" {
		t.Errorf("buildUpdatePlan(All) = %+v, want [pi extensions models]", got)
	}
	if got := buildUpdatePlan(UpdateSelf, ""); len(got) != 1 || got[0].kind != "pi" {
		t.Errorf("buildUpdatePlan(Self) = %+v, want [pi]", got)
	}
	if got := buildUpdatePlan(UpdateExtensions, ""); len(got) != 1 || got[0].kind != "extensions" {
		t.Errorf("buildUpdatePlan(Extensions) = %+v, want [extensions]", got)
	}
	if got := buildUpdatePlan(UpdateModels, ""); len(got) != 1 || got[0].kind != "models" {
		t.Errorf("buildUpdatePlan(Models) = %+v, want [models]", got)
	}
	if got := buildUpdatePlan(UpdateOne, "npm:pi-subagents"); len(got) != 1 ||
		got[0].kind != "one" || got[0].source != "npm:pi-subagents" {
		t.Errorf("buildUpdatePlan(One) = %+v, want [one npm:pi-subagents]", got)
	}
}

// fakeUpdatePiEnv 搭建隔离环境：HOME/RICK_PI_AGENT_DIR 指向 temp，PATH 放一个
// 会把所有调用参数记录到 $FAKE_PI_LOG 的假 pi，并支持 list/--version/update。
type fakeUpdatePiEnv struct {
	agentDir string
	logPath  string
}

func setupFakeUpdatePiEnv(t *testing.T, listOutput string, version string) *fakeUpdatePiEnv {
	t.Helper()
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("RICK_PI_AGENT_DIR", agentDir)

	logPath := filepath.Join(home, "pi-calls.log")
	script := "#!/bin/sh\n" +
		"echo \"$@ >> $FAKE_PI_LOG\" >> \"$FAKE_PI_LOG\"\n" +
		"case \"$1\" in\n" +
		"  list) printf '%s\\n' \"" + strings.ReplaceAll(listOutput, `"`, `\"`) + "\" ;;\n" +
		"  --version) echo \"" + version + "\" ;;\n" +
		"  update) exit 0 ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(binDir, "pi"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	// 假 pi 目录前置 + 保留原 PATH：假 pi 优先解析，但 python3 等系统工具仍可用。
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	t.Setenv("FAKE_PI_LOG", logPath)
	return &fakeUpdatePiEnv{agentDir: agentDir, logPath: logPath}
}

// piCalls 读取假 pi 的调用日志。
func (e *fakeUpdatePiEnv) piCalls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(e.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var calls []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			calls = append(calls, line)
		}
	}
	return calls
}

// TestUpdatePi_One_PathPi 验证单扩展更新：源名解析为注册形态（npm: 前缀），
// `pi update npm:<name>` 确实被调用，且结果记录 OneUpdated。
func TestUpdatePi_One_PathPi(t *testing.T) {
	env := setupFakeUpdatePiEnv(t,
		"User packages:\n  npm:pi-subagents\n  npm:pi-web-access\n", "0.84.1")

	res, err := UpdatePi(UpdateOne, "pi-subagents")
	if err != nil {
		t.Fatalf("UpdatePi(One, pi-subagents): %v", err)
	}
	if res.OneUpdated != "npm:pi-subagents" {
		t.Errorf("OneUpdated = %q, want npm:pi-subagents", res.OneUpdated)
	}
	calls := env.piCalls(t)
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "update npm:pi-subagents") {
		t.Errorf("expected `pi update npm:pi-subagents` call, got calls:\n%s", joined)
	}
}

// TestUpdatePi_OneUnknownExtension 验证未注册扩展直接报错（并列出已注册项）。
func TestUpdatePi_OneUnknownExtension(t *testing.T) {
	setupFakeUpdatePiEnv(t,
		"User packages:\n  npm:pi-subagents\n  npm:pi-web-access\n", "0.84.1")

	_, err := UpdatePi(UpdateOne, "no-such-extension")
	if err == nil {
		t.Fatal("expected error for unknown extension")
	}
	if !strings.Contains(err.Error(), "not registered") ||
		!strings.Contains(err.Error(), "npm:pi-subagents") {
		t.Errorf("error should mention registration and known extensions, got: %v", err)
	}
}

// TestUpdatePi_Extensions_PathPi 验证全部扩展更新走 `pi update --extensions`。
func TestUpdatePi_Extensions_PathPi(t *testing.T) {
	env := setupFakeUpdatePiEnv(t, "User packages:\n  npm:pi-subagents\n", "0.84.1")

	res, err := UpdatePi(UpdateExtensions, "")
	if err != nil {
		t.Fatalf("UpdatePi(Extensions): %v", err)
	}
	if !res.ExtensionsUpdated {
		t.Error("ExtensionsUpdated = false, want true")
	}
	calls := strings.Join(env.piCalls(t), "\n")
	if !strings.Contains(calls, "update --extensions") {
		t.Errorf("expected `pi update --extensions` call, got:\n%s", calls)
	}
	if res.PiUpdated {
		t.Error("PiUpdated should stay false for extensions-only target")
	}
}

// TestUpdatePi_Self_ManagedRuntime 验证托管 runtime 更新走 rick 自己的
// InstallManagedPI（npm --prefix），不依赖 `pi update --self`；npm 用假二进制
// 记录调用参数。
func TestUpdatePi_Self_ManagedRuntime(t *testing.T) {
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	runtimeBin := filepath.Join(agentDir, "runtime", "node_modules", ".bin", "pi")
	binDir := filepath.Join(home, "bin")
	for _, d := range []string{filepath.Dir(runtimeBin), binDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("RICK_PI_AGENT_DIR", agentDir)

	// 托管 runtime 已存在的假 pi（--version 输出旧版本号）。
	if err := os.WriteFile(runtimeBin, []byte("#!/bin/sh\ncase \"$1\" in --version) echo 0.84.1;; esac\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// 假 npm：记录调用参数 + 制造 node_modules/.bin/pi 已安装的假象。
	npmLog := filepath.Join(home, "npm-calls.log")
	npmScript := "#!/bin/sh\necho \"$@\" >> \"" + npmLog + "\"\n" +
		"mkdir -p \"$3/node_modules/.bin\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(binDir, "npm"), []byte(npmScript), 0755); err != nil {
		t.Fatal(err)
	}
	// 假 npm 前置 + 保留原 PATH（mkdir 等外部命令来自系统 PATH）。
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	res, err := UpdatePi(UpdateSelf, "")
	if err != nil {
		t.Fatalf("UpdatePi(Self, managed): %v", err)
	}
	if !res.PiUpdated || !res.ManagedRuntime {
		t.Errorf("PiUpdated=%v ManagedRuntime=%v, want true/true", res.PiUpdated, res.ManagedRuntime)
	}
	if res.PiBefore != "0.84.1" {
		t.Errorf("PiBefore = %q, want 0.84.1", res.PiBefore)
	}
	data, err := os.ReadFile(npmLog)
	if err != nil {
		t.Fatalf("fake npm not invoked: %v", err)
	}
	npmCall := string(data)
	// InstallManagedPI 用 npm install --prefix <RuntimeDir> @earendil-works/pi-coding-agent
	if !strings.Contains(npmCall, "--prefix") ||
		!strings.Contains(npmCall, "pi-coding-agent") {
		t.Errorf("expected npm install --prefix ... pi-coding-agent, got: %s", npmCall)
	}
	// 托管路径更新不得调用 `pi update --self`（该路径对非全局安装不可用）。
	if strings.Contains(npmCall, "update") {
		t.Errorf("managed runtime update must not use pi update --self, got: %s", npmCall)
	}
}

// TestUpdatePi_Self_NoPiAnywhere 验证既无托管 runtime 也无 PATH pi 时给出
// 可操作的错误（引导 init-pi）。
func TestUpdatePi_Self_NoPiAnywhere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RICK_PI_AGENT_DIR", filepath.Join(home, ".rick", "pi", "agent"))
	t.Setenv("PATH", t.TempDir()) // 空 PATH：无 pi、无 npm

	_, err := UpdatePi(UpdateSelf, "")
	if err == nil {
		t.Fatal("expected error when pi is missing everywhere")
	}
	if !strings.Contains(err.Error(), "init-pi") {
		t.Errorf("error should point to init-pi, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// QuickCheck（职责 4 的更新侧自检）
// ---------------------------------------------------------------------------

// TestQuickCheck_ReadyAndNotReady 验证快速自检各字段：就绪环境全绿；缺扩展 +
// helper.py 语法坏时给出对应告警。
func TestQuickCheck_ReadyAndNotReady(t *testing.T) {
	// python3 缺失时跳过语法分支（极少数环境）。
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}

	// --- 场景 1：就绪 ---
	env := setupFakeUpdatePiEnv(t,
		"User packages:\n  npm:pi-subagents\n  npm:pi-web-access\n", "9.9.9")
	if err := os.MkdirAll(filepath.Join(env.agentDir, "agents"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"think", "research", "exporter"} {
		if err := os.WriteFile(filepath.Join(env.agentDir, "agents", name+".md"),
			[]byte("---\nname: "+name+"\n---\nbody\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	hooksDir := filepath.Join(env.agentDir, "extensions", "rick-gates")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "helper.py"),
		[]byte("x = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res := QuickCheck()
	if res.PiVersion != "9.9.9" {
		t.Errorf("PiVersion = %q, want 9.9.9", res.PiVersion)
	}
	if len(res.MissingExtensions) != 0 {
		t.Errorf("MissingExtensions = %v, want empty", res.MissingExtensions)
	}
	if len(res.NotReady) != 0 {
		t.Errorf("NotReady = %v, want empty", res.NotReady)
	}
	if !res.GatesHelperOK {
		t.Errorf("GatesHelperOK = false, note: %s", res.GatesHelperNote)
	}

	// --- 场景 2：缺扩展 + helper 语法错误 + agent 缺失 ---
	env2 := setupFakeUpdatePiEnv(t, "User packages:\n  npm:pi-subagents\n", "9.9.9")
	hooksDir2 := filepath.Join(env2.agentDir, "extensions", "rick-gates")
	if err := os.MkdirAll(hooksDir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir2, "helper.py"),
		[]byte("def broken(:\n"), 0644); err != nil {
		t.Fatal(err)
	}

	res2 := QuickCheck()
	if len(res2.MissingExtensions) != 1 || res2.MissingExtensions[0] != "pi-web-access" {
		t.Errorf("MissingExtensions = %v, want [pi-web-access]", res2.MissingExtensions)
	}
	if res2.GatesHelperOK {
		t.Error("GatesHelperOK should be false for broken helper.py")
	}
	if res2.GatesHelperNote == "" {
		t.Error("GatesHelperNote should explain the failure")
	}
	if len(res2.NotReady) == 0 {
		t.Error("NotReady should be non-empty (agents missing)")
	}
}
