package release

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/env/procfs"
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

# P0-1/P0-2（job_49）：release 链路现在用 --version 做可执行性冒烟
#（verifyRunnable）——fake 必须能应答，否则所有 promote/rollback 用例会被
# 新校验误伤。
if "--version" in args:
    print("fake rick (test double) version=fake")
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
	// 真实提升（未设 StagingDir）必须落到发布目录 <prodRepo>/bin/releases/<ver>/
	if rel.Dir != p.VersionDir(rel.Version) {
		t.Fatalf("真实提升构建落点应为发布目录 %s，实际 %s", p.VersionDir(rel.Version), rel.Dir)
	}
	if rel.Staged {
		t.Fatal("非 dry-run 构建不应标记 Staged")
	}
	if !strings.HasPrefix(rel.Bin, p.ReleasesDir()) {
		t.Fatalf("产物必须落在 <prodRepo>/bin/releases/ 下: %s", rel.Bin)
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

// TestRestartRefusesToKillCwdSquatterInProdTree job_49 发现的误杀防线：
// 生产树根 = git 仓库根，开发会话/human 终端/agent 都在那里工作（cwd 匹配
// 「start-web.sh cd 进 prodRepo」的断言本意，但断言过宽）。cwd 在生产树的
// 非 rick 进程（sleep/开发 shell）绝不能被 stopProd 认领——否则 release 一跑
// 就把承载它的 agent 会话全杀了。
func TestRestartRefusesToKillCwdSquatterInProdTree(t *testing.T) {
	p := newFakeProd(t)
	// 旁观进程：cwd 就在生产树根（比 ForeignProcess 更危险的形态）
	foreign := exec.Command("sleep", "300")
	foreign.Dir = p.ProdRepo
	foreign.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
	if err := foreign.Start(); err != nil {
		t.Fatalf("start foreign: %v", err)
	}
	defer func() { _ = foreign.Process.Kill() }()

	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v\nskipped_why=%v", err, res.SkippedWhy)
	}
	if !res.Matches {
		t.Fatalf("重启未生效: %+v", res)
	}
	if !alive(foreign.Process.Pid) {
		t.Fatal("❌ 生产树 cwd 的非 rick 进程被误杀——cwd 断言过宽（会杀开发会话）")
	}
	for _, pid := range res.Stopped {
		if pid == foreign.Process.Pid {
			t.Fatal("❌ 旁观进程进了 stopped 清单")
		}
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
	var out strings.Builder
	res, err := p.DryRun(&out)
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

	// F3：dry-run 的**构建产物**也不得落进生产树。
	// 旧实现复用 Build()，会写 <ProdRepo>/bin/releases/<version>/{rick,dist}
	//（实测残留 bin/releases/1091f62-260921201517/）。
	if got := listFilesUnder(p.ReleasesDir()); len(got) != 0 {
		t.Fatalf("dry-run 在生产树写了构建产物（应零写入）: %v", got)
	}
	// 产物必须落在临时暂存目录里，且该目录在 dry-run 结束时被清理
	if res.StagingDir == "" || res.TargetBin == "" || res.TargetDist == "" {
		t.Fatalf("dry-run 未报告暂存/目标路径: %+v", res)
	}
	if dirExists(res.StagingDir) {
		t.Fatalf("dry-run 未清理暂存目录: %s", res.StagingDir)
	}
	if !strings.Contains(res.StagingDir, os.TempDir()) {
		t.Fatalf("暂存目录应位于系统临时目录下，实际 %s", res.StagingDir)
	}
	// 目标路径必须预告「正式提升本会落到哪」（相对生产树）
	wantBin := filepath.Join(p.ProdRepo, "bin", ReleasesDirName, res.Version, "rick")
	if res.TargetBin != wantBin {
		t.Fatalf("target_bin = %s, want %s", res.TargetBin, wantBin)
	}
	body := out.String()
	for _, want := range []string{"RELEASE_DRYRUN", "prod_untouched=true", "cleanup=ok", wantBin} {
		if !strings.Contains(body, want) {
			t.Fatalf("dry-run 输出缺少 %q:\n%s", want, body)
		}
	}
}

// TestPromoteRejectsStagedRelease 纵深防御：暂存（dry-run）产物即使被手工塞进
// Promote 也必须被拒绝 —— 否则「暂存目录 → 生产链」的路径就绕过了 DryRun 的承诺。
func TestPromoteRejectsStagedRelease(t *testing.T) {
	p := newFakeProd(t)
	staged := p
	staged.StagingDir = t.TempDir()
	rel, err := staged.Build(io.Discard)
	if err != nil {
		t.Fatalf("staged Build: %v", err)
	}
	if !rel.Staged {
		t.Fatal("StagingDir 生效时 rel.Staged 应为 true")
	}
	if _, err := p.Promote(rel); err == nil {
		t.Fatal("Promote 竟然接受了暂存产物")
	} else if stageOf(t, err) != "promote" {
		t.Fatalf("失败阶段 = %s, want promote", stageOf(t, err))
	}
	if fileExists(p.CurrentLink()) {
		t.Fatal("被拒绝的提升不该留下 current 链")
	}
}

// listFilesUnder 递归列出目录下所有条目（相对路径）；目录不存在返回 nil。
func listFilesUnder(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		out = append(out, rel)
		return nil
	})
	return out
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
	// 与生产 RealHomeDir 同序：user.LookupId（macOS 走 DirectoryService、Linux 走
	// /etc/passwd）→ 环境回退。原实现只读 /etc/passwd：macOS 本地用户不在该文件
	// → 回退到测试被覆盖的 $HOME（tmpdir）→ 与生产解析不一致（job_47 跨机后暴露）。
	if u, err := user.LookupId(fmt.Sprintf("%d", os.Getuid())); err == nil && u.HomeDir != "" {
		return u.HomeDir
	}
	if data, err := os.ReadFile("/etc/passwd"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Split(line, ":")
			if len(parts) >= 6 && parts[2] == fmt.Sprintf("%d", os.Getuid()) {
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

// ─── P0 修复用例（job_49）──────────────────────────────────────────────────

// fakeELF 写一份「本机不可执行」的二进制（ELF 头；linux 主机上则写 Mach-O 头，
// 两平台都能构造出 wrongPlatformMagic 命中的产物）。
func fakeELF(t *testing.T, path string) {
	t.Helper()
	head := "\x7fELF\x02\x01\x01\x00" + strings.Repeat("0", 64) // ELF 64-bit LSB
	if runtime.GOOS == "linux" {
		// linux 上 ELF 是本机格式 → 用 Mach-O 头构造「外平台产物」
		head = "\xfe\xed\xfa\xcf" + strings.Repeat("0", 64)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent of %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(head), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestPromoteRejectsNonExecutableTarget P0-1：版本目录里躺着外平台 ELF 时，
// Promote 必须在切链前拒绝（stage=build），current 不动、零进程被杀。
// （11:53 事故的自动回滚切到 eb8048c ELF → 生产彻底起不来——R1/R3 链头。）
func TestPromoteRejectsNonExecutableTarget(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}

	// 构造坏版本目录：releases/<badver>/rick 是 ELF
	bad := "badbad0-261030120000"
	fakeELF(t, filepath.Join(p.VersionDir(bad), "rick"))
	mustWrite(t, filepath.Join(p.VersionDir(bad), "dist", "index.html"), "<html>bad</html>")

	_, err := p.Promote(Release{Version: bad, Dir: p.VersionDir(bad), Bin: filepath.Join(p.VersionDir(bad), "rick")})
	if err == nil {
		t.Fatal("Promote 竟然接受了不可执行的版本目录")
	}
	if got := stageOf(t, err); got != "build" {
		t.Fatalf("失败阶段 = %q，期望 build（产物质量缺陷）", got)
	}
	if cur, _ := p.CurrentVersion(); cur != v1.Version {
		t.Fatalf("current 被动过: %q，期望保持 %q", cur, v1.Version)
	}
	if fileExists(p.LastRecord()) {
		t.Fatal("被拒绝的提升不该写回滚点")
	}
}

// TestRollbackRejectsNonExecutableTarget P0-2：.last 指向 ELF 版本时 Rollback
// 必须拒绝切链（点名「回滚点不可执行」），current 不变、服务继续在跑。
func TestRollbackRejectsNonExecutableTarget(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart v1: %v", err)
	}

	// 坏回滚点：.last 指向 ELF 版本目录
	bad := "badbad0-261030120001"
	fakeELF(t, filepath.Join(p.VersionDir(bad), "rick"))
	mustWrite(t, p.LastRecord(), bad+"\n")

	_, err := p.Rollback()
	if err == nil {
		t.Fatal("Rollback 竟然切到了不可执行的回滚点")
	}
	if got := stageOf(t, err); got != "promote" {
		t.Fatalf("失败阶段 = %q，期望 promote", got)
	}
	if !strings.Contains(err.Error(), "回滚点不可执行") {
		t.Fatalf("错误未点名回滚点不可执行: %v", err)
	}
	if cur, _ := p.CurrentVersion(); cur != v1.Version {
		t.Fatalf("current 被切走: %q，期望保持 %q", cur, v1.Version)
	}
	// 服务未受影响：旧实例仍在服务
	st := p.Status()
	if !st.Healthy || st.RunningBuildID != v1.Version {
		t.Fatalf("拒绝回滚后生产不可用: %+v", st)
	}
}

// TestReleaseAutorollbackRefusesBadRollbackTarget P0-2 自动回滚路径：
// 新版重启失败 + .last 是坏 ELF → forceSwitch 拒绝切链，把真实原因
// （回滚点不可执行）返回而不是默默切到另一份坏产物。
func TestReleaseAutorollbackRefusesBadRollbackTarget(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}
	if _, err := p.Restart(v1.Version); err != nil {
		t.Fatalf("Restart v1: %v", err)
	}

	// 11:53 事故精确复现：PrevVersion（自动回滚目标）本身是坏产物（EBF）。
	// 运行中的 v1 进程不受磁盘文件替换影响，但下一轮 release 的 startProd
	// 与 autorollback 都会碰到坏二进制 → 回滚必须拒绝切链并点名，而不是把
	// current 切到不可执行版本（R3 根因：回滚把「发布失败」升级成「起不来」）。
	fakeELF(t, filepath.Join(p.VersionDir(v1.Version), "rick"))

	// 部署脚本钉死 v1（模拟新版起不来）→ Release 失败 → 自动回滚 → 拒绝坏点
	mustWrite(t, p.StartScript, fmt.Sprintf(`#!/bin/bash
cd %s
exec %s web --listen 127.0.0.1 --port %d --state-dir %s --token %s
`, p.ProdRepo, filepath.Join(p.VersionDir(v1.Version), "rick"), p.Port, p.StateDir, p.Token))

	time.Sleep(1100 * time.Millisecond) // 版本号防同秒撞号
	var buf strings.Builder
	res, err := p.Release(&buf)
	if err == nil {
		t.Fatal("钉死旧版 + 坏回滚点，Release 必须失败")
	}
	if !strings.Contains(err.Error(), "回滚点不可执行") {
		t.Fatalf("自动回滚失败未点名坏回滚点: %v", err)
	}
	if !strings.Contains(buf.String(), "RELEASE_AUTOROLLBACK") {
		t.Fatalf("缺少自动回滚标记:\n%s", buf.String())
	}
	if cur, _ := p.CurrentVersion(); cur != res.Version {
		t.Fatalf("自动回滚被拒后 current 应保持新版本 %q，实际 %q", res.Version, cur)
	}
}

// TestStopProdClaimsBareBuildHoldingLock P0-3 锁持有者认领（本案 11135 场景）：
// 裸构建的 rick web（cwd/exe 都不在生产形态）占着生产单例锁 → 必须被认领停止，
// skipped 为空——否则新进程被锁挡住、health 读到旧实例、报「新构建没生效」。
func TestStopProdClaimsBareBuildHoldingLock(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}

	// 裸构建实例：exe 在生产树之外（baredir/rick），cwd 也在外面；
	// 用 prod 的 state-dir（会写 web.pid）+ 自己的端口——正是 11135 形态。
	bareDir := t.TempDir()
	bareBin := filepath.Join(bareDir, "rick")
	mustWrite(t, bareBin, fakeServerPy)
	if err := os.Chmod(bareBin, 0o755); err != nil {
		t.Fatalf("chmod bare: %v", err)
	}
	bare := exec.Command(bareBin, "web", "--listen", "127.0.0.1", "--port", strconv.Itoa(freePort(t)), "--state-dir", p.StateDirPath())
	bare.Dir = bareDir // cwd 不在生产树
	bare.Env = []string{"HOME=" + bareDir, "PATH=" + os.Getenv("PATH")}
	if err := bare.Start(); err != nil {
		t.Fatalf("start bare: %v", err)
	}
	defer func() { _ = bare.Process.Kill() }()
	waitFor := func(cond string) {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if data, err := os.ReadFile(p.ProdPidPath()); err == nil {
				if pid, _ := strconv.Atoi(strings.TrimSpace(string(data))); pid == bare.Process.Pid {
					return
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("裸实例未就绪（web.pid 未变成其 pid）: %s", cond)
	}
	waitFor("web.pid")

	// 模拟它持有生产单例锁（内核 flock 的持有者诊断记录）
	mustWrite(t, p.ProdLockPath(), fmt.Sprintf("%d\n", bare.Process.Pid))

	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v\nskipped_why=%v", err, res.SkippedWhy)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("裸实例仍被拒杀: %v（why=%v）", res.Skipped, res.SkippedWhy)
	}
	found := false
	for _, pid := range res.Stopped {
		if pid == bare.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("裸实例未被认领停止: stopped=%v skipped=%v why=%v", res.Stopped, res.Skipped, res.SkippedWhy)
	}
	if alive(bare.Process.Pid) {
		t.Fatal("裸实例仍存活（锁持有者认领失败）")
	}
	if !res.Matches {
		t.Fatalf("重启后 build_id 不匹配: %+v", res)
	}
}

// stubProcInfo 覆写个别查询（cwd 失败 / exe 修正），其余委托真实实现
// （Alive 必须⊥真实：terminate 的收尸轮询依赖它）。
type stubProcInfo struct {
	inner  procfs.Info
	cwdErr map[int]bool // 这些 pid 的 Cwd 返回错误（模拟 lsof 受限）
	exeFix map[int]string
}

func (s stubProcInfo) Alive(pid int) bool        { return s.inner.Alive(pid) }
func (s stubProcInfo) PPID(pid int) (int, error) { return s.inner.PPID(pid) }
func (s stubProcInfo) Exe(pid int) (string, error) {
	if p, ok := s.exeFix[pid]; ok {
		return p, nil
	}
	return s.inner.Exe(pid)
}
func (s stubProcInfo) ExeHashPath(pid int) (string, error) { return s.inner.ExeHashPath(pid) }
func (s stubProcInfo) Cwd(pid int) (string, error) {
	if s.cwdErr[pid] {
		return "", fmt.Errorf("lsof: permission denied (simulated)")
	}
	return s.inner.Cwd(pid)
}
func (s stubProcInfo) Command(pid int) (string, error) {
	return s.inner.Command(pid)
}
func (s stubProcInfo) Environ(pid int) (map[string]string, error)     { return s.inner.Environ(pid) }
func (s stubProcInfo) Enumerate() ([]int, error)                      { return s.inner.Enumerate() }
func (s stubProcInfo) Snapshot() (map[int]procfs.ProcSnapshot, error) { return s.inner.Snapshot() }

// withStubProc 注入 stub 并在测试结束时还原。
func withStubProc(t *testing.T, s stubProcInfo) {
	t.Helper()
	old := proc
	proc = s
	t.Cleanup(func() { proc = old })
}

// TestStopProdFallsBackToExeWhenCwdUnreadable P0-3 场景①：cwd 取不到
// （lsof 受限）时所有权断言应回退 exe 前缀断言认领进程，而不是静默判 false。
func TestStopProdFallsBackToExeWhenCwdUnreadable(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}

	// 旁观进程（pid 文件指向它）：cwd 不可读，但 exe 在 prod/bin 下
	foreign := exec.Command("sleep", "300")
	foreign.Dir = t.TempDir()
	foreign.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
	if err := foreign.Start(); err != nil {
		t.Fatalf("start foreign: %v", err)
	}
	defer func() { _ = foreign.Process.Kill() }()
	mustWrite(t, p.ProdPidPath(), fmt.Sprintf("%d\n", foreign.Process.Pid))

	withStubProc(t, stubProcInfo{
		inner:  procfs.New(),
		cwdErr: map[int]bool{foreign.Process.Pid: true},
		exeFix: map[int]string{foreign.Process.Pid: p.ProdBinLink()},
	})

	stopped, skipped, why := p.stopProd()
	if len(skipped) != 0 {
		t.Fatalf("cwd 不可读时不应拒杀（应回退 exe 断言）: skipped=%v why=%v", skipped, why)
	}
	found := false
	for _, pid := range stopped {
		if pid == foreign.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("exe 断言未认领进程: stopped=%v why=%v", stopped, why)
	}
	if alive(foreign.Process.Pid) {
		t.Fatal("被认领的进程未被停止")
	}
}

