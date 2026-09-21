package prompt

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// RSI（Recursive Self-Improvement）会话：把「改进 rick 自身」这件事绑定到一条
// loop 上，使「启动 rick 改进」这个动作**必然携带制度化的流程**，而不是依赖
// agent 自觉去 .rick/loops/ 里找。
//
// 三段职责（本文件）：
//   1. 制度载体定位：RSILoopPath（<ws>/.rick/loops/rick-rsi-loop.md）
//   2. fail-fast 校验：ValidateRSIWorkspace —— 必须是 rick 源码树、必须含该 loop、
//      且不得是生产仓库根（在生产树里跑 RSI 会话 = 直接改生产源码）
//   3. 注入产物：BuildRSIPrompt 产出「bootstrap 提示词（内嵌 loop 全文）」+ methodFile
//      （= loop 自身的绝对路径，供 --append-system-prompt 与 resume 复用）
//
// 交付路径（dev 工作区）与生产仓库的区分见 RSIProdRepoRoot 的注释。

const (
	// RSILoopFileName 是制度载体的文件名（唯一的 loop 来源）。
	RSILoopFileName = "rick-rsi-loop.md"
	// RSIAllowProdTreeEnv 显式放行「workspace == 生产仓库根」的逃生开关，
	// 仅供 E2E 的模拟生产使用（真实使用中绝不应设置）。
	RSIAllowProdTreeEnv = "RICK_RSI_ALLOW_PROD_TREE"
	// RSIProdRepoEnv 显式声明生产仓库根（优先于自动探测）——测试与运维显式
	// 覆盖用；自动探测见 RSIProdRepoRoot。
	RSIProdRepoEnv = "RICK_RSI_PROD_REPO"
	// rsiDraftDirName 是 <rickDir>/draft/ 下分配 RSI 会话目录的名字。
	rsiDraftDirName = "rsi"
)

// RSILoopPath returns <workspacePath>/.rick/loops/rick-rsi-loop.md.
func RSILoopPath(workspacePath string) string {
	return filepath.Join(workspacePath, ".rick", "loops", RSILoopFileName)
}

// ValidateRSIWorkspace fail-fast 校验 RSI 会话的工作区（web 与 CLI 共用同一套
// 判据，避免两条入口出现不一致的宽松度）。错误信息面向用户（中文、含下一步）。
func ValidateRSIWorkspace(workspacePath string) error {
	if strings.TrimSpace(workspacePath) == "" {
		return fmt.Errorf("RSI 会话需要一个工作区路径（当前为空）")
	}
	ws := filepath.Clean(workspacePath)
	st, err := os.Stat(ws)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("RSI 工作区不存在或不是目录：%s", ws)
	}

	// ① 必须是 rick 源码树：RSI 的产出就是编译后的 rick 本身。
	if st, err := os.Stat(filepath.Join(ws, "cmd", "rick", "main.go")); err != nil || st.IsDir() {
		return fmt.Errorf("RSI 会话必须指向 rick 源码树：%s 下没有 cmd/rick/main.go", ws)
	}
	if st, err := os.Stat(filepath.Join(ws, "internal", "web")); err != nil || !st.IsDir() {
		return fmt.Errorf("RSI 会话必须指向 rick 源码树：%s 下没有 internal/web/", ws)
	}

	// ② 必须有制度载体：没有 loop 的 RSI 会话等于没有流程。
	loop := RSILoopPath(ws)
	if st, err := os.Stat(loop); err != nil || st.IsDir() {
		return fmt.Errorf(
			"RSI 会话需要 loop 文件：%s 不存在。请先在 dev 工作树补齐该 loop（`rick tools dev-web init` 会准备 dev 环境），再启动 RSI 会话",
			loop)
	}

	// ③ 绝不能在生产仓库根上跑：RSI 会话会直接编辑工作区源码，
	//    在生产树上编辑 = 绕过隔离与 release 门禁。
	if strings.TrimSpace(os.Getenv(RSIAllowProdTreeEnv)) != "1" {
		if prod := RSIProdRepoRoot(); prod != "" && sameCleanPath(ws, prod) {
			return fmt.Errorf(
				"RSI 会话必须在 dev 工作区运行：当前指向生产仓库根 %s —— 在那里编辑会直接改动生产源码、绕过 release 门禁。请改用 dev 工作区（`rick tools dev-web init` 产出），或（仅供模拟环境）设置 %s=1",
				prod, RSIAllowProdTreeEnv)
		}
	}
	return nil
}

