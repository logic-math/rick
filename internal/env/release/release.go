// Package release implements `rick tools release`: 把 dev 工作树的产物（二进制 +
// 前端 dist）**原子提升**为生产、受控重启生产、并支持一键回滚。
//
// 设计依据（research-L6，均为实测事实）：
//   - 生产二进制真身是**生产仓库工作树**里的 `bin/rick`（不是 `~/.rick/bin/rick`，
//     那是 8/24 的历史遗留）；`start-web.sh` 用 `./bin/rick` 启动。
//   - 同一文件系统内 `rename(2)` 原子，且可以覆盖**正在运行**的二进制（`cp` 会
//     ETXTBSY；`mv` 成功、旧进程继续跑直到退出）。跨文件系统的 mv 会退化为
//     copy+unlink（路径短暂消失）——因此所有切换都走同目录 rename。
//   - 前端覆盖层 `<stateDir>/web/dist` 优先于二进制内嵌 dist（逐请求判断、
//     all-or-nothing）→ **前端必须与二进制同版推进**，否则出现「新后端 + 旧前端」。
//   - 重启必然中断进程内的 pi worker（worker 是 web 的子进程、0/1 是匿名管道，
//     新进程无法接管）。因此本命令**只保证**「平台自动回来 + 会话挂起待人工
//     一键恢复」，绝不自动 resume/续跑（human 裁决 J-L6-6/J-L6-7：自动续跑会
//     因 pi 不修复悬挂 toolCall 而重复副作用，也会在配额耗尽时静默空转）。
//
// 安全边界（本包的硬纪律）：所有破坏性动作只针对 **Plan 里显式给出的路径**；
// 杀进程前必须通过所有权断言（cwd 属于生产工作树 / 可执行文件在生产 bin 下 /
// environ 的 HOME 是生产 HOME）。测试全部用 t.TempDir() 造的「模拟生产」。
package release

