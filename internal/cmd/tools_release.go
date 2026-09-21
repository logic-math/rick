package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env/release"
)

// releaseDetachEnv 是 `--detach` 的子进程标记：父进程确认后以 setsid 重新拉起
// 自己（重启承载自己的实例时不被一起杀掉），子进程看到该变量即跳过再次自举与
// 交互确认。
const releaseDetachEnv = "RICK_RELEASE_DETACHED"

// releasePlanFor 是 Plan 解析的间接层：测试可替换它，从而在**不碰真实生产**的
// 前提下验证 CLI 行为（默认实现从部署事实推导，见 release.DefaultPlan）。
var releasePlanFor = func(opts releaseOptions) (release.Plan, error) {
	p, err := release.DefaultPlan(opts.prodRepo, opts.devTree, opts.port)
	if err != nil {
		return p, err
	}
	if opts.prodHome != "" {
		p.ProdHome = opts.prodHome
	}
	if opts.stateDir != "" {
		p.StateDir = opts.stateDir
	}
	if opts.startScript != "" {
		p.StartScript = opts.startScript
	}
	if opts.keep > 0 {
		p.KeepVersions = opts.keep
	}
	if opts.skipFrontend {
		p.SkipFrontend = true
	}
	p.SkipGates = opts.noGates
	return p, nil
}

type releaseOptions struct {
	prodRepo     string
	devTree      string
	prodHome     string
	stateDir     string
	startScript  string
	port         int
	keep         int
	noGates      bool
	skipFrontend bool
	yes          bool
	dryRun       bool
	rollback     bool
	detach       bool
	jsonOut      bool
}

// NewReleaseCmd 创建 `rick tools release`：把 dev 工作树的产物原子提升为生产、
// 受控重启生产、并支持一键回滚。**人类执行这条命令 = 人类确认**（research-L6 §5：
// web token 存在 localStorage，任何「网页点一下」都不构成确认；只有终端里的显式
// 动作才算）。
//
// 语义要点（human 裁决 J-L6-6/J-L6-7）：
//   - 提升会重启生产 → 所有在跑会话/后台 job 一律变「挂起（suspended）」，
//     **不自动恢复、不自动续跑**（避免重复副作用与配额静默空转）；
//   - 人类刷新页面后点「恢复继续」即可（会话走 --session resume；doing/dream 走
//     归一化 running→pending 后续跑）；
//   - 平台本身必须自动回来：本命令重启后会轮询 /api/health 并要求 build_id 等于
//     本次提升版本，否则判定失败并自动回滚到上一版。
//
// Print:
//
//	RELEASE_PLAN    报价单（版本/端口/路径/当前版本/回滚点）
//	RELEASE_GATE    门禁结果
//	RELEASE_BUILD   构建产物 + sha256
//	RELEASE_PROMOTE 换链 + 覆盖层 + GC 结果
//	RELEASE_RESTART 重启结果（pid/health_ms/build_id 是否匹配）
//	RELEASE_RECOVER 挂起清单（谁被挂起，去 UI 一键恢复）
//	RELEASE_OK      成功回执
//	RELEASE_FAIL    stage=<gate|build|promote|restart|health> detail=<原因>
//
// Exit codes: 0 ok / 2 gate fail / 3 build fail / 4 promote+restart fail / 5 health fail
func NewReleaseCmd() *cobra.Command {
	opts := releaseOptions{}

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Atomically promote the dev build to production, restart it, or roll back",
		Long: `Promote the dev worktree's artifacts (binary + frontend) to production, restart
production, and verify that the running build really is the new one. Supports one
command rollback.

  生产 = 仓库工作树（<prod-repo>/bin/rick → releases/current/rick）+ <prod-home>/.rick
  dev  = git worktree + 独立 HOME（rick tools dev-web 管理）

What it does:
  1) 门禁（go test ./... + npm run build）—— 在 **dev 树**里跑，绝不写生产 bin/
  2) 构建到 <prod-repo>/bin/releases/<version>/{rick,dist}（版本号=<sha7>-<时间戳>，
     并以 -ldflags 注入 build_id）
  3) 记录回滚点 → 原子换链（current）→ bin/rick → releases/current/rick
  4) 前端同版推进到 <state>/.rick/web/dist（旧覆盖层留 dist.prev）
  5) 重启生产 → 轮询 /api/health 并要求 build_id == 本次版本（不是则失败）
  6) 打印恢复报告：谁因重启被挂起（去 web UI 点「恢复继续」）

Impact: 生产会重启，期间浏览器自动重连；所有在跑会话/后台 job 变为「挂起」，
不会自动续跑（避免重复副作用）——这是刻意设计，不是故障。

Exit codes:
  0 ok / 2 gate fail / 3 build fail / 4 promote or restart fail / 5 health+build_id fail`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRelease(cmd, opts)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.yes, "yes", false, "跳过交互确认（等价于人类已在终端里点头）")
	f.BoolVar(&opts.rollback, "rollback", false, "回滚到上一个版本（bin/releases/.last）并重启")
	f.BoolVar(&opts.dryRun, "dry-run", false, "只跑门禁+构建+打印计划，不动生产、不重启")
	f.BoolVar(&opts.detach, "detach", false, "setsid 脱离后在后台执行（避免重启承载自己的实例时被一起杀掉）")
	f.StringVar(&opts.prodRepo, "prod-repo", "", "生产仓库工作树（默认：本 CLI 所在仓库）")
	f.StringVar(&opts.devTree, "dev-tree", "", "dev 工作树（默认：<祖父目录>/rick-dev 或 RICK_DEV_TREE）")
	f.StringVar(&opts.prodHome, "prod-home", "", "生产 HOME（默认：真实家目录）")
	f.StringVar(&opts.stateDir, "state-dir", "", "生产状态目录（默认：部署脚本/生产 HOME 约定）")
	f.StringVar(&opts.startScript, "start-script", "", "部署启动脚本（默认：<prod-home>/start-web.sh）")
	f.IntVar(&opts.port, "port", 0, "生产端口（默认：从部署脚本解析；解析不到即报错）")
	f.IntVar(&opts.keep, "keep", 0, "保留最近 N 个版本（默认 3）")
	f.BoolVar(&opts.noGates, "no-gates", false, "跳过门禁（仅演练/测试；正式提升会打印警告）")
	f.BoolVar(&opts.skipFrontend, "skip-frontend", false, "不提升前端（只换二进制）")
	f.BoolVar(&opts.jsonOut, "json", false, "结尾输出 JSON 结果（脚本/AI 解析用）")
	return cmd
}

