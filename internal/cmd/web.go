// web.go registers the `rick web` command tree: the server (default), plus
// the customize/reset subcommands driving the frontend self-iteration layer
// (env.DeployWebScaffold / env.ResetWebCustomization).
//
// 本文件同时是 rick web 的组合根（rick-spec 例外三：cmd RunE 越级豁免）——
// internal/web 的装配（registries/supervisor/hub/session manager/NewServer）
// 在此完成并注入 handler.Web：internal/web 依赖 internal/handler（sessions
// 消费 DoingIn/DreamIn 等导出），handler 不能反向 import（循环）。
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/env"
	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/web"
	webembed "github.com/sunquan/rick/web"
)

// NewWebCmd builds the rick web command tree. version flows from
// cmd/rick/main.go's VERSION constant (B7: package main is unreachable from
// internal/*, so root.go passes it down — NewRootCmd already holds it).
func NewWebCmd(version string) *cobra.Command {
	var (
		port     int
		listen   string
		token    string
		stateDir string
	)

	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Run the rick web UI server (multi-workspace, multi-device)",
		Long: `rick web 启动 rick 的 Web UI 中心服务（全机唯一单实例）。

浏览器（桌面/移动多端）访问 http://127.0.0.1:6137 操作 rick 的完整功能：
多工作区管理、会话型交互（plan/easy/ctrl/human-loop/learning/dream）、
doing/dream 后台监控、jobs/tasks 看板与 knowledge 浏览。

默认仅监听 loopback；--listen 0.0.0.0 显式开放局域网（单用户 token 认证）。
HTTPS 建议交由反向代理（见 wiki/web-ui.md）。

子命令：
  customize  抽取前端源码基线到 ~/.rick/web/（自迭代入口）
  reset      清除前端自定义层，恢复内嵌 baseline`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 状态目录显式化（--state-dir > RICK_STATE_DIR > $HOME/.rick）：
			// 默认路径行为与改造前一致。
			resolved, err := web.ResolveStateDir(stateDir)
			if err != nil {
				return err
			}
			web.SetStateDir(resolved)

			// dev 隔离守卫（两种 dev 形态都装 —— F1 修复）：
			// ① 拒绝注册生产注册表已拥有的工作区（防 dev 会话写生产 .rick）；
			// ② 只换状态目录（HOME 未换）时强制显式 RICK_PI_AGENT_DIR。
			if _, err := configureDevIsolation(resolved); err != nil {
				return err
			}

			// flock 强约束：同一状态目录只允许一个实例（pid 文件可被删，
			// 内核锁不会）。
			release, err := web.AcquireStateLock(resolved)
			if err != nil {
				return err
			}
			defer release()

			opts := handler.WebOptions{
				Port:     port,
				Listen:   listen,
				Token:    token,
				Verbose:  GetVerbose(),
				Version:  version,
				StateDir: resolved,
				PidPath:  filepath.Join(resolved, "web.pid"),
			}
			return handler.Web(opts, webServeComposition)
		},
	}

	webCmd.Flags().IntVar(&port, "port", 6137, "listen port (C-137)")
	webCmd.Flags().StringVar(&listen, "listen", "127.0.0.1", "bind address (0.0.0.0 opens to LAN — deliberate)")
	webCmd.Flags().StringVar(&token, "token", "", "auth token (overrides config web_token; auto-generated when unset)")
	webCmd.Flags().StringVar(&stateDir, "state-dir", "",
		"web state directory (default $HOME/.rick; also via RICK_STATE_DIR) — dev isolation: pin it together with RICK_PI_AGENT_DIR")

	webCmd.AddCommand(newWebCustomizeCmd())
	webCmd.AddCommand(newWebResetCmd())
	return webCmd
}

// orNone renders an optional path for the dev banner.
func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unset)"
	}
	return s
}

