// Package devweb 实现 `rick tools dev-web` 的编排逻辑：把「隔离的 dev 实例」
// （独立 worktree + 独立 HOME/状态目录 + 独立 pi 沙盒 + overlay 软链）的
// 初始化、构建、起停与健康/指纹校验做成可重入的函数，供 CLI 与测试复用。
//
// 为什么需要它（调研 research-L5 §2/§4/§5）：
//   - 生产内核不能因为「在它自己里面改代码」而受影响 —— dev 必须是一个**完全
//     独立的实例**（端口/HOME/agent dir/二进制/dist 全隔离）；
//   - 「改动生效了吗」必须能被一条命令判定：因此构建产物用**唯一文件名**
//     （`rick.dev.<buildID>`）当指纹，起新进程后再用 `/api/health` 的 build_id
//     复核，而不是靠人肉看日志；
//   - 构建依赖必须显式复用生产缓存（GOCACHE/GOMODCACHE/npm cache）：HOME 一旦
//     切到 dev，这些缓存会全部落空，`go build` 会退化成冷构建甚至触发 214MB
//     工具链下载（实测）。
//
// 纪律（task17 规格，违反即误伤生产）：
//   - 本包只操作 dev 树与 dev HOME；生产 `~/.rick` 仅**只读**读取 pi 种子文件；
//   - 任何 TERM/KILL 之前必须断言目标进程属于该 dev HOME
//     （/proc/<pid>/environ 或 cwd），断言不通过就**拒绝**，宁可不杀。
package devweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// 默认值与环境变量覆盖（全部显式，方便测试与脚本注入）。
const (
	// DefaultPort 是 dev 实例的默认端口。8414 在调研中实测空闲（生产占用
	// 8410-8413/8415/8420）。
	DefaultPort = 8414
	// EnvTree/EnvHome/EnvPort 允许脚本/测试重定向 dev 布局（默认按 prod 仓库
	// 的祖父目录推导，与人工搭建的 /workdir/sunquan20/rick-dev 一致）。
	EnvTree = "RICK_DEV_TREE"
	EnvHome = "RICK_DEV_HOME"
	EnvPort = "RICK_DEV_PORT"
	// DevBinPrefix 是 dev 二进制前缀：`<prefix><buildID>`，文件名即构建指纹。
	DevBinPrefix = "rick.dev."
	// listenAddr 固定 0.0.0.0：用户的真实工作流是局域网浏览器访问，dev UI 也
	// 必须能从别的机器打开（human 裁决 J-L4-4）。
	listenAddr = "0.0.0.0"
	// stopGrace 是 TERM → KILL 的宽限（有 SSE 长连接时优雅退出实测 ~10s）。
	stopGrace = 12 * time.Second
	// healthTimeout 是起新进程后的健康轮询预算。
	healthTimeout = 20 * time.Second
	// outputTail 是构建失败时回传给调用者的 stderr 尾部长度。
	outputTail = 2048
)

// Layout 描述一个 dev 实例的全部路径与运行参数（纯数据，便于测试注入）。
type Layout struct {
	Tree     string // dev 源码工作树（git worktree）
	Home     string // dev HOME（隔离后的 ~/.rick 就在它下面）
	AgentDir string // dev 独占的 pi 沙盒（RICK_PI_AGENT_DIR）
	ProdRepo string // 生产仓库工作树（worktree 的来源，只读）
	ProdHome string // 真实家目录（只读：种子文件 + 构建缓存来源）
	Port     int
	Token    string

	// 构建缓存：显式指向生产缓存，避免 HOME 切换导致冷构建。
	GoCache    string
	GoModCache string
	NpmCache   string
}

// BuildID 是构建指纹（`<sha7>-<YYMMDDHHMMSS>`）；与 task18 注入的
// cmd.BuildID 及 `/api/health` 的 `build_id` 是同一个值。
type BuildID string

// StateDir 返回 dev 实例的状态目录（= `--state-dir` 的取值，等价于生产的
// `~/.rick`）。
func (l Layout) StateDir() string { return filepath.Join(l.Home, ".rick") }

// OverlayDist 返回前端覆盖层路径（dev 实例优先服务它）。
func (l Layout) OverlayDist() string { return filepath.Join(l.StateDir(), "web", "dist") }

// BinDir 返回 dev 二进制目录。
func (l Layout) BinDir() string { return filepath.Join(l.Home, "bin") }

// LogPath 返回 dev 服务日志路径。
func (l Layout) LogPath() string { return filepath.Join(l.Home, "dev.log") }

// PidPath 返回 dev 实例的 singleton pid 文件。
func (l Layout) PidPath() string { return filepath.Join(l.StateDir(), "web.pid") }

// StatePath 返回 dev 运行态记录文件（给人/脚本/AI 读的一行真相）。
func (l Layout) StatePath() string { return filepath.Join(l.Home, "dev-state.json") }

// EnvPath 返回 dev 布局快照（Tree/Home/AgentDir/Port/Token）。
func (l Layout) EnvPath() string { return filepath.Join(l.Home, "dev.env") }

// BaseURL 返回本机访问 dev 实例的基址。
func (l Layout) BaseURL() string { return "http://127.0.0.1:" + strconv.Itoa(l.Port) }

