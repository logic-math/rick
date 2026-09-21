package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env/release"
)

// TestReleaseExitCodeContract 锁定 `rick tools release` 的退出码契约
// （AI 会话据此判断该重试哪一步）：0 ok / 2 gate / 3 build / 4 promote+restart / 5 health。
func TestReleaseExitCodeContract(t *testing.T) {
	cases := map[string]int{
		"gate":    release.ExitGate,
		"build":   release.ExitBuild,
		"promote": release.ExitRestart,
		"restart": release.ExitRestart,
		"health":  release.ExitHealth,
		"unknown": release.ExitRestart,
	}
	for stage, want := range cases {
		if got := release.ExitCodeForStage(stage); got != want {
			t.Errorf("ExitCodeForStage(%q) = %d, want %d", stage, got, want)
		}
	}
	if release.ExitGate != 2 || release.ExitBuild != 3 || release.ExitRestart != 4 || release.ExitHealth != 5 {
		t.Fatalf("退出码常量被改动: gate=%d build=%d restart=%d health=%d",
			release.ExitGate, release.ExitBuild, release.ExitRestart, release.ExitHealth)
	}
}

// TestReleaseStageName 验证失败阶段提取（非 StageError 归到 restart）。
func TestReleaseStageName(t *testing.T) {
	if got := stageName(&release.StageError{Stage: "gate", Err: fmt.Errorf("x")}); got != "gate" {
		t.Fatalf("stageName = %q", got)
	}
	if got := stageName(fmt.Errorf("plain")); got != "restart" {
		t.Fatalf("stageName(plain) = %q", got)
	}
}

// TestReleaseFailPrintsReceiptAndExits 验证失败回执格式 + 退出码（不真退出：替换 devExit）。
func TestReleaseFailPrintsReceiptAndExits(t *testing.T) {
	recorded := -1
	restore := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restore }()

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := releaseFail(cmd, "health", fmt.Errorf("build_id 不一致\n多行")); err == nil {
		t.Fatal("releaseFail 应把错误返回给调用方")
	}
	out := buf.String()
	if !strings.Contains(out, "RELEASE_FAIL stage=health") || !strings.Contains(out, "|") {
		t.Fatalf("回执格式不对: %q", out)
	}
	if recorded != release.ExitHealth {
		t.Fatalf("退出码 = %d，期望 %d（health）", recorded, release.ExitHealth)
	}
}

// TestReleaseCmdDocumentsFlagsAndExitCodes 锁定 CLI 契约面：人类/AI 靠 --help 发现
// 能力，gate9 也据此断言 flag 齐备。
func TestReleaseCmdDocumentsFlagsAndExitCodes(t *testing.T) {
	cmd := NewReleaseCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Help(); err != nil {
		t.Fatalf("help: %v", err)
	}
	help := buf.String()
	for _, flag := range []string{"--yes", "--rollback", "--dry-run", "--detach"} {
		if !strings.Contains(help, flag) {
			t.Errorf("--help 缺少 %s", flag)
		}
	}
	for _, code := range []string{"2 gate fail", "3 build fail", "4 promote or restart fail", "5 health"} {
		if !strings.Contains(help, code) {
			t.Errorf("--help 未文档化退出码 %q", code)
		}
	}
	// 「不会自动续跑」这一安全语义必须写在 help 里（人类执行前就知道会发生什么）
	if !strings.Contains(help, "不会自动续跑") {
		t.Error("--help 未说明「会话挂起、不自动续跑」的影响面")
	}
}

