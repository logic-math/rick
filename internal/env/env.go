// Package env 收口 rick 的 agent runtime 环境管理（env 四职责）。
//
// env 保证 rick 在当前机器的运行环境就绪：pi 及扩展、rick 自有定制。它是四层
// 架构第三层「执行」的一员，对下依赖 internal/runtime 暴露的共享路径工具
// （AgentDir/RuntimeDir/RuntimeBin/SettingsPath/AgentEnv/FileExists/EnsureAgentDir），
// 对上被 cli（薄入口）与 handler（调度聚合）调用。
//
// 四职责：
//  1. 安装/更新 pi agent（pi.go）
//  2. 安装/更新 pi 生态扩展/插件/skill（extensions.go）
//  3. 安装/更新 rick 自有 hook/skill/agent 定制（customizations.go）
//  4. 提供 pi 功能点就绪 check 函数，不含 session（check.go）
//
// 扩展 seam：RuntimeEnv 接口 + piEnv 实现，将来新增 dsh runtime 时只实现
// 对应的 RuntimeEnv 并注册，cli/handler 不改。
package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunquan/rick/internal/runtime"
)

// RuntimeEnv 是 env 抽象的扩展 seam（单一 runtime pi + dsh 预留）。pi 是当前
// 唯一实现；将来 dsh 只新增对应的 RuntimeEnv（安装方式/扩展机制/定制落盘格式/就绪 check
// 各自实现）并注册，cli/handler/方法层 templates 不改。
type RuntimeEnv interface {
	// Ensure 保证 runtime 就绪（安装/更新 runtime + 生态扩展 + 功能点校验）。
	Ensure() error
	// DeployCustomizations 落盘 rick 自有定制（hook/skill/agent）。
	DeployCustomizations() error
	// CheckReady 返回未就绪的功能点（空切片 = 就绪）。
	CheckReady() []string
}

// piEnv 是 RuntimeEnv 的 pi 实现，四职责收口于此。
type piEnv struct{}

// NewPiEnv 构造 pi env 实现。
func NewPiEnv() *piEnv { return &piEnv{} }

// Ensure 委托给包级 Ensure（完整 init-pi 流程）。
func (e *piEnv) Ensure() error { return Ensure() }

// DeployCustomizations 委托给包级 DeployRickCustomizations。
func (e *piEnv) DeployCustomizations() error { return DeployRickCustomizations() }

// CheckReady 委托给包级 CheckReady。
func (e *piEnv) CheckReady() []string { return CheckReady() }

// Ensure 执行完整的「保证 pi 就绪」流程并返回错误。仅在 pi 完全不可用（rick 无法
// 运行）时返回错误；缺失扩展/主题/定制仅告警，不阻断。输出与迁移前 runInitPi 逐字
// 一致（职责 3 定制落盘静默执行，不新增输出行）。
func Ensure() error {
	// Step 0: 前置检查 —— 仅当 rick 托管 pi runtime 尚未安装时。pi 是 Node.js
	// 程序（>= 22.19.0），其 npm 安装需要 node/npm 在 PATH 上。rick 不安装 node
	// （用户管理的环境依赖）。托管 runtime 已存在时视为环境就绪，跳过此检查。
	if !runtime.FileExists(runtime.RuntimeBin()) {
		if err := RequireNodeForPiInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			return err
		}
	}

	// Step 1: rick 自闭环 pi runtime 就绪（缺失则安装）。
	piPath, piNewlyInstalled, err := EnsurePI()
	if err != nil {
		// Fatal: 无 pi 则 rick 无法执行任何 agent 命令。
		fmt.Fprintf(os.Stderr, "❌ pi is not available and could not be installed: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Install manually: npm install -g @earendil-works/pi-coding-agent\n")
		return err
	}
	ver := PiVersion(piPath)
	fmt.Printf("✅ rick pi runtime ready: %s", piPath)
	if ver != "" {
		fmt.Printf(" (v%s)", ver)
	}
	if piNewlyInstalled {
		fmt.Printf(" (newly installed)")
	}
	fmt.Println()

	// Step 1.5: rick 托管 agent 目录 + settings 引导（隔离配置 + hideThinkingBlock）。
	if err := BootstrapAgentSettings(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  managed agent dir: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick will still run, but pi config isolation may be incomplete.\n")
	} else {
		fmt.Printf("✅ pi agent dir ready: %s (hideThinkingBlock=true)\n", runtime.AgentDir())
	}

	// Step 2: subagent 扩展注册（缺失则安装）。非致命。
	if err := EnsureNpmExtension("pi-subagents", "pi-subagents"); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  subagent extension: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it, but Sub Agent delegation is unavailable.\n")
	} else {
		fmt.Println("✅ pi subagent extension ready")
	}

	// Step 3: web-access 扩展注册（缺失则安装）。非致命。
	if err := EnsureNpmExtension("pi-web-access", "web-access"); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  web-access extension: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it, but external web search/fetch is unavailable.\n")
	} else {
		fmt.Println("✅ pi web-access extension ready")
	}

	// Step 4: 从托管配置清除 Tokyo Night 包。非致命。
	if err := PurgeTokyoNight(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  tokyo-night purge: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it; the package may still be listed.\n")
	} else {
		fmt.Println("✅ tokyo-night purged from managed config (theme/packages)")
	}

	// Step 5: 校验所有必需扩展 + 主题已注册。
	missing := VerifyExtensions()
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  verification: these extensions are NOT registered: %s\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "   rick may be degraded. Re-run `rick tools init-pi` or install manually.\n")
	} else {
		fmt.Println("✅ verification: all required extensions registered")
	}
	if cur := CurrentTheme(); cur == "" {
		fmt.Fprintf(os.Stderr, "⚠️  verification: no theme set in settings.json\n")
	} else {
		fmt.Printf("✅ verification: theme %s active\n", cur)
	}

	// 职责 3：rick 自有定制（静默落盘，不影响 init-pi 输出基线）。
	_ = DeployRickCustomizations()

	fmt.Println("✅ pi environment ready")
	return nil
}
