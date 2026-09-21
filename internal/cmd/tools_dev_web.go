package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env/devweb"
)

// dev-web 退出码语义（AI 会话据此判定「哪一步失败」，不要去猜日志）：
//
//	0 ok / 2 build fail / 3 start fail / 4 health（含指纹）fail
const (
	devExitBuild  = 2
	devExitStart  = 3
	devExitHealth = 4
)

// NewDevWebCmd 创建 `rick tools dev-web`：把「隔离的 dev 实例」的初始化、构建、
// 起停与健康/指纹校验做成一条命令，供 AI 会话在 rick web 里自改进 rick 时反复
// 调用（自举闭环的**可操作面**：改代码 → dev-web build → dev-web restart →
// 一条回执证明新构建在跑）。
func NewDevWebCmd() *cobra.Command {
	var (
		prodRepo   string
		devTree    string
		noBuild    bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "dev-web",
		Short: "Manage the isolated dev instance (dev worktree + dev HOME) used to develop rick itself",
		Long: `Manage the isolated dev instance used to develop rick itself.

  生产实例（8413 / ~/.rick / 生产工作树）与 dev 实例（8414 / dev HOME / dev 工作树）
  完全隔离：dev 侧可以随便重建、重启、甚至 kill -9，都不会影响生产上正在跑的会话。

Subcommands:
  init     幂等准备 dev 环境：git worktree + dev HOME + pi 沙盒种子 + 前端依赖 + overlay 软链
  build    构建 dev 产物（前端 npm build + 后端 go build；唯一命名 = 构建指纹）
  up       停旧 + 起新 + 健康轮询 + 指纹复核（--no-build 复用最近一次构建）
  restart  build + up（改完后端代码后的标准动作）
  status   汇报：进程 / 端口 / 期望指纹 / 运行中指纹 / 是否一致
  down     停止 dev 实例（只杀确认属于该 dev HOME 的进程）

Print:
  DEV_UP bin=<路径> pid=<pid> health_ms=<n> build_id=<指纹>
  DEV_FAIL stage=<init|build|start|health|fingerprint> detail=<原因>

Exit codes:
  0 ok / 2 build fail / 3 start fail / 4 health or fingerprint fail`,
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	resolveRepo := func() string {
		if prodRepo != "" {
			return prodRepo
		}
		return defaultProdRepo()
	}
	// F4：--dev-tree 显式优先；否则 devweb 按「env → 当前目录向上找到的 dev 工作树
	// → prodRepo 的祖父目录」解析（从 dev 树内执行也能正确解析）。需要已有树的
	// 子命令在解析阶段就会拿到清晰的中文错误。
	newLayout := func() (devweb.Layout, error) { return devweb.LayoutFor(resolveRepo(), devTree) }
	newLayoutForInit := func() (devweb.Layout, error) { return devweb.LayoutForInit(resolveRepo(), devTree) }

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Prepare the dev worktree, HOME, pi sandbox seed, frontend deps and overlay symlink (idempotent)",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := newLayoutForInit()
			if err != nil {
				return err
			}
			if err := l.RequireInitTarget(); err != nil {
				return devFail(cmd, "init", err)
			}
			out := cmd.OutOrStdout()
			if err := devweb.Init(l, out); err != nil {
				return devFail(cmd, "init", err)
			}
			fmt.Fprintf(out, "DEV_INIT tree=%s home=%s agent_dir=%s port=%d overlay=%s\n",
				l.Tree, l.Home, l.AgentDir, l.Port, l.OverlayDist())
			return nil
		},
	}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build dev artifacts (frontend + backend); the binary name is the build fingerprint",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := newLayout()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			bin, err := devweb.Build(l, "", "")
			if err != nil {
				return devFail(cmd, "build", err)
			}
			fmt.Fprintf(out, "DEV_BUILD bin=%s build_id=%s\n", bin, devweb.BuildIDFromBin(bin))
			return nil
		},
	}

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Start (or restart) the dev instance and verify health + build fingerprint",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := newLayout()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			bin := ""
			if !noBuild {
				if bin, err = devweb.Build(l, "", ""); err != nil {
					return devFail(cmd, "build", err)
				}
			}
			res, err := devweb.Up(l, bin)
			if err != nil {
				return devFail(cmd, stageOf(err), err)
			}
			fmt.Fprintf(out, "DEV_UP bin=%s pid=%d health_ms=%d build_id=%s port=%d\n",
				res.Bin, res.PID, res.HealthMS, res.BuildID, res.Port)
			if len(res.Skipped) > 0 {
				// 拒绝杀掉的进程要显式暴露（防误杀断言的可见性）
				fmt.Fprintf(out, "  skipped_pids=%v (未确认属于该 dev HOME，已拒杀)\n", res.Skipped)
			}
			return nil
		},
	}

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Rebuild and restart the dev instance (standard action after backend changes)",
		RunE:  upCmd.RunE,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Report dev instance status (process / port / expected vs running fingerprint)",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := newLayout()
			if err != nil {
				return err
			}
			res, err := devweb.Status(l)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if jsonOutput {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			state := "down"
			if res.Running {
				state = "up"
			}
			fp := "-"
			if res.Running {
				if res.FingerprintOK {
					fp = "ok"
				} else {
					fp = "MISMATCH"
				}
			}
			fmt.Fprintf(out, "DEV_STATUS state=%s pid=%d port=%d build_id=%s fingerprint=%s state_file=%s\n",
				state, res.PID, res.Port, res.RunningID, fp, res.StatePath)
			return nil
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop the dev instance (only processes proven to belong to the dev HOME are signalled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := newLayout()
			if err != nil {
				return err
			}
			stopped, skipped, err := devweb.Down(l)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "DEV_DOWN stopped=%v skipped=%v\n", stopped, skipped)
			return nil
		},
	}

	for _, sub := range []*cobra.Command{initCmd, buildCmd, upCmd, restartCmd, statusCmd, downCmd} {
		sub.Flags().StringVar(&prodRepo, "prod-repo", "", "生产仓库工作树路径（默认：本 CLI 所在仓库）")
		sub.Flags().StringVar(&devTree, "dev-tree", "", "dev 工作树路径（默认：RICK_DEV_TREE → 当前目录向上找到的 dev 工作树 → <prodRepo 祖父>/rick-dev）")
	}
	upCmd.Flags().BoolVar(&noBuild, "no-build", false, "复用最近一次构建（不重新编译）")
	restartCmd.Flags().BoolVar(&noBuild, "no-build", false, "复用最近一次构建（不重新编译）")
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "以 JSON 输出（脚本/AI 解析用）")

	cmd.AddCommand(initCmd, buildCmd, upCmd, restartCmd, statusCmd, downCmd)
	return cmd
}

