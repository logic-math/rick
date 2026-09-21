package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// NewRSICmd 是 `rick rsi`：启动**自进化（RSI）会话**——把「改进 rick 自身」
// 绑定到 <cwd>/.rick/loops/rick-rsi-loop.md（后端注入 loop 全文 + 以
// --append-system-prompt 常驻），并做与 web 入口一致的 workspace 硬校验。
//
// 与 web 的关系：`rick rsi` 是同一套语义的 CLI 入口（同一 prompt.BuildRSIPrompt、
// 同一校验）。区别只是会话落在 `.rick/draft/rsi/rsi_N/`（CLI 侧持久化 session_id，
// 可用 pi 的 --session 语义恢复），而 web 侧把会话登记到 web 注册表。
func NewRSICmd() *cobra.Command {
	rsiCmd := &cobra.Command{
		Use:   "rsi",
		Short: "Start an RSI session that improves rick itself (loads .rick/loops/rick-rsi-loop.md)",
		Long: `Start a recursive-self-improvement (RSI) session.

本次会话的唯一职责是改进 rick 自身：后端把 .rick/loops/rick-rsi-loop.md 的全文
注入系统提示词，agent 必须按该 loop 的状态机执行（isolated dev instance →
layered gates → human approval → rick tools release）。loop 的产出评估由
` + "`rick tools rsi_check --job <job>`" + ` 机器校验。

硬校验（fail-fast）：
  1. 当前工作区必须是 rick 源码树（含 cmd/rick/main.go 与 internal/web/）
  2. 必须存在 .rick/loops/rick-rsi-loop.md
  3. 不得是生产仓库根（生产改动只能经人类确认后的 rick tools release）`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetResume() != "" {
				return resumeRSI(GetResume())
			}
			wsPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve working directory: %w", err)
			}
			rickDir, err := workspace.GetRickDir()
			if err != nil {
				return fmt.Errorf("resolve .rick dir: %w", err)
			}

			promptText, loopFile, err := prompt.BuildRSIPrompt(wsPath)
			if err != nil {
				return err
			}

			if GetDryRun() {
				fmt.Printf("[DRY-RUN] 工作区：%s\n", wsPath)
				fmt.Printf("[DRY-RUN] loop：%s\n", loopFile)
				fmt.Println(promptText)
				return nil
			}

			rsiDir, err := prompt.EnsureRSIDirs(rickDir)
			if err != nil {
				return err
			}
			promptFile := filepath.Join(rsiDir, "prompt.md")
			if err := os.WriteFile(promptFile, []byte(promptText), 0644); err != nil {
				return fmt.Errorf("write RSI prompt: %w", err)
			}
			sessionID, err := prompt.EnsureRSISessionID(rsiDir)
			if err != nil {
				return err
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if GetVerbose() {
				fmt.Printf("[INFO] RSI session dir: %s\n", rsiDir)
				fmt.Printf("[INFO] prompt: %s\n", promptFile)
				fmt.Printf("[INFO] loop (method): %s\n", loopFile)
			}
			fmt.Printf("RSI 自进化会话：%s（loop %s）\n", filepath.Base(rsiDir), loopFile)
			fmt.Printf("Session ID: %s\n", sessionID)

			if err := runtime.CallCLI(GetVerbose(), cfg, promptFile, runtime.ModeInteractive,
				"--session-id", sessionID, "--append-system-prompt", loopFile); err != nil {
				return fmt.Errorf("failed to start pi CLI: %w", err)
			}
			fmt.Printf("本次迭代的落盘位置：%s/.rick/jobs/<job>/doing/rsi/（可用 `rick tools rsi_check --job <job> --init` 生成骨架）\n", wsPath)
			return nil
		},
	}
	return rsiCmd
}

// resumeRSI 以 --session-id 恢复既有 RSI 会话（参数为 rsi_N）。
func resumeRSI(rsiID string) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("resolve .rick dir: %w", err)
	}
	dir := filepath.Join(rickDir, "draft", "rsi", rsiID)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("RSI 会话目录不存在：%s（查看 .rick/draft/rsi/ 下可恢复的会话）", dir)
	}
	data, err := os.ReadFile(filepath.Join(dir, "session_id"))
	if err != nil {
		return fmt.Errorf("RSI 会话无会话记录（%s/session_id 不存在）——先正常运行一次 `rick rsi`", dir)
	}
	sessionID := string(data)
	for len(sessionID) > 0 && (sessionID[len(sessionID)-1] == '\n' || sessionID[len(sessionID)-1] == '\r' || sessionID[len(sessionID)-1] == ' ') {
		sessionID = sessionID[:len(sessionID)-1]
	}
	if sessionID == "" {
		return fmt.Errorf("RSI 会话记录为空：%s/session_id", dir)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Printf("恢复 RSI 会话：%s（session %s）\n", rsiID, sessionID)
	loopFile := prompt.RSILoopPath(mustGetwd())
	return runtime.CallCLI(GetVerbose(), cfg, "", runtime.ModeInteractive,
		"--session-id", sessionID, "--append-system-prompt", loopFile)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
