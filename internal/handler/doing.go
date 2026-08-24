package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// Doing executes the complete doing workflow. Scheduling is now delegated to
// pi's workflowScript orchestration (parent single-writer + runs.run with await),
// and the gate is a deterministic rick-side script run after the pi session
// settles. rick's retry loop is only a safety net: it regenerates an
// orchestration of the remaining pending tasks on each attempt, bounded by
// cfg.MaxRetries.
//
// rt is the agent runtime (DIP composition root in cmd constructs the concrete
// implementation and injects it here). Handler depends on the Runtime interface
// only, so a future dsh runtime only needs a new impl + registration without
// touching this orchestrator.
// ResumeDoing resumes a previous doing parent session interactively: reads
// doing/session_id (persisted by piRuntime.Run) and re-enters the same pi
// session (--session-id) so a human (or the agent) can inspect state, fix a
// stuck pipeline, or continue unfinished work with full context.
func ResumeDoing(jobID string, opts Options) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return fmt.Errorf("job %s doing directory does not exist", jobID)
	}
	sessionID, err := loadSessionIDStrict(doingDir, fmt.Sprintf("job %s doing", jobID))
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Printf("恢复 doing 会话：job %s（session %s，交互模式）\n", jobID, sessionID)
	if err := runtime.CallCLI(opts.Verbose, cfg, "", runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("failed to resume pi CLI: %w", err)
	}
	return nil
}

