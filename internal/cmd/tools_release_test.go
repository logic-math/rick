package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// 回执契约（F3 后）：版本/目标路径/暂存路径/零触碰/清理结果
	for _, want := range []string{"RELEASE_DRYRUN", "target_bin=", "target_dist=", "staging=",
		"prod_untouched=true", "cleanup=ok"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run 回执缺少 %q:\n%s", want, out)
		}
	}
	if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该切 current 链")
	}
	if _, err := os.Lstat(plan.ProdBinLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该创建 bin/rick")
	}
	// F3：dry-run 的构建产物**不得**落进生产树（模拟生产 = 假生产 repo）。
	if entries := listUnder(plan.ReleasesDir()); len(entries) != 0 {
		t.Fatalf("dry-run 在生产树写了产物（应零写入）: %v", entries)
	}
	// 目标路径必须是「正式提升本会落到」的生产发布目录
	if !strings.Contains(out, filepath.Join(plan.ReleasesDir(), "")) {
		t.Fatalf("回执未预告正式提升的目标发布目录 %s:\n%s", plan.ReleasesDir(), out)
	}
	// 暂存必须是系统临时目录且已清理
	staging := extractField(out, "RELEASE_DRYRUN staging=")
	if staging == "" || !strings.HasPrefix(staging, os.TempDir()) {
		t.Fatalf("暂存目录应位于系统临时目录: %q", staging)
	}
	if _, err := os.Stat(strings.SplitN(staging, " ", 2)[0]); !os.IsNotExist(err) {
		t.Fatalf("dry-run 未清理暂存目录: %s", staging)
	}
	// 产物只落在临时模拟生产内
	if !strings.HasPrefix(root, os.TempDir()) {
		t.Fatalf("模拟生产不在临时目录: %s", root)
	}
}

// listUnder 递归列出目录下所有条目（相对路径）；目录不存在返回 nil。
func listUnder(root string) []string {
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

// extractField 从回执里取 "PREFIX=<token>" 形式的值（取到第一个空格为止）。
func extractField(out, prefix string) string {
	i := strings.Index(out, prefix)
	if i < 0 {
		return ""
	}
	rest := out[i+len(prefix):]
	if j := strings.IndexAny(rest, " \n"); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
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

// withHostedByProd 注入「被生产实例托管」的判定（默认实现依赖真实 /proc 祖谱）。
func withHostedByProd(t *testing.T, hosted bool, why string) {
	t.Helper()
	restore := hostedByProd
	hostedByProd = func(release.Plan) (bool, string) { return hosted, why }
	t.Cleanup(func() { hostedByProd = restore })
}

// TestReleaseCmdDryRunBypassesHostedSelfProtection 覆盖 F2：由生产实例托管的会话
// 执行 `release --dry-run` 必须成功（dry-run 不重启任何东西，自保拦截不适用），
// 但仍要如实打印 hosted_by_prod 警告，且绝不产生提升副作用。
func TestReleaseCmdDryRunBypassesHostedSelfProtection(t *testing.T) {
	plan, root := fakeReleaseProd(t)
	withFakePlan(t, plan)
	withHostedByProd(t, true, "祖先进程 12345 是生产实例")

	out, err := runReleaseCmd(t, "", "--dry-run")
	if err != nil {
		t.Fatalf("hosted 场景下 dry-run 应成功（F2 回归）: %v\n%s", err, out)
	}
	if !strings.Contains(out, "RELEASE_WARN hosted_by_prod=true") {
		t.Fatalf("dry-run 仍应如实打印托管警告:\n%s", out)
	}
	if !strings.Contains(out, "RELEASE_DRYRUN") {
		t.Fatalf("dry-run 未输出计划回执:\n%s", out)
	}
	// 纯演练：不得切换 current 链 / 不得创建生产 bin/rick / 不得重启
	if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该切 current 链")
	}
	if _, err := os.Lstat(plan.ProdBinLink()); !os.IsNotExist(err) {
		t.Fatal("dry-run 不该创建 bin/rick")
	}
	if !strings.HasPrefix(root, os.TempDir()) {
		t.Fatalf("模拟生产不在临时目录: %s", root)
	}
}

// TestReleaseCmdHostedNonDryRunStillRequiresConsent 反向断言：自保语义在非 dry-run
// 路径上**没有被削弱** —— 仍要求 --yes（或 --detach）。
func TestReleaseCmdHostedNonDryRunStillRequiresConsent(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)
	withHostedByProd(t, true, "祖先进程 12345 是生产实例")

	// releaseFail 会走 devExit（生产是 os.Exit）——测试里替换成记录函数，绝不真退出
	recorded := -1
	restoreExit := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restoreExit }()

	out, err := runReleaseCmd(t, "y\n")
	if err == nil {
		t.Fatalf("非 dry-run 且未给 --yes 时应被自保拦下:\n%s", out)
	}
	if recorded != release.ExitRestart {
		t.Fatalf("退出码 = %d，期望 %d（restart）", recorded, release.ExitRestart)
	}
	if !strings.Contains(out, "RELEASE_WARN hosted_by_prod=true") ||
		!strings.Contains(out, "承载当前会话") {
		t.Fatalf("自保拦截回执缺失:\n%s", out)
	}
	// 断言没有真正提升（当前链不存在）
	if _, statErr := os.Lstat(plan.CurrentLink()); !os.IsNotExist(statErr) {
		t.Fatal("被自保拦下时不该产生提升副作用")
	}
}