func runRelease(cmd *cobra.Command, opts releaseOptions) error {
	out := cmd.OutOrStdout()
	plan, err := releasePlanFor(opts)
	if err != nil {
		return releaseFail(cmd, "promote", err)
	}

	// ---- 自保：本进程是否被生产实例托管（我们正要去重启它）----
	hosted, why := release.HostedByProd(plan)
	if hosted && !opts.detach {
		fmt.Fprintf(out, "RELEASE_WARN hosted_by_prod=true (%s)\n", why)
		if !opts.yes {
			return releaseFail(cmd, "restart", fmt.Errorf(
				"本命令会重启承载当前会话的生产实例，进程可能半途被杀；请加 --yes 明确同意，或用 --detach 脱离进程组执行"))
		}
	}

	// ---- --detach：父进程确认后以 setsid 重新拉起自己 ----
	if opts.detach && os.Getenv(releaseDetachEnv) == "" {
		return spawnDetached(cmd, plan)
	}

	if opts.noGates {
		fmt.Fprintf(out, "RELEASE_WARN no_gates=true（仅演练；正式提升必须跑门禁）\n")
	}
	if err := plan.Validate(); err != nil {
		return releaseFail(cmd, "promote", err)
	}
	printPlan(out, plan, opts)

	// ---- 回滚路径（不构建）----
	if opts.rollback {
		if !opts.yes && !confirm(cmd, plan, "rollback") {
			fmt.Fprintln(out, "RELEASE_ABORTED by=user")
			return nil
		}
		res, err := plan.Rollback()
		if err != nil {
			_ = plan.WriteState(release.ReleaseState{Action: "rollback", Stage: stageName(err), OK: false, Error: err.Error()})
			return releaseFail(cmd, stageName(err), err)
		}
		fmt.Fprintf(out, "RELEASE_PROMOTE rollback_to=%s prev=%s bin=%s overlay=%s\n",
			res.Version, res.PrevVersion, res.Bin, res.Overlay)
		rr, err := plan.Restart(res.Version)
		res.Restart = &rr
		if err != nil {
			return finishFail(cmd, plan, res, err)
		}
		printRestart(out, rr)
		return finishOK(cmd, plan, res, opts)
	}

	// ---- dry-run（构建 + 打印，不动生产）----
	if opts.dryRun {
		res, err := plan.DryRun(out)
		if err != nil {
			return releaseFail(cmd, stageName(err), err)
		}
		return emitJSON(cmd, opts, res)
	}

	// ---- 正常提升：先构建（门禁+产物）→ 报价 → 人类确认 → 落盘 ----
	rel, err := plan.Build(out)
	if err != nil {
		return releaseFail(cmd, stageName(err), err)
	}
	printQuote(out, plan, rel)
	if !opts.yes && !confirm(cmd, plan, "promote") {
		fmt.Fprintln(out, "RELEASE_ABORTED by=user（产物已构建但未提升）")
		return nil
	}

	res, err := plan.Rollout(rel, out)
	if err != nil {
		return finishFail(cmd, plan, res, err)
	}
	if res.Restart != nil {
		printRestart(out, *res.Restart)
	}
	fmt.Fprintf(out, "RELEASE_PROMOTE version=%s prev=%s bin=%s overlay=%s dist_files=%d gc=%v\n",
		res.Version, res.PrevVersion, res.Bin, res.Overlay, res.DistFiles, res.GCRemoved)
	return finishOK(cmd, plan, res, opts)
}