// configureDevIsolation wires the dev-instance isolation guards and returns a
// human-readable mode label ("" when this is a production instance).
//
// F1 修复：旧实现用 IsDevStateDir(resolved)（= resolved != $HOME/.rick）当开关，
// 而 `rick tools dev-web` 启动 dev 实例用的是**换 HOME**形态（HOME=<dev-home>，
// 状态目录仍是默认的 $HOME/.rick）→ 开关恒为 false → **守卫根本没装**。实测：
// 换 HOME 起 dev 后 `POST /api/workspaces` 注册生产工作区 /workdir/.../BERT_KEETA
// 返回 HTTP 201（本应 4xx），dev 会话因此可以写坏生产工作区的 .rick/。
//
// 现在按「状态目录是否等于**真实用户**的生产状态目录」判定，两种形态都装守卫：
//   - home-swapped  ：HOME 已换 → pi 沙盒随 HOME 隔离，不必强制 RICK_PI_AGENT_DIR；
//   - state-dir-only：HOME 未换 → pi 沙盒仍指生产，必须显式 RICK_PI_AGENT_DIR。
func configureDevIsolation(resolved string) (string, error) {
	isDev, prodState, homeSwapped := web.DevModeInfo(resolved)
	if !isDev {
		// 生产实例：确保不残留 guard（进程内单例，测试/重复调用安全）
		web.SetWorkspaceAddGuard(nil)
		return "", nil
	}

	agentDir := strings.TrimSpace(os.Getenv("RICK_PI_AGENT_DIR"))
	if !homeSwapped && agentDir == "" {
		return "", fmt.Errorf("dev 模式（--state-dir=%s）必须同时显式指定 RICK_PI_AGENT_DIR。"+
			"\n否则 web 状态隔离了、pi 沙盒仍指向生产（$HOME/.rick/pi/agent），"+
			"\ndev 跑 tools init-pi/update-pi 会就地覆写生产的 agent runtime。", resolved)
	}

	web.SetWorkspaceAddGuard(web.ProdWorkspaceGuard(prodState))
	mode := "state-dir-only"
	if homeSwapped {
		mode = "home-swapped"
	}
	agentLabel := agentDir
	if agentLabel == "" {
		agentLabel = "(HOME-derived)"
	}
	fmt.Printf("[rick-web] DEV MODE mode=%s state-dir=%s agent-dir=%s prod-state=%s\n",
		mode, resolved, agentLabel, orNone(prodState))
	return mode, nil
}

