package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/workspace"
)

// rsi_check：把 `.rick/loops/rick-rsi-loop.md` 的「产出评估」表变成**机器契约**。
//
// 为什么需要它：loop 里那张表如果是纯文档，就等于「靠自觉」——一次自进化迭代
// 有没有做人类确认、有没有留回滚点、生产是不是真的跑上了新构建，全都没人验。
// 本命令把那 6 行逐条变成可执行断言，缺证据即「本次迭代未完成」（exit 1）。
//
// 与 loop 的关系：**loop 是权威**。若 loop 的评估表变了，改本文件去对齐；
// 绝不允许为了让现状通过而放松本校验。
//
// 6 项与 loop「产出评估」表一一对应：
//  1. dev 迭代记录   doing/rsi/dev-iterations.md  至少一条构建指纹
//  2. 门禁全绿       doing/rsi/gates.md           每行 pass=true，无 pass=false
//  3. 人类确认       doing/rsi/approval.md        APPROVED by=human + 时间戳（最关键）
//  4. 提升记录       doing/rsi/release.md         version=<sha7>-<ts> + rollback_point=
//  5. 挂起-恢复      doing/rsi/resume.md          挂起清单 + 人工恢复结果
//  6. 生产健康       实时 GET <prod>/api/health   status=ok 且 build_id == 本次 version
const rsiEvidenceDir = "rsi"

// rsiSkeletonMarker 是 --init 生成的骨架里的占位标记。**残留该标记 = 未填写**，
// 因此骨架永远不会「看起来通过」（这是本校验器最重要的一条防伪规则：没有它，
// --init 之后立刻 rsi_check 就会 pass，等于把契约变成自欺）。
const rsiSkeletonMarker = "<!-- TODO"

// rsiBuildIDPattern 匹配构建指纹：`rick.dev.<sha7>-<HHMMSS>` 或 `build_id=<sha7>-<YYMMDDHHMMSS>`。
// 两段都允许：dev 二进制名产物形态与 --json 回执里的 build_id= 形态。
var (
	rsiBuildIDRe   = regexp.MustCompile(`[0-9a-f]{7,40}-[0-9]{6,20}`)
	rsiGateLineRe  = regexp.MustCompile(`(?im)^\s*(?:GATE\s+)?\S*\s*pass\s*=\s*(true|false)`)
	rsiApprovedRe  = regexp.MustCompile(`(?i)APPROVED\s+by\s*=\s*human`)
	rsiTimestampRe = regexp.MustCompile(`(?i)(at\s*=|20[0-9]{2}-[0-9]{2}-[0-9]{2})`)
	rsiVersionRe   = regexp.MustCompile(`(?i)version\s*=\s*([0-9a-f]{7,40}-[0-9]{6,20})`)
	rsiRollbackRe  = regexp.MustCompile(`(?i)(rollback_point\s*=\s*\S|released\s*=\s*yes)`)
	rsiSuspendedRe = regexp.MustCompile(`(?i)(suspended|挂起)`)
	rsiResumedRe   = regexp.MustCompile(`(?i)(resumed|恢复)`)
)

// rsiCheck is one evaluation row's verdict (evidence = the file/URL it was judged on).
type rsiCheck struct {
	Name     string `json:"name"`
	Pass     bool   `json:"pass"`
	Evidence string `json:"evidence"`
	Detail   string `json:"detail"`
}

// rsiCheckResult is the --json contract of `rick tools rsi_check`.
type rsiCheckResult struct {
	Pass   bool       `json:"pass"`
	Job    string     `json:"job"`
	Dir    string     `json:"dir"`
	Checks []rsiCheck `json:"checks"`
	Errors []string   `json:"errors"`
}

// rsiEvidenceFiles lists the five on-disk evidence files in evaluation order.
var rsiEvidenceFiles = []struct{ Name, File string }{
	{"dev-iterations", "dev-iterations.md"},
	{"gates", "gates.md"},
	{"approval", "approval.md"},
	{"release", "release.md"},
	{"resume", "resume.md"},
}