// finishFail 记录失败留痕并映射退出码（自动回滚的结局也在这里体现）。
func finishFail(cmd *cobra.Command, plan release.Plan, res release.Result, err error) error {
	stage := stageName(err)
	st := release.ReleaseState{Action: res.Action, Version: res.Version, Stage: stage, OK: false, Error: err.Error(), Result: &res}
	if res.Restart != nil {
		st.PID, st.BuildID, st.Recovery = res.Restart.PID, res.Restart.BuildID, res.Restart.Recovery
	}
	_ = plan.WriteState(st)
	return releaseFail(cmd, stage, err)
}

// finishOK 打印成功回执 + 挂起清单，并落留痕。
func finishOK(cmd *cobra.Command, plan release.Plan, res release.Result, opts releaseOptions) error {
	out := cmd.OutOrStdout()
	st := release.ReleaseState{Action: res.Action, Version: res.Version, Stage: "done", OK: true, Result: &res}
	if res.Restart != nil {
		st.PID, st.BuildID, st.Recovery = res.Restart.PID, res.Restart.BuildID, res.Restart.Recovery
	}
	if err := plan.WriteState(st); err != nil {
		fmt.Fprintf(out, "RELEASE_WARN write_state_failed=%v\n", err)
	}
	fmt.Fprintf(out, "RELEASE_OK version=%s state=%s\n", res.Version, plan.StatePath())
	return emitJSON(cmd, opts, res)
}

// printPlan 打印计划（人类确认前的事实清单）。
func printPlan(out io.Writer, p release.Plan, opts releaseOptions) {
	s := p.Summary()
	fmt.Fprintf(out, "RELEASE_PLAN prod_repo=%s dev_tree=%s state_dir=%s port=%d listen=%s\n",
		s.ProdRepo, s.DevTree, s.StateDir, s.Port, s.Listen)
	fmt.Fprintf(out, "RELEASE_PLAN start_script=%s current=%s rollback_point=%s keep=%d\n",
		firstNonEmptyStr(s.StartScript, "(直接执行 bin/rick)"), firstNonEmptyStr(s.Current, "(无)"),
		firstNonEmptyStr(s.Last, "(无)"), p.KeepOr())
	if opts.rollback {
		fmt.Fprintln(out, "RELEASE_INTENT 回滚到上一个版本并重启生产")
		return
	}
	fmt.Fprintln(out, "RELEASE_IMPACT 生产将重启（期间浏览器自动重连）；所有在跑会话/后台 job 变为「挂起」——刷新页面后点「恢复继续」即可，**不会自动续跑**（避免重复副作用）")
}

// printQuote 打印「报价单」：人类批准的是冻结了 sha256 的确定产物。
func printQuote(out io.Writer, p release.Plan, rel release.Release) {
	devHead := rel.Version
	if i := strings.Index(rel.Version, "-"); i > 0 {
		devHead = rel.Version[:i]
	}
	fmt.Fprintf(out, "RELEASE_QUOTE version=%s sha256=%s dist_files=%d bin=%s dev_head=%s\n",
		rel.Version, rel.SHA256, rel.DistFiles, rel.Bin, devHead)
	if file := filepath.Join(rel.Dir, release.ChangesName); fileExistsCmd(file) {
		if body, err := os.ReadFile(file); err == nil {
			fmt.Fprintf(out, "RELEASE_CHANGES %s\n%s", file, tailStr(string(body), 1200))
		}
	}
	fmt.Fprintf(out, "RELEASE_CONFIRM 输入 y 确认提升到 %s（或回车放弃）: ", rel.Version)
}

