package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// EasyDryRun prints the easy prompt without creating any job or calling pi
// (migrated from cmd.runEasyDryRun).
func EasyDryRun(requirement, ctxPath string) error {
	if requirement == "" {
		return fmt.Errorf("requirement cannot be empty")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}
	_, content, err := builder.NewPIBuilder().BuildEasy(requirement, map[string]string{
		"rick_dir":  rickDir,
		"doing_dir": filepath.Join(rickDir, "jobs", "job_N", "doing"),
		"ctx_path":  ctxPath,
		"job_id":    "job_N",
	})
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate prompt: %v\n", err)
		return nil
	}
	fmt.Println("[DRY-RUN] Easy prompt:")
	fmt.Println()
	fmt.Println(content)
	return nil
}

// Easy creates a new job and starts an easy interactive session (migrated from
// cmd.runEasyMode). Interactive requirement prompting stays in the CLI layer.
func Easy(requirement, ctxPath string, opts Options) error {
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

	return StartEasySession(jobID, requirement, rickDir, cfg, ctxPath, opts)
}

// validateCtxInheritance checks that:
// 1. The target ctxPath exists and looks like a .rick directory.
// 2. The local .rick does NOT already have loops/ or skills/ (to prevent accidental overwrite).
func validateCtxInheritance(localRickDir, ctxPath string) error {
	// Verify the source ctx path exists
	if _, err := os.Stat(ctxPath); os.IsNotExist(err) {
		return fmt.Errorf("--ctx path does not exist: %s", ctxPath)
	}
	// Verify local .rick has no existing loops/ or skills/
	for _, name := range []string{"loops", "skills"} {
		p := filepath.Join(localRickDir, name)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(p)
			if len(entries) > 0 {
				return fmt.Errorf("local context already exists (%s). Remove it first or omit --ctx", p)
			}
		}
	}
	return nil
}

// ResumeEasy resumes an existing easy session by job ID (migrated from
// cmd.resumeEasyMode).
func ResumeEasy(jobID string, opts Options) error {
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

	if err := runtime.CallCLI(opts.Verbose, cfg, "", runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("session resume failed: %w", err)
	}

	// Update tasks.json timestamp on resume so dream sees the latest run
	if err := writeEasyTasksJSON(doingDir); err != nil {
		fmt.Printf("[WARN] failed to write tasks.json: %v\n", err)
	}

	return nil
}

// StartEasySession runs the full easy session flow for a new job (migrated from
// cmd.startEasySession).
func StartEasySession(jobID, requirement, rickDir string, cfg *config.Config, ctxPath string, opts Options) error {
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

	mainFile, _, _, err := builder.NewPIBuilder().SaveEasyPrompt(jobID, requirement, rickDir, ctxPath)
	if err != nil {
		return fmt.Errorf("failed to generate easy prompt: %w", err)
	}

	fmt.Printf("Job ID: %s\n", jobID)
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Println("🤖 Starting Easy interactive session...")

	if err := runtime.CallCLI(opts.Verbose, cfg, mainFile, runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("easy session failed: %w", err)
	}

	// Write synthetic tasks.json so dream can discover this job
	if err := writeEasyTasksJSON(doingDir); err != nil {
		fmt.Printf("[WARN] failed to write tasks.json: %v\n", err)
	}

	return nil
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

// writeEasyTasksJSON writes a synthetic tasks.json so dream can discover this
// easy job. Easy mode has no task breakdown, so we write a single
// "easy_session" task as success.
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
