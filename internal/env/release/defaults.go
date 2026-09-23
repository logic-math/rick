package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// DefaultPlan 从**部署事实**推导提升计划：端口/监听地址/状态目录都从部署脚本
// （默认 `~/.rick/start-web.sh`）或环境变量读，读不到就报错——**绝不猜**一个
// 「常见端口」：猜错意味着重启错对象，代价不可逆。
//
// 优先级：显式参数 > 环境变量（RICK_PROD_PORT / RICK_PROD_REPO / RICK_DEV_TREE）
// > 部署脚本 > 生产 HOME 下的约定路径。
func DefaultPlan(prodRepo, devTree string, portOverride int) (Plan, error) {
	repo := firstNonEmpty(prodRepo, os.Getenv(EnvProdRepo))
	if repo == "" {
		return Plan{}, fmt.Errorf("无法确定生产仓库路径：请显式传 --prod-repo")
	}
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve prod repo: %w", err)
	}
	prodHome, err := realHomeDir()
	if err != nil {
		return Plan{}, err
	}

	script := filepath.Join(prodHome, ".rick", "start-web.sh")
	var scriptPort int
	scriptListen, scriptStateDir := "", ""
	if fileExists(script) {
		body, err := os.ReadFile(script)
		if err != nil {
			return Plan{}, fmt.Errorf("read %s: %w", script, err)
		}
		scriptPort = parseScriptPort(string(body))
		scriptListen = parseScriptFlag(string(body), "listen")
		scriptStateDir = parseScriptFlag(string(body), "state-dir")
	} else {
		script = "" // 部署脚本不存在 → 由调用方决定怎么拉起（直接用 bin/rick）
	}

	stateDir := firstNonEmpty(os.Getenv("RICK_STATE_DIR"), scriptStateDir, filepath.Join(prodHome, ".rick"))

	// ReleasesRoot 覆盖（发布目录被外部重置属主/ACL 时的逃生通道）：
	// RICK_RELEASES_DIR > 默认 <ProdRepo>/bin/releases。见 Plan.ReleasesRoot 注释。
	releasesRoot := strings.TrimSpace(os.Getenv("RICK_RELEASES_DIR"))

	port := portOverride
	if port <= 0 {
		if v := strings.TrimSpace(os.Getenv(EnvProdPort)); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 || n > 65535 {
				return Plan{}, fmt.Errorf("invalid %s=%q", EnvProdPort, v)
			}
			port = n
		}
	}
	if port <= 0 {
		port = scriptPort
	}
	if port <= 0 {
		return Plan{}, errNoPort
	}

	tree := firstNonEmpty(devTree, os.Getenv(EnvDevTree))
	if tree == "" {
		tree = filepath.Join(filepath.Dir(filepath.Dir(absRepo)), "rick-dev")
	}

	return Plan{
		ProdRepo:     absRepo,
		DevTree:      tree,
		ProdHome:     prodHome,
		ReleasesRoot: releasesRoot,
		StateDir:     stateDir,
		Port:         port,
		Listen:       firstNonEmpty(scriptListen, DefaultListen),
		StartScript:  script,
		Token:        readProdToken(stateDir),
		GoCache:      firstNonEmpty(os.Getenv("GOCACHE"), filepath.Join(prodHome, ".cache", "go-build")),
		GoModCache:   firstNonEmpty(os.Getenv("GOMODCACHE"), filepath.Join(prodHome, "go", "pkg", "mod")),
		NpmCache:     firstNonEmpty(os.Getenv("npm_config_cache"), filepath.Join(prodHome, ".npm")),
	}, nil
}

// Validate 做提升前的基本自检：产物目录必须存在、生产工作树必须像个仓库、
// dev 树不能就是生产树（自己提升自己会在重启瞬间把构建源也换掉）。
func (p Plan) Validate() error {
	if p.ProdRepo == "" {
		return fmt.Errorf("生产仓库路径为空")
	}
	if samePath(p.ProdRepo, p.DevTree) {
		return fmt.Errorf("dev 工作树与生产工作树是同一个目录（%s）——提升无意义且危险", p.ProdRepo)
	}
	for _, f := range []string{filepath.Join(p.ProdRepo, "go.mod")} {
		if !fileExists(f) {
			return fmt.Errorf("%s 不存在：%s 不像 rick 生产仓库", f, p.ProdRepo)
		}
	}
	if !fileExists(filepath.Join(p.DevTree, "go.mod")) {
		return fmt.Errorf("%s/go.mod 不存在：dev 工作树不完整（先 `rick tools dev-web init`）", p.DevTree)
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("生产端口非法: %d", p.Port)
	}
	return nil
}

// parseScriptPort 从部署脚本里取 `--port N`（端口是部署事实，不写死在代码里）。
func parseScriptPort(script string) int {
	v := parseScriptFlag(script, "port")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.Trim(v, `"'`))
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}

// parseScriptFlag 从部署脚本里取 `--<name> <value>`（支持 `--name=value` 写法）。
func parseScriptFlag(script, name string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`--` + regexp.QuoteMeta(name) + `[=\s]+([^\s"'\\]+)`),
		regexp.MustCompile(`--` + regexp.QuoteMeta(name) + `[=\s]+"([^"]+)"`),
		regexp.MustCompile(`--` + regexp.QuoteMeta(name) + `[=\s]+'([^']+)'`),
	}
	for _, re := range patterns {
		if m := re.FindStringSubmatch(script); len(m) == 2 && strings.TrimSpace(m[1]) != "" {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// readProdToken 只读取生产 web token（用于重启后查询会话/恢复报告）。
// 读不到不算错：token 缺失只影响「带鉴权的附加查询」，不影响提升本身。
func readProdToken(stateDir string) string {
	data, err := os.ReadFile(filepath.Join(stateDir, "config.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		WebToken string `json:"web_token"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return strings.TrimSpace(cfg.WebToken)
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, err1 := filepath.EvalSymlinks(a)
	cb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ca == cb
}

// realHomeDir 返回**真实**家目录（不跟随 $HOME）：提升命令可能被 dev 实例
// 托管的会话调用（其 HOME 是 dev HOME），但生产 HOME 必须始终是真实家目录。
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
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}
	return "", fmt.Errorf("cannot determine real home directory")
}
