package release

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeServerPy 是「模拟生产二进制」：一个最小 HTTP 服务，把自己所在**版本目录名**
// 当作 build_id 上报（`releases/<ver>/rick` → build_id=<ver>），并写 singleton pid
// 文件。这样测试就能真实地验证「新构建真的在跑」这条断言（而不是 mock 掉它）。
const fakeServerPy = `#!/usr/bin/env python3
import http.server, json, os, pathlib, socketserver, sys

args = sys.argv[1:]

def opt(name, default=None):
    if name in args:
        i = args.index(name)
        if i + 1 < len(args):
            return args[i + 1]
    return default

if "--help" in args:
    print("fake rick web (test double)")
    sys.exit(0)

state = opt("--state-dir") or os.path.join(os.environ.get("HOME", "."), ".rick")
port = int(opt("--port") or "0")
# realpath 会跟随 bin/rick -> releases/current -> releases/<ver>/rick，取到版本号
exe = os.path.realpath(__file__)
version = pathlib.Path(os.path.dirname(exe)).name
os.makedirs(state, exist_ok=True)
with open(os.path.join(state, "web.pid"), "w") as fh:
    fh.write(str(os.getpid()))

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/api/health"):
            payload = {"status": "ok", "build_id": version}
        else:
            payload = {"build_id": version}
        body = json.dumps(payload).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer(("127.0.0.1", port), Handler) as srv:
    srv.serve_forever()
`

// newFakeProd 搭一个完整的「模拟生产」：生产仓库 + 生产 HOME + 部署脚本 +
// dev 工作树（含 web/dist）。**全部在 t.TempDir 内**，绝不接触真实 ~/.rick。
func newFakeProd(t *testing.T) Plan {
	t.Helper()
	root := t.TempDir()
	plan := Plan{
		ProdRepo:     filepath.Join(root, "prod"),
		DevTree:      filepath.Join(root, "dev"),
		ProdHome:     filepath.Join(root, "prodhome"),
		Port:         freePort(t),
		Listen:       "127.0.0.1",
		Token:        "faketok",
		SkipGates:    true,
		KeepVersions: 3,
		HealthWait:   15 * time.Second,
		StopGrace:    6 * time.Second,
		BuildBinary:  fakeBuildBinary,
	}
	plan.StateDir = filepath.Join(plan.ProdHome, ".rick")
	mustMkdir(t, filepath.Join(plan.ProdRepo, "bin"))
	mustMkdir(t, filepath.Join(plan.StateDir, "web"))
	mustMkdir(t, filepath.Join(plan.DevTree, "web", "dist"))
	mustWrite(t, filepath.Join(plan.ProdRepo, "go.mod"), "module github.com/sunquan/rick\n")
	mustWrite(t, filepath.Join(plan.DevTree, "go.mod"), "module github.com/sunquan/rick\n")
	mustWrite(t, filepath.Join(plan.DevTree, "web", "dist", "index.html"), "<html>dist-v1</html>")

	plan.StartScript = filepath.Join(plan.ProdHome, "start-web.sh")
	mustWrite(t, plan.StartScript, fmt.Sprintf(`#!/bin/bash
cd %s
exec ./bin/rick web --listen 127.0.0.1 --port %d --state-dir %s --token %s
`, plan.ProdRepo, plan.Port, plan.StateDir, plan.Token))

	// 清理：无论测试怎么结束，都把自己拉起的模拟生产收掉。
	t.Cleanup(func() { plan.stopProd() })
	return plan
}

// fakeBuildBinary 把「模拟生产二进制」写到目标路径（模拟 go build 的产物）。
func fakeBuildBinary(_ Plan, dest string, _ io.Writer) error {
	return os.WriteFile(dest, []byte(fakeServerPy), 0o755)
}

// ---- 基础工具 ----

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent of %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func buildAndCheck(t *testing.T, p Plan) Release {
	t.Helper()
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	rel, err := p.Build(io.Discard)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return rel
}

func stageOf(t *testing.T, err error) string {
	t.Helper()
	var se *StageError
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !asStage(err, &se) {
		t.Fatalf("expected *StageError, got %T: %v", err, err)
	}
	return se.Stage
}