// seededFiles 是从生产 pi 沙盒**只读**拷贝过来的种子（认证/设置/模型目录）。
// 不拷 sessions/ 与 runtime/：前者是 1.6G 历史，后者是 145MB 且绝不能被 dev
// 覆写（human 裁决 J-L4-2 = 独立 agent dir）。
var seededFiles = []string{"auth.json", "settings.json", "models-store.json"}

// IsDevWorktree 报告 dir 是否像一个 rick dev 工作树。判据刻意用**结构**而非目录名：
//   - `.git` 是**文件**（git worktree 的标记；生产主工作树的 .git 是目录）
//   - 文件内容含 `gitdir:`（指向 <repo>/.git/worktrees/<name>）
//   - 同时存在 `cmd/rick/` 与 `web/package.json`（确实是 rick 源码树）
func IsDevWorktree(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	// 自动探测必须严：`.git` 是 worktree 标记**文件**（生产主工作树的 .git 是目录，
	// 借此把「从生产仓库执行」与「从 dev 树执行」分开）。
	fi, err := os.Lstat(filepath.Join(dir, ".git"))
	if err != nil || fi.IsDir() {
		return false
	}
	data, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil || !strings.Contains(string(data), "gitdir:") {
		return false
	}
	return IsRickSourceTree(dir)
}

// IsRickSourceTree 判断 dir 是否是一棵 rick 源码树（含 cmd/rick 与
// web/package.json）。**显式指定的路径**（--dev-tree / RICK_DEV_TREE）用这个宽松
// 判据：dev 树也可以是普通 clone 而不是 git worktree，不该被无谓拒绝。
func IsRickSourceTree(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	if wfi, err := os.Stat(filepath.Join(dir, "web", "package.json")); err != nil || wfi.IsDir() {
		return false
	}
	if cfi, err := os.Stat(filepath.Join(dir, "cmd", "rick")); err != nil || !cfi.IsDir() {
		return false
	}
	return true
}

// DevTreeFromCwd 从 dir 起向上查找 dev 工作树（最多 8 层），供「在树内任意子目录
// 执行命令」的情形使用。
func DevTreeFromCwd(dir string) (string, bool) {
	cur, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for i := 0; i < 8; i++ {
		if IsDevWorktree(cur) {
			return cur, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "", false
}

// ProdRepoFromDevTree 从 dev 工作树的 `.git`（`gitdir: <repo>/.git/worktrees/<name>`）
// 反推生产仓库工作树路径。形状不符或目标缺 go.mod 时返回 false。
func ProdRepoFromDevTree(tree string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(tree, ".git"))
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(data))
	const prefix = "gitdir:"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if gitdir == "" {
		return "", false
	}
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(tree, gitdir)
	}
	gitdir = filepath.Clean(gitdir)
	if filepath.Base(filepath.Dir(gitdir)) != "worktrees" {
		return "", false
	}
	dotGit := filepath.Dir(filepath.Dir(gitdir))
	if filepath.Base(dotGit) != ".git" {
		return "", false
	}
	repo := filepath.Dir(dotGit)
	if _, err := os.Stat(filepath.Join(repo, "go.mod")); err != nil {
		return "", false
	}
	return repo, true
}

// looksLikeProdRepo 判断 dir 是否是生产仓库主工作树（有 go.mod 且 .git 是目录）。
func looksLikeProdRepo(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	fi, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && fi.IsDir()
}

// RequireTree 在「需要已存在的 dev 工作树」的子命令（build/up/restart/status/down）
// 开头校验，给出**可操作的中文错误**——旧实现要等到 build 阶段才报
// `stat <tree>/web/package.json: no such file or directory`，看不出是树解析错了。
// init 刻意不调用它（init 的职责就是创建树）。
func (l Layout) RequireTree() error {
	if IsRickSourceTree(l.Tree) {
		return nil
	}
	if _, err := os.Stat(l.Tree); err != nil {
		return fmt.Errorf("dev 工作树不存在：%s\n"+
			"  尚未初始化 → 先执行：rick tools dev-web init\n"+
			"  树在别处     → 设置 RICK_DEV_TREE=<路径>，或传 --dev-tree <路径>", l.Tree)
	}
	return fmt.Errorf("目录不是 rick 源码树：%s\n"+
		"  （期望含 cmd/rick 与 web/package.json）\n"+
		"  请检查 --dev-tree / RICK_DEV_TREE 是否指错目录", l.Tree)
}

// RequireInitTarget 是 init 的目标校验：不存在 → 允许（init 会创建）；已存在但
// 不是 dev 工作树 → 明确拒绝，避免把无关目录当成 dev 树去折腾。
func (l Layout) RequireInitTarget() error {
	entries, err := os.ReadDir(l.Tree)
	if err != nil {
		return nil // 不存在（或不可读）→ init 会创建/报错
	}
	if len(entries) == 0 || IsRickSourceTree(l.Tree) {
		return nil // 空目录可被 git worktree 占用；已是源码树则幂等
	}
	return fmt.Errorf("目标 dev 树已存在且非空，但不是 rick 源码树：%s\n"+
		"  （期望含 cmd/rick 与 web/package.json）\n"+
		"  如确要用别处目录，请显式设置 --dev-tree / RICK_DEV_TREE", l.Tree)
}

