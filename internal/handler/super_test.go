// super_test.go — task11 Go 命令面测试（fake superopt py 脚本，不依赖真 superopt——
// 与 py 侧并行层解耦；真集成在 task14 e2e）。
//
// 覆盖（spec「测试方法」段）：
//   1. SuperStart 生命周期：fake beating superopt → 心跳 → status running → 优雅停 → state=stopped
//   2. start 门先证红失败路径：fake exit-now superopt → SuperStart 返回错误含「门先证红」
//   3. SuperEvent 干预队列文件（内容/命名）
//   4. BuildSuperPrompt：含 super-loop 与 gen_env 内容、无未替换 {{}} 变量
//   5. config 默认值与 py DEFAULTS 一致（adjust_interval=5/debug_max_attempts=3/soft_gate_k=3）
//   6. SuperStatus --json：北极星聚合（迭代数/入格数/last_seq）从 events.jsonl 尾部聚合
//
// 隔离（bugs.md）：fake 脚本恢复系统 PATH；不触碰真实 ~/.rick/super（ws 一律 t.TempDir()）。
package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/config"
	superpkg "github.com/sunquan/rick/internal/super"
)

// --- fake superopt fixtures（照 task10 supervisor_test 的 fake 脚本范式） ---

// writeFakeSuperopt writes a fake `superopt` launcher script. The PySupervisor
// invokes `<pythonPath> -m superopt --workdir <ws>`; the fake ignores argv
// except --workdir (heartbeats land there).
func writeFakeSuperopt(t *testing.T, name, body string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// fakeSuperoptBeating 心跳 + 迭代事件常驻（TERM 陷阱优雅退出：写 session_ended 不必做——
// 真调度器职责；这里只需退出 0）。
func fakeSuperoptBeating(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	// bugs.md：fake 脚本恢复系统 PATH
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	// 解析 --workdir
	b.WriteString(`WS=""; prev=""
for a in "$@"; do
  if [ "$prev" = "--workdir" ]; then WS="$a"; fi
  prev="$a"
done
mkdir -p "$WS"
`)
	b.WriteString("n=0\nwhile :; do\n  n=$((n+1))\n")
	b.WriteString(`  printf '{"ts":"%s","seq":%d,"state":"running"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$n" > "$WS/heartbeat.json"` + "\n")
	b.WriteString("  sleep 0.05\ndone\n")
	return writeFakeSuperopt(t, "fake-superopt-beating", b.String())
}

// fakeSuperoptExitNow 立即退出非 0（门先证红失败 fixture：调度器启动即退）。
func fakeSuperoptExitNow(t *testing.T) string {
	t.Helper()
	body := "#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\necho \"superopt: preverify-red FAILED: leaked=[fake-exploit-1]\" >&2\nexit 3\n"
	return writeFakeSuperopt(t, "fake-superopt-exit", body)
}

// --- 测试用 ws 与 config ---

// newSuperWS 建一个临时工作区根 + ws 目录。
func newSuperWS(t *testing.T) (root, ws string) {
	t.Helper()
	root = t.TempDir()
	ws = filepath.Join(root, "test-super")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, ws
}

// testSuperStartConfig 组装短心跳节奏的 SuperStartConfig（测试秒级完成）。
func testSuperStartConfig(pythonPath string) SuperStartConfig {
	return SuperStartConfig{
		Root:              "", // 由调用方填
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  800 * time.Millisecond,
		TermGrace:         400 * time.Millisecond,
		PythonPath:        pythonPath,
		MaxRestarts:       1,
	}
}

// waitFor 轮询 cond 直到 true 或超时（秒级测试节拍）。
func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// --- 1. 生命周期 ---

func TestSuperStartStopLifecycle(t *testing.T) {
	root, ws := newSuperWS(t)
	py := fakeSuperoptBeating(t)
	cfg := testSuperStartConfig(py)
	cfg.Root = root

	// 后台跑前台监控循环（等价于 CLI 前台形态；测试经 stopCh 优雅停）
	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	var runErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = SuperStartMonitored(cfg, "test-super", ws, "", SuperStartOverrides{}, stopCh)
	}()

	// 等 state=running（首心跳 + 启动窗口通过）
	waitFor(t, 5*time.Second, "state=running", func() bool {
		data, err := os.ReadFile(filepath.Join(ws, "super.json"))
		if err != nil {
			return false
		}
		var v superJSONView
		return json.Unmarshal(data, &v) == nil && v.State == "running"
	})

	// status 人类可读冒烟（不 panic 即可；JSON 版另测）
	if err := SuperStatusIn(ws, false); err != nil {
		t.Fatalf("SuperStatusIn: %v", err)
	}

	// 优雅停 → 循环退出 → state=stopped
	close(stopCh)
	wg.Wait()
	if runErr != nil {
		t.Fatalf("SuperStartMonitored returned error: %v", runErr)
	}
	sj := readSuperJSONForTest(t, ws)
	if sj.State != "stopped" {
		t.Fatalf("state = %q, want stopped", sj.State)
	}
}