// devFail 打印一行结构化失败回执（DEV_FAIL stage=... detail=...）并按 stage
// 以专用退出码退出——AI 会话只看这一行就知道该重试什么，不必解析日志。
//
// 为什么直接 os.Exit：main.go 对 RunE 的错误一律 os.Exit(1)，而本命令需要
// 区分 2/3/4（build/start/health）。（devExit 可被测试替换，避免测试进程退出。）
func devFail(cmd *cobra.Command, stage string, err error) error {
	detail := strings.ReplaceAll(err.Error(), "\n", " | ")
	if len(detail) > 600 {
		detail = detail[:600] + "..."
	}
	fmt.Fprintf(cmd.OutOrStdout(), "DEV_FAIL stage=%s detail=%s\n", stage, detail)
	devExit(exitCodeFor(stage))
	return err
}

// devExit 是 os.Exit 的间接层（测试里替换成记录函数，绝不真退出）。
var devExit = os.Exit

// exitCodeFor 把失败阶段映射为退出码：0 ok / 2 build / 3 start / 4 health+fingerprint。
func exitCodeFor(stage string) int {
	switch stage {
	case "build":
		return devExitBuild
	case "health", "fingerprint":
		return devExitHealth
	default: // init / start / 未知 → 按启动失败处理
		return devExitStart
	}
}

// stageOf 从 devweb.StageError 里提取阶段（未知阶段归到 start）。
func stageOf(err error) string {
	var se *devweb.StageError
	if errors.As(err, &se) {
		return se.Stage
	}
	return "start"
}

// defaultProdRepo 推导生产仓库工作树：CLI 可执行文件所在目录的父目录
// （生产布局是 `<repo>/bin/rick`；本命令通常由生产实例托管的 AI 会话调用）。
// 找不到时退回当前工作目录。
func defaultProdRepo() string {
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Dir(filepath.Dir(exe))
		if _, err := os.Stat(filepath.Join(cand, "go.mod")); err == nil {
			return cand
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