// TreePolicy 决定解析时是否要求 dev 工作树已经存在。
//   - RequireExistingTree：build/up/restart/status/down 等「要操作已有树」的命令；
//     树缺失或不像 dev 树时**在解析阶段**给出可操作中文错误（F4 修复：旧实现要等到
//     build 阶段才报 `stat <tree>/web/package.json`,更早还会先被 EnsureToken 的
//     `mkdir <home>` 权限错误掩盖）。
//   - AllowCreateTree：init 专用（它的职责就是创建树）。
type TreePolicy int

const (
	RequireExistingTree TreePolicy = iota
	AllowCreateTree
)

// DefaultLayout 推导 dev 布局（等价于 LayoutFor(prodRepo, "")，要求树已存在）。
func DefaultLayout(prodRepo string) (Layout, error) { return LayoutFor(prodRepo, "") }

// LayoutFor 推导 dev 布局。treeOverride 来自 `--dev-tree`（空则看环境变量）。
//
// **dev 树解析顺序**（F4 修复：旧实现只看 prodRepo 的祖父目录，于是从 dev 树内
// 执行命令会把自己解析成 `<祖父>/rick-dev`，得到一条不存在的路径）：
//
//	① treeOverride（--dev-tree）
//	② RICK_DEV_TREE
//	③ 当前目录向上查找出的 dev 工作树（判据见 IsDevWorktree：`.git` 是 worktree
//	   标记**文件** + 含 cmd/rick 与 web/package.json —— 生产主工作树的 .git 是
//	   目录，因此不会被误判）
//	④ prodRepo 本身是 dev 工作树时就用它（`go run ./cmd/rick` 的常见情形）
//	⑤ 回退 `<prodRepo 的祖父目录>/rick-dev`
//
// home 默认 = `<tree>-home`（与现网实际布局一致），不再从 base 独立推导——
// 这样即使 tree 来自 ③④，home 也必然正确；RICK_DEV_HOME 仍可覆盖。
func LayoutFor(prodRepo, treeOverride string) (Layout, error) {
	return layout(prodRepo, treeOverride, RequireExistingTree)
}

// LayoutForInit 与 LayoutFor 同解析逻辑，但允许 dev 工作树尚不存在（init 会创建）。
func LayoutForInit(prodRepo, treeOverride string) (Layout, error) {
	return layout(prodRepo, treeOverride, AllowCreateTree)
}

func layout(prodRepo, treeOverride string, policy TreePolicy) (Layout, error) {
	absRepo, err := filepath.Abs(prodRepo)
	if err != nil {
		return Layout{}, fmt.Errorf("resolve prod repo: %w", err)
	}

	tree := strings.TrimSpace(treeOverride)
	if tree == "" {
		tree = strings.TrimSpace(os.Getenv(EnvTree))
	}
	if tree == "" {
		if wd, err := os.Getwd(); err == nil {
			if t, ok := DevTreeFromCwd(wd); ok {
				tree = t
			}
		}
	}
	if tree == "" {
		if t, ok := DevTreeFromCwd(absRepo); ok {
			tree = t
		}
	}
	if tree == "" {
		tree = filepath.Join(filepath.Dir(filepath.Dir(absRepo)), "rick-dev")
	}
	if abs, err := filepath.Abs(tree); err == nil {
		tree = abs
	}

	// 生产仓库：传入值本身不像生产仓库（缺 go.mod 或 .git 不是目录）时，
	// 改从 dev 工作树的 `.git`（`gitdir: <prod>/.git/worktrees/<name>`）反推——
	// 这条路径是权威的，能修正「从 dev 树内执行导致 prodRepo=cwd=dev 树」。
	if !looksLikeProdRepo(absRepo) {
		if repo, ok := ProdRepoFromDevTree(tree); ok {
			absRepo = repo
		}
	}

	home := strings.TrimSpace(os.Getenv(EnvHome))
	if home == "" {
		home = tree + "-home"
	}
	if abs, err := filepath.Abs(home); err == nil {
		home = abs
	}
	port := DefaultPort
	if v := strings.TrimSpace(os.Getenv(EnvPort)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 65535 {
			return Layout{}, fmt.Errorf("invalid %s=%q", EnvPort, v)
		}
		port = n
	}
	prodHome, err := realHomeDir()
	if err != nil {
		return Layout{}, err
	}
	l := Layout{
		Tree:       tree,
		Home:       home,
		AgentDir:   filepath.Join(home, ".rick", "pi", "agent"),
		ProdRepo:   absRepo,
		ProdHome:   prodHome,
		Port:       port,
		GoCache:    firstNonEmpty(os.Getenv("GOCACHE"), filepath.Join(prodHome, ".cache", "go-build")),
		GoModCache: firstNonEmpty(os.Getenv("GOMODCACHE"), filepath.Join(prodHome, "go", "pkg", "mod")),
		NpmCache:   firstNonEmpty(os.Getenv("npm_config_cache"), filepath.Join(prodHome, ".npm")),
	}
	// 关键顺序：先校验 dev 工作树，再准备 HOME（EnsureToken 会 mkdir + 写
	// DEV_TOKEN）——否则一个指向不存在的树的 RICK_DEV_TREE 会先报
	// `mkdir <home>: permission denied`，把真正的问题（树解析错了）掩盖掉。
	if policy == RequireExistingTree {
		if err := l.RequireTree(); err != nil {
			return Layout{}, err
		}
	}
	token, err := EnsureToken(l)
	if err != nil {
		return Layout{}, fmt.Errorf("准备 dev HOME 失败（%s）：%w", l.Home, err)
	}
	l.Token = token
	return l, nil
}