// rsiHealthProbe is the seam for the production-health row so tests (and the
// simulated-production E2E) never have to touch the real 8413.
var rsiHealthProbe = func(baseURL string) (buildID string, err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/health")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var body struct {
		Status  string `json:"status"`
		BuildID string `json:"build_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("解析 /api/health 响应: %w", err)
	}
	if body.Status != "ok" {
		return "", fmt.Errorf("status=%q（期望 ok）", body.Status)
	}
	return body.BuildID, nil
}

// resolveRSIJobID resolves the job to evaluate: explicit flag > global --job >
// the most recently modified job under <cwd>/.rick/jobs/.
//
// 自动取「最近修改的 job」是因为 RSI 会话的典型用法就是「刚干完活，顺手校验」——
// 强制每次都手写 job 编号只会让人漏校验。
func resolveRSIJobID(explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		return id, nil
	}
	if id := strings.TrimSpace(GetJobID()); id != "" {
		return id, nil
	}
	jobsDir, err := workspace.GetJobsDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		return "", fmt.Errorf("无法读取 %s（应指向含 .rick/jobs/ 的工作区根）: %w", jobsDir, err)
	}
	type cand struct {
		name string
		mod  time.Time
	}
	var cands []cand
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "job_") {
			continue
		}
		info, iErr := e.Info()
		if iErr != nil {
			continue
		}
		cands = append(cands, cand{e.Name(), info.ModTime()})
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("%s 下没有 job_* 目录；请用 --job job_N 显式指定", jobsDir)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })
	return cands[0].name, nil
}

// rsiEvidencePath returns <doing>/rsi/<file>.
func rsiEvidencePath(doingDir, file string) string {
	return filepath.Join(doingDir, rsiEvidenceDir, file)
}

// readRSIEvidence reads an evidence file, returning (content, exists, isSkeleton).
func readRSIEvidence(path string) (string, bool, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, false
	}
	content := string(data)
	return content, true, strings.Contains(content, rsiSkeletonMarker)
}

// filled reports whether a file exists, is non-empty, and is not a raw skeleton.
func filled(path string) (string, string) {
	content, exists, skeleton := readRSIEvidence(path)
	switch {
	case !exists:
		return "", "文件不存在"
	case strings.TrimSpace(content) == "":
		return content, "文件为空"
	case skeleton:
		return content, fmt.Sprintf("仍是 --init 生成的骨架（残留 %q 占位标记）", rsiSkeletonMarker)
	}
	if len(strings.TrimSpace(content)) < 20 {
		return content, "内容过短，未记录实质证据"
	}
	return content, ""
}

// checkRSIDevIterations: 至少一条构建指纹（证明真的在 dev 实例上迭代过）。
func checkRSIDevIterations(doingDir string) rsiCheck {
	path := rsiEvidencePath(doingDir, "dev-iterations.md")
	c := rsiCheck{Name: "dev-iterations", Evidence: path}
	content, problem := filled(path)
	if problem != "" {
		c.Detail = fmt.Sprintf("%s；下一步：跑 `rick tools dev-web status`（或 `up`/`restart`）并把回执里的 build_id 贴进 %s", problem, path)
		return c
	}
	if !rsiBuildIDRe.MatchString(content) {
		c.Detail = fmt.Sprintf("未找到构建指纹（形如 %s）；下一步：把 `DEV_UP ... build_id=<sha7>-<ts>` 或 `rick.dev.<sha7>-<ts>` 记进 %s", "c539c60-260921192330", path)
		return c
	}
	c.Pass = true
	c.Detail = "已记录 dev 实例构建指纹"
	return c
}

// checkRSIGates: 每行门禁记录都是 pass=true（出现 pass=false 即失败）。
func checkRSIGates(doingDir string) rsiCheck {
	path := rsiEvidencePath(doingDir, "gates.md")
	c := rsiCheck{Name: "gates", Evidence: path}
	content, problem := filled(path)
	if problem != "" {
		c.Detail = fmt.Sprintf("%s；下一步：逐层跑 `python3 .rick/jobs/<job>/plan/gates/gateN.py` 并把 `GATE gateN pass=true` 记进 %s", problem, path)
		return c
	}
	matches := rsiGateLineRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		c.Detail = fmt.Sprintf("没有任何 `pass=` 记录（形如 `GATE gate7 pass=true`）；下一步：记录每层门禁结果到 %s", path)
		return c
	}
	var red []string
	for _, m := range matches {
		if strings.EqualFold(m[1], "false") {
			red = append(red, "pass=false")
		}
	}
	if len(red) > 0 {
		c.Detail = fmt.Sprintf("存在 %d 条 pass=false —— 门禁未全绿，禁止继续（先修好再重跑）；见 %s", len(red), path)
		return c
	}
	c.Pass = true
	c.Detail = fmt.Sprintf("%d 条门禁记录全绿", len(matches))
	return c
}

// checkRSIApproval: 人类确认痕迹（loop 里最关键的一项）。
func checkRSIApproval(doingDir string) rsiCheck {
	path := rsiEvidencePath(doingDir, "approval.md")
	c := rsiCheck{Name: "approval", Evidence: path}
	content, problem := filled(path)
	if problem != "" {
		c.Detail = fmt.Sprintf("%s；下一步：先跑 `rick tools release --dry-run` 把计划给人看，得到同意后把 `APPROVED by=human at=<时间>` 写进 %s", problem, path)
		return c
	}
	if !rsiApprovedRe.MatchString(content) {
		c.Detail = fmt.Sprintf("缺 `APPROVED by=human`（人类确认是本 loop 的硬门槛）；见 %s", path)
		return c
	}
	if !rsiTimestampRe.MatchString(content) {
		c.Detail = fmt.Sprintf("有确认标记但缺时间戳（`at=<时间>`）；见 %s", path)
		return c
	}
	c.Pass = true
	c.Detail = "已记录人类确认（by=human + 时间戳）"
	return c
}

// checkRSIRelease: 提升记录（版本 + 回滚点）。
func checkRSIRelease(doingDir string) rsiCheck {
	path := rsiEvidencePath(doingDir, "release.md")
	c := rsiCheck{Name: "release", Evidence: path}
	content, problem := filled(path)
	if problem != "" {
		c.Detail = fmt.Sprintf("%s；下一步：人类确认后跑 `rick tools release --merge-source` 并把 `version=` 与 `rollback_point=` 记进 %s", problem, path)
		return c
	}
	if !rsiVersionRe.MatchString(content) {
		c.Detail = fmt.Sprintf("缺 `version=<sha7>-<ts>`；见 %s", path)
		return c
	}
	if !rsiRollbackRe.MatchString(content) {
		c.Detail = fmt.Sprintf("缺回滚点（`rollback_point=` 或 `released=yes`）；见 %s", path)
		return c
	}
	c.Pass = true
	c.Detail = "已记录提升版本与回滚点"
	return c
}

// checkRSIResume: 重启后的挂起清单与人工恢复结果。
func checkRSIResume(doingDir string) rsiCheck {
	path := rsiEvidencePath(doingDir, "resume.md")
	c := rsiCheck{Name: "resume", Evidence: path}
	content, problem := filled(path)
	if problem != "" {
		c.Detail = fmt.Sprintf("%s；下一步：release 重启后把「挂起清单 + 逐条人工恢复结果」记进 %s", problem, path)
		return c
	}
	if !rsiSuspendedRe.MatchString(content) {
		c.Detail = fmt.Sprintf("未记录挂起清单（缺 `suspended`/`挂起`）；见 %s", path)
		return c
	}
	if !rsiResumedRe.MatchString(content) {
		c.Detail = fmt.Sprintf("未记录人工恢复结果（缺 `resumed`/`恢复`）；见 %s", path)
		return c
	}
	c.Pass = true
	c.Detail = "已记录挂起清单与人工恢复结果"
	return c
}

// checkRSIProdHealth: 生产实时健康 + build_id 与 release 记录的 version 一致。
//
// 为什么必须是实时的：loop 的成功判据是「生产真的跑上了这次构建」，而
// release.md 只是自述 —— 只有 GET /api/health 才能证伪。
func checkRSIProdHealth(doingDir, prodURL string) rsiCheck {
	c := rsiCheck{Name: "prod-health", Evidence: prodURL}
	relPath := rsiEvidencePath(doingDir, "release.md")
	relContent, exists, _ := readRSIEvidence(relPath)
	if !exists {
		c.Detail = fmt.Sprintf("无法校验：%s 不存在（先完成提升并记录）", relPath)
		return c
	}
	m := rsiVersionRe.FindStringSubmatch(relContent)
	if m == nil {
		c.Detail = fmt.Sprintf("无法校验：%s 里没有 `version=`", relPath)
		return c
	}
	want := m[1]
	if strings.TrimSpace(prodURL) == "" {
		c.Detail = "无法确定生产地址；下一步：用 `--prod-url http://host:port` 指定（或设 RICK_RSI_PROD_URL）"
		return c
	}
	got, err := rsiHealthProbe(prodURL)
	if err != nil {
		c.Detail = fmt.Sprintf("生产探测失败（%s）：%v；下一步：确认生产可达（`curl %s/api/health`）", prodURL, err, strings.TrimRight(prodURL, "/"))
		return c
	}
	if got != want {
		c.Detail = fmt.Sprintf("生产 build_id=%q ≠ 本次 version=%q —— 生产没真的跑上这次构建（或另有实例占着端口）；见 %s", got, want, relPath)
		return c
	}
	c.Pass = true
	c.Detail = fmt.Sprintf("生产健康且 build_id=%s 与本次提升一致", got)
	return c
}