// TestStopProdClaimsPortSquatterRickProcess P0-3 场景②端口认领：裸 rick 实例
// 监听生产端口（未持锁、cwd/exe 均不在生产形态）→ 认领停止；否则新进程
// 永远绑不上端口。
func TestStopProdClaimsPortSquatterRickProcess(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}

	// 裸 rick 直接监听生产端口，state-dir 用自己的（不写 prod 的 web.pid）
	bareDir := t.TempDir()
	bareBin := filepath.Join(bareDir, "rick")
	mustWrite(t, bareBin, fakeServerPy)
	if err := os.Chmod(bareBin, 0o755); err != nil {
		t.Fatalf("chmod bare: %v", err)
	}
	bareState := filepath.Join(bareDir, "state")
	bare := exec.Command(bareBin, "web", "--listen", "127.0.0.1", "--port", strconv.Itoa(p.Port), "--state-dir", bareState)
	bare.Dir = bareDir
	bare.Env = []string{"HOME=" + bareDir, "PATH=" + os.Getenv("PATH")}
	if err := bare.Start(); err != nil {
		t.Fatalf("start bare: %v", err)
	}
	defer func() { _ = bare.Process.Kill() }()

	// 等它真的监听上端口（端口认靠 lsof 必须能看到 LISTEN）
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if portOwner(p.Port) == bare.Process.Pid {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got := portOwner(p.Port); got != bare.Process.Pid {
		t.Fatalf("裸实例未监听生产端口（lsof=%d want=%d）", got, bare.Process.Pid)
	}

	res, err := p.Restart(v1.Version)
	if err != nil {
		t.Fatalf("Restart: %v\nskipped_why=%v", err, res.SkippedWhy)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("端口占用的裸 rick 仍被拒杀: %v（why=%v）", res.Skipped, res.SkippedWhy)
	}
	found := false
	for _, pid := range res.Stopped {
		if pid == bare.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("端口占用者未被认领: stopped=%v skipped=%v why=%v", res.Stopped, res.Skipped, res.SkippedWhy)
	}
	if alive(bare.Process.Pid) {
		t.Fatal("端口占用的裸 rick 仍存活")
	}
	if !res.Matches {
		t.Fatalf("重启后 build_id 不匹配: %+v", res)
	}
}