// EnsureToken 读取 `<Home>/DEV_TOKEN`，缺失则生成 `devtok-<8hex>` 并落盘
// （0600）。dev 绝不能复用生产 token：两个实例的会话凭证必须互不可用。
func EnsureToken(l Layout) (string, error) {
	path := filepath.Join(l.Home, "DEV_TOKEN")
	if data, err := os.ReadFile(path); err == nil {
		if tok := strings.TrimSpace(string(data)); tok != "" {
			return tok, nil
		}
	}
	if err := os.MkdirAll(l.Home, 0o755); err != nil {
		return "", fmt.Errorf("mkdir dev home: %w", err)
	}
	tok, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(tok+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write dev token: %w", err)
	}
	return tok, nil
}

// ServerEnv 返回启动 dev 实例时的**环境白名单**。刻意不继承整个 os.Environ()：
// 生产 web 进程的 environ 里带着 PI_SESSION_ID/PI_PROVIDER 之类的变量，透传给
// dev 的 pi 子进程会污染其行为（research-L4 §2.1）。
func (l Layout) ServerEnv() []string {
	env := []string{
		"HOME=" + l.Home,
		"PATH=" + os.Getenv("PATH"),
		"RICK_STATE_DIR=" + l.StateDir(),
		"RICK_PI_AGENT_DIR=" + l.AgentDir,
		// 让 dev 的隔离守卫知道生产状态目录在哪（只读比对，用于拒绝注册生产工作区）
		"RICK_PROD_STATE_DIR=" + filepath.Join(l.ProdHome, ".rick"),
		"GOCACHE=" + l.GoCache,
		"GOMODCACHE=" + l.GoModCache,
		"npm_config_cache=" + l.NpmCache,
	}
	for _, k := range []string{"TMPDIR", "LANG", "LC_ALL", "USER", "TERM"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// Init 幂等地准备 dev 环境：worktree → pi 沙盒种子 → 前端依赖/dist → overlay
// 软链 → 布局快照。重复执行不会破坏已有状态（worktree 已在、种子已在、
// node_modules 已在 → 全部跳过）。
func Init(l Layout, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if err := ensureWorktree(l, out); err != nil {
		return &StageError{Stage: "init", Err: err}
	}
	if err := ensureAgentSeed(l); err != nil {
		return &StageError{Stage: "init", Err: err}
	}
	if err := ensureFrontendDeps(l, out); err != nil {
		return &StageError{Stage: "init", Err: err}
	}
	if err := ensureOverlay(l); err != nil {
		return &StageError{Stage: "init", Err: err}
	}
	if err := l.WriteEnvFile(); err != nil {
		return &StageError{Stage: "init", Err: err}
	}
	return nil
}

// ensureWorktree 保证 dev 工作树存在（git worktree，秒级 + 共享 object store）。
func ensureWorktree(l Layout, out io.Writer) error {
	if _, err := os.Stat(filepath.Join(l.Tree, ".git")); err == nil {
		return nil
	}
	branch := "dev/self-evolve"
	if !isGitWorkTree(l.ProdRepo) {
		return fmt.Errorf("prod repo %s is not a git work tree", l.ProdRepo)
	}
	// 分支可能已被占用（重复 init）→ 退化为直接检出该分支。
	if err := runGit(l.ProdRepo, out, "worktree", "add", "-b", branch, l.Tree, "HEAD"); err != nil {
		if err2 := runGit(l.ProdRepo, out, "worktree", "add", l.Tree, branch); err2 != nil {
			return fmt.Errorf("git worktree add: %w", err2)
		}
	}
	return nil
}

// ensureAgentSeed 从生产 pi 沙盒拷贝种子文件（只读来源；已存在不覆盖）。
func ensureAgentSeed(l Layout) error {
	if err := os.MkdirAll(l.AgentDir, 0o755); err != nil {
		return fmt.Errorf("mkdir dev agent dir: %w", err)
	}
	srcDir := filepath.Join(l.ProdHome, ".rick", "pi", "agent")
	for _, name := range seededFiles {
		dst := filepath.Join(l.AgentDir, name)
		if _, err := os.Stat(dst); err == nil {
			continue // 已存在 → 幂等跳过（不覆盖 dev 自己改过的配置）
		}
		src := filepath.Join(srcDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue // 生产没有该文件（如未登录）→ 跳过，pi 首次运行会自建
		}
		if err := os.WriteFile(dst, data, 0o600); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}
	return nil
}

// ensureFrontendDeps 保证 dev 树 `web/` 能就地构建：node_modules 缺失时用生产
// npm 缓存 `npm ci`（实测热缓存 ~3s）；dist 缺失时构建一次。
// ⚠️ 必须在**dev 树内**构建：Tailwind v4 会把 dist 里的类名算进 CSS，换目录构建
// 会得到不同的 hash（research-L5 §5 实测）。
func ensureFrontendDeps(l Layout, out io.Writer) error {
	webDir := filepath.Join(l.Tree, "web")
	if _, err := os.Stat(filepath.Join(webDir, "package.json")); err != nil {
		return fmt.Errorf("dev tree missing web/package.json: %w", err)
	}
	if _, err := os.Stat(filepath.Join(webDir, "node_modules")); err != nil {
		if err := runNode(l, webDir, out, "npm", "ci"); err != nil {
			return fmt.Errorf("npm ci (in %s) failed: %w", webDir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(webDir, "dist", "index.html")); err != nil {
		if err := runNpmBuild(l, webDir, out); err != nil {
			return fmt.Errorf("npm run build failed: %w", err)
		}
	}
	return nil
}

// ensureOverlay 把 dev 实例的前端覆盖层指到 dev 树的 dist（目录软链，零拷贝）：
// `npm run build` 的产物立刻可见，无需重新部署。
func ensureOverlay(l Layout) error {
	target := filepath.Join(l.Tree, "web", "dist")
	if _, err := os.Stat(filepath.Join(target, "index.html")); err != nil {
		return fmt.Errorf("dev tree dist not built (missing index.html): %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(l.OverlayDist()), 0o755); err != nil {
		return fmt.Errorf("mkdir overlay parent: %w", err)
	}
	if fi, err := os.Lstat(l.OverlayDist()); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if cur, _ := os.Readlink(l.OverlayDist()); cur == target {
				return nil // 已是正确软链 → 幂等
			}
			_ = os.Remove(l.OverlayDist())
		} else {
			// 真实目录（旧的人工拷贝）：挪到一边再建软链，绝不静默丢数据
			bak := fmt.Sprintf("%s.bak-%d", l.OverlayDist(), time.Now().Unix())
			if err := os.Rename(l.OverlayDist(), bak); err != nil {
				return fmt.Errorf("move aside existing overlay: %w", err)
			}
		}
	}
	if err := os.Symlink(target, l.OverlayDist()); err != nil {
		return fmt.Errorf("link overlay: %w", err)
	}
	return nil
}

// WriteEnvFile 落盘布局快照（供脚本/人/AI 直接 source 或读取）。
func (l Layout) WriteEnvFile() error {
	body := strings.Join([]string{
		"# rick dev 实例布局快照（由 `rick tools dev-web init` 生成）",
		"RICK_DEV_TREE=" + l.Tree,
		"RICK_DEV_HOME=" + l.Home,
		"RICK_DEV_AGENT_DIR=" + l.AgentDir,
		"RICK_DEV_PORT=" + strconv.Itoa(l.Port),
		"RICK_DEV_TOKEN=" + l.Token,
		"RICK_DEV_STATE_DIR=" + l.StateDir(),
		"",
	}, "\n")
	if err := os.MkdirAll(l.Home, 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.EnvPath(), []byte(body), 0o600)
}

// Build 构建 dev 产物：前端 `npm run build` + 后端 `go build`（唯一命名 =
// 构建指纹）。goCache/goModCache 为空时用 Layout 里的默认（显式指向生产缓存）。
// 返回的 bin 路径即指纹本体：`<BinDir>/rick.dev.<sha7>-<YYMMDDHHMMSS>`。
func Build(l Layout, goCache, goModCache string) (string, error) {
	if goCache == "" {
		goCache = l.GoCache
	}
	if goModCache == "" {
		goModCache = l.GoModCache
	}
	binaryID, err := NextBuildID(l)
	if err != nil {
		return "", &StageError{Stage: "build", Err: err}
	}
	webDir := filepath.Join(l.Tree, "web")
	if _, err := os.Stat(filepath.Join(webDir, "package.json")); err != nil {
		return "", &StageError{Stage: "build", Err: fmt.Errorf("dev tree missing web/: %w", err)}
	}
	var npmOut strings.Builder
	if err := runNpmBuild(l, webDir, &npmOut); err != nil {
		return "", &StageError{Stage: "build", Err: fmt.Errorf("npm run build: %w", err)}
	}
	if err := os.MkdirAll(l.BinDir(), 0o755); err != nil {
		return "", &StageError{Stage: "build", Err: err}
	}
	bin := filepath.Join(l.BinDir(), DevBinPrefix+string(binaryID))
	args := []string{"build", "-o", bin,
		"-ldflags", "-X github.com/sunquan/rick/internal/cmd.BuildID=" + string(binaryID),
		"./cmd/rick"}
	cmd := exec.Command("go", args...)
	cmd.Dir = l.Tree
	cmd.Env = l.buildEnv(goCache, goModCache)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "", &StageError{Stage: "build", Err: fmt.Errorf("go build: %v\n%s", err, tail(string(raw), outputTail))}
	}
	return bin, nil
}

// NextBuildID 生成下一个构建指纹（<sha7>-<YYMMDDHHMMSS>）。
func NextBuildID(l Layout) (BuildID, error) {
	sha, err := gitShortSHA(l.Tree)
	if err != nil {
		sha = "nogit"
	}
	return BuildID(sha + "-" + time.Now().Format("060102150405")), nil
}

// BuildIDFromBin 从唯一文件名反解构建指纹（Up 用它做「新构建真的在跑」的校验）。
func BuildIDFromBin(bin string) BuildID {
	return BuildID(strings.TrimPrefix(filepath.Base(bin), DevBinPrefix))
}

// LatestBin 返回 dev 树里最近构建的二进制（供 `up --no-build`）。
func LatestBin(l Layout) (string, error) {
	entries, err := os.ReadDir(l.BinDir())
	if err != nil {
		return "", fmt.Errorf("read %s: %w", l.BinDir(), err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), DevBinPrefix) {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no dev build in %s (run `rick tools dev-web build`)", l.BinDir())
	}
	// 指纹里带 YYMMDDHHMMSS → 字典序即时间序
	sort.Strings(names)
	return filepath.Join(l.BinDir(), names[len(names)-1]), nil
}

// UpResult 是一次成功启动的回执（一行结构化的真相）。
type UpResult struct {
	Bin      string   `json:"bin"`
	BuildID  BuildID  `json:"build_id"`
	PID      int      `json:"pid"`
	Port     int      `json:"port"`
	HealthMS int64    `json:"health_ms"`
	Stopped  []int    `json:"stopped"`
	Skipped  []int    `json:"skipped"`
	LogTail  []string `json:"log_tail,omitempty"`
}

// StatusResult 是 `dev-web status` 的输出。
type StatusResult struct {
	Running       bool    `json:"running"`
	PID           int     `json:"pid,omitempty"`
	Port          int     `json:"port"`
	Bin           string  `json:"bin,omitempty"`
	ExpectedID    BuildID `json:"expected_build_id,omitempty"`
	RunningID     string  `json:"running_build_id,omitempty"`
	FingerprintOK bool    `json:"fingerprint_ok"`
	HealthMS      int64   `json:"health_ms,omitempty"`
	StatePath     string  `json:"state_path"`
}

// Up 起（或重启）dev 实例：停旧 → 起新 → 健康轮询 → 指纹复核 → 落盘状态。
// bin 为空时取最近一次构建。
func Up(l Layout, bin string) (UpResult, error) {
	var res UpResult
	if strings.TrimSpace(bin) == "" {
		latest, err := LatestBin(l)
		if err != nil {
			return res, &StageError{Stage: "start", Err: err}
		}
		bin = latest
	}
	if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
		return res, &StageError{Stage: "start", Err: fmt.Errorf("dev binary not usable: %s", bin)}
	}
	stopped, skipped := StopOld(l)

	logf, err := os.OpenFile(l.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return res, &StageError{Stage: "start", Err: fmt.Errorf("open dev log: %w", err)}
	}
	defer logf.Close()
	fmt.Fprintf(logf, "\n[dev-web] ---- starting %s at %s ----\n", filepath.Base(bin), time.Now().Format(time.RFC3339))

	cmd := exec.Command(bin, "web",
		"--listen", listenAddr,
		"--port", strconv.Itoa(l.Port),
		"--token", l.Token,
		"--state-dir", l.StateDir(),
	)
	cmd.Dir = l.Tree
	cmd.Env = l.ServerEnv()
	cmd.Stdout = logf
	cmd.Stderr = logf
	// 自成进程组：dev 实例（及其 pi 子进程）与调用者解耦，AI 会话后续命令不会
	// 因为进程组信号被连带干掉。
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return res, &StageError{Stage: "start", Err: fmt.Errorf("start dev server: %w", err)}
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release() // 不持有子进程，避免调用者被 Wait 阻塞

	healthMS, err := waitHealthy(l, pid, healthTimeout)
	if err != nil {
		return UpResult{Bin: bin, PID: pid, Port: l.Port, Stopped: stopped, Skipped: skipped,
			LogTail: logTail(l)}, &StageError{Stage: "health", Err: err}
	}

	expected := BuildIDFromBin(bin)
	running, ferr := runningBuildID(l)
	if ferr != nil || running == "" {
		// 指纹缺失（老构建）→ 降级为 /proc/<pid>/exe 的 sha256 比对
		if ok, why := exeMatches(l, pid, bin); !ok {
			return UpResult{Bin: bin, PID: pid, Port: l.Port, HealthMS: healthMS, Stopped: stopped, Skipped: skipped,
					LogTail: logTail(l)}, &StageError{Stage: "fingerprint",
					Err: fmt.Errorf("running build fingerprint unverifiable: %s", why)}
		}
	} else if BuildID(running) != expected {
		return UpResult{Bin: bin, PID: pid, Port: l.Port, HealthMS: healthMS, Stopped: stopped, Skipped: skipped,
				LogTail: logTail(l)}, &StageError{Stage: "fingerprint",
				Err: fmt.Errorf("running build_id=%q but started %q (旧进程没被换掉？)", running, expected)}
	}

	res = UpResult{Bin: bin, BuildID: expected, PID: pid, Port: l.Port, HealthMS: healthMS,
		Stopped: stopped, Skipped: skipped}
	if err := l.writeState(res); err != nil {
		return res, &StageError{Stage: "start", Err: err}
	}
	return res, nil
}

// Down 停止 dev 实例（只杀确认属于本 dev HOME 的进程）。
func Down(l Layout) ([]int, []int, error) {
	stopped, skipped := StopOld(l)
	if err := l.clearState(); err != nil {
		return stopped, skipped, err
	}
	return stopped, skipped, nil
}

// Status 汇报 dev 实例状态（进程 + 健康 + 指纹）。
func Status(l Layout) (StatusResult, error) {
	res := StatusResult{Port: l.Port, StatePath: l.StatePath()}
	state, err := l.readState()
	if err == nil {
		res.Bin = state.Bin
		res.ExpectedID = BuildIDFromBin(state.Bin)
		res.PID = state.PID
	}
	if pids := findDevPIDs(l); len(pids) > 0 {
		res.Running = true
		res.PID = pids[0]
	}
	if !res.Running {
		return res, nil
	}
	start := time.Now()
	if _, err := getJSON(l.BaseURL()+"/api/health", ""); err == nil {
		res.HealthMS = time.Since(start).Milliseconds()
		res.RunningID, _ = runningBuildID(l)
		res.FingerprintOK = res.ExpectedID == "" ||
			res.RunningID == "" || BuildID(res.RunningID) == res.ExpectedID
	}
	return res, nil
}

// ---- 进程发现与安全收尸 ----

// StopOld 停掉属于本 dev 实例的旧进程：先读 pid 文件，再用 /proc 扫描兜底
// （pid 文件可能缺失/陈旧）。**只有通过所有权断言的进程才会被杀**，其余记录在
// skipped 里返回给调用者（宁可不杀，也不能误杀生产）。
func StopOld(l Layout) (stopped, skipped []int) {
	seen := map[int]bool{}
	var candidates []int
	if data, err := os.ReadFile(l.PidPath()); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 {
			candidates = append(candidates, pid)
			seen[pid] = true
		}
	}
	for _, pid := range findDevPIDs(l) {
		if !seen[pid] {
			candidates = append(candidates, pid)
			seen[pid] = true
		}
	}
	for _, pid := range candidates {
		if pid == os.Getpid() {
			continue
		}
		if !alive(pid) {
			continue // 陈旧 pid（进程早已不在）→ 静默清理，不制造噪音
		}
		if !ownedByDev(pid, l) {
			// 候选存在但不是我们的进程（例如 pid 文件被别的东西占用）→ 拒杀并上报
			skipped = append(skipped, pid)
			continue
		}
		if terminate(pid, stopGrace) {
			stopped = append(stopped, pid)
		} else {
			skipped = append(skipped, pid)
		}
	}
	_ = os.Remove(l.PidPath()) // 陈旧 pid 文件不该让下一次 up 误判
	return stopped, skipped
}