func Doing(jobID string, opts Options, rt runtime.Runtime) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	jobDir := filepath.Join(rickDir, "jobs", jobID)
	planDir := filepath.Join(jobDir, "plan")
	doingDir := filepath.Join(jobDir, "doing")

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return fmt.Errorf("job directory not found: %s", jobDir)
	}
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		return fmt.Errorf("plan directory not found: %s", planDir)
	}
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		return fmt.Errorf("failed to create doing directory: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Job directory: %s\n", jobDir)
		fmt.Printf("[INFO] Plan directory: %s\n", planDir)
		fmt.Printf("[INFO] Doing directory: %s\n", doingDir)
	}

	// Initial tasks.json draft: builder scans plan/task*.md (all pending).
	if err := builder.EnsureTasksJSON(doingDir, planDir); err != nil {
		return fmt.Errorf("failed to initialize tasks.json: %w", err)
	}

	// 确定性前置：cwd 不是 git 仓库时先初始化（worker 的 commit 契约与门禁的
	// commit_hash 校验都依赖仓库存在；不留给 parent agent 纠结）。
	if err := ensureGitRepo(rickDir); err != nil {
		fmt.Printf("[WARN] git repo init failed: %v\n", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	if rt == nil {
		return fmt.Errorf("runtime is nil (composition root must inject a Runtime)")
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if opts.Verbose {
			fmt.Printf("[INFO] Attempt %d/%d\n", attempt, maxRetries)
		}

		// builder regenerates an orchestration of only the remaining pending
		// tasks (completed tasks are filtered out at generation time).
		promptFile, method, _, err := builder.NewPIBuilder().SaveDoingPrompt(doingDir, planDir, rickDir, jobID)
		if err != nil {
			return fmt.Errorf("failed to build doing prompt: %w", err)
		}

		// v4.4.6: 执行反馈——编排摘要（分层结构）+ 运行中实时事件（runtime 打 stderr）。
		printOrchestrationSummary(doingDir, planDir, attempt, maxRetries)

		// v4.4.7: 确定性进度——tasks.json watcher（hook 写状态，watcher 轮询 diff，
		// 状态变更打一行）。pi 会话内 assistant 文本不固定，不作为进度信号。
		watchDone := make(chan struct{})
		go watchTasksJSON(watchDone, doingDir)

		sessionID, trace, err := rt.Run(method, promptFile, cfg)
		close(watchDone)
		if err != nil {
			fmt.Printf("[WARN] pi run did not settle (attempt %d/%d): %v\n", attempt, maxRetries, err)
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("pi run failed after %d attempts: %w", maxRetries, err)
		}
		if trace != nil {
			fmt.Printf("[rick] 会话 %s 结束：耗时 %s，%d 次工具调用，%d 次失败\n",
				sessionID, trace.Duration.Round(time.Second).String(), len(trace.ToolCalls), countErrors(trace.ToolCalls))
		}

		// Persist the pi session ID to the job directory (handler 持久化契约)。
		if sessionID != "" {
			if err := saveSessionID(doingDir, sessionID); err != nil {
				fmt.Printf("[WARN] failed to persist session_id: %v\n", err)
			}
		}

		// Deterministic gate after the session settles (agent_settled).
		if gateErr := runDoingGate(rickDir, doingDir); gateErr != nil {
			fmt.Printf("[WARN] gate failed (attempt %d/%d): %v\n", attempt, maxRetries, gateErr)
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("doing gate failed after %d attempts: %w", maxRetries, gateErr)
		}

		fmt.Printf("Job %s execution completed!\n", jobID)
		fmt.Printf("\n下一步（沉淀本次执行的知识——不跑则 domain/skills 不更新，经验会丢）：\n")
		fmt.Printf("  rick learning %s     # 单 job 沉淀（SUMMARY + skills/loops + domain 事实）\n", jobID)
		fmt.Printf("  rick dream            # 跨 job 全局反思（可攒多个 job 一起跑，演化 loops/domain）\n")
		return nil
	}

	return fmt.Errorf("job execution incomplete after %d attempts", maxRetries)
}

// runDoingGate runs the deterministic rick-side gate script:
// python3 .rick/skills/rick-gates/helper.py <doingDir>. Exit non-zero = gate
// failure (unparseable tasks.json / zombie running / success without commit_hash).
// v4.4.6: 工作仓库的 .rick/skills/rick-gates/ 可能不存在（非 rick 仓库的
// doing 工作区）——降级到 env 已部署副本（~/.rick/pi/agent/extensions/
// rick-gates/helper.py，init-pi 职责 3 落盘）；两处都没有则跳过本兜底
//（层门禁 pipeline_gate/level_complete 才是主验收，helper 只做终态校验）。
func runDoingGate(rickDir, doingDir string) error {
	helper := filepath.Join(rickDir, "skills", "rick-gates", "helper.py")
	if _, err := os.Stat(helper); err != nil {
		home, homeErr := os.UserHomeDir()
		deployed := ""
		if homeErr == nil {
			deployed = filepath.Join(home, ".rick", "pi", "agent", "extensions", "rick-gates", "helper.py")
		}
		if deployed != "" {
			if _, err2 := os.Stat(deployed); err2 == nil {
				helper = deployed
			} else {
				fmt.Println("[rick] ⚠️ rick-gates helper 不可达（工作仓库与部署目录均无）——跳过终态兜底校验（层门禁为主验收）")
				return nil
			}
		} else {
			fmt.Println("[rick] ⚠️ rick-gates helper 不可达（无法定位 HOME）——跳过终态兜底校验（层门禁为主验收）")
			return nil
		}
	}
	cmd := exec.Command("python3", helper, doingDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// DoingDryRun generates and prints the doing prompt (with the pi workflowScript
// orchestration) without executing it.
func DoingDryRun(jobID string) error {
	if jobID == "" {
		fmt.Println("[DRY-RUN] No job ID provided")
		return nil
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	planDir := filepath.Join(rickDir, "jobs", jobID, "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		doingDirFallback := filepath.Join(rickDir, "jobs", jobID, "doing")
		if _, e := os.Stat(filepath.Join(doingDirFallback, "requirement.md")); e == nil {
			fmt.Printf("[DRY-RUN] %s is an easy mode job (no plan/). Use: rick doing --easy --dry-run\n", jobID)
		} else {
			fmt.Printf("[DRY-RUN] plan directory not found: %s\n", planDir)
		}
		return nil
	}

	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")

	promptFile, _, _, err := builder.NewPIBuilder().SaveDoingPrompt(doingDir, planDir, rickDir, jobID)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate prompt: %v\n", err)
		return nil
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to read prompt file: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Doing prompt:\n\n")
	fmt.Println(string(content))
	return nil
}

// ensureGitRepo 保证 rickDir 的父目录（工作仓库根）是 git 仓库：已初始化则跳过；
// 否则 git init + 初始 commit（把当前工作区纳入版本管理）。幂等。
func ensureGitRepo(rickDir string) error {
	repoRoot := filepath.Dir(rickDir)
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
		return nil
	}
	fmt.Printf("[INFO] %s is not a git repo — initializing ...\n", repoRoot)
	if out, err := exec.Command("git", "-C", repoRoot, "init").CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %v: %s", err, string(out))
	}
	if out, err := exec.Command("git", "-C", repoRoot, "add", "-A").CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A: %v: %s", err, string(out))
	}
	commit := exec.Command("git", "-C", repoRoot, "commit", "-m", "chore(rick): initial commit before doing")
	commit.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=rick", "GIT_AUTHOR_EMAIL=rick@local",
		"GIT_COMMITTER_NAME=rick", "GIT_COMMITTER_EMAIL=rick@local")
	if out, err := commit.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "nothing to commit") &&
			!strings.Contains(string(out), "no changes added to commit") {
			return fmt.Errorf("git commit: %v: %s", err, string(out))
		}
	}
	fmt.Printf("[INFO] git repo initialized at %s\n", repoRoot)
	return nil
}