// TestStopProdNeverClaimsForeignPortSquatter P0-3 场景③不误杀：非 rick 进程
// 占用生产端口 → 宁可不杀（它根本不进 stopped/skipped——不是 candidate）；
// 失败显式报「端口被 pid=X 占用」，绝不误杀 python/其他服务。
func TestStopProdNeverClaimsForeignPortSquatter(t *testing.T) {
	p := newFakeProd(t)
	v1 := buildAndCheck(t, p)
	if _, err := p.Promote(v1); err != nil {
		t.Fatalf("Promote v1: %v", err)
	}

	// 非 rick 的端口占用者：python http.server（cwd/exe 与 rick 无关）
	squatter := exec.Command("python3", "-m", "http.server", strconv.Itoa(p.Port), "--bind", "127.0.0.1")
	squatter.Dir = t.TempDir()
	squatter.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
	if err := squatter.Start(); err != nil {
		t.Fatalf("start squatter: %v", err)
	}
	defer func() { _ = squatter.Process.Kill() }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if portOwner(p.Port) == squatter.Process.Pid {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got := portOwner(p.Port); got != squatter.Process.Pid {
		t.Fatalf("squatter 未监听（lsof=%d want=%d）", got, squatter.Process.Pid)
	}

	_, err := p.Restart(v1.Version)
	if err == nil {
		t.Fatal("端口被非 rick 进程占着，Restart 必须失败")
	}
	if !alive(squatter.Process.Pid) {
		t.Fatal("❌ 非 rick 进程被误杀——认领断言越界")
	}
	// P0-4：错误必须点名端口占用者 pid
	if !strings.Contains(err.Error(), fmt.Sprintf("pid=%d", squatter.Process.Pid)) {
		t.Fatalf("错误未点名端口占用者: %v", err)
	}
}

// TestRestartMismatchNamesOldInstance P0-4：build_id 不一致时，错误必须含
// old_pid 与 old_build_id（旧实例是誰一目了然——本案旧文案「新构建没生效？」
// 在 11135 漏杀场景下是误导）。
func TestRestartMismatchNamesOldInstance(t *testing.T) {
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

	// 部署脚本钉死 v1：重启后仍是 v1 实例 → mismatch
	mustWrite(t, p.StartScript, fmt.Sprintf(`#!/bin/bash
cd %s
exec %s web --listen 127.0.0.1 --port %d --state-dir %s --token %s
`, p.ProdRepo, filepath.Join(p.VersionDir(v1.Version), "rick"), p.Port, p.StateDir, p.Token))
	if _, err := p.Promote(v2); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	res, err := p.Restart(v2.Version)
	if got := stageOf(t, err); got != "health" {
		t.Fatalf("失败阶段 = %q，期望 health", got)
	}
	msg := err.Error()
	if !strings.Contains(msg, v1.Version) {
		t.Fatalf("错误未含旧 build_id: %v", msg)
	}
	if !strings.Contains(msg, fmt.Sprintf("端口 %d", p.Port)) {
		t.Fatalf("错误未含生产端口: %v", msg)
	}
	if !strings.Contains(msg, "pid=") {
		t.Fatalf("错误未含旧实例 pid: %v", msg)
	}
	// 诊断两分支文案
	if !strings.Contains(msg, "诊断: 端口") {
		t.Fatalf("错误缺端口诊断: %v", msg)
	}
	// 留痕带 20 行日志尾（P0-4）——重启失败信息可定位
	if len(res.LogTail) == 0 {
		t.Fatal("失败未附日志尾部")
	}
}