// webServeComposition is the composition root injected into handler.Web:
// assembles the internal/web server (registries + supervisor + hub +
// session manager + static) and serves until ctx is cancelled.
func webServeComposition(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
	// Concrete pi runtime here (cmd/doing.go pattern — composition root).
	rt := runtime.NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...)

	workspaces, err := web.LoadWorkspaceRegistry(web.WebConfigPath())
	if err != nil {
		return fmt.Errorf("load workspace registry: %w", err)
	}
	archived, err := web.LoadArchived(web.ArchivedPath())
	if err != nil {
		return fmt.Errorf("load archived registry: %w", err)
	}
	sessions, err := web.LoadSessionRegistry(web.SessionsPath())
	if err != nil {
		return fmt.Errorf("load session registry: %w", err)
	}
	jobNames, err := web.LoadJobNameStore(web.JobNamesPath())
	if err != nil {
		return fmt.Errorf("load job names: %w", err)
	}
	hub := web.NewHub(0)
	sup := runtime.NewSupervisor(runtime.SupervisorConfig{
		PiPath:    cfg.PiPath,
		ExtraArgs: cfg.PiExtraArgs,
		// MaxActive/IdleTimeout/Heartbeat: supervisor defaults (8 / 30m / 30s).
	})
	sm := web.NewSessionManager(sessions, workspaces, sup, hub, rt, nil, nil)
	// 用户自定义 job 任务名（侧栏会话行 + Jobs 页展示）
	sm.SetJobNames(jobNames)
	// 重启对账：active/running 但无 worker 的会话标记 error（前端显示 Resume 恢复）
	sm.ReconcileOnStart()
	// 后台型会话的历史空标题回填（doing job_N / dream ×N）——旧数据侧栏只显示会话 id
	sm.BackfillTitles()
	// 历史会话的公开 job 参数回填（只有 _job_id 的行 → 前端可见/可重命名）
	sm.BackfillJobParams()

	addr := fmt.Sprintf("%s:%d", opts.Listen, opts.Port)
	if opts.Listen == "" && opts.Port == 0 {
		addr = web.DefaultWebAddr
	}

	serverCfg := web.ServerConfig{
		Addr:     addr,
		Token:    token,
		WebFS:    webembed.DistWeb(),
		StateDir: web.WebStateDir(),
		Version:  opts.Version,
	}
	deps := web.Deps{
		Hub:        hub,
		Sessions:   sm,
		Workspaces: workspaces,
		Token:      token,
		Version:    opts.Version,
		Archived:   archived,
		JobNames:   jobNames,
	}
	server, err := web.NewServer(serverCfg, deps)
	if err != nil {
		return fmt.Errorf("assemble web server: %w", err)
	}

	// Watchers share the serve lifetime (frontend_reload + jobs_update).
	go server.StartWatchers(ctx)

	if opts.Verbose {
		fmt.Printf("[INFO] rick web composition: workspaces=%d sessions=%d addr=%s\n",
			len(workspaces.List()), len(sessions.Items), addr)
	}

	return server.Serve(ctx)
}

// newWebCustomizeCmd deploys the frontend source scaffold for agent-driven
// self-iteration.
func newWebCustomizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "customize",
		Short: "Deploy the frontend source scaffold to ~/.rick/web/ for agent self-iteration",
		Long: `抽取内嵌的前端源码基线到 ~/.rick/web/src/（幂等——已存在则跳过）。

之后在任意 agent 会话中对话改造：让 agent 编辑 ~/.rick/web/src/ 并执行
npm ci && npm run build（node 已是 rick 的环境依赖），新 dist 会被 rick web
优先服务并触发浏览器自动刷新（frontend_reload）。
改坏了随时 ` + "`rick web reset`" + ` 恢复内嵌 baseline。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deployed, err := env.DeployWebScaffold()
			if err != nil {
				return fmt.Errorf("deploy web scaffold: %w", err)
			}
			if deployed {
				fmt.Println("✅ 前端源码基线已抽取到 ~/.rick/web/")
			} else {
				fmt.Println("ℹ️  ~/.rick/web/ 已存在自定义层，跳过抽取（reset 后可重新抽取）")
			}
			fmt.Println("下一步：在任意 agent 会话中让它编辑 ~/.rick/web/src/ 并 npm ci && npm run build；")
			fmt.Println("构建产物会自动生效（rick web 服务中时浏览器自动刷新）。")
			return nil
		},
	}
}

// newWebResetCmd removes the customization overlay, restoring the embedded
// baseline. --yes skips the confirmation prompt (scripting friendly).
func newWebResetCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove the frontend customization overlay (restore embedded baseline)",
		Long: `删除 ~/.rick/web/ 的前端自定义层（src/dist/根配置），恢复内嵌 baseline。

~/.rick/web.json（工作区注册表）与 ~/.rick/web/sessions.json（会话注册表）
不受影响——reset 只清前端自定义，不清机器级用户数据。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("将删除 ~/.rick/web/ 的前端自定义层（注册表数据保留）。确认？[y/N] ")
				reader := bufio.NewReader(os.Stdin)
				answer, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					fmt.Println("已取消。")
					return nil
				}
			}
			if err := env.ResetWebCustomization(); err != nil {
				return fmt.Errorf("reset web customization: %w", err)
			}
			fmt.Println("✅ 前端自定义层已清除，恢复内嵌 baseline。")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}