// printOrchestrationSummary prints the pending-task pipeline snapshot before
// a pi run: layer structure + per-task status (real-time feedback part 1).
func printOrchestrationSummary(doingDir, planDir string, attempt, maxRetries int) {
	tasksJSON := filepath.Join(doingDir, "tasks.json")
	data, err := os.ReadFile(tasksJSON)
	if err != nil {
		return
	}
	var tj struct {
		Tasks []struct {
			TaskID string   `json:"task_id"`
			Status string   `json:"status"`
			Deps   []string `json:"dependencies"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &tj); err != nil {
		return
	}
	pending, total := 0, 0
	for _, t := range tj.Tasks {
		total++
		if t.Status != "success" {
			pending++
		}
	}
	fmt.Printf("[rick] ── doing 执行（第 %d/%d 轮）：%d/%d task 待完成 ──\n", attempt, maxRetries, pending, total)
	for _, t := range tj.Tasks {
		mark := "✅"
		if t.Status != "success" {
			mark = "⬜"
		}
		fmt.Printf("[rick]   %s %s\n", mark, t.TaskID)
	}
}

func countErrors(calls []runtime.ToolCall) int {
	n := 0
	for _, c := range calls {
		if c.IsError {
			n++
		}
	}
	return n
}

// taskSnapshot is a point-in-time view of tasks.json statuses.
type taskSnapshot map[string]struct {
	Status     string
	CommitHash string
}

// readTaskSnapshot loads tasks.json into id → (status, commit) pairs; a
// missing/unparsable file returns an empty snapshot (watcher stays silent).
func readTaskSnapshot(doingDir string) taskSnapshot {
	snap := taskSnapshot{}
	data, err := os.ReadFile(filepath.Join(doingDir, "tasks.json"))
	if err != nil {
		return snap
	}
	var tj struct {
		Tasks []struct {
			TaskID     string `json:"task_id"`
			Status     string `json:"status"`
			CommitHash string `json:"commit_hash"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &tj); err != nil {
		return snap
	}
	for _, t := range tj.Tasks {
		snap[t.TaskID] = struct {
			Status     string
			CommitHash string
		}{t.Status, t.CommitHash}
	}
	return snap
}

// watchTasksJSON polls tasks.json every 2s and prints one line per status
// change until done is closed. Deterministic progress: the rick-gates hook is
// the only writer, so every transition (level_complete batch success, manual
// fixes) surfaces here immediately.
func watchTasksJSON(done <-chan struct{}, doingDir string) {
	last := readTaskSnapshot(doingDir)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			// 收尾时补一次 diff（Run 返回前的最后一次写可能未被 tick 捕获）
			printTaskDiff(last, readTaskSnapshot(doingDir))
			return
		case <-ticker.C:
			cur := readTaskSnapshot(doingDir)
			printTaskDiff(last, cur)
			last = cur
		}
	}
}

// printTaskDiff prints one progress line per changed/new task.
func printTaskDiff(last, cur taskSnapshot) {
	// 稳定输出顺序：按新增/变更的 task id 排序
	type change struct {
		id   string
		from string
		to   string
		hash string
	}
	var changes []change
	for id, s := range cur {
		prev, ok := last[id]
		if !ok {
			changes = append(changes, change{id: id, from: "new", to: s.Status, hash: s.CommitHash})
			continue
		}
		if prev.Status != s.Status || (s.Status == "success" && prev.CommitHash != s.CommitHash) {
			changes = append(changes, change{id: id, from: prev.Status, to: s.Status, hash: s.CommitHash})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].id < changes[j].id })
	for _, c := range changes {
		switch {
		case c.to == "success":
			fmt.Printf("[rick] ✅ %s 完成（commit %s）\n", c.id, shortHash(c.hash))
		case c.to == "running":
			fmt.Printf("[rick] 🔄 %s 开始执行\n", c.id)
		case c.from == "new":
			fmt.Printf("[rick] ＋ %s（%s）\n", c.id, c.to)
		default:
			fmt.Printf("[rick] %s：%s → %s\n", c.id, c.from, c.to)
		}
	}
}

func shortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}