// printRestart 打印重启结果 + 挂起清单。
func printRestart(out io.Writer, rr release.RestartResult) {
	fmt.Fprintf(out, "RELEASE_RESTART pid=%d health_ms=%d build_id=%s matches=%v stopped=%v skipped=%v\n",
		rr.PID, rr.HealthMS, rr.BuildID, rr.Matches, rr.Stopped, rr.Skipped)
	if len(rr.Skipped) > 0 {
		fmt.Fprintf(out, "RELEASE_WARN skipped_pids=%v（未确认属于该生产部署，已拒杀）\n", rr.Skipped)
	}
	fmt.Fprintf(out, "RELEASE_RECOVER suspended=%d recovered=%d failed=%d report=%s\n",
		rr.Recovery.Suspended, rr.Recovery.Recovered, rr.Recovery.Failed,
		firstNonEmptyStr(rr.Recovery.Path, "(无)"))
	if len(rr.Recovery.IDs) > 0 {
		fmt.Fprintf(out, "RELEASE_RECOVER_SESSIONS %s\n", strings.Join(rr.Recovery.IDs, ", "))
		if rr.Recovery.Suspended > 0 {
			fmt.Fprintln(out, "RELEASE_RECOVER_HINT 去 rick web 会话列表点「恢复继续」（会话=resume；doing/dream=归一化 running→pending 后续跑）")
		}
	}
	if len(rr.LogTail) > 0 {
		fmt.Fprintf(out, "RELEASE_LOG_TAIL %s\n", strings.Join(rr.LogTail, " | "))
	}
}

// confirm 读终端确认（默认拒绝）。可输入 y/yes，或直接输入版本号（更强的确认）。
func confirm(cmd *cobra.Command, p release.Plan, action string) bool {
	fmt.Fprint(cmd.OutOrStdout(), "[y/N] ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "y" || ans == "yes" || ans == "确认" {
		return true
	}
	if cur, _ := p.CurrentVersion(); cur != "" && ans == strings.ToLower(cur) {
		return true
	}
	return false
}

// spawnDetached 用 setsid 重新拉起自己执行提升（日志落 <state>/release.log），
// 父进程立刻返回——避免「提升过程把自己杀掉」导致半途而废。
func spawnDetached(cmd *cobra.Command, p release.Plan) error {
	out := cmd.OutOrStdout()
	if err := os.MkdirAll(p.StateDirPath(), 0o755); err != nil {
		return releaseFail(cmd, "restart", err)
	}
	logPath := filepath.Join(p.StateDirPath(), "release.log")
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return releaseFail(cmd, "restart", err)
	}
	defer logf.Close()
	args := os.Args[1:]
	if !containsFlag(args, "--yes") {
		args = append(args, "--yes")
	}
	child := exec.Command(os.Args[0], args...)
	child.Env = append(os.Environ(), releaseDetachEnv+"=1")
	child.Dir = p.ProdRepo
	child.Stdout = logf
	child.Stderr = logf
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return releaseFail(cmd, "restart", err)
	}
	_ = child.Process.Release()
	fmt.Fprintf(out, "RELEASE_DETACHED pid=%d log=%s state=%s\n", child.Process.Pid, logPath, p.StatePath())
	fmt.Fprintf(out, "RELEASE_DETACHED_HINT 提升在后台进行；用 `cat %s` 看进度，`curl -s %s/api/health` 看新实例\n",
		p.StatePath(), p.BaseURL())
	return nil
}

func emitJSON(cmd *cobra.Command, opts releaseOptions, payload any) error {
	if !opts.jsonOut {
		return nil
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// releaseFail 打印结构化失败回执并按阶段以专用退出码退出。
func releaseFail(cmd *cobra.Command, stage string, err error) error {
	detail := strings.ReplaceAll(err.Error(), "\n", " | ")
	if len(detail) > 600 {
		detail = detail[:600] + "..."
	}
	fmt.Fprintf(cmd.OutOrStdout(), "RELEASE_FAIL stage=%s detail=%s\n", stage, detail)
	devExit(release.ExitCodeForStage(stage))
	return err
}

// stageName 从错误里取阶段（未知 → restart）。
func stageName(err error) string {
	var se *release.StageError
	if errors.As(err, &se) {
		return se.Stage
	}
	return "restart"
}

func containsFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return true
		}
	}
	return false
}

func fileExistsCmd(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func tailStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

// firstNonEmptyStr 返回第一个非空串（打印兜底）。
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