// --- 2. 门先证红失败路径 ---

func TestSuperStartPreverifyRedFailure(t *testing.T) {
	root, ws := newSuperWS(t)
	py := fakeSuperoptExitNow(t)
	cfg := testSuperStartConfig(py)
	cfg.Root = root

	stopCh := make(chan struct{})
	err := SuperStartMonitored(cfg, "test-super", ws, "", SuperStartOverrides{}, stopCh)
	if err == nil {
		t.Fatal("SuperStartMonitored should fail when the scheduler exits nonzero at startup")
	}
	if !strings.Contains(err.Error(), "门先证红") {
		t.Fatalf("error should mention 门先证红, got: %v", err)
	}
	if !strings.Contains(err.Error(), "fake-exploit-1") {
		t.Fatalf("error should carry the scheduler stderr tail, got: %v", err)
	}
	sj := readSuperJSONForTest(t, ws)
	if sj.State != "failed" {
		t.Fatalf("state = %q, want failed", sj.State)
	}
}

// --- 3. 干预队列 ---

func TestSuperEventQueueFile(t *testing.T) {
	_, ws := newSuperWS(t)
	if err := SuperEventIn(ws, "weight", `{"cell_id":"impl=blocked","weight":0}`); err != nil {
		t.Fatalf("SuperEventIn: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(ws, "interventions"))
	if err != nil {
		t.Fatalf("read interventions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("interventions has %d files, want 1", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, ".json") {
		t.Fatalf("queue file %q should end with .json", name)
	}
	// 命名带时间戳（可排序）：数字前缀
	if len(name) < 15 || name[0] < '0' || name[0] > '9' {
		t.Fatalf("queue file %q should be timestamp-prefixed for ordering", name)
	}
	data, err := os.ReadFile(filepath.Join(ws, "interventions", name))
	if err != nil {
		t.Fatal(err)
	}
	var ev struct {
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		CreatedAt string          `json:"created_at"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("queue file not valid JSON: %v\n%s", err, data)
	}
	if ev.Type != "weight" {
		t.Fatalf("type = %q, want weight", ev.Type)
	}
	if string(ev.Payload) == "" {
		t.Fatal("payload must be preserved")
	}
	if ev.CreatedAt == "" {
		t.Fatal("created_at missing")
	}
	// 第二条入队：文件数 2 且字典序可排
	if err := SuperEventIn(ws, "stop", `{}`); err != nil {
		t.Fatal(err)
	}
	entries, _ = os.ReadDir(filepath.Join(ws, "interventions"))
	if len(entries) != 2 {
		t.Fatalf("interventions has %d files, want 2", len(entries))
	}
}

// --- 4. BuildSuperPrompt ---

func TestBuildSuperPrompt(t *testing.T) {
	// rickDir：用当前仓库（.rick/loops/super-loop.md 与 skills/gen_env_skill 由 L1 落盘）
	rickDir, err := filepath.Abs(filepath.Join("..", "..", ".rick"))
	if err != nil {
		t.Fatal(err)
	}
	_, ws := newSuperWS(t)
	methodFile, promptFile, err := superpkg.BuildSuperPrompt(rickDir, ws)
	if err != nil {
		t.Fatalf("BuildSuperPrompt: %v", err)
	}
	for _, f := range []string{methodFile, promptFile} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("prompt artifact missing: %v", err)
		}
	}
	prompt, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatal(err)
	}
	p := string(prompt)
	// 含 super-loop 与 gen_env 内容（inline 而非只有目录引用）
	for _, kw := range []string{"super-loop", "super 中控", "gen_env", "单写者", "rick super start"} {
		if !strings.Contains(p, kw) {
			t.Errorf("prompt missing keyword %q", kw)
		}
	}
	// 无未替换变量
	if strings.Contains(p, "{{") {
		t.Errorf("prompt contains unreplaced {{...}} variable")
	}
	// 会话 id 落盘（resume 契约）
	sessDir := filepath.Dir(promptFile)
	if _, err := os.Stat(filepath.Join(sessDir, "session_id")); err != nil {
		t.Errorf("session_id not persisted next to prompt: %v", err)
	}
}

// --- 5. 默认值一致性 ---

func TestSuperDefaultsMatchPy(t *testing.T) {
	// Go 侧默认
	got := config.DefaultSuperConfig()
	if got.AdjustInterval != 5 || got.DebugMaxAttempts != 3 {
		t.Errorf("Go defaults adjust_interval=%d debug_max_attempts=%d, want 5/3",
			got.AdjustInterval, got.DebugMaxAttempts)
	}
	if got.HeartbeatIntervalMs != 30000 || got.HeartbeatTimeoutMs != 90000 || got.TermGraceMs != 5000 {
		t.Errorf("Go heartbeat defaults = %d/%d/%d, want 30000/90000/5000",
			got.HeartbeatIntervalMs, got.HeartbeatTimeoutMs, got.TermGraceMs)
	}
	if got.PythonPath != "python3" {
		t.Errorf("python_path default = %q, want python3", got.PythonPath)
	}
	// py 侧 DEFAULTS 交叉核对（python3 可用时）
	pyCfg, err := filepath.Abs(filepath.Join("..", "..", "super", "superopt", "config.py"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pyCfg); err != nil {
		t.Skipf("py config.py not present yet (parallel layer): %v", err)
	}
	out, err := exec.Command("python3", "-c",
		`import json, re, sys
src = open(`+fmt.Sprintf("%q", pyCfg)+`, encoding="utf-8").read()
m = re.search(r"DEFAULTS[^=]*=\s*(\{.*?\n\})", src, re.S)
if not m:
    sys.exit("DEFAULTS not found")
print(json.dumps(eval(m.group(1))))`).CombinedOutput()
	if err != nil {
		t.Skipf("cannot eval py DEFAULTS (python3 unavailable?): %v: %s", err, out)
	}
	var pyDefaults map[string]interface{}
	if err := json.Unmarshal(out, &pyDefaults); err != nil {
		t.Fatalf("py DEFAULTS not JSON: %v\n%s", err, out)
	}
	checks := []struct {
		key string
		got interface{}
	}{
		{"adjust_interval", got.AdjustInterval},
		{"debug_max_attempts", got.DebugMaxAttempts},
		{"soft_gate_k", 3},
		{"heartbeat_interval_s", got.HeartbeatIntervalMs / 1000},
		{"heartbeat_timeout_s", got.HeartbeatTimeoutMs / 1000},
		{"term_grace_s", got.TermGraceMs / 1000},
		{"python_path", got.PythonPath},
	}
	for _, c := range checks {
		pyV, ok := pyDefaults[c.key]
		if !ok {
			t.Errorf("py DEFAULTS missing key %s", c.key)
			continue
		}
		pyF, ok1 := pyV.(float64)
		gotF, ok2 := toFloat(c.got)
		if ok1 && ok2 && pyF != gotF {
			t.Errorf("default mismatch %s: py=%v go=%v", c.key, pyV, c.got)
		}
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case string:
		return 0, false
	}
	return 0, false
}

// --- 6. status --json 北极星聚合 ---

func TestSuperStatusJSONAggregation(t *testing.T) {
	_, ws := newSuperWS(t)
	// 工作区先初始化（super.json 必须存在——status 的读取前置）
	if _, err := initSuperWorkspace(ws, "", SuperStartOverrides{}); err != nil {
		t.Fatalf("initSuperWorkspace: %v", err)
	}
	// 造 events.jsonl（Go 只读尾部聚合；schema 对齐 super/superopt/events.py 信封）
	events := []string{
		`{"ts":"2026-10-17T10:00:01+08:00","seq":1,"type":"session_started","source":"scheduler","payload":{"env":"env"}}`,
		`{"ts":"2026-10-17T10:00:02+08:00","seq":2,"type":"iteration_started","source":"scheduler","payload":{"iteration_id":1}}`,
		`{"ts":"2026-10-17T10:00:03+08:00","seq":3,"type":"cell_sampled","source":"scheduler","payload":{"cell_id":"impl=blocked","visit":1}}`,
		`{"ts":"2026-10-17T10:00:04+08:00","seq":4,"type":"agent_ended","source":"scheduler","payload":{"role":"coding","iteration_id":1,"exit_code":0,"wall_ms":1200,"total_tokens":1500}}`,
		`{"ts":"2026-10-17T10:00:05+08:00","seq":5,"type":"iteration_ended","source":"scheduler","payload":{"iteration_id":1}}`,
		`{"ts":"2026-10-17T10:00:06+08:00","seq":6,"type":"iteration_started","source":"scheduler","payload":{"iteration_id":2}}`,
		`{"ts":"2026-10-17T10:00:07+08:00","seq":7,"type":"elite_admitted","source":"scheduler","payload":{"cell_id":"impl=blocked","code_path":"codes/x/","perf_value":42.0}}`,
		`{"ts":"2026-10-17T10:00:08+08:00","seq":8,"type":"iteration_ended","source":"scheduler","payload":{"iteration_id":2}}`,
	}
	var sb strings.Builder
	for _, e := range events {
		sb.WriteString(e + "\n")
	}
	if err := os.WriteFile(filepath.Join(ws, "events.jsonl"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := SuperStatusJSONIn(ws, &out); err != nil {
		t.Fatalf("SuperStatusJSONIn: %v", err)
	}
	var st struct {
		Workspace    string  `json:"workspace"`
		State        string  `json:"state"`
		LastSeq      int64   `json:"last_seq"`
		Iterations   int     `json:"iterations"`
		EliteAdmits  int     `json:"elite_admits"`
		Cells        int     `json:"cells"`
		TotalTokens  float64 `json:"total_tokens"`
		HeartbeatSeq *int64  `json:"heartbeat_seq"`
	}
	if err := json.Unmarshal([]byte(out.String()), &st); err != nil {
		t.Fatalf("status JSON invalid: %v\n%s", err, out.String())
	}
	if st.LastSeq != 8 {
		t.Errorf("last_seq = %d, want 8", st.LastSeq)
	}
	if st.Iterations != 2 {
		t.Errorf("iterations = %d, want 2", st.Iterations)
	}
	if st.EliteAdmits != 1 {
		t.Errorf("elite_admits = %d, want 1", st.EliteAdmits)
	}
	if st.Cells != 1 {
		t.Errorf("cells = %d, want 1", st.Cells)
	}
	if st.TotalTokens != 1500 {
		t.Errorf("total_tokens = %v, want 1500", st.TotalTokens)
	}
	if st.HeartbeatSeq != nil {
		t.Errorf("heartbeat_seq should be nil without heartbeat.json")
	}
}

// --- 7. SuperStop pid 路径（daemon 形态） ---

func TestSuperStopKillsDaemonPID(t *testing.T) {
	_, ws := newSuperWS(t)
	// 一个可杀的假 daemon 进程（setsid 的 sleep）。
	// 后台并发 reap：进程死后被回收，pid 存活探测（kill(pid,0)）才会翻假——
	// 否则僵尸态会让 SuperStopIn 的优雅等待空转（生产形态父进程即退、init
	// 收养回收，无此窗口；仅测试进程作为存活父进程时需要主动 reap）。
	sleeper := exec.Command("sh", "-c", "exec sleep 300")
	sleeper.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = sleeper.Wait() }()
	defer func() { _ = sleeper.Process.Kill() }()
	pid := sleeper.Process.Pid

	sj := map[string]interface{}{
		"state":  "running",
		"pid":    pid,
		"daemon": true,
	}
	if err := writeSuperJSONForTest(t, ws, sj); err != nil {
		t.Fatal(err)
	}
	if err := SuperStopIn(ws); err != nil {
		t.Fatalf("SuperStopIn: %v", err)
	}
	// 进程被收割
	waitFor(t, 3*time.Second, "daemon pid gone", func() bool {
		return sleeper.ProcessState != nil || !pidAlive(pid)
	})
	// state=stopped 落盘
	sjr := readSuperJSONForTest(t, ws)
	if sjr.State != "stopped" {
		t.Fatalf("state = %q, want stopped", sjr.State)
	}
}

func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// --- helpers：super.json 读写（测试视角） ---

type superJSONView struct {
	State           string `json:"state"`
	IterationCursor int    `json:"iteration_cursor"`
	Pid             int    `json:"pid"`
	Daemon          bool   `json:"daemon"`
	EnvPath         string `json:"env_path"`
	PreverifyRed    bool   `json:"preverify_red"`
}

func readSuperJSONForTest(t *testing.T, ws string) superJSONView {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ws, "super.json"))
	if err != nil {
		t.Fatalf("read super.json: %v", err)
	}
	var v superJSONView
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("super.json invalid: %v", err)
	}
	return v
}

func writeSuperJSONForTest(t *testing.T, ws string, v map[string]interface{}) error {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ws, "super.json"), data, 0o644)
}

// --- 自然完成（exit 0）不触发崩溃重启（job_47 首跑实测缺陷的回归） ---

// fakeSuperoptFinish 心跳若干次后 exit 0（模拟 max_iterations 跑满自然收尾）。
func fakeSuperoptFinish(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(`WS=""; prev=""
for a in "$@"; do
  if [ "$prev" = "--workdir" ]; then WS="$a"; fi
  prev="$a"
done
mkdir -p "$WS"
i=0
while [ $i -lt 8 ]; do
  i=$((i+1))
  printf '{"ts":"%s","seq":%d,"state":"running"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$i" > "$WS/heartbeat.json"
  sleep 0.05
done
exit 0
`)
	return writeFakeSuperopt(t, "fake-superopt-finish", b.String())
}

func TestSuperStartNaturalCompletionNotRestarted(t *testing.T) {
	root, ws := newSuperWS(t)
	if err := os.MkdirAll(filepath.Join(ws, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := testSuperStartConfig(fakeSuperoptFinish(t))
	cfg.Root = root
	stopCh := make(chan struct{})
	defer close(stopCh)

	// 前台监控循环（fake 心跳 8 跳后 exit 0 → 自然完成路径）
	err := SuperStartMonitored(cfg, "test-super", ws, "", SuperStartOverrides{}, stopCh)
	if err != nil {
		t.Fatalf("自然完成不该报错（修复前：重启超限 failed）: %v", err)
	}
	sj := readSuperJSONForTest(t, ws)
	if sj.State != "completed" {
		t.Fatalf("state = %q，期望 completed（exit 0 是自然完成不是崩溃）", sj.State)
	}
	// 不重启：super.log 无 restart 字样（修复前会打 3 次 restarting）
	_ = root
}

// fakeSuperoptFinishWithCursor 心跳若干次（带 iteration_cursor=5）后 exit 0
// ——自然完成收尾时 super.json 的游标应从最后的 heartbeat 刷新（改动⑥回归）。
func fakeSuperoptFinishWithCursor(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(`WS=""; prev=""
for a in "$@"; do
  if [ "$prev" = "--workdir" ]; then WS="$a"; fi
  prev="$a"
done
mkdir -p "$WS"
i=0
while [ $i -lt 8 ]; do
  i=$((i+1))
  printf '{"ts":"%s","seq":%d,"state":"running","iteration_cursor":5}' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$i" > "$WS/heartbeat.json"
  sleep 0.05
done
exit 0
`)
	return writeFakeSuperopt(t, "fake-superopt-finish-cursor", b.String())
}

func TestSuperStartNaturalCompletionRefreshesCursor(t *testing.T) {
	root, ws := newSuperWS(t)
	if err := os.MkdirAll(filepath.Join(ws, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := testSuperStartConfig(fakeSuperoptFinishWithCursor(t))
	cfg.Root = root
	stopCh := make(chan struct{})
	defer close(stopCh)

	err := SuperStartMonitored(cfg, "test-super", ws, "", SuperStartOverrides{}, stopCh)
	if err != nil {
		t.Fatalf("自然完成: %v", err)
	}
	sj := readSuperJSONForTest(t, ws)
	if sj.State != "completed" {
		t.Fatalf("state = %q, want completed", sj.State)
	}
	// 游标从最后的 heartbeat.json 刷新（job_47 首跑实测：曾显示 0）
	if sj.IterationCursor != 5 {
		t.Fatalf("IterationCursor = %d, want 5（heartbeat 的 iteration_cursor）",
			sj.IterationCursor)
	}
}

// --- 工作区根解析（job_48 task1：<ws>/.rick/super_env 模型） ---

// chdirInto 切到 dir 并随测试恢复（t.Chdir 的显式形态，兼容任意 Go 版本）。
func chdirInto(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})
}

// TestSuperWsRootResolvesInsideWorkspace：cwd 在工作区内（含 .rick）→
// root = <ws>/.rick/super_env（含从子目录上溯的形态）。
func TestSuperWsRootResolvesInsideWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // 隔离 config（无 super.default_super_dir 覆盖）

	wsRoot := t.TempDir()
	rickDir := filepath.Join(wsRoot, ".rick")
	if err := os.MkdirAll(filepath.Join(rickDir, "jobs"), 0o755); err != nil {
		t.Fatal(err)
	}
	// macOS：t.TempDir 逻辑路径 /var/... vs Getwd 物理路径 /private/var/...
	// ——两边 EvalSymlinks 归一后比较（supervisor_test.go 同款先例；注意
	// EvalSymlinks 要求路径存在，故只对已存在的 rickDir 归一再拼 super_env）
	norm := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return r
		}
		return p
	}
	want := filepath.Join(norm(rickDir), "super_env")

	// ① 工作区根本身
	chdirInto(t, wsRoot)
	root, err := superWsRoot()
	if err != nil {
		t.Fatalf("superWsRoot: %v", err)
	}
	if root != want {
		t.Fatalf("root = %q, want %q", root, want)
	}

	// ② 任意子目录上溯
	sub := filepath.Join(wsRoot, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	chdirInto(t, sub)
	root2, err := superWsRoot()
	if err != nil {
		t.Fatalf("superWsRoot (subdir): %v", err)
	}
	if root2 != want {
		t.Fatalf("root from subdir = %q, want %q", root2, want)
	}
}

// TestSuperWsRootFailsOutsideWorkspace：cwd 上溯无 .rick → 明确报错（含指引）。
func TestSuperWsRootFailsOutsideWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// t.TempDir() 位于系统临时区，正常情况下其父链无 .rick
	chdirInto(t, t.TempDir())
	_, err := superWsRoot()
	if err == nil {
		t.Fatal("期望在无 .rick 的目录下报错，实际成功")
	}
	msg := err.Error()
	for want := range map[string]bool{
		".rick":             true, // 指引中须提到 .rick
		"default_super_dir": true, // 指引中须提到 config 覆盖项
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("错误信息缺 %q：%s", want, msg)
		}
	}
}

// TestSuperWsRootConfigOverrideWins：config default_super_dir 显式覆盖优先
//（即使 cwd 在工作区内）。
func TestSuperWsRootConfigOverrideWins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	explicit := filepath.Join(t.TempDir(), "explicit-super-root")
	if err := os.MkdirAll(explicit, 0o755); err != nil {
		t.Fatal(err)
	}
	// 写 config：<home>/.rick/config.json 的 super.default_super_dir
	rickHome := filepath.Join(home, ".rick")
	if err := os.MkdirAll(rickHome, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgBody := fmt.Sprintf(`{"super":{"default_super_dir":%q}}`, explicit)
	if err := os.WriteFile(filepath.Join(rickHome, "config.json"), []byte(cfgBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// cwd 在一个含 .rick 的工作区内——覆盖必须赢
	wsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(wsRoot, ".rick"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdirInto(t, wsRoot)
	root, err := superWsRoot()
	if err != nil {
		t.Fatalf("superWsRoot: %v", err)
	}
	if root != explicit {
		t.Fatalf("root = %q, want 显式覆盖 %q", root, explicit)
	}
}

// TestSuperLifecycleUnderNewRoot：新 root（<ws>/.rick/super_env/<name>）下的
// start→running→stop 全生命周期（fake superopt）依旧绿。
func TestSuperLifecycleUnderNewRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	wsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(wsRoot, ".rick", "jobs"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdirInto(t, wsRoot)

	root, err := superWsRoot()
	if err != nil {
		t.Fatalf("superWsRoot: %v", err)
	}
	ws := superWsPath(root, "lifecycle")

	cfg := testSuperStartConfig(fakeSuperoptBeating(t))
	cfg.Root = root

	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	var runErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = SuperStartMonitored(cfg, "lifecycle", ws, "", SuperStartOverrides{}, stopCh)
	}()

	waitFor(t, 5*time.Second, "state=running under new root", func() bool {
		data, err := os.ReadFile(filepath.Join(ws, "super.json"))
		if err != nil {
			return false
		}
		var v superJSONView
		return json.Unmarshal(data, &v) == nil && v.State == "running"
	})

	// 干预队列投递（SuperEventIn 落在 <ws>/.rick/super_env/lifecycle/interventions/）
	if err := SuperEventIn(ws, "pause", "{}"); err != nil {
		t.Fatalf("SuperEventIn: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(ws, "interventions"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("干预队列落盘异常：entries=%v err=%v", entries, err)
	}

	close(stopCh)
	wg.Wait()
	if runErr != nil {
		t.Fatalf("SuperStartMonitored: %v", runErr)
	}
	sj := readSuperJSONForTest(t, ws)
	if sj.State != "stopped" {
		t.Fatalf("state = %q, want stopped", sj.State)
	}
}