// RSIProdRepoRoot 尽力定位**生产仓库根**（用于拒绝在生产树上开 RSI 会话）。
// 两个来源，任一命中即可：
//  1. 当前可执行文件的 git 祖先仓库：prod 的 bin/rick（或 bin/releases/<ver>/rick）
//     向上一定能找到生产仓库根；若祖先里 .git 是**文件**（worktree），则解析其
//     gitdir 得到真正的主仓库根。
//  2. 部署脚本兜底：<真实用户家目录>/.rick/start-web.sh 里的 `cd <生产仓库>`。
//     dev 实例的二进制位于 dev HOME 下（不在任何 git 仓库内），来源 1 找不到；
//     兜底让 dev 实例同样能识别并拒绝生产仓库根。
//
// 找不到时返回 ""（此时只依赖规则 ①②）。
func RSIProdRepoRoot() string {
	if v := strings.TrimSpace(os.Getenv(RSIProdRepoEnv)); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs
		}
		return filepath.Clean(v)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if root := gitMainRepoRootFrom(filepath.Dir(exe)); root != "" {
			return root
		}
	}
	return prodRepoFromDeployScript()
}

// gitMainRepoRootFrom walks up from dir looking for a git repository and returns
// the *main* repository root (a linked worktree reports the repo it belongs to).
// Returns "" when no repository is found before the filesystem root.
func gitMainRepoRootFrom(dir string) string {
	cur := filepath.Clean(dir)
	for i := 0; i < 16; i++ {
		gitPath := filepath.Join(cur, ".git")
		if st, err := os.Stat(gitPath); err == nil {
			if st.IsDir() {
				return cur // 主仓库（生产检出）
			}
			// worktree：.git 是文件，内容形如 `gitdir: <repo>/.git/worktrees/<name>`
			if data, err := os.ReadFile(gitPath); err == nil {
				if main := mainRepoFromGitdir(string(data)); main != "" {
					return main
				}
			}
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
	return ""
}

// mainRepoFromGitdir parses a worktree `.git` file's `gitdir:` pointer and
// returns the main repository root ("" when the shape is unexpected).
func mainRepoFromGitdir(content string) string {
	line := strings.TrimSpace(content)
	for _, l := range strings.Split(line, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "gitdir:") {
			line = strings.TrimSpace(strings.TrimPrefix(l, "gitdir:"))
			break
		}
	}
	if line == "" || !filepath.IsAbs(line) {
		return ""
	}
	// <repo>/.git/worktrees/<name> → <repo>
	clean := filepath.Clean(line)
	marker := string(filepath.Separator) + ".git" + string(filepath.Separator) + "worktrees" + string(filepath.Separator)
	if idx := strings.Index(clean, marker); idx > 0 {
		return clean[:idx]
	}
	return ""
}

// prodRepoFromDeployScript reads `<realUserHome>/.rick/start-web.sh` (the
// production launch script, same convention the release command parses) and
// returns the repo it `cd`s into. Read-only; "" when absent/unparsable.
func prodRepoFromDeployScript() string {
	u, err := userHomeDirReal()
	if err != nil || u == "" {
		return ""
	}
	for _, name := range []string{filepath.Join(".rick", "start-web.sh"), "start-web.sh"} {
		data, err := os.ReadFile(filepath.Join(u, name))
		if err != nil {
			continue
		}
		if repo := repoFromDeployScript(string(data)); repo != "" {
			return repo
		}
	}
	return ""
}

// repoFromDeployScript extracts the first `cd <path>` target of a deploy script.
func repoFromDeployScript(content string) string {
	for _, l := range strings.Split(content, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "cd ") {
			continue
		}
		target := strings.TrimSpace(strings.TrimPrefix(l, "cd "))
		target = strings.Trim(target, `"'`)
		if !filepath.IsAbs(target) {
			continue
		}
		return filepath.Clean(target)
	}
	return ""
}

func sameCleanPath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// EnsureRSIDirs allocates the next <rickDir>/draft/rsi/rsi_<n> directory
// (mirrors the human-loop draft/loops/loop_N convention) and returns it.
func EnsureRSIDirs(rickDir string) (string, error) {
	base := filepath.Join(rickDir, "draft", rsiDraftDirName)
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", fmt.Errorf("create RSI draft dir %s: %w", base, err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("read RSI draft dir %s: %w", base, err)
	}
	maxN := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "rsi_%d", &n); err == nil && n > 0 && n <= 9999 && n > maxN {
			maxN = n
		}
	}
	dir := filepath.Join(base, fmt.Sprintf("rsi_%d", maxN+1))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create RSI session dir %s: %w", dir, err)
	}
	return dir, nil
}

// BuildRSIPrompt renders the RSI bootstrap prompt (loop 全文内嵌) and returns it
// together with methodFile —— the loop's own absolute path, so the spawn/resume
// layer can re-inject the authoritative copy via --append-system-prompt.
//
// workspacePath 是工作区根（不是 .rick）：校验按工作区根进行。
func BuildRSIPrompt(workspacePath string) (prompt string, methodFile string, err error) {
	if err := ValidateRSIWorkspace(workspacePath); err != nil {
		return "", "", err
	}
	ws := filepath.Clean(workspacePath)
	loopPath := RSILoopPath(ws)
	data, err := os.ReadFile(loopPath)
	if err != nil {
		return "", "", fmt.Errorf("读取 loop 失败（%s）：%w", loopPath, err)
	}
	loop := string(data)
	if strings.TrimSpace(loop) == "" {
		return "", "", fmt.Errorf("loop 文件为空：%s（RSI 会话无法在没有流程的情况下启动）", loopPath)
	}
	return buildRSIBootstrap(ws, loopPath, loop), loopPath, nil
}

