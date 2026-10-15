package relocate

import (
	"path"
	"strings"
)

// Root identifies which archive root an exclude rule applies to. Patterns
// are authored relative to their own root so one rule table serves the
// workspace tree(s), the rick state tree (~/.rick), and the pi agent tree
// (~/.rick/pi/agent) without pattern collisions between roots ("bin" means
// different things in a repo vs the pi agent dir).
type Root string

const (
	// RootWorkspace is a packed workspace's tree root.
	RootWorkspace Root = "workspace"
	// RootState is the rick state dir root (~/.rick).
	RootState Root = "state"
	// RootPIAgent is the pi agent dir root (~/.rick/pi/agent).
	RootPIAgent Root = "pi-agent"
)

// Rule kinds — the values recorded in Manifest.Excluded[].Kind.
const (
	// KindPlatformELF: machine-bound binary, rebuilt per target platform.
	KindPlatformELF = "platform-elf"
	// KindRegenerable: reproducible on the target (npm ci, go build).
	KindRegenerable = "regenerable"
	// KindCache: safe to drop, regrown lazily.
	KindCache = "cache"
	// KindRuntime: machine-local runtime state (pid / lock / log).
	KindRuntime = "runtime"
	// KindExternal: outside rick's ownership (human-ruled exclusion).
	KindExternal = "external"
)

// Rule excludes one path — a file or a directory subtree — from a dump.
// Pattern is a slash-separated path relative to Root. A pattern without
// glob metacharacters matches the exact path and, for directories, the
// whole subtree below it (path-segment aware). A pattern with
// metacharacters (e.g. "config.json.bak.*") is matched with path.Match
// against the relative path itself.
type Rule struct {
	Pattern string
	Root    Root
	Kind    string
	Reason  string
}

// DefaultRules is the human-approved default exclusion table (job_46
// design tree leaf layer, "无缝运行优先于字节级全量"). Everything excluded
// here is either machine-bound, regenerable, or deliberate scope ruling;
// the manifest records each hit so the dump stays auditable.
func DefaultRules() []Rule {
	return []Rule{
		{Pattern: "web/node_modules", Root: RootWorkspace, Kind: KindRegenerable, Reason: "npm ci 可再生；含 linux-x64 原生模块，跨平台必坏"},
		{Pattern: "bin", Root: RootWorkspace, Kind: KindRegenerable, Reason: "构建产物；归档内嵌多平台二进制"},
		{Pattern: "web/dist.prev", Root: RootWorkspace, Kind: KindRegenerable, Reason: "回滚覆盖层，release 重建"},
		{Pattern: ".pi", Root: RootWorkspace, Kind: KindCache, Reason: "pi 会话 artifacts 缓存"},
		{Pattern: ".morty", Root: RootWorkspace, Kind: KindCache, Reason: "外部工具状态"},

		{Pattern: "web.pid", Root: RootState, Kind: KindRuntime, Reason: "判活 pid，跨机无意义"},
		{Pattern: "web.lock", Root: RootState, Kind: KindRuntime, Reason: "flock 锁文件"},
		{Pattern: "web.log", Root: RootState, Kind: KindRuntime, Reason: "运行日志"},
		{Pattern: "release.log", Root: RootState, Kind: KindRuntime, Reason: "release 日志"},
		{Pattern: "config.json.bak.*", Root: RootState, Kind: KindCache, Reason: "人工备份"},

		{Pattern: "web-search-cache", Root: RootPIAgent, Kind: KindCache, Reason: "pi-web-access 搜索缓存"},
		{Pattern: "bin/rg", Root: RootPIAgent, Kind: KindPlatformELF, Reason: "Linux ELF；pi 会按目标平台重下"},
		{Pattern: "extensions/catpaw-proxy.ts*", Root: RootPIAgent, Kind: KindExternal, Reason: "provider 专属（软链指向仓库外 + 凭据在 ~/.config）；human 裁决不迁移"},
	}
}

// Matcher applies exclusion rules per root. The zero rules + disabled=false
// combination excludes nothing. disabled=true (dump --no-exclude) disables
// every rule regardless of origin.
type Matcher struct {
	rules    []Rule
	disabled bool
}

// NewMatcher builds a matcher over rules; disabled short-circuits every
// lookup to "keep" (used by dump --no-exclude).
func NewMatcher(rules []Rule, disabled bool) *Matcher {
	return &Matcher{rules: rules, disabled: disabled}
}

// Match reports whether relPath (relative to root, "" = the root itself)
// is excluded, returning the rule that hit. The root itself is never
// excluded; directory rules exclude their whole subtree.
func (m *Matcher) Match(root Root, relPath string) (Rule, bool) {
	if m == nil || m.disabled || relPath == "" {
		return Rule{}, false
	}
	relPath = strings.Trim(relPath, "/")
	if relPath == "" {
		return Rule{}, false
	}
	for _, r := range m.rules {
		if r.Root != root {
			continue
		}
		if ruleMatches(r.Pattern, relPath) {
			return r, true
		}
	}
	return Rule{}, false
}

// MatchFunc returns a func(relPath string) bool for one root — the per-item
// exclude hook shape consumed by archive.Item.Match.
func (m *Matcher) MatchFunc(root Root) func(relPath string) bool {
	return func(relPath string) bool {
		_, excluded := m.Match(root, relPath)
		return !excluded // archive semantics: true = keep
	}
}

// ruleMatches applies one pattern to one relative path.
func ruleMatches(pattern, relPath string) bool {
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := path.Match(pattern, relPath)
		return err == nil && ok
	}
	// Exact file, or directory subtree (segment-aware prefix).
	return relPath == pattern || strings.HasPrefix(relPath, pattern+"/")
}