func asStage(err error, target **StageError) bool {
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

// ---- 构建 ----

func TestBuildProducesVersionedArtifacts(t *testing.T) {
	p := newFakeProd(t)
	rel := buildAndCheck(t, p)

	if !versionRe.MatchString(rel.Version) {
		t.Fatalf("版本号格式不对: %q", rel.Version)
	}
	if !fileExists(rel.Bin) || !fileExists(filepath.Join(rel.Dir, ChecksumName)) {
		t.Fatalf("产物缺失: bin=%v checksums=%v", fileExists(rel.Bin), fileExists(filepath.Join(rel.Dir, ChecksumName)))
	}
	if rel.DistFiles == 0 {
		t.Fatal("前端产物未随版本打包（dist_files=0）")
	}
	if rel.SHA256 == "" {
		t.Fatal("缺少二进制 sha256")
	}
	// 构建不得触碰生产入口与覆盖层
	if fileExists(p.ProdBinLink()) {
		t.Fatal("构建阶段不应创建生产 bin/rick")
	}
	if dirExists(p.OverlayDist()) {
		t.Fatal("构建阶段不应写入生产覆盖层")
	}
}

func TestBuildGateFailureBlocksEverything(t *testing.T) {
	p := newFakeProd(t)
	p.SkipGates = false
	p.RunGates = func(_ Plan, _ io.Writer) (GateResult, error) {
		return GateResult{Ran: true}, fmt.Errorf("go test ./... 失败")
	}
	_, err := p.Build(io.Discard)
	if got := stageOf(t, err); got != "gate" {
		t.Fatalf("失败阶段 = %q，期望 gate", got)
	}
	entries, _ := os.ReadDir(p.ReleasesDir())
	for _, e := range entries {
		if versionRe.MatchString(e.Name()) {
			t.Fatalf("门禁失败却留下了版本目录: %s", e.Name())
		}
	}
	if fileExists(p.ProdBinLink()) || fileExists(p.CurrentLink()) {
		t.Fatal("门禁失败不得改动生产入口")
	}
}

// ---- 提升 ----

func TestPromoteSwitchesChainAtomicallyAndRecordsRollbackPoint(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	cur, _ := p.CurrentVersion()
	if cur != v1.Version {
		t.Fatalf("current = %q，期望 %q", cur, v1.Version)
	}
	link, err := os.Readlink(p.ProdBinLink())
	if err != nil || link != filepath.Join(ReleasesDirName, CurrentLinkName, "rick") {
		t.Fatalf("bin/rick 未指向 releases/current/rick: %q (%v)", link, err)
	}
	if got, err := os.ReadFile(filepath.Join(p.ProdBinLink(), "..", "..", "..", "..", "bin", "rick")); err == nil && len(got) == 0 {
		t.Fatal("bin/rick 内容为空")
	}
	if !dirExists(p.OverlayDist()) {
		t.Fatal("覆盖层未投放")
	}
	if body, err := os.ReadFile(filepath.Join(p.OverlayDist(), "index.html")); err != nil || !strings.Contains(string(body), "dist-v1") {
		t.Fatalf("覆盖层内容不对: %q (%v)", body, err)
	}

	// 第二次提升：回滚点必须记录上一版，且覆盖层留 .prev
	// （版本号含秒级时间戳：同秒内连跑会撞号 → 强制推进一秒）
	time.Sleep(1100 * time.Millisecond)
	mustWrite(t, filepath.Join(p.DevTree, "web", "dist", "index.html"), "<html>dist-v2</html>")
	v2 := buildAndCheck(t, p)
	if _, err := p.Promote(v2); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	if last, _ := p.LastVersion(); last != v1.Version {
		t.Fatalf("回滚点 = %q，期望 %q", last, v1.Version)
	}
	if cur, _ := p.CurrentVersion(); cur != v2.Version {
		t.Fatalf("current = %q，期望 %q", cur, v2.Version)
	}
	prevBody, err := os.ReadFile(filepath.Join(p.OverlayPrev(), "index.html"))
	if err != nil || !strings.Contains(string(prevBody), "dist-v1") {
		t.Fatalf("覆盖层备份不对: %q (%v)", prevBody, err)
	}
}

func TestPromoteGcKeepsCurrentAndRollbackPoint(t *testing.T) {
	p := newFakeProd(t)
	p.KeepVersions = 2
	var versions []string
	for i := 0; i < 4; i++ {
		mustWrite(t, filepath.Join(p.DevTree, "web", "dist", "index.html"), fmt.Sprintf("<html>dist-%d</html>", i))
		rel := buildAndCheck(t, p)
		// 版本号含秒级时间戳：同秒内连跑会撞号，强制推进一秒
		time.Sleep(1100 * time.Millisecond)
		if _, err := p.Promote(rel); err != nil {
			t.Fatalf("Promote #%d: %v", i, err)
		}
		versions = append(versions, rel.Version)
	}
	cur, _ := p.CurrentVersion()
	last, _ := p.LastVersion()
	ent, _ := os.ReadDir(p.ReleasesDir())
	var dirs []string
	for _, e := range ent {
		if e.IsDir() && versionRe.MatchString(e.Name()) {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) > p.keep()+1 { // keep=2，但 current + last 可能各占一个 → 上限 3
		t.Fatalf("GC 后残留 %d 个版本目录: %v", len(dirs), dirs)
	}
	for _, needle := range []string{cur, last} {
		if !dirExists(p.VersionDir(needle)) {
			t.Fatalf("GC 删掉了受保护版本 %s（dirs=%v）", needle, dirs)
		}
	}
	if fileExists(p.VersionDir(versions[0])) && len(dirs) > 2 {
		t.Fatalf("最旧版本 %s 未被 GC（dirs=%v）", versions[0], dirs)
	}
}

func TestPromoteRejectsMissingVersionDir(t *testing.T) {
	p := newFakeProd(t)
	_, err := p.Promote(Release{Version: "deadbeef-202601010101", Dir: p.VersionDir("deadbeef-202601010101")})
	if got := stageOf(t, err); got != "promote" {
		t.Fatalf("失败阶段 = %q，期望 promote", got)
	}
}

// ---- 重启 ----

func TestRestartVerifiesBuildIDAndReplacesProcess(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	res1, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart v1: %v", err)
	}
	if !res1.Matches || res1.BuildID != v1.Version {
		t.Fatalf("build_id 校验失败: matches=%v build_id=%q 期望 %q", res1.Matches, res1.BuildID, v1.Version)
	}
	if res1.PID == 0 || !alive(res1.PID) {
		t.Fatalf("重启后进程不在: pid=%d", res1.PID)
	}

	// 第二次提升 + 重启：老进程必须被换掉
	time.Sleep(1100 * time.Millisecond)
	mustWrite(t, filepath.Join(p.DevTree, "web", "dist", "index.html"), "<html>dist-v2</html>")
	v2 := buildAndCheck(t, p)
	if _, err := p.Promote(v2); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	res2, err := p.Restart(v2.Version)
	if err != nil {
		t.Fatalf("Restart v2: %v", err)
	}
	if res2.BuildID != v2.Version {
		t.Fatalf("重启后 build_id = %q，期望 %q", res2.BuildID, v2.Version)
	}
	if res2.PID == res1.PID {
		t.Fatalf("进程没被换掉（pid 仍是 %d）", res2.PID)
	}
	if alive(res1.PID) {
		t.Fatalf("旧生产进程 %d 仍在运行", res1.PID)
	}
	if len(res2.Stopped) == 0 {
		t.Fatal("未记录被停掉的旧进程")
	}
}

func TestRestartFailsWhenRunningBuildIsStale(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart v1: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	v2 := buildAndCheck(t, p)

	// 部署脚本被「钉死」在旧版本上：重启后跑的仍是 v1 → 必须判定失败
	mustWrite(t, p.StartScript, fmt.Sprintf(`#!/bin/bash
cd %s
exec %s web --listen 127.0.0.1 --port %d --state-dir %s --token %s
`, p.ProdRepo, filepath.Join(p.VersionDir(v1.Version), "rick"), p.Port, p.StateDir, p.Token))
	if _, err := p.Promote(v2); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	_, err := p.Restart(v2.Version)
	if got := stageOf(t, err); got != "health" {
		t.Fatalf("失败阶段 = %q，期望 health", got)
	}
	if !strings.Contains(err.Error(), v1.Version) {
		t.Fatalf("错误信息未指出实际运行的旧版本: %v", err)
	}
}

func TestRestartRefusesToKillForeignProcess(t *testing.T) {
	p := newFakeProd(t)
	// 一个「不属于生产」的旁观进程
	foreign := exec.Command("sleep", "300")
	foreign.Dir = t.TempDir()
	foreign.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
	if err := foreign.Start(); err != nil {
		t.Fatalf("start foreign: %v", err)
	}
	defer func() { _ = foreign.Process.Kill() }()
	// pid 文件被这个旁观进程的 pid 占着（模拟陈旧/串号）
	mustWrite(t, p.ProdPidPath(), fmt.Sprintf("%d\n", foreign.Process.Pid))

	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if !res.Matches {
		t.Fatalf("重启未生效: %+v", res)
	}
	if len(res.Skipped) == 0 || res.Skipped[0] != foreign.Process.Pid {
		t.Fatalf("未把旁观进程记入 skipped: %v", res.Skipped)
	}
	if !alive(foreign.Process.Pid) {
		t.Fatal("❌ 旁观进程被误杀——所有权断言失效")
	}
}

func TestRestartToleratesMissingRecoveryReport(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if res.Recovery.Suspended != 0 || len(res.Recovery.IDs) != 0 {
		t.Fatalf("无报告时不该有挂起项: %+v", res.Recovery)
	}
}

func TestRestartSurfacesRecoveryReport(t *testing.T) {
	p := newFakeProd(t)
	report := map[string]any{
		"version": 1,
		"at":      "2026-01-01T00:00:00+08:00",
		"suspended": []map[string]any{
			{"id": "s1", "type": "easy", "title": "job_5"},
			{"id": "s2", "type": "doing", "job_id": "job_9"},
		},
		"recovered": []map[string]any{},
		"failed":    []map[string]any{},
	}
	body, _ := json.Marshal(report)
	mustWrite(t, filepath.Join(p.StateDirPath(), RecoveryReportFile), string(body))

	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if res.Recovery.Suspended != 2 {
		t.Fatalf("挂起数 = %d，期望 2", res.Recovery.Suspended)
	}
	if len(res.Recovery.IDs) != 2 || !strings.Contains(res.Recovery.IDs[0], "s1") {
		t.Fatalf("挂起 id 未解析: %v", res.Recovery.IDs)
	}
	if res.Recovery.Path != filepath.Join(p.StateDirPath(), RecoveryReportFile) {
		t.Fatalf("报告路径不对: %s", res.Recovery.Path)
	}
}

// ---- 回滚 ----

func TestRollbackRestoresPreviousVersion(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart v1: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	mustWrite(t, filepath.Join(p.DevTree, "web", "dist", "index.html"), "<html>dist-v2</html>")
	v2 := buildAndCheck(t, p)
	if _, err := p.Promote(v2); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	if _, err := p.Restart(v2.Version); err != nil {
		t.Fatalf("Restart v2: %v", err)
	}

	res, err := p.Rollback()
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if res.Version != v1.Version {
		t.Fatalf("回滚目标 = %q，期望 %q", res.Version, v1.Version)
	}
	restart, err := p.Restart(res.Version)
	if err != nil {
		t.Fatalf("Rollback 后 Restart: %v", err)
	}
	if restart.BuildID != v1.Version {
		t.Fatalf("回滚后运行的 build_id = %q，期望 %q", restart.BuildID, v1.Version)
	}
	body, err := os.ReadFile(filepath.Join(p.OverlayDist(), "index.html"))
	if err != nil || !strings.Contains(string(body), "dist-v1") {
		t.Fatalf("覆盖层未随回滚还原: %q (%v)", body, err)
	}
}

func TestRollbackWithoutRollbackPointFails(t *testing.T) {
	p := newFakeProd(t)
	_, err := p.Rollback()
	if got := stageOf(t, err); got != "promote" {
		t.Fatalf("失败阶段 = %q，期望 promote", got)
	}
}

// ---- dry-run / status / 校验 ----

func TestDryRunNeverTouchesProd(t *testing.T) {
	p := newFakeProd(t)
	res, err := p.DryRun(io.Discard)
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if res.Version == "" || res.Bin == "" {
		t.Fatalf("dry-run 未给出产物信息: %+v", res)
	}
	if fileExists(p.CurrentLink()) || fileExists(p.ProdBinLink()) || dirExists(p.OverlayDist()) {
		t.Fatal("dry-run 改动了生产（链/入口/覆盖层）")
	}
	if pids := findProdPIDs(p); len(pids) != 0 {
		t.Fatalf("dry-run 启动了生产进程: %v", pids)
	}
}

func TestStatusReportsVersionsAndRecovery(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	st := p.Status()
	if st.CurrentVersion != v1.Version || !st.Healthy || !st.Matches {
		t.Fatalf("status 不对: %+v", st)
	}
	if st.PID == 0 || !alive(st.PID) {
		t.Fatalf("status 未报出运行进程: %+v", st)
	}
}

func TestValidateRejectsDevTreeEqualToProd(t *testing.T) {
	p := newFakeProd(t)
	p.DevTree = p.ProdRepo
	if err := p.Validate(); err == nil {
		t.Fatal("dev 树 == 生产树 时必须拒绝")
	}
	// 缺 go.mod 也必须拒绝
	p2 := newFakeProd(t)
	_ = os.Remove(filepath.Join(p2.DevTree, "go.mod"))
	if err := p2.Validate(); err == nil {
		t.Fatal("dev 树不完整时必须拒绝")
	}
}

func TestDefaultPlanReadsPortFromDeploymentScript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(EnvProdPort, "")
	repo := filepath.Join(t.TempDir(), "prod")
	mustMkdir(t, repo)
	mustWrite(t, filepath.Join(repo, "go.mod"), "module github.com/sunquan/rick\n")
	stateDir := filepath.Join(home, ".rick")
	mustMkdir(t, stateDir)
	mustWrite(t, filepath.Join(stateDir, "start-web.sh"), "cd "+repo+"\nexec ./bin/rick web --listen 0.0.0.0 --port 19999 --state-dir "+stateDir+"\n")
	mustWrite(t, filepath.Join(stateDir, "config.json"), `{"web_token":"tok-from-config"}`)

	// realHomeDir 优先 /etc/passwd（真实家目录），因此这里直接验证「脚本解析」本身
	body, _ := os.ReadFile(filepath.Join(stateDir, "start-web.sh"))
	if got := parseScriptPort(string(body)); got != 19999 {
		t.Fatalf("parseScriptPort = %d，期望 19999", got)
	}
	if got := parseScriptFlag(string(body), "state-dir"); got != stateDir {
		t.Fatalf("parseScriptFlag(state-dir) = %q，期望 %q", got, stateDir)
	}
	if got := parseScriptFlag(string(body), "listen"); got != "0.0.0.0" {
		t.Fatalf("parseScriptFlag(listen) = %q", got)
	}
	// DefaultPlan 的端口来自**真实家目录**下的部署脚本（realHomeDir 走 /etc/passwd，
	// 不跟随 $HOME）——因此断言「等于从该脚本解析出来的端口」，而不是硬编码值：
	// 这正是「不写死端口」的证据。
	realScript := filepath.Join(realHomeOf(t), ".rick", "start-web.sh")
	if body, err := os.ReadFile(realScript); err == nil {
		want := parseScriptPort(string(body))
		p, err := DefaultPlan(repo, filepath.Join(t.TempDir(), "dev"), 0)
		if err != nil {
			t.Fatalf("DefaultPlan: %v", err)
		}
		if p.Port != want {
			t.Fatalf("DefaultPlan 端口 = %d，期望等于部署脚本里的 %d", p.Port, want)
		}
		if p.ProdRepo != repo {
			t.Fatalf("DefaultPlan prodRepo = %q", p.ProdRepo)
		}
		if p.StartScript != realScript {
			t.Fatalf("DefaultPlan 未采用部署脚本: %q", p.StartScript)
		}
	} else {
		t.Fatalf("部署脚本不可读: %v", err)
	}
}