// buildRSIBootstrap 组装 bootstrap 提示词：先给「本次会话是什么 + 硬约束 + 机制
// 命令表」，再内嵌 loop 全文（权威副本另见 methodFile，随系统提示词常驻）。
func buildRSIBootstrap(ws, loopPath, loop string) string {
	var b strings.Builder
	b.WriteString("# RSI 会话：rick 自进化（由 rick-rsi-loop 驱动）\n\n")
	b.WriteString("本次会话的唯一职责是**改进 rick 自身**（源码 `cmd/` `internal/` `web/`、prompt 模板、或 `.rick` 知识库），并让改动**安全地**生效到生产。\n")
	b.WriteString("后端已把 `rick-rsi-loop` 注入本会话——**你必须完整遵循它的状态机（S0→S7），不得自行发明流程**。\n\n")

	b.WriteString("## 制度载体（权威副本）\n\n")
	fmt.Fprintf(&b, "- loop 路径：`%s`\n", loopPath)
	b.WriteString("- 本提示词下方内嵌了该 loop 的全文（与上述文件同源）；`--append-system-prompt` 也已注入同一份 loop，resume 后依然生效。\n")
	b.WriteString("- 本次迭代的**完成判据由机器校验**：`rick tools rsi_check --job <job> --json` → `pass=true`（缺证据即未完成）。\n\n")

	b.WriteString("## 机制工具（loop 的执行面，全部由 rick 提供）\n\n")
	b.WriteString("| 命令 | 用途 |\n|---|---|\n")
	b.WriteString("| `rick tools dev-web init` / `status` / `up` / `restart` / `down` | 隔离 dev 实例（8414）：改后端后 `restart` 会重建并复核构建指纹 |\n")
	b.WriteString("| `npm --prefix web run build`（在 dev 树） | 改前端：产物直接热更（dev overlay 是指向 dev 树 dist 的软链，刷新即生效） |\n")
	b.WriteString("| `python3 .rick/jobs/<job>/plan/gates/gateN.py` | 层门禁（逐层必须全绿才下钻） |\n")
	b.WriteString("| `rick tools release --dry-run` | 演练：给人类看计划，不碰生产 |\n")
	b.WriteString("| `rick tools release --merge-source`（**人类确认后**） | 源码合并 + 二进制/dist 原子换链 + 重启 + build_id 校验；冲突即中止 |\n")
	b.WriteString("| `rick tools rsi_check --job <job> [--init]` | 产出评估（机器校验）与证据骨架 |\n\n")

	b.WriteString("## 硬约束（不可协商）\n\n")
	b.WriteString("1. **只在 dev 工作区编辑**（当前工作区即 dev 工作区）；生产仓库工作树只读。\n")
	b.WriteString("2. 不得 kill/重启生产实例（8413）；生产只经**人类确认后**的 `rick tools release` 被改动。\n")
	b.WriteString("3. **不自动续跑**被打断的会话（重启后一律挂起，由人类一键恢复）——自动重放工具调用有重复副作用风险。\n")
	b.WriteString("4. 进展只以**门禁 `pass=true`** 为准，不以「代码看起来对」为准。\n\n")

	b.WriteString("## 落盘位置（`rsi_check` 逐个校验）\n\n")
	fmt.Fprintf(&b, "`%s/.rick/jobs/<job>/doing/rsi/{dev-iterations.md,gates.md,approval.md,release.md,resume.md}`\n", ws)
	b.WriteString("（缺骨架时先跑 `rick tools rsi_check --job <job> --init`。）\n\n")

	b.WriteString("---\n\n## Loop 全文（权威副本：上述 loop 路径）\n\n")
	b.WriteString(loop)
	if !strings.HasSuffix(loop, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

// EnsureRSISessionID returns <dir>/session_id, creating it (uuid v4) when absent
// —— 与 human-loop 的 `--resume loop_N` 语义对齐：RSI 会话同样可被恢复。
func EnsureRSISessionID(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create RSI session dir: %w", err)
	}
	path := filepath.Join(dir, "session_id")
	if data, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id, nil
		}
	}
	id, err := newUUIDv4()
	if err != nil {
		return "", fmt.Errorf("generate RSI session id: %w", err)
	}
	if err := os.WriteFile(path, []byte(id), 0644); err != nil {
		return "", fmt.Errorf("persist RSI session id: %w", err)
	}
	return id, nil
}

// newUUIDv4 mirrors handler.generateUUID / web.newUUID (crypto/rand v4).
// 第三份实现是已知重复：待后续抽成共享 helper（本 task 的写域不允许改那些文件）。
func newUUIDv4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// userHomeDirReal returns the *real* user's home (passwd entry), independent of
// the $HOME the process was started with —— dev 实例用换 HOME 隔离，但仍需能
// 定位真实用户家目录下的生产部署脚本。
func userHomeDirReal() (string, error) {
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return u.HomeDir, nil
	}
	return os.UserHomeDir()
}
