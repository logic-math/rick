package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

// runEasyMode creates a new job and starts an easy interactive session.
func runEasyMode(requirement, ctxPath string) error {
	if requirement == "" {
		var err error
		requirement, err = promptForRequirement()
		if err != nil {
			return fmt.Errorf("failed to get requirement: %w", err)
		}
	}
	if requirement == "" {
		return fmt.Errorf("requirement cannot be empty")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	// --ctx: guard against overwriting existing context
	if ctxPath != "" {
		if err := validateCtxInheritance(rickDir, ctxPath); err != nil {
			return err
		}
	}

	jobID, err := workspace.NextJobID()
	if err != nil {
		return fmt.Errorf("failed to determine next job ID: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return startEasySession(jobID, requirement, rickDir, cfg, ctxPath)
}

// validateCtxInheritance checks that:
// 1. The target ctxPath exists and looks like a .rick directory.
// 2. The local .rick does NOT already have OKR.md or SPEC.md (to prevent accidental overwrite).
func validateCtxInheritance(localRickDir, ctxPath string) error {
	// Verify the source ctx path exists
	if _, err := os.Stat(ctxPath); os.IsNotExist(err) {
		return fmt.Errorf("--ctx path does not exist: %s", ctxPath)
	}
	// Verify local .rick has no existing OKR.md / SPEC.md
	for _, name := range []string{"OKR.md", "SPEC.md"} {
		p := filepath.Join(localRickDir, name)
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("local context already exists (%s). Remove it first or omit --ctx", p)
		}
	}
	return nil
}

// resumeEasyMode resumes an existing easy session by job ID.
func resumeEasyMode(jobID string) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return fmt.Errorf("job %s does not exist or has no doing directory", jobID)
	}

	sessionID, err := loadSessionID(doingDir)
	if err != nil {
		return fmt.Errorf("no session found for job %s: %w", jobID, err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Job ID: %s\n", jobID)
	fmt.Printf("Resuming session: %s\n", sessionID)

	if err := callClaudeCodeCLI(cfg, "", "--resume", sessionID); err != nil {
		return fmt.Errorf("session resume failed: %w", err)
	}

	// Update tasks.json timestamp on resume so dream sees the latest run
	if err := writeEasyTasksJSON(doingDir); err != nil {
		fmt.Printf("[WARN] failed to write tasks.json: %v\n", err)
	}

	// Generate fresh learning prompt (doingDir now fully populated after resume)
	learningFile, err := prompt.GenerateEasyLearningPromptFile(jobID, rickDir)
	if err != nil {
		fmt.Printf("[WARN] failed to generate learning prompt: %v\n", err)
	}

	// Auto-trigger learning on exit
	fmt.Println("\n✅ Easy 会话结束，开始自动 learning...")
	if err := triggerAutoLearning(cfg, learningFile); err != nil {
		fmt.Printf("[WARN] auto learning failed: %v\n", err)
	} else {
		fmt.Println("✅ Learning + Merge 完成！")
	}

	return nil
}

// startEasySession runs the full easy session flow for a new job.
func startEasySession(jobID, requirement, rickDir string, cfg *config.Config, ctxPath string) error {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		return fmt.Errorf("failed to create doing directory: %w", err)
	}

	// Write requirement
	if err := os.WriteFile(filepath.Join(doingDir, "requirement.md"), []byte(requirement), 0644); err != nil {
		return fmt.Errorf("failed to write requirement: %w", err)
	}

	// Generate session UUID and persist
	sessionID, err := generateUUID()
	if err != nil {
		return fmt.Errorf("failed to generate session ID: %w", err)
	}
	if err := saveSessionID(doingDir, sessionID); err != nil {
		return fmt.Errorf("failed to save session ID: %w", err)
	}

	mainFile, _, err := prompt.GenerateEasyPromptFile(jobID, requirement, rickDir, ctxPath)
	if err != nil {
		return fmt.Errorf("failed to generate easy prompt: %w", err)
	}

	fmt.Printf("Job ID: %s\n", jobID)
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Println("🤖 Starting Easy interactive session...")

	if err := callClaudeCodeCLI(cfg, mainFile, "--session-id", sessionID); err != nil {
		return fmt.Errorf("easy session failed: %w", err)
	}

	// Write synthetic tasks.json so dream can discover this job
	if err := writeEasyTasksJSON(doingDir); err != nil {
		fmt.Printf("[WARN] failed to write tasks.json: %v\n", err)
	}

	// Generate learning prompt now that doingDir is fully populated
	learningFile, err := prompt.GenerateEasyLearningPromptFile(jobID, rickDir)
	if err != nil {
		fmt.Printf("[WARN] failed to generate learning prompt: %v\n", err)
	}

	// Auto-trigger learning on exit
	fmt.Println("\n✅ Easy 会话结束，开始自动 learning...")
	if err := triggerAutoLearning(cfg, learningFile); err != nil {
		fmt.Printf("[WARN] auto learning failed: %v\n", err)
	} else {
		fmt.Println("✅ Learning + Merge 完成！")
	}

	return nil
}

// triggerAutoLearning runs the learning prompt in background (non-interactive) mode.
func triggerAutoLearning(cfg *config.Config, learningPromptFile string) error {
	return callClaudeCodeCLIBackground(cfg, learningPromptFile)
}

// saveSessionID persists the session UUID to doingDir/session_id.
func saveSessionID(doingDir, sessionID string) error {
	return os.WriteFile(filepath.Join(doingDir, "session_id"), []byte(sessionID), 0644)
}

// loadSessionID reads the session UUID from doingDir/session_id.
func loadSessionID(doingDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(doingDir, "session_id"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeEasyTasksJSON writes a synthetic tasks.json so dream can discover this easy job.
// Easy mode has no task breakdown, so we write a single "easy_session" task as success.
func writeEasyTasksJSON(doingDir string) error {
	now := time.Now()
	tasksJSON := map[string]interface{}{
		"version":    "1.0",
		"created_at": now,
		"updated_at": now,
		"tasks": []map[string]interface{}{
			{
				"task_id":      "easy_session",
				"task_name":    "Easy Mode Session",
				"task_file":    "",
				"status":       "success",
				"dependencies": []string{},
				"attempts":     1,
				"created_at":   now,
				"updated_at":   now,
			},
		},
	}
	data, err := json.MarshalIndent(tasksJSON, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(doingDir, "tasks.json"), data, 0644)
}

// generateUUID generates a random UUID (v4).
func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