// runRSICheck evaluates all rows against <doingDir> and the production URL.
func runRSICheck(jobID, doingDir, prodURL string) rsiCheckResult {
	checks := []rsiCheck{
		checkRSIDevIterations(doingDir),
		checkRSIGates(doingDir),
		checkRSIApproval(doingDir),
		checkRSIRelease(doingDir),
		checkRSIResume(doingDir),
		checkRSIProdHealth(doingDir, prodURL),
	}
	res := rsiCheckResult{Job: jobID, Dir: filepath.Join(doingDir, rsiEvidenceDir), Checks: checks}
	res.Errors = []string{}
	for _, c := range checks {
		if !c.Pass {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %s", c.Name, c.Detail))
		}
	}
	res.Pass = len(res.Errors) == 0
	return res
}

// rsiSkeleton renders the template for one evidence file. Headers state the
// pass criteria so the agent has an unambiguous落盘位置与目标。
func rsiSkeleton(name, file, criteria string) string {
	return fmt.Sprintf(`# RSI 证据：%s

%s

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->

`, name, criteria)
}

// rsiSkeletonCriteria is the per-file 通过标准（与 rick-rsi-loop 产出评估表同源）。
var rsiSkeletonCriteria = map[string]string{
	"dev-iterations.md": "通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。\n示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`",
	"gates.md":          "通过标准：每层门禁一行且全部 pass=true，**不得**出现 pass=false。\n示例：`GATE gate11 pass=true`",
	"approval.md":       "通过标准：含 `APPROVED by=human` 与时间戳（人类确认是本 loop 的硬门槛）。\n示例：`APPROVED by=human at=2026-09-21T20:40:00+08:00`",
	"release.md":        "通过标准：含 `version=<sha7>-<ts>` 与回滚点（`rollback_point=` 或 `released=yes`）。\n示例：\n```\nversion=c539c60-260921192330\nrollback_point=bin/releases/a60035a-260921195959\nreleased=yes\n```",
	"resume.md":         "通过标准：记录挂起清单与人工恢复结果（含 `suspended`/`挂起` 与 `resumed`/`恢复`）。\n示例：\n```\nsuspended=3（本会话 + 2 个 doing job）\nresumed=3（逐条人工点击恢复）\n```",
}