// errMergeConflictStub 是一个 stage=merge 的假冲突错误（CLI 单测不碰真实 git）。
var errMergeConflictStub = errors.New("源码合并存在冲突：f.txt（已 git merge --abort 回滚）")

// stubReleaseExit 让 releaseFail 不真的 os.Exit（生产是 os.Exit，测试里会直接
// 结束测试进程——之前的 exit status 5 就是这么来的）。
func stubReleaseExit(t *testing.T) *int {
	t.Helper()
	code := -1
	restore := devExit
	devExit = func(c int) { code = c }
	t.Cleanup(func() { devExit = restore })
	return &code
}

// fastProdPlan 把健康轮询/停机宽限压到 200ms：这些单测只关心「步骤顺序与回执」，
// 不关心真的把生产拉起来（假生产没有服务在 18999 监听，重启阶段必然失败）。
func fastProdPlan(p release.Plan) release.Plan {
	p.HealthWait = 200 * time.Millisecond
	p.StopGrace = 200 * time.Millisecond
	return p
}

// withMergeSourceFn 替换源码合并的调用点，记录是否被调用 + 可控返回。
func withMergeSourceFn(t *testing.T, fn func(release.Plan) (release.MergeResult, error)) *int {
	t.Helper()
	calls := 0
	restore := mergeSourceFn
	mergeSourceFn = func(p release.Plan) (release.MergeResult, error) {
		calls++
		return fn(p)
	}
	t.Cleanup(func() { mergeSourceFn = restore })
	return &calls
}

// TestReleaseCmdRejectsConflictingMergeFlags：--merge-source 与 --no-merge-source 互斥。
func TestReleaseCmdRejectsConflictingMergeFlags(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)

	recorded := -1
	restoreExit := devExit
	devExit = func(code int) { recorded = code }
	defer func() { devExit = restoreExit }()

	out, err := runReleaseCmd(t, "y\n", "--merge-source", "--no-merge-source", "--yes")
	if err == nil {
		t.Fatal("互斥 flag 必须报错")
	}
	if !strings.Contains(out, "参数冲突") {
		t.Fatalf("回执未指出参数冲突:\n%s", out)
	}
	if !strings.Contains(out, "RELEASE_FAIL stage=promote") {
		t.Fatalf("回执未标出 promote 阶段:\n%s", out)
	}
	if recorded != release.ExitRestart {
		t.Fatalf("退出码 = %d，期望 %d", recorded, release.ExitRestart)
	}
	// 冲突检查发生在任何构建/提升之前：产物目录不应出现
	if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
		t.Fatal("参数冲突时不得切 current 链")
	}
}

// TestReleaseCmdDefaultDoesNotMergeSource：不带 --merge-source 时**绝不**调用合并
// （默认语义与改造前完全一致）。
func TestReleaseCmdDefaultDoesNotMergeSource(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	plan = fastProdPlan(plan)
	withFakePlan(t, plan)
	stubReleaseExit(t) // 假生产没有服务监听 → 重启阶段会失败并走到 releaseFail
	calls := withMergeSourceFn(t, func(p release.Plan) (release.MergeResult, error) {
		t.Error("默认路径不应调用源码合并")
		return release.MergeResult{}, nil
	})

	out, _ := runReleaseCmd(t, "y\n", "--yes")
	if *calls != 0 {
		t.Fatalf("合并被调用 %d 次，期望 0", *calls)
	}
	if strings.Contains(out, "RELEASE_MERGE") {
		t.Fatalf("默认路径不应打印合并回执:\n%s", out)
	}
}