import (
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// BuildIDLdflag is the linker flag that injects the build fingerprint
// (task18's cmd.BuildID → /api/health 的 build_id)。提升后用它判定
// 「新构建真的在跑」，而不是靠版本号硬编码常量。
const BuildIDLdflag = "github.com/sunquan/rick/internal/cmd.BuildID"

// 版本目录布局（全部在**生产仓库**内；bin/ 已被 .gitignore，产物不入库）：
//
//	<prodRepo>/bin/releases/<version>/rick
//	<prodRepo>/bin/releases/<version>/dist
//	<prodRepo>/bin/releases/<version>/SHA256SUMS
//	<prodRepo>/bin/releases/<version>/CHANGES
//	<prodRepo>/bin/releases/current     -> <version>      (符号链接)
//	<prodRepo>/bin/releases/.last       -> <version>      (回滚点，纯文本)
//	<prodRepo>/bin/rick                 -> releases/current/rick
const (
	ReleasesDirName = "releases"
	CurrentLinkName = "current"
	LastRecordName  = ".last"
	ChecksumName    = "SHA256SUMS"
	ChangesName     = "CHANGES"

	// RecoveryReportFile 与 internal/web 的报告文件同名。刻意**不 import**
	// internal/web：env 层不反向依赖 web 层（internal/env/web.go 有同样的注解），
	// 报告用本地结构防御式解析（字段缺失也不影响提升）。
	RecoveryReportFile = "recovery-report.json"

	// EnvProdRepo/EnvDevTree/EnvProdPort 允许脚本与测试重定向（默认从部署事实推导）。
	EnvProdRepo = "RICK_PROD_REPO"
	EnvDevTree  = "RICK_DEV_TREE"
	EnvProdPort = "RICK_PROD_PORT"

	// 默认值
	DefaultListen       = "0.0.0.0"
	DefaultKeepVersions = 3
	defaultHealthWait   = 25 * time.Second
	defaultStopGrace    = 14 * time.Second
	defaultGateTimeout  = 1800 * time.Second
)

// versionRe 匹配 `<sha7>-<YYMMDDHHMMSS>`（防止 GC 误删非版本目录）。
var versionRe = regexp.MustCompile(`^(dev|[0-9a-f]{7,40})-\d{12}$`)

// Plan 描述「从哪提升到哪」的全部参数（纯数据 → 测试可完全注入）。
type Plan struct {
	ProdRepo string // 生产仓库工作树（提升目标；start-web.sh 的 cd 目标）
	DevTree  string // dev 源码工作树（构建来源）
	ProdHome string // 生产 HOME（其下的 .rick 是生产状态目录）

	// ReleasesRoot 覆盖发布目录的根（默认 <ProdRepo>/bin/releases）。存在的原因：
	// 生产仓库树可能被外部（cloud-ide 快照/同事/其它用户）重置属主与 ACL mask，
	// 导致 release 进程（以部署用户跑）无法在 <ProdRepo>/bin/releases 下建目录
	// （实测：bin/releases 属主 sankuai、mask r-x → mkdir permission denied，
	// 且 bin/rick 软链同因消失）。指向用户可控目录（如 ~/.rick/releases）即可
	// 继续发布；bin/rick 软链仍指向 <ReleasesRoot>/current/rick，start-web.sh 不变。
	ReleasesRoot string

	// StateDir 是生产状态目录（默认 <ProdHome>/.rick）。显式给出便于测试与
	// 「生产用了 --state-dir」的部署。
	StateDir string

	// StagingDir 覆盖「本次构建写到哪」：
	//   - 空（默认）：<ProdRepo>/bin/releases/<version> —— 真实提升的发布目录；
	//   - 非空：<StagingDir>/<version> —— dry-run 用的一次性暂存目录。
	// 存在的理由：`--dry-run` 承诺「不动生产」，而旧实现复用 Build()，会把产物
	// 写进生产树 <ProdRepo>/bin/releases/<version>（实测残留 bin/releases/1091f62-…）。
	// 暂存目录里的产物**永远不可提升**（Promote 显式拒绝 Staged 产物）。
	StagingDir string

	Port   int    // 生产监听端口（从部署脚本解析，**不硬编码**）
	Listen string // 生产监听地址（默认 0.0.0.0）

	// StartScript 是部署自带的启动脚本（默认 <ProdHome>/start-web.sh）。存在时
	// 提升重启沿用它 —— 它就是「生产如何被拉起」的唯一事实来源。
	StartScript string

	Token string // 生产 web token（只读自 config.json；用于重启后查询会话/报告）

	GoCache    string
	GoModCache string
	NpmCache   string

	// MergeVersion 是「--merge-source 合并提交信息里写哪个版本号」。
	// CLI 在构建后把它设为本次提升的版本（rel.Version），使 merge commit 与
	// 二进制版本一一对应；留空则用 NewVersion() 现取。
	MergeVersion string

	KeepVersions  int
	HealthWait    time.Duration
	StopGrace     time.Duration
	GateTimeout   time.Duration
	SkipGates     bool
	SkipFrontend  bool
	GateResultOut *GateResult // 门禁结果回填（供报价单打印）

	// 测试注入点（nil = 真实实现）
	BuildBinary func(p Plan, dest string, out io.Writer) error
	RunGates    func(p Plan, out io.Writer) (GateResult, error)
	Now         func() time.Time
	HTTPGet     func(url, token string) ([]byte, error)
}

func (p Plan) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// KeepOr 返回保留版本数（默认 3），供 CLI 打印。
func (p Plan) KeepOr() int { return p.keep() }

func (p Plan) keep() int {
	if p.KeepVersions > 0 {
		return p.KeepVersions
	}
	return DefaultKeepVersions
}

func (p Plan) healthWait() time.Duration {
	if p.HealthWait > 0 {
		return p.HealthWait
	}
	return defaultHealthWait
}

func (p Plan) stopGrace() time.Duration {
	if p.StopGrace > 0 {
		return p.StopGrace
	}
	return defaultStopGrace
}

func (p Plan) listen() string {
	if strings.TrimSpace(p.Listen) != "" {
		return p.Listen
	}
	return DefaultListen
}

// StateDirPath 返回生产状态目录。
func (p Plan) StateDirPath() string {
	if strings.TrimSpace(p.StateDir) != "" {
		return p.StateDir
	}
	return filepath.Join(p.ProdHome, ".rick")
}

// ReleasesDir 返回版本目录根。
func (p Plan) ReleasesDir() string {
	if p.ReleasesRoot != "" {
		return p.ReleasesRoot
	}
	return filepath.Join(p.ProdRepo, "bin", ReleasesDirName)
}

// CurrentLink 返回 `current` 符号链接路径。
func (p Plan) CurrentLink() string { return filepath.Join(p.ReleasesDir(), CurrentLinkName) }

// LastRecord 返回回滚点记录文件路径。
func (p Plan) LastRecord() string { return filepath.Join(p.ReleasesDir(), LastRecordName) }

// ProdBinLink 返回生产二进制入口（symbolic link → releases/current/rick）。
func (p Plan) ProdBinLink() string { return filepath.Join(p.ProdRepo, "bin", "rick") }

// OverlayDist 返回前端覆盖层目录。
func (p Plan) OverlayDist() string { return filepath.Join(p.StateDirPath(), "web", "dist") }

// OverlayPrev 返回覆盖层备份目录（回滚用）。
func (p Plan) OverlayPrev() string { return p.OverlayDist() + ".prev" }

// ProdPidPath 返回生产 singleton pid 文件路径。
func (p Plan) ProdPidPath() string { return filepath.Join(p.StateDirPath(), "web.pid") }

// LogPath 返回生产日志追加路径。
func (p Plan) LogPath() string { return filepath.Join(p.StateDirPath(), "web.log") }

// StatePath 返回提升留痕文件（release-state.json）。
func (p Plan) StatePath() string { return filepath.Join(p.StateDirPath(), "release-state.json") }

// BaseURL 返回本机访问生产实例的基址。
func (p Plan) BaseURL() string { return "http://127.0.0.1:" + strconv.Itoa(p.Port) }

// VersionDir 返回某版本号对应的目录。
func (p Plan) VersionDir(version string) string {
	return filepath.Join(p.ReleasesDir(), version)
}

// BuildDir 返回本次构建**实际写入**的目录。默认就是 VersionDir（发布目录）；
// 只有 dry-run 通过 StagingDir 把它指向一次性暂存目录，从而在生产树上零写入。
func (p Plan) BuildDir(version string) string {
	if strings.TrimSpace(p.StagingDir) != "" {
		return filepath.Join(p.StagingDir, version)
	}
	return p.VersionDir(version)
}

// ---- 失败阶段（CLI 据此映射退出码）----

// StageError 携带失败阶段：gate / build / promote / restart / health。
type StageError struct {
	Stage string
	Err   error
}

func (e *StageError) Error() string { return e.Stage + ": " + e.Err.Error() }
func (e *StageError) Unwrap() error { return e.Err }

func stageErr(stage string, format string, args ...any) error {
	return &StageError{Stage: stage, Err: fmt.Errorf(format, args...)}
}

// ---- 产物与结果 ----

// Release 是一次构建的产物（版本目录内的东西）。
type Release struct {
	Version string
	// Dir 是**实际构建落点**（真实提升=发布目录；dry-run=暂存目录）。
	Dir string
	// TargetDir 是「正式提升时产物应当所在」的发布目录（<prodRepo>/bin/releases/<ver>）。
	// dry-run 用它把「本来会落到哪」告诉人类，避免暂存路径造成误解。
	TargetDir string
	// Staged 为真表示这是暂存（dry-run）产物 —— Promote 会拒绝提升它。
	Staged    bool
	Bin       string
	Dist      string
	DistFiles int
	SHA256    string
	BuildID   string
}

// GateResult 是提升前门禁的结果（报价单的核心输入）。
type GateResult struct {
	Ran      bool   `json:"ran"`
	Tests    string `json:"tests,omitempty"`
	Frontend string `json:"frontend,omitempty"`
	Took     string `json:"took,omitempty"`
	Output   string `json:"output,omitempty"`
}

// SuspendSummary 是重启后读到的挂起清单（来自 web 层的恢复报告）。
// IDs 只带前若干个会话 id：提升回执要能直接告诉人被挂起的是谁（人再去 UI
// 一键恢复），但不必把整份报告塞进回执。
type SuspendSummary struct {
	At        string   `json:"at,omitempty"`
	Suspended int      `json:"suspended"`
	Recovered int      `json:"recovered"`
	Failed    int      `json:"failed"`
	IDs       []string `json:"ids,omitempty"`
	Path      string   `json:"path,omitempty"`
}

// RestartResult 是一次生产重启的结果。
type RestartResult struct {
	Version  string         `json:"version"`
	PID      int            `json:"pid"`
	HealthMS int64          `json:"health_ms"`
	BuildID  string         `json:"build_id"`
	Matches  bool           `json:"build_id_matches"`
	Stopped  []int          `json:"stopped"`
	Skipped  []int          `json:"skipped"`
	Overlay  string         `json:"overlay,omitempty"`
	Recovery SuspendSummary `json:"recovery"`
	LogTail  []string       `json:"log_tail,omitempty"`
}

// Result 是提升/回滚的结果（一行真相 + 给 AI 会话解析的结构）。
type Result struct {
	Action      string `json:"action"` // promote | rollback | dry-run
	Version     string `json:"version"`
	PrevVersion string `json:"prev_version,omitempty"`
	Bin         string `json:"bin,omitempty"`
	DistFiles   int    `json:"dist_files,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	// dry-run 专用：正式提升本会写到的目标路径 + 本次临时暂存路径
	TargetBin  string         `json:"target_bin,omitempty"`
	TargetDist string         `json:"target_dist,omitempty"`
	StagingDir string         `json:"staging_dir,omitempty"`
	GCRemoved  []string       `json:"gc_removed,omitempty"`
	Overlay    string         `json:"overlay,omitempty"`
	Restart    *RestartResult `json:"restart,omitempty"`
	// Merge 记录 --merge-source 的源码合并结果（未开启时为 nil）。
	Merge *MergeResult `json:"merge,omitempty"`
	Plan  PlanSummary  `json:"plan"`
}

// PlanSummary 是可打印的「提升计划」（人类审核的报价单主体）。
type PlanSummary struct {
	ProdRepo    string `json:"prod_repo"`
	DevTree     string `json:"dev_tree"`
	StateDir    string `json:"state_dir"`
	Port        int    `json:"port"`
	Listen      string `json:"listen"`
	StartScript string `json:"start_script,omitempty"`
	Current     string `json:"current,omitempty"`
	Last        string `json:"last,omitempty"`
}

// Summary 返回可打印的计划摘要。
func (p Plan) Summary() PlanSummary {
	cur, _ := p.CurrentVersion()
	last, _ := p.LastVersion()
	return PlanSummary{
		ProdRepo: p.ProdRepo, DevTree: p.DevTree, StateDir: p.StateDirPath(),
		Port: p.Port, Listen: p.listen(), StartScript: p.StartScript,
		Current: cur, Last: last,
	}
}

// ---- 构建 ----

// NewVersion 生成版本号 `<sha7>-<YYMMDDHHMMSS>`（sha 取自 dev 树 HEAD，读不
// 到时退回 `dev`——仍是合法版本号，不阻断构建）。
func (p Plan) NewVersion() string {
	sha := shortSHA(p.DevTree)
	if sha == "" {
		sha = "dev"
	}
	return sha + "-" + p.now().Format("060102150405")
}

// Build 跑门禁并把 dev 产物封装进 `<prodRepo>/bin/releases/<version>/`。
// **不触碰正在运行的生产**：只写新目录（新文件，不覆盖任何在跑的二进制）。
func (p Plan) Build(out io.Writer) (Release, error) {
	if out == nil {
		out = io.Discard
	}
	version := p.NewVersion()
	target := p.VersionDir(version)
	dir := p.BuildDir(version)
	rel := Release{Version: version, Dir: dir, TargetDir: target, Staged: dir != target, BuildID: version}
	rel.Bin = filepath.Join(rel.Dir, "rick")
	rel.Dist = filepath.Join(rel.Dir, "dist")

	if !p.SkipGates {
		gates := p.RunGates
		if gates == nil {
			gates = runGates
		}
		res, err := gates(p, out)
		if p.GateResultOut != nil {
			*p.GateResultOut = res
		}
		if err != nil {
			return rel, &StageError{Stage: "gate", Err: err}
		}
		if out != nil {
			fmt.Fprintf(out, "RELEASE_GATE pass=true tests=%s frontend=%s took=%s\n",
				res.Tests, res.Frontend, res.Took)
		}
	}

	if err := os.MkdirAll(rel.Dir, 0o755); err != nil {
		return rel, stageErr("build", "mkdir %s: %v", rel.Dir, err)
	}
	build := p.BuildBinary
	if build == nil {
		build = buildBinary
	}
	if err := build(p, rel.Bin, out); err != nil {
		return rel, &StageError{Stage: "build", Err: err}
	}
	if fi, err := os.Stat(rel.Bin); err != nil || fi.IsDir() {
		return rel, stageErr("build", "构建产物不可用: %s", rel.Bin)
	}
	if err := os.Chmod(rel.Bin, 0o755); err != nil {
		return rel, stageErr("build", "chmod 构建产物: %v", err)
	}

	if p.SkipFrontend {
		if out != nil {
			fmt.Fprintln(out, "RELEASE_BUILD frontend=skipped")
		}
	} else if err := copyTree(filepath.Join(p.DevTree, "web", "dist"), rel.Dist); err != nil {
		return rel, &StageError{Stage: "build", Err: fmt.Errorf("copy frontend dist: %w", err)}
	}
	rel.DistFiles, _ = countFiles(rel.Dist)

	sum, err := fileSHA256(rel.Bin)
	if err != nil {
		return rel, stageErr("build", "sha256 构建产物: %v", err)
	}
	rel.SHA256 = sum
	if err := writeChecksums(rel); err != nil {
		return rel, &StageError{Stage: "build", Err: err}
	}
	writeChanges(p, rel, out)

	// 冒烟：新二进制可执行（`web --help` 是最轻的入口，不监听端口）。
	if err := smokeCheck(rel.Bin); err != nil {
		return rel, &StageError{Stage: "build", Err: err}
	}
	if out != nil {
		fmt.Fprintf(out, "RELEASE_BUILD bin=%s dist_files=%d sha256=%s\n",
			rel.Bin, rel.DistFiles, short(sum))
	}
	return rel, nil
}

// runGates 是真实门禁：dev 树内 `go test ./...` + `web/` 内 `npm run build`。
// 刻意**在生产仓库之外**跑（research-L6 §4.2 缺陷 1：老门禁会 `go build -o bin/rick`
// 覆盖生产二进制——这里绝不写生产路径）。
func runGates(p Plan, out io.Writer) (GateResult, error) {
	start := time.Now()
	res := GateResult{Ran: true}
	timeout := p.GateTimeout
	if timeout <= 0 {
		timeout = defaultGateTimeout
	}

	goCmd := exec.Command("go", "test", "./...", "-timeout", "900s")
	goCmd.Dir = p.DevTree
	goCmd.Env = p.buildEnv()
	raw, err := goCmd.CombinedOutput()
	if out != nil {
		fmt.Fprint(out, tail(string(raw), 2048))
	}
	if err != nil {
		res.Output = tail(string(raw), 2048)
		return res, fmt.Errorf("go test ./... 失败（dev 树 %s）: %v", p.DevTree, err)
	}
	res.Tests = "pass"

	if p.SkipFrontend {
		res.Frontend = "skipped"
		res.Took = time.Since(start).Round(time.Second).String()
		return res, nil
	}
	npmCmd := exec.Command("npm", "run", "build")
	npmCmd.Dir = filepath.Join(p.DevTree, "web")
	npmCmd.Env = []string{"HOME=" + p.DevHomeOr(), "PATH=" + os.Getenv("PATH"),
		"npm_config_cache=" + firstNonEmpty(p.NpmCache, filepath.Join(p.DevHomeOr(), ".npm"))}
	raw, err = npmCmd.CombinedOutput()
	if out != nil {
		fmt.Fprint(out, tail(string(raw), 2048))
	}
	if err != nil {
		res.Output = tail(string(raw), 2048)
		return res, fmt.Errorf("npm run build 失败（%s/web）: %v", p.DevTree, err)
	}
	res.Frontend = "pass"
	res.Took = time.Since(start).Round(time.Second).String()
	return res, nil
}

// buildBinary 是真实构建：`go build -ldflags "-X <buildID>=<version>" -o <dest> ./cmd/rick`。
// 只写 releases/<version>/rick（新文件），永远不写正在运行的 bin/rick。
func buildBinary(p Plan, dest string, out io.Writer) error {
	// 版本号 = 产物所在目录名（releases/<version>/rick）→ 注入后可由
	// /api/health 的 build_id 反查「跑的是哪一份」，重启后的校验就靠它。
	version := filepath.Base(filepath.Dir(dest))
	args := []string{"build",
		"-ldflags", "-X " + BuildIDLdflag + "=" + version,
		"-o", dest, "./cmd/rick"}
	cmd := exec.Command("go", args...)
	cmd.Dir = p.DevTree
	cmd.Env = p.buildEnv()
	raw, err := cmd.CombinedOutput()
	if len(raw) > 0 && out != nil {
		fmt.Fprint(out, tail(string(raw), 2048))
	}
	if err != nil {
		return fmt.Errorf("go build: %v\n%s", err, tail(string(raw), 2048))
	}
	return nil
}

func (p Plan) buildEnv() []string {
	home := p.DevHomeOr()
	env := []string{
		"HOME=" + home,
		"PATH=" + os.Getenv("PATH"),
	}
	for _, kv := range []struct{ k, v string }{
		{"GOCACHE", firstNonEmpty(p.GoCache, os.Getenv("GOCACHE"))},
		{"GOMODCACHE", firstNonEmpty(p.GoModCache, os.Getenv("GOMODCACHE"))},
		{"npm_config_cache", firstNonEmpty(p.NpmCache, os.Getenv("npm_config_cache"))},
		{"TMPDIR", os.Getenv("TMPDIR")},
		{"LANG", os.Getenv("LANG")},
	} {
		if kv.v != "" {
			env = append(env, kv.k+"="+kv.v)
		}
	}
	return env
}

// DevHomeOr 返回构建用的 HOME：优先 plan.ProdHome 之外的 deddev 家目录（dev 树
// 的祖父目录下的 rick-dev-home），没有就退回当前 HOME。
func (p Plan) DevHomeOr() string {
	if c := filepath.Join(filepath.Dir(p.DevTree), "rick-dev-home"); dirExists(c) {
		return c
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return p.ProdHome
}

func smokeCheck(bin string) error {
	cmd := exec.Command(bin, "web", "--help")
	cmd.Env = []string{"HOME=" + os.TempDir(), "PATH=" + os.Getenv("PATH")}
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("新二进制冒烟失败（%s web --help）: %v\n%s", bin, err, tail(string(raw), 512))
	}
	return nil
}

// ---- 提升 ----

// Promote 把已构建好的 Release 原子提升为生产：
// ① 记录回滚点 ② 原子换链 ③ bin/rick → releases/current/rick ④ 覆盖层同版推进
// ⑤ 保留最近 N 版并 GC。**不重启**（重启由调用方显式发起，便于先打印计划）。
func (p Plan) Promote(rel Release) (Result, error) {
	res := Result{Action: "promote", Version: rel.Version, Plan: p.Summary()}
	if rel.Staged {
		return res, stageErr("promote",
			"拒绝提升暂存（dry-run）产物: %s —— 暂存目录不属于发布目录，请走真实提升路径重新构建", rel.Dir)
	}
	if fi, err := os.Stat(rel.Dir); err != nil || !fi.IsDir() {
		return res, stageErr("promote", "版本目录不存在: %s", rel.Dir)
	}
	prev, err := p.CurrentVersion()
	if err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	res.PrevVersion = prev
	if prev != "" && prev != rel.Version {
		if err := os.WriteFile(p.LastRecord(), []byte(prev+"\n"), 0o644); err != nil {
			return res, stageErr("promote", "写回滚点 %s: %v", p.LastRecord(), err)
		}
	}
	if err := p.switchCurrent(rel.Version); err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	if err := p.ensureBinLink(); err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	overlay, err := p.syncOverlay(rel)
	if err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	gc, err := p.gcReleases()
	if err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	res.Bin = rel.Bin
	res.SHA256 = rel.SHA256
	res.DistFiles = rel.DistFiles
	res.Overlay = overlay
	res.GCRemoved = gc
	return res, nil
}

// switchCurrent 原子切换 `current` 链（同目录 symlink + rename，绝不 ln -sfn：
// 那是「先 unlink 再建」，中间存在符号链接缺失的窗口）。
func (p Plan) switchCurrent(version string) error {
	if err := os.MkdirAll(p.ReleasesDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir releases: %w", err)
	}
	tmp := filepath.Join(p.ReleasesDir(), ".current.tmp")
	_ = os.Remove(tmp)
	if err := os.Symlink(version, tmp); err != nil {
		return fmt.Errorf("symlink %s: %w", version, err)
	}
	if err := os.Rename(tmp, p.CurrentLink()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename current -> %s: %w", version, err)
	}
	return nil
}

// ensureBinLink 让 `bin/rick` 成为指向 `releases/current/rick` 的相对软链
// （相对目标保证 `./bin/rick` 在 start-web.sh 里照旧可用，仓库整体搬走也不坏）。
// 首次提升时把原来的普通文件改名留存（`.legacy-<ts>`），不静默丢历史。
func (p Plan) ensureBinLink() error {
	link := p.ProdBinLink()
	if fi, err := os.Lstat(link); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			// 已是软链：只需确保指向正确（幂等重建，走 tmp+rename 原子替换）
		} else if fi.Mode().IsRegular() {
			legacy := fmt.Sprintf("%s.legacy-%s", link, p.now().Format("20060102-150405"))
			if err := os.Rename(link, legacy); err != nil {
				return fmt.Errorf("留存旧二进制 %s: %w", legacy, err)
			}
		} else {
			return fmt.Errorf("%s 既不是普通文件也不是软链，拒绝覆盖", link)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", link, err)
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return fmt.Errorf("mkdir bin: %w", err)
	}
	// 默认：相对链 releases/current/rick（bin/ 与 releases/ 同在 ProdRepo/bin 下）。
	// ReleasesRoot 覆盖时（发布目录被迁到 ~/.rick/releases 之类）：必须用绝对路径，
	// 否则相对链会指向 <ProdRepo>/bin/<ReleasesRoot> 这个不存在的位置。
	var target string
	if p.ReleasesRoot != "" {
		target = filepath.Join(p.ReleasesRoot, CurrentLinkName, "rick")
	} else {
		target = filepath.Join(ReleasesDirName, CurrentLinkName, "rick")
	}
	tmp := link + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("symlink bin/rick: %w", err)
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename bin/rick: %w", err)
	}
	return nil
}

// syncOverlay 把版本内的前端产物原子投放到覆盖层，并把旧覆盖层留成
// `<overlay>.prev`（回滚用）。版本内没有 dist 时保持现状（纯后端提升）。
func (p Plan) syncOverlay(rel Release) (string, error) {
	if !dirExists(rel.Dist) {
		return p.OverlayDist(), nil
	}
	ov := p.OverlayDist()
	prev := p.OverlayPrev()
	if err := os.MkdirAll(filepath.Dir(ov), 0o755); err != nil {
		return ov, fmt.Errorf("mkdir overlay parent: %w", err)
	}
	if dirExists(ov) {
		_ = os.RemoveAll(prev)
		if err := os.Rename(ov, prev); err != nil {
			return ov, fmt.Errorf("备份覆盖层 %s: %w", prev, err)
		}
	}
	tmpNew := ov + ".new"
	_ = os.RemoveAll(tmpNew)
	if err := copyTree(rel.Dist, tmpNew); err != nil {
		return ov, fmt.Errorf("投放覆盖层: %w", err)
	}
	if err := os.Rename(tmpNew, ov); err != nil {
		return ov, fmt.Errorf("rename overlay: %w", err)
	}
	return ov, nil
}

// restoreOverlay 回滚覆盖层（dist.prev → dist）。
func (p Plan) restoreOverlay() (bool, error) {
	ov, prev := p.OverlayDist(), p.OverlayPrev()
	if !dirExists(prev) {
		return false, nil
	}
	tmp := ov + ".rollback-tmp"
	_ = os.RemoveAll(tmp)
	if dirExists(ov) {
		if err := os.Rename(ov, tmp); err != nil {
			return false, fmt.Errorf("腾挪当前覆盖层: %w", err)
		}
	}
	if err := os.Rename(prev, ov); err != nil {
		// 尽力回退，避免留下「两边都不在」的状态
		_ = os.Rename(tmp, ov)
		return false, fmt.Errorf("恢复覆盖层: %w", err)
	}
	_ = os.RemoveAll(tmp)
	return true, nil
}

// gcReleases 保留最近 keep 版，删除更早的版本目录；**永不删** current 与 .last
// 指向的版本（回滚点必须一直在）。
func (p Plan) gcReleases() ([]string, error) {
	entries, err := os.ReadDir(p.ReleasesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read releases dir: %w", err)
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && versionRe.MatchString(e.Name()) {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions) // 版本号内含时间戳 → 字典序即时间序
	keep := p.keep()
	if len(versions) <= keep {
		return nil, nil
	}
	cur, _ := p.CurrentVersion()
	last, _ := p.LastVersion()
	var removed []string
	for _, v := range versions[:len(versions)-keep] {
		if v == cur || v == last {
			continue
		}
		if err := os.RemoveAll(filepath.Join(p.ReleasesDir(), v)); err != nil {
			return removed, fmt.Errorf("gc %s: %w", v, err)
		}
		removed = append(removed, v)
	}
	return removed, nil
}

// ---- 重启 ----

// Restart 受控重启生产：停旧（TERM → grace → KILL，只杀通过所有权断言的进程）
// → 起新（沿部署脚本，否则直接跑 bin/rick）→ 健康轮询 → **build_id 必须等于
// 期望版本** → 汇总挂起清单。任一步失败都返回对应阶段错误，供调用方决定回滚。
func (p Plan) Restart(expectedVersion string) (RestartResult, error) {
	res := RestartResult{Version: expectedVersion}
	stopped, skipped := p.stopProd()
	res.Stopped, res.Skipped = stopped, skipped

	if err := p.startProd(); err != nil {
		res.LogTail = p.logTail()
		return res, &StageError{Stage: "restart", Err: err}
	}
	pid, healthMS, buildID, err := p.waitHealthy()
	if err != nil {
		res.LogTail = p.logTail()
		return res, &StageError{Stage: "health", Err: err}
	}
	res.PID, res.HealthMS, res.BuildID = pid, healthMS, buildID
	if expectedVersion != "" {
		res.Matches = buildID == expectedVersion
		if !res.Matches {
			res.LogTail = p.logTail()
			return res, stageErr("health",
				"运行中的 build_id=%q 与本次提升版本 %q 不一致（新构建没生效？）", buildID, expectedVersion)
		}
	}
	res.Recovery = p.readRecovery()
	res.Overlay = p.OverlayDist()
	return res, nil
}

// stopProd 停掉生产实例：pid 文件 + /proc 扫描，逐个做所有权断言；
// 断言不通过的进程记录在 skipped（宁可不杀，也绝不误杀）。
func (p Plan) stopProd() (stopped, skipped []int) {
	seen := map[int]bool{}
	var candidates []int
	if data, err := os.ReadFile(p.ProdPidPath()); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 {
			candidates = append(candidates, pid)
			seen[pid] = true
		}
	}
	for _, pid := range findProdPIDs(p) {
		if !seen[pid] {
			candidates = append(candidates, pid)
			seen[pid] = true
		}
	}
	for _, pid := range candidates {
		if pid == os.Getpid() || !alive(pid) {
			continue
		}
		if !ownedByProd(p, pid) {
			skipped = append(skipped, pid)
			continue
		}
		if terminate(pid, p.stopGrace()) {
			stopped = append(stopped, pid)
		} else {
			skipped = append(skipped, pid)
		}
	}
	// 进程已消失但 pid 文件还在 → 清理，否则新实例会被 singleton 门禁拒绝
	if !alivePIDFile(p.ProdPidPath()) {
		_ = os.Remove(p.ProdPidPath())
	}
	return stopped, skipped
}

// startProd 拉起生产：优先用部署脚本（研究实测它就是「生产如何被拉起」的事实
// 来源），否则直接执行 bin/rick。detach（Setsid）+ 日志追加，调用方不被阻塞。
func (p Plan) startProd() error {
	name, args, dir := p.startCommand()
	if err := os.MkdirAll(p.StateDirPath(), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	logf, err := os.OpenFile(p.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open prod log: %w", err)
	}
	defer logf.Close()
	fmt.Fprintf(logf, "\n[release] ---- starting %s at %s ----\n", name, p.now().Format(time.RFC3339))

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = p.prodEnv()
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s %s: %w", name, strings.Join(args, " "), err)
	}
	_ = cmd.Process.Release()
	return nil
}

// startCommand 决定如何拉起生产。
func (p Plan) startCommand() (name string, args []string, dir string) {
	if p.StartScript != "" && fileExists(p.StartScript) {
		return "bash", []string{p.StartScript}, p.ProdRepo
	}
	return p.ProdBinLink(), []string{
		"web", "--listen", p.listen(), "--port", strconv.Itoa(p.Port),
	}, p.ProdRepo
}

// prodEnv 是生产进程的环境（白名单 + 生产 HOME/状态目录）。
func (p Plan) prodEnv() []string {
	env := []string{
		"HOME=" + p.ProdHome,
		"PATH=" + os.Getenv("PATH"),
		"RICK_STATE_DIR=" + p.StateDirPath(),
	}
	if v := os.Getenv("RICK_PI_AGENT_DIR"); v != "" {
		// 生产 pi 沙盒保持部署既有值（release 不改动 pi 运行时）
		env = append(env, "RICK_PI_AGENT_DIR="+v)
	}
	for _, k := range []string{"TMPDIR", "LANG", "LC_ALL", "TERM"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	// 网络代理必须透传：本机直连外网不通，pi worker 的模型调用（如 deepseek）
	// 遵循 *_proxy。丢了它们 → release 重启后的生产 web 会话全部 "Connection
	// error."（实测：release 拉起的生产进程 proxy 变量数=0，而用户终端拉起的有 4 个）。
	for _, k := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY", "no_proxy", "NO_PROXY"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// waitHealthy 轮询 /api/health 直到 200，返回 (pid, 耗时, build_id)。
func (p Plan) waitHealthy() (int, int64, string, error) {
	start := time.Now()
	var lastErr error
	for time.Since(start) < p.healthWait() {
		body, err := p.getJSON(p.BaseURL() + "/api/health")
		if err == nil {
			var payload struct {
				Status  string `json:"status"`
				BuildID string `json:"build_id"`
			}
			if err := json.Unmarshal(body, &payload); err == nil && payload.Status == "ok" {
				pid := 0
				if data, err := os.ReadFile(p.ProdPidPath()); err == nil {
					pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
				}
				if pid == 0 {
					if pids := findProdPIDs(p); len(pids) > 0 {
						pid = pids[0]
					}
				}
				return pid, time.Since(start).Milliseconds(), payload.BuildID, nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("health 未在 %s 内就绪", p.healthWait())
	}
	return 0, 0, "", fmt.Errorf("生产未就绪（%s /api/health）: %w", p.BaseURL(), lastErr)
}

// readRecovery 读挂起清单（文件缺失视为「本次没有挂起」）。防御式解析：
// 字段缺失/格式变化都不应让提升失败。
func (p Plan) readRecovery() SuspendSummary {
	sum := SuspendSummary{Path: filepath.Join(p.StateDirPath(), RecoveryReportFile)}
	data, err := os.ReadFile(sum.Path)
	if err != nil {
		return sum
	}
	var rep struct {
		At        string            `json:"at"`
		Suspended []json.RawMessage `json:"suspended"`
		Recovered []json.RawMessage `json:"recovered"`
		Failed    []json.RawMessage `json:"failed"`
	}
	if err := json.Unmarshal(data, &rep); err != nil {
		return sum
	}
	sum.At = rep.At
	sum.Suspended, sum.Recovered, sum.Failed = len(rep.Suspended), len(rep.Recovered), len(rep.Failed)
	for _, raw := range rep.Suspended {
		if len(sum.IDs) >= 10 {
			break
		}
		if id := recoveryItemID(raw); id != "" {
			sum.IDs = append(sum.IDs, id)
		}
	}
	return sum
}

// recoveryItemID 从一条挂起记录里取 id（对象 `{"id":…}` 或裸字符串都接受）。
func recoveryItemID(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "{") {
		var obj struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Type  string `json:"type"`
		}
		if json.Unmarshal(raw, &obj) == nil {
			// id 为主（人对得上 UI/注册表），类型+标题作为人类可读的补充
			hint := strings.TrimSpace(strings.Join(nonEmpty(obj.Type, obj.Title), " "))
			switch {
			case obj.ID != "" && hint != "":
				return obj.ID + " (" + hint + ")"
			case obj.ID != "":
				return obj.ID
			default:
				return hint
			}
		}
		return ""
	}
	return strings.Trim(trimmed, "\"")
}

func (p Plan) logTail() []string {
	data, err := os.ReadFile(p.LogPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return lines
}

// ---- 回滚 ----

// Rollback 切回上一个版本（`current` → `.last`）并把覆盖层一起回滚，然后重启。
func (p Plan) Rollback() (Result, error) {
	res := Result{Action: "rollback", Plan: p.Summary()}
	last, err := p.LastVersion()
	if err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	if last == "" {
		return res, stageErr("promote", "没有回滚点（%s 为空）——从未提升过？", p.LastRecord())
	}
	if !dirExists(p.VersionDir(last)) {
		return res, stageErr("promote", "回滚点 %s 的版本目录已不存在: %s", last, p.VersionDir(last))
	}
	cur, _ := p.CurrentVersion()
	res.Version = last
	res.PrevVersion = cur
	if err := p.switchCurrent(last); err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	restored, err := p.restoreOverlay()
	if err != nil {
		return res, &StageError{Stage: "promote", Err: err}
	}
	if restored {
		res.Overlay = p.OverlayDist()
	}
	res.Bin = filepath.Join(p.VersionDir(last), "rick")
	return res, nil
}

// DryRun 只做门禁 + 构建 + 打印计划，**不动生产、不重启**。
// DryRun 演练完整构建（门禁 + 二进制 + 前端），但**生产树零写入**：
// 产物落到 os.MkdirTemp 的一次性暂存目录，打印「正式提升本会落到哪」，
// 结束后清理暂存目录（清理失败会明确报告残留路径，不静默）。
func (p Plan) DryRun(out io.Writer) (Result, error) {
	if out == nil {
		out = io.Discard
	}
	staging, err := os.MkdirTemp("", "rick-release-dryrun-")
	if err != nil {
		return Result{Action: "dry-run", Plan: p.Summary()},
			stageErr("build", "创建 dry-run 暂存目录: %v", err)
	}
	staged := p
	staged.StagingDir = staging

	rel, buildErr := staged.Build(out)
	if buildErr != nil {
		// 构建失败也要清场（暂存目录可能已有半成品）
		if rmErr := os.RemoveAll(staging); rmErr != nil {
			fmt.Fprintf(out, "RELEASE_WARN dryrun_staging_left=%s err=%v（请手工清理）\n", staging, rmErr)
		}
		return Result{Action: "dry-run", Plan: p.Summary()}, buildErr
	}

	res := Result{Action: "dry-run", Version: rel.Version, Bin: rel.Bin,
		TargetBin:  filepath.Join(rel.TargetDir, "rick"),
		TargetDist: filepath.Join(rel.TargetDir, "dist"),
		StagingDir: rel.Dir,
		DistFiles:  rel.DistFiles, SHA256: rel.SHA256, Plan: p.Summary()}
	fmt.Fprintf(out, "RELEASE_DRYRUN version=%s dist_files=%d sha256=%s\n",
		rel.Version, rel.DistFiles, short(rel.SHA256))
	fmt.Fprintf(out, "RELEASE_DRYRUN target_bin=%s\n", res.TargetBin)
	fmt.Fprintf(out, "RELEASE_DRYRUN target_dist=%s\n", res.TargetDist)
	fmt.Fprintf(out, "RELEASE_DRYRUN staging=%s (临时；不影响生产树)\n", rel.Dir)
	fmt.Fprintln(out, "RELEASE_DRYRUN prod_untouched=true（未写生产树、未换链、未重启；所有会话/任务不会被打断）")

	// 清理暂存：失败必须显式报告（不静默留垃圾）
	if rmErr := os.RemoveAll(staging); rmErr != nil {
		fmt.Fprintf(out, "RELEASE_WARN dryrun_staging_left=%s err=%v（请手工清理）\n", staging, rmErr)
	} else {
		fmt.Fprintln(out, "RELEASE_DRYRUN cleanup=ok")
	}
	return res, nil
}

// ExitOK / ExitGate / ExitBuild / ExitRestart / ExitHealth 是 `rick tools release`
// 的退出码语义（AI 会话据此判断该重试哪一步，不必解析日志）：
//
//	0 ok / 2 门禁失败 / 3 构建失败 / 4 提升或重启失败 / 5 健康（含 build_id 校验）失败
const (
	ExitOK      = 0
	ExitGate    = 2
	ExitBuild   = 3
	ExitRestart = 4
	ExitHealth  = 5
)

// ExitCodeForStage 把失败阶段映射为退出码。
func ExitCodeForStage(stage string) int {
	switch stage {
	case "gate":
		return ExitGate
	case "build":
		return ExitBuild
	case "health":
		return ExitHealth
	default: // promote / restart / 未知 → 按重启失败处理
		return ExitRestart
	}
}

// Release 是完整提升：Build → Promote → Restart。
//
// **重启失败自动回滚**：如果提升前存在可用版本（PrevVersion 非空），则切回上一版
// 并重启，让生产先活下来，再把原始失败原因返回给调用方（宁可回到旧版可用，也不能
// 留在「新二进制起不来」的状态）。首次提升无回滚点时不做自动回滚。
func (p Plan) Release(out io.Writer) (Result, error) {
	if out == nil {
		out = io.Discard
	}
	rel, err := p.Build(out)
	if err != nil {
		return Result{Action: "release", Plan: p.Summary()}, err
	}
	return p.Rollout(rel, out)
}

// Rollout 用**已构建好**的 Release 执行 Promote + Restart（含重启失败自动回滚）。
// 单独暴露是为了让 CLI 能先构建、把「报价单」（版本/sha256/门禁结果）拿给人确认，
// 确认之后再落盘 —— 人类批准的是**冻结了 sha256 的确定产物**，不是「一个分支」。
func (p Plan) Rollout(rel Release, out io.Writer) (Result, error) {
	if out == nil {
		out = io.Discard
	}
	res, err := p.Promote(rel)
	if err != nil {
		return res, err
	}
	rr, err := p.Restart(rel.Version)
	res.Restart = &rr
	if err == nil {
		return res, nil
	}
	if res.PrevVersion == "" {
		return res, err // 首次提升：没有可回滚的版本
	}
	fmt.Fprintf(out, "RELEASE_AUTOROLLBACK reason=%v rollback_to=%s\n", err, res.PrevVersion)
	if rbErr := p.forceSwitch(res.PrevVersion); rbErr != nil {
		return res, stageErr("promote", "自动回滚失败: %v（原始错误: %v）", rbErr, err)
	}
	if _, rbErr := p.Restart(res.PrevVersion); rbErr != nil {
		return res, stageErr("health", "自动回滚后生产仍未就绪: %v（原始错误: %v）", rbErr, err)
	}
	res.Action = "release-rolled-back"
	fmt.Fprintf(out, "RELEASE_ROLLED_BACK version=%s\n", res.PrevVersion)
	return res, err
}

// forceSwitch 把 current 链与覆盖层切到指定版本（自动回滚用；不读 .last）。
func (p Plan) forceSwitch(version string) error {
	if !dirExists(p.VersionDir(version)) {
		return fmt.Errorf("版本目录不存在: %s", p.VersionDir(version))
	}
	if err := p.switchCurrent(version); err != nil {
		return err
	}
	_, err := p.restoreOverlay()
	return err
}

// ---- 状态与审计 ----

// Status 汇报生产当前版本 / 运行指纹 / 回滚点。
type Status struct {
	CurrentVersion string         `json:"current_version"`
	LastVersion    string         `json:"last_version,omitempty"`
	RunningBuildID string         `json:"running_build_id,omitempty"`
	PID            int            `json:"pid,omitempty"`
	Matches        bool           `json:"matches"`
	Healthy        bool           `json:"healthy"`
	Overlay        string         `json:"overlay,omitempty"`
	OverlayExists  bool           `json:"overlay_exists"`
	Recovery       SuspendSummary `json:"recovery"`
	Plan           PlanSummary    `json:"plan"`
}

// Status 采集状态（只读，不做任何修改）。
func (p Plan) Status() Status {
	cur, _ := p.CurrentVersion()
	last, _ := p.LastVersion()
	st := Status{CurrentVersion: cur, LastVersion: last, Plan: p.Summary()}
	st.Overlay = p.OverlayDist()
	st.OverlayExists = dirExists(st.Overlay)
	st.Recovery = p.readRecovery()
	if body, err := p.getJSON(p.BaseURL() + "/api/health"); err == nil {
		var payload struct {
			Status  string `json:"status"`
			BuildID string `json:"build_id"`
		}
		if json.Unmarshal(body, &payload) == nil {
			st.Healthy = payload.Status == "ok"
			st.RunningBuildID = payload.BuildID
			st.Matches = payload.BuildID != "" && payload.BuildID == cur
		}
	}
	if data, err := os.ReadFile(p.ProdPidPath()); err == nil {
		st.PID, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	return st
}

// ReleaseState 是提升过程的留痕（供 --detach 后台执行后轮询）。
type ReleaseState struct {
	At       string         `json:"at"`
	Action   string         `json:"action"`
	Version  string         `json:"version,omitempty"`
	Stage    string         `json:"stage"`
	OK       bool           `json:"ok"`
	Error    string         `json:"error,omitempty"`
	PID      int            `json:"pid,omitempty"`
	BuildID  string         `json:"build_id,omitempty"`
	Recovery SuspendSummary `json:"recovery,omitempty"`
	Restart  *RestartResult `json:"restart,omitempty"`
	Result   *Result        `json:"result,omitempty"`
}

// WriteState 原子写提升留痕。
func (p Plan) WriteState(st ReleaseState) error {
	if st.At == "" {
		st.At = p.now().Format(time.RFC3339)
	}
	if err := os.MkdirAll(p.StateDirPath(), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(p.StatePath(), append(body, '\n'), 0o644)
}

// ReadState 读最近一次提升留痕（缺失不算错）。
func (p Plan) ReadState() (ReleaseState, error) {
	var st ReleaseState
	data, err := os.ReadFile(p.StatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	return st, json.Unmarshal(data, &st)
}

// ---- 版本链读取 ----

// CurrentVersion 返回 `current` 链指向的版本号（无链 → 空）。
func (p Plan) CurrentVersion() (string, error) {
	target, err := os.Readlink(p.CurrentLink())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("readlink %s: %w", p.CurrentLink(), err)
	}
	return filepath.Base(target), nil
}

// LastVersion 返回回滚点版本号（无记录 → 空）。
func (p Plan) LastVersion() (string, error) {
	data, err := os.ReadFile(p.LastRecord())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", p.LastRecord(), err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ---- 进程工具（与 devweb 同构，但断言对象是生产）----

// ownedByProd 是**防误杀的核心断言**：只有能证明进程属于该生产部署才返回 true
// ——cwd 是生产工作树、或可执行文件在生产 bin 目录下、或 environ 的 HOME 是生产 HOME。
func ownedByProd(p Plan, pid int) bool {
	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil && cwd == p.ProdRepo {
		return true
	}
	if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
		clean := strings.TrimSuffix(exe, " (deleted)")
		if strings.HasPrefix(clean, filepath.Join(p.ProdRepo, "bin")+string(filepath.Separator)) {
			return true
		}
	}
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid)); err == nil {
		for _, kv := range strings.Split(string(data), "\x00") {
			if kv == "HOME="+p.ProdHome {
				return true
			}
		}
	}
	return false
}

// findProdPIDs 在 /proc 里找属于该生产部署的进程（pid 文件缺失/陈旧时的兜底）。
func findProdPIDs(p Plan) []int {
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
		if ownedByProd(p, pid) {
			out = append(out, pid)
		}
	}
	return out
}

func alive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	idx := strings.LastIndexByte(string(data), ')')
	if idx < 0 || idx+2 >= len(data) {
		return true
	}
	return string(data[idx+2]) != "Z"
}

func alivePIDFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	return alive(pid)
}

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
		time.Sleep(150 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	for i := 0; i < 30; i++ {
		if !alive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !alive(pid)
}

// HostedByProd 判断「当前进程是否被该生产的实例托管」（AI 会话正在被 release
// 重启掉自己时的自保信号）：向上遍历祖先进程，命中生产 pid 或 cwd 在生产工作树
// 即视为托管。
func HostedByProd(p Plan) (bool, string) {
	prodPIDs := map[int]bool{}
	for _, pid := range findProdPIDs(p) {
		prodPIDs[pid] = true
	}
	pid := os.Getpid()
	for i := 0; i < 12; i++ {
		ppid := parentPID(pid)
		if ppid <= 1 || ppid == pid {
			return false, ""
		}
		if prodPIDs[ppid] {
			return true, fmt.Sprintf("祖先进程 %d 是生产实例", ppid)
		}
		if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", ppid)); err == nil && cwd == p.ProdRepo {
			return true, fmt.Sprintf("祖先进程 %d 的工作目录是生产工作树", ppid)
		}
		pid = ppid
	}
	return false, ""
}

func parentPID(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	idx := strings.LastIndexByte(string(data), ')')
	if idx < 0 || idx+2 >= len(data) {
		return 0
	}
	fields := strings.Fields(string(data)[idx+2:])
	if len(fields) < 2 {
		return 0
	}
	ppid, _ := strconv.Atoi(fields[1])
	return ppid
}

// ---- HTTP / 文件工具 ----

func (p Plan) getJSON(url string) ([]byte, error) {
	if p.HTTPGet != nil {
		return p.HTTPGet(url, p.Token)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
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

func writeChecksums(rel Release) error {
	sum, err := fileSHA256(rel.Bin)
	if err != nil {
		return fmt.Errorf("sha256: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n", sum, filepath.Base(rel.Bin))
	if dirExists(rel.Dist) {
		_ = filepath.WalkDir(rel.Dist, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if s, err := fileSHA256(path); err == nil {
				rel, _ := filepath.Rel(rel.Dist, path)
				fmt.Fprintf(&b, "%s  dist/%s\n", s, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	return os.WriteFile(filepath.Join(rel.Dir, ChecksumName), []byte(b.String()), 0o644)
}

// writeChanges 记录「这次提升带来了什么」（人类审核的输入之一）：prod HEAD →
// dev HEAD 的提交与 diffstat。git 不可用时不阻断（best-effort）。
func writeChanges(p Plan, rel Release, out io.Writer) {
	var b strings.Builder
	if sha := shortSHA(p.DevTree); sha != "" {
		b.WriteString("dev HEAD: " + sha + "\n")
	}
	for _, args := range [][]string{{"log", "--oneline", "-20"}, {"diff", "--stat", "HEAD~1..HEAD"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = p.DevTree
		raw, err := cmd.Output()
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "\n$ git %s\n%s", strings.Join(args, " "), string(raw))
	}
	if b.Len() == 0 {
		return
	}
	_ = os.WriteFile(filepath.Join(rel.Dir, ChangesName), []byte(b.String()), 0o644)
	if out != nil {
		fmt.Fprintf(out, "RELEASE_CHANGES %s\n", filepath.Join(rel.Dir, ChangesName))
	}
}

func shortSHA(dir string) string {
	if dir == "" {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "--short=7", "HEAD")
	cmd.Dir = dir
	raw, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func copyTree(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 没有前端产物 = 跳过
		}
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s 不是目录", src)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil // 跳过符号链接/设备文件（前端产物不应含）
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func countFiles(dir string) (int, error) {
	if !dirExists(dir) {
		return 0, nil
	}
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n, err
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// nonEmpty 返回入参里的非空项（保持顺序）。
func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

// errNoPort 表示无法从部署事实推导生产端口（拒绝猜：猜错会重启错对象）。
var errNoPort = errors.New("无法确定生产端口：请显式传 --port 或设置 " + EnvProdPort)