// initRSIEvidence creates <doing>/rsi/ and the five skeletons. Idempotent:
// existing files are never overwritten.
func initRSIEvidence(doingDir string) (created []string, skipped []string, err error) {
	dir := filepath.Join(doingDir, rsiEvidenceDir)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("创建 %s: %w", dir, err)
	}
	for _, item := range rsiEvidenceFiles {
		path := filepath.Join(dir, item.File)
		if _, statErr := os.Stat(path); statErr == nil {
			skipped = append(skipped, path)
			continue
		}
		body := rsiSkeleton(item.Name, item.File, rsiSkeletonCriteria[item.File])
		if writeErr := os.WriteFile(path, []byte(body), 0o644); writeErr != nil {
			return created, skipped, fmt.Errorf("写 %s: %w", path, writeErr)
		}
		created = append(created, path)
	}
	return created, skipped, nil
}

// resolveProdURLForRSI resolves the production health URL: flag > env >
// `<prod-home>/start-web.sh` 或 `<prod-home>/.rick/start-web.sh` 解析 --port >
// 空（由检查项给出可操作指引）。
//
// 刻意不写死 8413：端口可能被换（历史上就换过一次），硬编码会让检查项悄悄测错对象。
func resolveProdURLForRSI(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(rsiProdURLEnv)); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, p := range []string{
		filepath.Join(home, "start-web.sh"),
		filepath.Join(home, ".rick", "start-web.sh"),
	} {
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		if port := parseStartScriptPort(string(data)); port != "" {
			return "http://127.0.0.1:" + port
		}
	}
	return ""
}