// TestReleaseCmdMergeSourceRunsBeforePromote：开启合并时先合并、且回执打印结果；
// 合并失败则**中止且不切链**（不继续提升）。
func TestReleaseCmdMergeSourceRunsBeforePromote(t *testing.T) {
	t.Run("merge ok then promote", func(t *testing.T) {
		plan, _ := fakeReleaseProd(t)
		plan = fastProdPlan(plan)
		withFakePlan(t, plan)
		stubReleaseExit(t)
		var gotVersion string
		withMergeSourceFn(t, func(p release.Plan) (release.MergeResult, error) {
			gotVersion = p.MergeVersion
			return release.MergeResult{Merged: true, Branch: "dev/self-evolve", Main: "main", Commit: "abc1234", Files: 3}, nil
		})

		// 假生产没有服务在监听：提升会走到重启阶段失败——但**合并必须已经发生且已换链**，
		// 且失败阶段不能是 merge（那才是本用例要防的回归）。
		out, _ := runReleaseCmd(t, "y\n", "--merge-source", "--yes")
		if strings.Contains(out, "RELEASE_FAIL stage=merge") {
			t.Fatalf("合并成功后不应在 merge 阶段失败:\n%s", out)
		}
		if !strings.Contains(out, "RELEASE_MERGE merged=true branch=dev/self-evolve main=main commit=abc1234 files=3") {
			t.Fatalf("缺少合并回执:\n%s", out)
		}
		// MergeVersion 必须是本次构建的版本号（merge commit 与二进制同版本）
		if gotVersion == "" {
			t.Fatal("调用合并时 MergeVersion 为空（应与本次构建版本一致）")
		}
		if !strings.Contains(out, "sha256=") || !strings.Contains(out, gotVersion) {
			t.Fatalf("MergeVersion 与报价单版本不一致（got %q）:\n%s", gotVersion, out)
		}
		if _, err := os.Lstat(plan.CurrentLink()); err != nil {
			t.Fatalf("提升应切换 current 链: %v", err)
		}
	})

	t.Run("already up to date is not a failure", func(t *testing.T) {
		plan, _ := fakeReleaseProd(t)
		plan = fastProdPlan(plan)
		withFakePlan(t, plan)
		stubReleaseExit(t)
		withMergeSourceFn(t, func(p release.Plan) (release.MergeResult, error) {
			return release.MergeResult{Merged: false, Branch: "dev/self-evolve", Main: "main", Reason: "already up to date"}, nil
		})

		out, _ := runReleaseCmd(t, "y\n", "--merge-source", "--yes")
		if strings.Contains(out, "RELEASE_FAIL stage=merge") {
			t.Fatalf("already up to date 不应被当成 merge 失败:\n%s", out)
		}
		if !strings.Contains(out, "RELEASE_MERGE merged=false") || !strings.Contains(out, "already up to date") {
			t.Fatalf("幂等回执缺失:\n%s", out)
		}
		if _, err := os.Lstat(plan.CurrentLink()); err != nil {
			t.Fatalf("幂等路径仍应完成提升: %v", err)
		}
	})

	t.Run("merge conflict aborts promote", func(t *testing.T) {
		plan, _ := fakeReleaseProd(t)
		withFakePlan(t, plan)
		withMergeSourceFn(t, func(p release.Plan) (release.MergeResult, error) {
			return release.MergeResult{}, &release.StageError{Stage: "merge", Err: errMergeConflictStub}
		})

		recorded := stubReleaseExit(t)

		out, err := runReleaseCmd(t, "y\n", "--merge-source", "--yes")
		if err == nil {
			t.Fatal("合并失败必须中止（不得继续提升）")
		}
		if !strings.Contains(out, "RELEASE_FAIL stage=merge") {
			t.Fatalf("回执未标出 merge 阶段:\n%s", out)
		}
		if *recorded != release.ExitRestart {
			t.Fatalf("退出码 = %d，期望 %d（提升失败类）", *recorded, release.ExitRestart)
		}
		// 关键：源码合并失败 → **不切链**（生产产物保持原样）
		if _, err := os.Lstat(plan.CurrentLink()); !os.IsNotExist(err) {
			t.Fatal("合并失败后不得切换 current 链")
		}
		// 失败留痕要能被人/AI 看到阶段=merge
		st, err := plan.ReadState()
		if err != nil {
			t.Fatalf("读留痕失败: %v", err)
		}
		if st.Stage != "merge" || st.OK {
			t.Fatalf("留痕 = %+v，期望 stage=merge ok=false", st)
		}
	})
}

// TestReleaseCmdMergeSourceWithDryRunWarns：演练不合并源码，但必须明确告知。
func TestReleaseCmdMergeSourceWithDryRunWarns(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)
	calls := withMergeSourceFn(t, func(p release.Plan) (release.MergeResult, error) {
		t.Error("dry-run 不应合并源码")
		return release.MergeResult{}, nil
	})

	out, err := runReleaseCmd(t, "", "--dry-run", "--merge-source")
	if err != nil {
		t.Fatalf("dry-run 应成功: %v\n%s", err, out)
	}
	if *calls != 0 {
		t.Fatalf("dry-run 调用了合并 %d 次，期望 0", *calls)
	}
	if !strings.Contains(out, "merge_source_skipped=dry-run") {
		t.Fatalf("dry-run 未提示「不合并源码」:\n%s", out)
	}
}

// TestReleaseCmdRollbackWarnsSourceNotReverted：--rollback 只回滚产物。
func TestReleaseCmdRollbackWarnsSourceNotReverted(t *testing.T) {
	plan, _ := fakeReleaseProd(t)
	withFakePlan(t, plan)

	// 本用例走到失败路径（没有回滚点）→ releaseFail 会调 devExit，必须替换掉，
	// 否则 os.Exit 直接结束测试进程（观察到的 exit status 5 就是这个）。
	stubReleaseExit(t)

	out, err := runReleaseCmd(t, "", "--rollback", "--yes", "--merge-source")
	if err == nil {
		t.Fatalf("没有回滚点时应失败（本用例只断言告警先打印）:\n%s", out)
	}
	if !strings.Contains(out, "rollback_does_not_revert_source=true") {
		t.Fatalf("回滚未说明「不含源码回滚」:\n%s", out)
	}
}
