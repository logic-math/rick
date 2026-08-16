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

	if err := runtime.CallCLI(opts.Verbose, cfg, mainFile, runtime.ModeInteractive); err != nil {
		return fmt.Errorf("failed to start pi CLI: %w", err)
	}

	fmt.Printf("思考记录已保存到 %s\n", loopDir)
	return nil
}

// HumanLoopDryRun generates and prints the SENSE (sense_loop) prompt without
// creating a session (migrated from the dry-run branch of cmd.runHumanLoop).
func HumanLoopDryRun(topic string) error {
	draftDir, rfcDir, _, err := prepareHumanLoopDirs()
	if err != nil {
		return err
	}

	_, content, err := builder.NewPIBuilder().BuildHumanLoop(topic, map[string]string{
		"rfc_dir":   rfcDir,
		"draft_dir": draftDir,
	})
	if err != nil {
		return fmt.Errorf("failed to generate human-loop prompt: %w", err)
	}

	fmt.Print(content)
	return nil
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
