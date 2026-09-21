package release

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// mergeStage 是源码合并失败的阶段名。它没有独立的退出码：ExitCodeForStage 的
// default 分支把未知阶段（含 merge）映射为 ExitRestart(4)——合并失败属于「提升
// 没做成」，与 promote/restart 同类，不应发明新的退出码。
const mergeStage = "merge"

// MergeResult 是一次源码合并的结果（CLI 据此打印 RELEASE_MERGE 回执；也会被塞进
// Result 供 `--json` 与 rsi_check 取证）。
type MergeResult struct {
	Merged bool   `json:"merged"`
	Branch string `json:"branch,omitempty"` // dev 分支名（被合并方）
	Main   string `json:"main,omitempty"`   // 生产主分支名（合并目标）
	Commit string `json:"commit,omitempty"` // merge commit 短 sha
	Files  int    `json:"files,omitempty"`  // 合并引入的变更文件数
	Reason string `json:"reason,omitempty"` // merged=false 的原因（already up to date）
}

// MergeSource 把 dev 工作树的分支合并进生产仓库当前分支（human 裁决 J-RSI-2：
// **冲突即终止报错**，把现场留给 AI 修复后重新 release，绝不自动解冲突）。
//
// 顺序与安全边界：
//  1. 解析 dev 分支 / 生产分支（detached HEAD 一律拒绝——「合并谁」必须无歧义）
//  2. 拒绝带未完成合并（MERGE_HEAD）的生产树
//  3. **前置校验生产工作树干净**：不干净直接中止，且不产生任何改动
//     （生产树里常有未提交的 job 文件，错误信息必须点明「先提交或暂存」，
//     否则用户只会看到笼统的 merge 失败）
//  4. dev 未领先 → merged=false（不算失败，幂等）
//  5. `git merge --no-ff --no-commit <devBranch>`；冲突 → `git merge --abort`
//     回滚到干净状态 + 返回含冲突清单与「怎么修」的错误
//  6. 无冲突 → 提交 merge commit（生产仓库缺 git 身份时补一个中性身份，
//     避免卡在 "Please tell me who you are"）
//
// 本函数只动**生产仓库工作树的 git 状态**（提交合并），不碰发布链、覆盖层与
// 在跑的进程——那些是 Promote/Restart 的职责。
func (p Plan) MergeSource() (MergeResult, error) {
	var res MergeResult

	if strings.TrimSpace(p.ProdRepo) == "" || strings.TrimSpace(p.DevTree) == "" {
		return res, mergeErrf("生产仓库或 dev 工作树路径为空，无法合并源码")
	}
	if !dirExists(p.ProdRepo) {
		return res, mergeErrf("生产仓库不存在: %s", p.ProdRepo)
	}
	if !dirExists(p.DevTree) {
		return res, mergeErrf("dev 工作树不存在: %s", p.DevTree)
	}

	devBranch, err := gitOut(p.DevTree, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return res, mergeErrf("无法解析 dev 工作树的分支: %v", err)
	}
	if devBranch == "" || devBranch == "HEAD" {
		return res, mergeErrf("dev 工作树处于 detached HEAD，无法确定要合并的分支（先 `git -C %s checkout <branch>`）", p.DevTree)
	}
	mainBranch, err := gitOut(p.ProdRepo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return res, mergeErrf("无法解析生产仓库的分支: %v", err)
	}
	if mainBranch == "" || mainBranch == "HEAD" {
		return res, mergeErrf("生产仓库处于 detached HEAD，无法确定合并目标分支（先 `git -C %s checkout <main-branch>`）", p.ProdRepo)
	}
	if devBranch == mainBranch {
		return res, mergeErrf("dev 工作树与生产仓库在**同一分支** %s 上——源码合并无意义（dev 必须是独立 worktree/分支）", devBranch)
	}
	res.Branch, res.Main = devBranch, mainBranch

	// 未完成的合并：直接拒绝（此时 merge/abort 都会踩到别人的现场）
	if sha, err := gitOut(p.ProdRepo, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil && sha != "" {
		return res, mergeErrf(
			"生产仓库存在未完成的合并（MERGE_HEAD=%s）——请先 `git -C %s merge --abort` 或完成该合并后重试",
			short(sha), p.ProdRepo)
	}

	// 生产工作树必须干净（不干净时 merge 可能覆盖本地改动，且失败后现场不可信）
	dirty, err := gitOut(p.ProdRepo, "status", "--porcelain")
	if err != nil {
		return res, mergeErrf("无法读取生产仓库状态: %v", err)
	}
	if strings.TrimSpace(dirty) != "" {
		return res, mergeErrf(
			"生产工作树不干净，已中止源码合并（**未产生任何改动**）。请先提交或暂存以下改动后重试：\n%s\n"+
				"提示：`--no-merge-source` 可只提升产物、不动源码。",
			indentBlock(dirty, "    "))
	}

	// dev 未领先 → 幂等成功（不是失败）
	ahead, err := gitOut(p.ProdRepo, "rev-list", "--count", mainBranch+".."+devBranch)
	if err != nil {
		return res, mergeErrf("无法比较 %s..%s（该分支在生产仓库里可见吗？）: %v", mainBranch, devBranch, err)
	}
	if strings.TrimSpace(ahead) == "0" {
		res.Merged = false
		res.Reason = "already up to date"
		return res, nil
	}

	// --no-commit：先把合并结果摆好，冲突时才能干净地 abort 而不留半成品提交
	if out, err := gitCombined(p.ProdRepo, "merge", "--no-ff", "--no-commit", devBranch); err != nil {
		conflicts, _ := gitOut(p.ProdRepo, "diff", "--name-only", "--diff-filter=U")
		abortErr := gitAbortMerge(p.ProdRepo)
		after, _ := gitOut(p.ProdRepo, "status", "--porcelain")
		var b strings.Builder
		b.WriteString("源码合并存在冲突，已中止（未提交、未提升）：\n")
		if strings.TrimSpace(conflicts) != "" {
			b.WriteString("冲突文件：\n" + indentBlock(conflicts, "    ") + "\n")
		} else {
			b.WriteString("冲突文件：git 未列出（详见下）\n")
		}
		b.WriteString("git 输出：\n" + indentBlock(truncateTail(strings.TrimSpace(out), 600), "    ") + "\n")
		switch {
		case abortErr != nil:
			b.WriteString(fmt.Sprintf("⚠ `git merge --abort` 未能完成: %v——请人工检查 %s 的合并状态\n", abortErr, p.ProdRepo))
		case strings.TrimSpace(after) != "":
			b.WriteString(fmt.Sprintf("⚠ 已执行 `git merge --abort`，但工作树仍有残留（%s）——请人工检查\n",
				strings.Join(strings.Fields(strings.TrimSpace(after)), " ")))
		default:
			b.WriteString(fmt.Sprintf("已执行 `git merge --abort` 自动回滚：%s 的工作树已恢复到合并前状态（无残留、无提交）。\n", p.ProdRepo))
		}
		b.WriteString("下一步：AI 修复冲突（可 `git -C " + p.ProdRepo + " merge " + devBranch + "` 手动合并并解决冲突、提交到生产分支），" +
			"然后重跑 `rick tools release --merge-source`；" +
			"若本次只想提升产物、不动源码，可改用 `rick tools release --no-merge-source`。")
		return res, mergeErr(b.String())
	}

	// 无冲突：提交合并
	msg := fmt.Sprintf("chore(release): merge %s → %s (version=%s)", devBranch, mainBranch, p.mergeVersionOrNew())
	commitArgs := []string{"commit", "-m", msg}
	if email, err := gitOut(p.ProdRepo, "config", "user.email"); err != nil || strings.TrimSpace(email) == "" {
		// 生产仓库没配 git 身份时补一个中性身份：release 不该因为缺 user.email 而停在半路
		commitArgs = append([]string{"-c", "user.name=rick release", "-c", "user.email=rick-release@localhost"}, commitArgs...)
	}
	if out, err := gitCombined(p.ProdRepo, commitArgs...); err != nil {
		// 提交失败时把合并现场留给人（不回滚：合并结果本身是好的，只是没提交）
		return res, mergeErrf(
			"源码合并已完成但提交失败（合并状态保留在生产工作树，请人工提交）：\n%s",
			indentBlock(truncateTail(strings.TrimSpace(out), 600), "    "))
	}

	res.Merged = true
	if sha, err := gitOut(p.ProdRepo, "rev-parse", "--short=7", "HEAD"); err == nil {
		res.Commit = sha
	}
	// merge commit 是双亲提交：`git show` 默认不展示 diff，必须显式与**第一父**
	// （即生产分支一侧）比较，才等于「本次合并带进来的文件」。
	if files, err := gitOut(p.ProdRepo, "diff", "--name-only", "HEAD^1", "HEAD"); err == nil {
		for _, line := range strings.Split(files, "\n") {
			if strings.TrimSpace(line) != "" {
				res.Files++
			}
		}
	}
	return res, nil
}

// mergeVersionOrNew 返回合并提交信息里用的版本号：优先本次提升的版本（CLI 会设
// MergeVersion），否则现取一个时间戳版本（`rick` 的版本号本身带时间）。
func (p Plan) mergeVersionOrNew() string {
	if v := strings.TrimSpace(p.MergeVersion); v != "" {
		return v
	}
	return p.NewVersion()
}

// ---- git 小工具（本包内私有；只作用于调用者给的目录，绝不隐式使用真实仓库）----

// gitOut 在 dir 里执行 git，返回 trim 后的 stdout（stderr 仅用于报错）。
func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(stdout.String()),
			fmt.Errorf("git %s: %s", strings.Join(args, " "), truncateTail(msg, 300))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// gitCombined 在 dir 里执行 git，返回 stdout+stderr 合并文本（用于把 git 的
// 冲突输出完整转述给用户）。
func gitCombined(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	return buf.String(), err
}

// gitAbortMerge 中止未完成的合并，尽量把工作树恢复到合并前（幂等：没有合并
// 在进行中时 git 会报错，这里视为「已经是干净的」）。
func gitAbortMerge(dir string) error {
	if _, err := gitOut(dir, "rev-parse", "-q", "--verify", "MERGE_HEAD"); err != nil {
		return nil // 没有 MERGE_HEAD = 没在合并中
	}
	if _, err := gitCombined(dir, "merge", "--abort"); err != nil {
		return fmt.Errorf("git merge --abort 失败: %v", err)
	}
	return nil
}

// mergeErrf 构造合并失败错误（stage=merge）。
func mergeErrf(format string, args ...any) error {
	return &StageError{Stage: mergeStage, Err: fmt.Errorf(format, args...)}
}

// mergeErr 包装既有错误为 stage=merge。
func mergeErr(msg string) error {
	return &StageError{Stage: mergeStage, Err: fmt.Errorf("%s", msg)}
}

// indentBlock 给多行文本加统一缩进（回执可读性）。
func indentBlock(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// truncateTail 保留尾部 n 个字符（git 报错的关键信息都在末尾）。
func truncateTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