// findDevPIDs 在 /proc 里找「属于本 dev 实例」的进程：environ 含
// RICK_STATE_DIR=<dev state dir> 或 HOME=<dev home>，或 cwd 恰好是 dev 树。
func findDevPIDs(l Layout) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 || pid == os.Getpid() {
			continue
		}
		if ownedByDev(pid, l) {
			out = append(out, pid)
		}
	}
	return out
}

// ownedByDev 是**防误杀的核心断言**：只有能证明进程确实属于该 dev 实例
// （environ 里的 HOME/RICK_STATE_DIR，或 cwd 是 dev 树）才返回 true。
// 读不到环境且 cwd 也不匹配 → false（拒绝杀）。
func ownedByDev(pid int, l Layout) bool {
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid)); err == nil {
		env := strings.Split(string(data), "\x00")
		for _, kv := range env {
			if kv == "HOME="+l.Home || kv == "RICK_STATE_DIR="+l.StateDir() {
				return true
			}
		}
		return false // 环境可读但不含 dev 标记 → 明确不属于本实例
	}
	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
		return cwd == l.Tree
	}
	return false
}

// terminate 先 TERM，宽限期后仍存活则 KILL；返回是否确认已消失。
func terminate(pid int, grace time.Duration) bool {
	if !alive(pid) {
		return true
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !alive(pid) {
			return true
		}
		time.Sleep(120 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	for i := 0; i < 20; i++ {
		if !alive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !alive(pid)
}

// alive 报告 pid 是否仍存在（供收尸轮询；僵尸态视为已退出）。
func alive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	// stat: pid (comm) state ...  —— comm 可能含空格/括号，取最后一个 ')'
	idx := strings.LastIndexByte(string(data), ')')
	if idx < 0 || idx+2 >= len(data) {
		return true
	}
	return string(data[idx+2]) != "Z"
}

// ---- 健康 / 指纹 / 状态文件 ----

// waitHealthy 轮询 /api/health（免认证）直到 200，返回耗时。
func waitHealthy(l Layout, pid int, timeout time.Duration) (int64, error) {
	start := time.Now()
	for time.Since(start) < timeout {
		if !alive(pid) {
			return 0, fmt.Errorf("dev server (pid %d) exited during startup", pid)
		}
		if _, err := getJSON(l.BaseURL()+"/api/health", ""); err == nil {
			return time.Since(start).Milliseconds(), nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return 0, fmt.Errorf("dev server on port %d not healthy within %s", l.Port, timeout)
}

// runningBuildID 读运行实例的构建指纹（/api/health 的 build_id）。
func runningBuildID(l Layout) (string, error) {
	body, err := getJSON(l.BaseURL()+"/api/health", "")
	if err != nil {
		return "", err
	}
	var payload struct {
		BuildID string `json:"build_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return payload.BuildID, nil
}

// exeMatches 是指纹校验的降级路径：比对 /proc/<pid>/exe 与目标二进制的 sha256。
func exeMatches(l Layout, pid int, bin string) (bool, string) {
	exe := fmt.Sprintf("/proc/%d/exe", pid)
	want, err := fileSHA256(bin)
	if err != nil {
		return false, err.Error()
	}
	got, err := fileSHA256(exe)
	if err != nil {
		return false, fmt.Sprintf("hash %s: %v", exe, err)
	}
	if want != got {
		return false, fmt.Sprintf("exe sha256 %s != %s", got[:12], want[:12])
	}
	return true, ""
}

func (l Layout) writeState(res UpResult) error {
	state := map[string]any{
		"bin":       res.Bin,
		"sha7":      strings.SplitN(string(res.BuildID), "-", 2)[0],
		"build_id":  string(res.BuildID),
		"pid":       res.PID,
		"built_at":  time.Now().Format(time.RFC3339),
		"health_ms": res.HealthMS,
		"port":      res.Port,
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(l.Home, 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.StatePath(), append(body, '\n'), 0o644)
}

func (l Layout) readState() (UpResult, error) {
	var res UpResult
	data, err := os.ReadFile(l.StatePath())
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (l Layout) clearState() error {
	if err := os.Remove(l.StatePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ---- 小工具 ----

// StageError 携带失败阶段，供 CLI 映射退出码（2 build / 3 start / 4 health）。
type StageError struct {
	Stage string
	Err   error
}

func (e *StageError) Error() string { return e.Stage + ": " + e.Err.Error() }
func (e *StageError) Unwrap() error { return e.Err }

func (l Layout) buildEnv(goCache, goModCache string) []string {
	env := l.ServerEnv()
	env = append(env, "GOCACHE="+goCache, "GOMODCACHE="+goModCache)
	return env
}

// runNpmBuild 在 dev 树 `web/` 内就地构建前端（缓存指向生产 npm cache）。
func runNpmBuild(l Layout, webDir string, out io.Writer) error {
	return runNode(l, webDir, out, "npm", "run", "build")
}

// runNode 在指定目录跑 node/npm 命令，注入 npm 缓存与 PATH。
func runNode(l Layout, dir string, out io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append([]string{
		"HOME=" + l.Home,
		"PATH=" + os.Getenv("PATH"),
		"npm_config_cache=" + l.NpmCache,
	}, envPairs("TMPDIR", "LANG", "LC_ALL", "USER")...)
	raw, err := cmd.CombinedOutput()
	if len(raw) > 0 && out != nil {
		fmt.Fprint(out, tail(string(raw), outputTail))
	}
	if err != nil {
		return fmt.Errorf("%s %s: %v\n%s", name, strings.Join(args, " "), err, tail(string(raw), outputTail))
	}
	return nil
}

func runGit(dir string, out io.Writer, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	raw, err := cmd.CombinedOutput()
	if len(raw) > 0 && out != nil {
		fmt.Fprint(out, string(raw))
	}
	if err != nil {
		return fmt.Errorf("git %s: %v\n%s", strings.Join(args, " "), err, tail(string(raw), 512))
	}
	return nil
}

func gitShortSHA(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short=7", "HEAD")
	cmd.Dir = dir
	raw, err := cmd.Output()
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(raw))
	if sha == "" {
		return "", fmt.Errorf("empty sha")
	}
	return sha, nil
}

func isGitWorkTree(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return false
	}
	return true
}

func getJSON(url, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func envPairs(keys ...string) []string {
	var out []string
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func logTail(l Layout) []string {
	data, err := os.ReadFile(l.LogPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return lines
}

// randomToken 生成 `devtok-<8hex>`（dev 专用凭证，绝不复用生产 token）。
func randomToken() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate dev token: %w", err)
	}
	return "devtok-" + hex.EncodeToString(buf), nil
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// realHomeDir 返回**真实**家目录（不跟随 $HOME）：dev 实例的 HOME 已被切换，
// 但种子文件与构建缓存在真实家目录下。优先 os/user（读 /etc/passwd），失败时
// 回退 $HOME。
func realHomeDir() (string, error) {
	if data, err := os.ReadFile("/etc/passwd"); err == nil {
		uid := strconv.Itoa(os.Getuid())
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Split(line, ":")
			if len(parts) >= 6 && parts[2] == uid {
				return parts[5], nil
			}
		}
	}
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}
	return "", fmt.Errorf("cannot determine real home directory")
}