const rsiProdURLEnv = "RICK_RSI_PROD_URL"

var rsiPortRe = regexp.MustCompile(`--port\s+([0-9]{2,5})`)

// parseStartScriptPort extracts the --port value from a deployment start script.
func parseStartScriptPort(script string) string {
	if m := rsiPortRe.FindStringSubmatch(script); m != nil {
		return m[1]
	}
	return ""
}

// NewRSICheckCmd implements `rick tools rsi_check`.
func NewRSICheckCmd() *cobra.Command {
	var (
		jobFlag  string
		jsonOut  bool
		initFlag bool
		prodURL  string
	)

	cmd := &cobra.Command{
		Use:   "rsi_check",
		Short: "Validate the rsi-loop evidence of one self-evolution iteration",
		Long: `Validate whether a self-evolution iteration left the evidence required by
.rick/loops/rick-rsi-loop.md（「产出评估」表的 6 行，逐项机器校验）:

  1. dev-iterations  doing/rsi/dev-iterations.md  至少一条构建指纹
  2. gates           doing/rsi/gates.md           每行 pass=true（出现 pass=false 即失败）
  3. approval        doing/rsi/approval.md        APPROVED by=human + 时间戳（最关键）
  4. release         doing/rsi/release.md         version=<sha7>-<ts> + rollback_point=
  5. resume          doing/rsi/resume.md          挂起清单 + 人工恢复结果
  6. prod-health     实时 GET <prod>/api/health   status=ok 且 build_id == 本次 version

loop 是权威：若 loop 的产出评估表变化，本校验器随之对齐（绝不为通过现状而放松）。

Arguments:
  --job       job 编号（默认：全局 --job，否则 <cwd>/.rick/jobs/ 下最近修改的 job）
  --prod-url  生产地址（默认：RICK_RSI_PROD_URL，否则从 start-web.sh 解析 --port）
  --init      幂等生成 doing/rsi/ 五项证据骨架（已存在不覆盖）
  --json      输出一行 JSON 结论

Output:
  ✅ rsi_check passed: job_36 (6/6)
  ❌ rsi_check failed: job_36
     - <检查项>: <原因与下一步>

Exit codes:
  0  全部证据齐备
  1  缺证据（本次迭代未完成）或参数/环境有误`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID, err := resolveRSIJobID(jobFlag)
			if err != nil {
				return rsiCheckFail(cmd, jsonOut, err)
			}
			doingDir, err := workspace.GetJobDoingDir(jobID)
			if err != nil {
				return rsiCheckFail(cmd, jsonOut, fmt.Errorf("解析 job 目录: %w", err))
			}

			if initFlag {
				created, skipped, iErr := initRSIEvidence(doingDir)
				if iErr != nil {
					return rsiCheckFail(cmd, jsonOut, iErr)
				}
				out := cmd.OutOrStdout()
				if jsonOut {
					body, _ := json.Marshal(map[string]any{
						"job": jobID, "dir": filepath.Join(doingDir, rsiEvidenceDir),
						"created": created, "skipped": skipped,
					})
					fmt.Fprintln(out, string(body))
				} else {
					fmt.Fprintf(out, "✅ rsi 证据骨架就绪：%s\n", filepath.Join(doingDir, rsiEvidenceDir))
					for _, p := range created {
						fmt.Fprintf(out, "   + %s\n", filepath.Base(p))
					}
					for _, p := range skipped {
						fmt.Fprintf(out, "   = %s（已存在，未覆盖）\n", filepath.Base(p))
					}
					fmt.Fprintf(out, "   下一步：按各文件头部的通过标准填写，然后 `rick tools rsi_check --job %s`\n", jobID)
				}
				return nil
			}

			res := runRSICheck(jobID, doingDir, resolveProdURLForRSI(prodURL))
			out := cmd.OutOrStdout()
			if jsonOut {
				body, mErr := json.Marshal(res)
				if mErr != nil {
					return fmt.Errorf("marshal result: %w", mErr)
				}
				fmt.Fprintln(out, string(body))
			} else if res.Pass {
				fmt.Fprintf(out, "✅ rsi_check passed: %s (%d/%d)\n", res.Job, len(res.Checks), len(res.Checks))
				for _, c := range res.Checks {
					fmt.Fprintf(out, "   ✓ %-15s %s\n", c.Name, c.Detail)
				}
			} else {
				passed := 0
				for _, c := range res.Checks {
					if c.Pass {
						passed++
					}
				}
				fmt.Fprintf(out, "❌ rsi_check failed: %s (%d/%d)\n", res.Job, passed, len(res.Checks))
				for _, c := range res.Checks {
					mark := "✓"
					if !c.Pass {
						mark = "✗"
					}
					fmt.Fprintf(out, "   %s %-15s %s\n", mark, c.Name, c.Detail)
				}
				fmt.Fprintf(out, "   证据目录：%s\n", res.Dir)
				fmt.Fprintf(out, "   （本 loop 的完成判据 = pass=true；缺证据先 `--init` 生成骨架再填）\n")
			}
			if !res.Pass {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&jobFlag, "job", "", "job 编号（默认：全局 --job 或最近修改的 job）")
	cmd.Flags().StringVar(&prodURL, "prod-url", "", "生产地址（默认 RICK_RSI_PROD_URL 或从 start-web.sh 解析）")
	cmd.Flags().BoolVar(&initFlag, "init", false, "幂等生成 doing/rsi/ 证据骨架（已存在不覆盖）")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "输出一行 JSON 结论")
	return cmd
}

// rsiCheckFail reports a resolution/IO failure in the same shape as a verdict.
func rsiCheckFail(cmd *cobra.Command, jsonOut bool, err error) error {
	if jsonOut {
		body, _ := json.Marshal(rsiCheckResult{Pass: false, Checks: []rsiCheck{}, Errors: []string{err.Error()}})
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	} else {
		fmt.Fprintf(os.Stderr, "❌ rsi_check failed: %v\n", err)
	}
	os.Exit(1)
	return nil
}