// fakeReleaseProd 造一个「模拟生产」：临时 repo + 临时 home + 假二进制。
// 只用于把 CLI 的**非重启路径**（dry-run / 拒绝确认 / 缺回滚点）跑通，
// 绝不接触真实 ~/.rick 与生产端口。
func fakeReleaseProd(t *testing.T) (release.Plan, string) {
	t.Helper()
	root := t.TempDir()
	prodRepo := filepath.Join(root, "prod")
	devTree := filepath.Join(root, "dev")
	prodHome := filepath.Join(root, "prodhome")
	for _, d := range []string{filepath.Join(prodRepo, "bin"), filepath.Join(devTree, "web", "dist"),
		prodHome, filepath.Join(prodHome, ".rick", "web")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for _, f := range []string{filepath.Join(prodRepo, "go.mod"), filepath.Join(devTree, "go.mod")} {
		if err := os.WriteFile(f, []byte("module github.com/sunquan/rick\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	if err := os.WriteFile(filepath.Join(devTree, "web", "dist", "index.html"), []byte("<html>x</html>"), 0o644); err != nil {
		t.Fatalf("write dist: %v", err)
	}
	plan := release.Plan{
		ProdRepo: prodRepo, DevTree: devTree, ProdHome: prodHome,
		StateDir: filepath.Join(prodHome, ".rick"), Port: 18999, Listen: "127.0.0.1",
		Token: "faketok", SkipGates: true,
		// 假构建：写一个可执行的占位二进制（dry-run 路径不会去重启它）
		BuildBinary: func(_ release.Plan, dest string, _ io.Writer) error {
			return os.WriteFile(dest, []byte("#!/bin/sh\nexit 0\n"), 0o755)
		},
	}
	return plan, root
}

// withFakePlan 替换 releasePlanFor（CLI 的 Plan 解析入口），测试结束自动还原。
// 返回的 plan 强制 SkipGates=true：CLI 单测绝不跑真实门禁（那要几分钟且与 CLI
// 行为无关）；`--no-gates` 的**告警**由 opts 驱动，与本桩无关。
func withFakePlan(t *testing.T, plan release.Plan) {
	t.Helper()
	restore := releasePlanFor
	releasePlanFor = func(o releaseOptions) (release.Plan, error) {
		p := plan
		p.SkipGates = true
		if o.port > 0 {
			p.Port = o.port
		}
		if o.keep > 0 {
			p.KeepVersions = o.keep
		}
		return p, nil
	}
	t.Cleanup(func() { releasePlanFor = restore })
}

// runReleaseCmd 执行命令并捕获输出/错误。
func runReleaseCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := NewReleaseCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestReleaseCmdDryRunUsesInjectedPlanAndNeverPromotes(t *testing.T) {
	plan, root := fakeReleaseProd(t)
	withFakePlan(t, plan)

	out, err := runReleaseCmd(t, "", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "RELEASE_DRYRUN") || !strings.Contains(out, "未触碰生产") {
		t.Fatalf("dry-run 回执缺失:\n%s", out)
	}
	if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该切 current 链")
	}
	if _, err := os.Lstat(plan.ProdBinLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该创建 bin/rick")
	}
	// 产物只落在临时模拟生产内
	if !strings.HasPrefix(root, os.TempDir()) {
		t.Fatalf("模拟生产不在临时目录: %s", root)
	}
}

func TestReleaseCmdDeclineAbortsWithoutPromoting(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)

	out, err := runReleaseCmd(t, "n\n")
	if err != nil {
		t.Fatalf("decline: %v\n%s", err, out)
	}
	if !strings.Contains(out, "RELEASE_ABORTED") {
		t.Fatalf("未提示放弃:\n%s", out)
	}
	if !strings.Contains(out, "RELEASE_QUOTE") {
		t.Fatalf("确认前应给出报价单（含 sha256/版本）:\n%s", out)
	}
	if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
		t.Fatal("放弃确认后不得切 current 链")
	}
}

func TestReleaseCmdRollbackWithoutRollbackPointFails(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)

	recorded := -1
	restore := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restore }()

	out, err := runReleaseCmd(t, "", "--rollback", "--yes")
	if err == nil {
		t.Fatal("没有回滚点时必须报错")
	}
	if !strings.Contains(out, "RELEASE_FAIL stage=promote") {
		t.Fatalf("回执未标出 promote 阶段:\n%s", out)
	}
	if recorded != release.ExitRestart {
		t.Fatalf("退出码 = %d，期望 %d", recorded, release.ExitRestart)
	}
}

func TestReleaseCmdRejectsDevTreeEqualToProd(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	plan.DevTree = plan.ProdRepo
	withFakePlan(t, plan)

	recorded := -1
	restore := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restore }()

	out, err := runReleaseCmd(t, "", "--yes")
	if err == nil {
		t.Fatal("dev 树 == 生产树 必须拒绝")
	}
	if !strings.Contains(out, "RELEASE_FAIL") {
		t.Fatalf("回执缺失:\n%s", out)
	}
	if recorded != release.ExitRestart {
		t.Fatalf("退出码 = %d，期望 %d", recorded, release.ExitRestart)
	}
}

// TestReleaseCmdNoGatesWarns 演练模式必须显著告警（防「以为跑了门禁」）。
func TestReleaseCmdNoGatesWarns(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	plan.SkipGates = false
	withFakePlan(t, plan)

	out, err := runReleaseCmd(t, "n\n", "--no-gates")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "RELEASE_WARN no_gates=true") {
		t.Fatalf("未告警跳过门禁:\n%s", out)
	}
}