// realHomeOf 复刻 realHomeDir 的解析（测试内独立实现，避免与实现共享同一 bug）。
func realHomeOf(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("/etc/passwd")
	if err == nil {
		uid := fmt.Sprintf("%d", os.Getuid())
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Split(line, ":")
			if len(parts) >= 6 && parts[2] == uid {
				return parts[5]
			}
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	t.Fatal("cannot determine real home")
	return ""
}

func TestDefaultPlanFailsWithoutAnyPortSource(t *testing.T) {
	t.Setenv(EnvProdPort, "")
	repo := filepath.Join(t.TempDir(), "prod")
	mustMkdir(t, repo)
	mustWrite(t, filepath.Join(repo, "go.mod"), "module github.com/sunquan/rick\n")
	// 生产 HOME 的 start-web.sh 不含 --port（用真实 HOME 时该文件存在且含端口，
	// 所以这里只能验证「没有来源时 parseScriptPort=0」这一事实 + 环境变量优先级）。
	if got := parseScriptPort("exec ./bin/rick web"); got != 0 {
		t.Fatalf("parseScriptPort = %d，期望 0", got)
	}
	t.Setenv(EnvProdPort, "17777")
	p, err := DefaultPlan(repo, filepath.Join(t.TempDir(), "dev"), 0)
	if err != nil {
		t.Fatalf("DefaultPlan: %v", err)
	}
	if p.Port != 17777 {
		t.Fatalf("环境变量端口未生效: %d", p.Port)
	}
	// 显式参数优先级最高
	p2, err := DefaultPlan(repo, filepath.Join(t.TempDir(), "dev"), 18888)
	if err != nil {
		t.Fatalf("DefaultPlan: %v", err)
	}
	if p2.Port != 18888 {
		t.Fatalf("显式端口未生效: %d", p2.Port)
	}
}

func TestHostedByProdDetectsForeignCaller(t *testing.T) {
	p := newFakeProd(t)
	if hosted, why := HostedByProd(p); hosted {
		t.Fatalf("测试进程不该被判定为被模拟生产托管（%s）", why)
	}
}

func TestWriteAndReadState(t *testing.T) {
	p := newFakeProd(t)
	if err := p.WriteState(ReleaseState{Action: "promote", Version: "v", Stage: "done", OK: true}); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	st, err := p.ReadState()
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if !st.OK || st.Action != "promote" || st.At == "" {
		t.Fatalf("留痕内容不对: %+v", st)
	}
}

func TestWriteStateMissingFileIsNotAnError(t *testing.T) {
	p := newFakeProd(t)
	st, err := p.ReadState()
	if err != nil || st.Action != "" {
		t.Fatalf("缺失留痕应静默: %+v (%v)", st, err)
	}
}

// ---- 完整流程与自动回滚 ----

// TestReleaseAutoRollsBackWhenHealthFails 验证「重启失败自动回滚」这条硬要求：
// 新版本起来后 build_id 不匹配（部署脚本被钉死在旧版）→ Release 自动切回上一版
// 并重启 → 生产继续可用（旧版本），原始失败原因照实返回（阶段 health）。
func TestReleaseAutoRollsBackWhenHealthFails(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart v1: %v", err)
	}

	// 把部署脚本钉死在 v1（模拟「新版本起不来 / 新构建没生效」）
	mustWrite(t, p.StartScript, fmt.Sprintf(`#!/bin/bash
cd %s
exec %s web --listen 127.0.0.1 --port %d --state-dir %s --token %s
`, p.ProdRepo, filepath.Join(p.VersionDir(v1.Version), "rick"), p.Port, p.StateDir, p.Token))

	time.Sleep(1100 * time.Millisecond) // 版本号含秒级时间戳，避免同秒撞号
	var buf strings.Builder
	res, err := p.Release(&buf)
	if err == nil {
		t.Fatal("部署脚本钉死旧版时，提升必须判定失败")
	}
	if got := stageOf(t, err); got != "health" {
		t.Fatalf("失败阶段 = %q，期望 health", got)
	}
	if res.Action != "release-rolled-back" {
		t.Fatalf("未标记自动回滚: action=%q", res.Action)
	}
	if res.PrevVersion != v1.Version {
		t.Fatalf("回滚目标 = %q，期望 %q", res.PrevVersion, v1.Version)
	}
	if !strings.Contains(buf.String(), "RELEASE_AUTOROLLBACK") {
		t.Fatalf("回执缺少自动回滚标记:\n%s", buf.String())
	}
	if cur, _ := p.CurrentVersion(); cur != v1.Version {
		t.Fatalf("自动回滚后 current = %q，期望 %q", cur, v1.Version)
	}
	st := p.Status()
	if !st.Healthy || st.RunningBuildID != v1.Version {
		t.Fatalf("自动回滚后生产不可用: healthy=%v build_id=%q", st.Healthy, st.RunningBuildID)
	}
}

// TestReleaseHappyPath 覆盖正常路径：Release = Build + Promote + Restart 全绿。
func TestReleaseHappyPath(t *testing.T) {
	p := newFakeProd(t)
	var buf strings.Builder
	res, err := p.Release(&buf)
	if err != nil {
		t.Fatalf("Release: %v\n%s", err, buf.String())
	}
	if res.Restart == nil || !res.Restart.Matches {
		t.Fatalf("未能确认新构建在跑: %+v", res.Restart)
	}
	if cur, _ := p.CurrentVersion(); cur != res.Version {
		t.Fatalf("current = %q，期望 %q", cur, res.Version)
	}
	if !strings.Contains(buf.String(), "RELEASE_BUILD") {
		t.Fatalf("回执缺少构建行:\n%s", buf.String())
	}
}

func TestExitCodeForStage(t *testing.T) {
	cases := map[string]int{"gate": ExitGate, "build": ExitBuild, "promote": ExitRestart,
		"restart": ExitRestart, "health": ExitHealth, "unknown": ExitRestart}
	for stage, want := range cases {
		if got := ExitCodeForStage(stage); got != want {
			t.Errorf("ExitCodeForStage(%q) = %d，期望 %d", stage, got, want)
		}
	}
}
