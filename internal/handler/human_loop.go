package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// HumanLoop executes the complete human-loop thinking session (migrated from
// cmd.runHumanLoop). It allocates a loop_N directory, builds the SENSE prompt
// files, and launches the interactive pi session.
func HumanLoop(topic string, opts Options) error {
	draftDir, rfcDir, loopDir, err := prepareHumanLoopDirs()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mainFile, _, _, _, _, err := builder.NewPIBuilder().SaveHumanLoopPrompt(topic, rfcDir, draftDir, loopDir)
	if err != nil {
		return fmt.Errorf("failed to generate human-loop prompt: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Loop directory: %s\n", loopDir)
		fmt.Printf("[INFO] Human-loop prompt saved to: %s\n", mainFile)
		fmt.Printf("[INFO] rfc directory: %s\n", rfcDir)
	}

	// v4.4.9: 会话持久化——loop_N/session_id；`rick human-loop --resume loop_N` 恢复。
	sessionID, err := ensureSessionID(loopDir)
	if err != nil {
		return fmt.Errorf("ensure human-loop session id: %w", err)
	}
	fmt.Printf("Session ID: %s\n", sessionID)

	if err := runtime.CallCLI(opts.Verbose, cfg, mainFile, runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("failed to start pi CLI: %w", err)
	}

	fmt.Printf("思考记录已保存到 %s\n", loopDir)
	return nil
}

// ResumeHumanLoop resumes a previous human-loop session by loop id (e.g.
// loop_1 → .rick/draft/loops/loop_1)：读 loop 目录的 session_id，以 --session-id
// 恢复完整 pi 会话（SENSÉ 五阶段进度、此前 human 判断全在上下文中）。
func ResumeHumanLoop(loopID string, opts Options) error {
	draftDir, err := workspace.GetDraftDir()
	if err != nil {
		return fmt.Errorf("failed to get draft directory: %w", err)
	}
	loopDir := filepath.Join(draftDir, "loops", loopID)
	if _, err := os.Stat(loopDir); os.IsNotExist(err) {
		return fmt.Errorf("loop 目录不存在：%s（查看 .rick/draft/loops/ 下可恢复的会话）", loopDir)
	}

	sessionID, err := loadSessionIDStrict(loopDir, fmt.Sprintf("loop %s", loopID))
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("恢复 human-loop 会话：%s（session %s）\n", loopID, sessionID)
	if err := runtime.CallCLI(opts.Verbose, cfg, "", runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("failed to resume pi CLI: %w", err)
	}
	return nil
}

// HumanLoopDryRun generates and prints the SENSE (sense_loop) prompt without
// creating a session (migrated from the dry-run branch of cmd.runHumanLoop).
func HumanLoopDryRun(topic string) error {
	content, err := RenderHumanLoopPrompt(topic)
	if err != nil {
		return err
	}
	fmt.Print(content)
	return nil
}

// RenderHumanLoopPrompt 构建并返回 SENSE（sense_loop）提示词全文，不落盘、不
// 创建 session、不打印。供 dry-run 与 update-pi 的渲染冒烟自检共用。
func RenderHumanLoopPrompt(topic string) (string, error) {
	draftDir, rfcDir, _, err := prepareHumanLoopDirs()
	if err != nil {
		return "", err
	}

	_, content, err := builder.NewPIBuilder().BuildHumanLoop(topic, map[string]string{
		"rfc_dir":   rfcDir,
		"draft_dir": draftDir,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate human-loop prompt: %w", err)
	}
	return content, nil
}

// prepareHumanLoopDirs resolves the draft/rfc/loop directories and ensures the
// draft directory tree exists (idempotent MkdirAll, same as the original
// cmd.runHumanLoop setup).
func prepareHumanLoopDirs() (draftDir, rfcDir, loopDir string, err error) {
	draftDir, err = workspace.GetDraftDir()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get draft directory: %w", err)
	}
	rfcDir = filepath.Join(draftDir, "rfc")

	// Allocate next loop_N directory
	loopID, err := workspace.NextLoopID(draftDir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to allocate loop id: %w", err)
	}
	loopDir = filepath.Join(draftDir, "loops", loopID)

	for _, sub := range []string{
		draftDir,
		rfcDir,
		filepath.Join(draftDir, "concepts"),
		filepath.Join(draftDir, "human-learning"),
		filepath.Join(draftDir, "loops"),
	} {
		if err := os.MkdirAll(sub, 0755); err != nil {
			return "", "", "", fmt.Errorf("failed to create directory %s: %w", sub, err)
		}
	}

	return draftDir, rfcDir, loopDir, nil
}
