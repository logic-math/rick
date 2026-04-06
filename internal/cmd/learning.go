package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

func NewLearningCmd() *cobra.Command {
	var jobID string

	learningCmd := &cobra.Command{
		Use:   "learning [job_id]",
		Short: "Analyze and document learnings from job execution",
		Long:  `Analyze execution results and update documentation (OKR, SPEC, wiki, skills).`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting learning phase...")
			}

			if GetDryRun() {
				return runLearningDryRun(jobID)
			}

			// Get job ID from args, local flag, or global flag
			if len(args) > 0 {
				jobID = args[0]
			} else if jobID == "" {
				jobID = GetJobID()
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick learning [job_id] or rick learning --job job_id")
			}

			// Validate job ID format
			if err := validateJobID(jobID); err != nil {
				return err
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Analyzing learnings for job: %s\n", jobID)
			}

			// Execute learning workflow
			if err := executeLearningWorkflow(jobID); err != nil {
				return err
			}

			fmt.Printf("✅ Learning phase completed for job %s!\n", jobID)
			return nil
		},
	}

	learningCmd.Flags().StringVar(&jobID, "job", "", "Job ID to analyze")

	return learningCmd
}

// ExecutionData holds all execution information for learning
type ExecutionData struct {
	JobID        string
	DebugContent string
	TasksJSON    *executor.TasksJSON
	OKRContent   string
	TaskMDContent string
}

// runLearningDryRun generates and prints the learning prompt without executing it.
func runLearningDryRun(jobID string) error {
	if jobID == "" {
		jobID = "job_N"
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	// Build minimal ExecutionData for dry-run
	data := &ExecutionData{
		JobID:         jobID,
		DebugContent:  "",
		TasksJSON:     nil,
		OKRContent:    "",
		TaskMDContent: "",
	}

	// Try to read real data if available
	jobDir := filepath.Join(rickDir, "jobs", jobID)
	doingDir := filepath.Join(jobDir, "doing")
	planDir := filepath.Join(jobDir, "plan")

	if content, err := os.ReadFile(filepath.Join(doingDir, "debug.md")); err == nil {
		data.DebugContent = string(content)
	}
	if content, err := os.ReadFile(filepath.Join(planDir, "OKR.md")); err == nil {
		data.OKRContent = string(content)
	}

	learningDir := filepath.Join(jobDir, "learning")

	promptContent, err := buildLearningPrompt(data, learningDir)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate learning prompt: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Learning prompt:\n\n")
	fmt.Println(promptContent)
	return nil
}

// executeLearningWorkflow executes the complete learning workflow
func executeLearningWorkflow(jobID string) error {
	fmt.Println("\n=== Learning Workflow ===")
	fmt.Println()

	// Step 1: Collect execution data
	fmt.Println("=== Step 1: Collecting execution data ===")
	data, err := collectExecutionData(jobID)
	if err != nil {
		return fmt.Errorf("failed to collect execution data: %w", err)
	}

	// Step 2: Call Claude for analysis (with simplified prompt)
	fmt.Println("\n=== Step 2: Analyzing with Claude ===")
	fmt.Println("Calling Claude Code CLI for analysis...")

	if err := callClaudeForAnalysis(data); err != nil {
		return fmt.Errorf("Claude analysis failed: %w", err)
	}

	fmt.Println("\n✅ Learning workflow completed!")
	return nil
}

// collectExecutionData collects all execution data for learning
func collectExecutionData(jobID string) (*ExecutionData, error) {
	// Get workspace
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get rick directory: %w", err)
	}

	jobDir := filepath.Join(rickDir, "jobs", jobID)
	doingDir := filepath.Join(jobDir, "doing")

	// Check if doing directory exists
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("doing directory not found: %s (has the job been executed?)", doingDir)
	}

	data := &ExecutionData{
		JobID: jobID,
	}

	// 1. Read debug.md
	debugPath := filepath.Join(doingDir, "debug.md")
	if _, err := os.Stat(debugPath); err == nil {
		content, err := os.ReadFile(debugPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read debug.md: %w", err)
		}
		data.DebugContent = string(content)
		fmt.Printf("✅ Read debug.md (%d bytes)\n", len(content))
	} else {
		fmt.Println("⚠ debug.md not found (no debugging was needed)")
		data.DebugContent = "No debugging information available."
	}

	// 2. Load tasks.json
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")
	if _, err := os.Stat(tasksJSONPath); err == nil {
		tasksJSON, err := executor.LoadTasksJSON(tasksJSONPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load tasks.json: %w", err)
		}
		data.TasksJSON = tasksJSON
		fmt.Printf("✅ Loaded tasks.json (%d tasks)\n", len(tasksJSON.Tasks))
	} else {
		return nil, fmt.Errorf("tasks.json not found: %s", tasksJSONPath)
	}

	// 3. Read OKR.md from plan directory (optional)
	planDir := filepath.Join(jobDir, "plan")
	okrPath := filepath.Join(planDir, "OKR.md")
	if content, err := os.ReadFile(okrPath); err == nil {
		data.OKRContent = string(content)
		fmt.Printf("✅ Read OKR.md (%d bytes)\n", len(content))
	} else {
		fmt.Println("⚠ OKR.md not found (skipping)")
	}

	// 4. Read all task*.md files from plan directory
	taskFiles, err := filepath.Glob(filepath.Join(planDir, "task*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob task files: %w", err)
	}
	var taskMDParts []string
	for _, tf := range taskFiles {
		content, err := os.ReadFile(tf)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", tf, err)
		}
		taskMDParts = append(taskMDParts, fmt.Sprintf("### %s\n\n%s", filepath.Base(tf), string(content)))
	}
	if len(taskMDParts) > 0 {
		data.TaskMDContent = strings.Join(taskMDParts, "\n\n---\n\n")
		fmt.Printf("✅ Read %d task*.md files\n", len(taskMDParts))
	} else {
		fmt.Println("⚠ No task*.md files found in plan directory")
	}

	return data, nil
}

// callClaudeForAnalysis calls Claude Code CLI for analysis
// Uses interactive mode so Claude can read git commits and create documentation
func callClaudeForAnalysis(data *ExecutionData) error {
	// Create learning directory structure
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	learningDir := filepath.Join(rickDir, "jobs", data.JobID, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return fmt.Errorf("failed to create learning directory: %w", err)
	}

	// Create subdirectories
	wikiDir := filepath.Join(learningDir, "wiki")
	skillsDir := filepath.Join(learningDir, "skills")
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		return fmt.Errorf("failed to create wiki directory: %w", err)
	}
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	fmt.Printf("✅ Created learning directory structure:\n")
	fmt.Printf("   - %s\n", learningDir)
	fmt.Printf("   - %s/wiki/\n", learningDir)
	fmt.Printf("   - %s/skills/\n", learningDir)
	fmt.Println()

	// Build learning prompt using template system
	prompt, err := buildLearningPrompt(data, learningDir)
	if err != nil {
		return fmt.Errorf("failed to build learning prompt: %w", err)
	}

	// Get Claude CLI path
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	claudePath := cfg.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Create temporary file for the prompt
	tmpFile, err := os.CreateTemp("", "rick-learning-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(prompt); err != nil {
		return fmt.Errorf("failed to write prompt to temporary file: %w", err)
	}
	tmpFile.Close()

	fmt.Printf("📝 提示词已保存到: %s\n", tmpFile.Name())
	fmt.Println("🤖 启动 Claude Code CLI 交互模式...")
	fmt.Println("📌 Claude 将在 learning 目录下生成文档（等待人工审核后合并）")
	fmt.Println()

	// Call Claude Code CLI in interactive mode (no --dangerously-skip-permissions)
	// This allows Claude to use tools like Read, Write, Bash (git show), etc.
	// No timeout - let Claude run as long as needed
	cmd := exec.Command(claudePath, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run without timeout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude Code CLI 执行失败: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Learning 阶段完成！")
	fmt.Printf("📁 生成的文档位于: %s\n", learningDir)
	fmt.Println()
	fmt.Println("⚠️  下一步操作:")
	fmt.Println("   1. 审核 learning 目录下的所有文档")
	fmt.Println("   2. 根据需要将更新合并到 .rick/ 目录")
	fmt.Println("   3. 提交最终的文档更新")
	fmt.Println()

	return nil
}

// buildLearningPrompt builds learning prompt using template system
func buildLearningPrompt(data *ExecutionData, learningDir string) (string, error) {
	// Create prompt manager to load template
	promptMgr := prompt.NewPromptManager("")

	// Load learning template
	template, err := promptMgr.LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	// Create prompt builder
	builder := prompt.NewPromptBuilder(template)

	// Set basic variables
	projectName, err := workspace.GetProjectName()
	if err != nil {
		projectName = filepath.Base(".")
	}
	builder.SetVariable("project_name", projectName)
	builder.SetVariable("project_description", "Context-First AI Coding Framework")
	builder.SetVariable("job_id", data.JobID)

	// Set learning directory path
	builder.SetVariable("learning_dir", learningDir)

	// Set rick_bin_path: path to the locally built rick binary in the project
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = "."
	}
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	builder.SetVariable("rick_bin_path", rickBinPath)

	// Build task execution results table
	var taskResults strings.Builder
	if data.TasksJSON != nil {
		taskResults.WriteString("| Task ID | 任务名称 | 状态 | 任务文件 | Commit Hash | 重试次数 |\n")
		taskResults.WriteString("|---------|---------|------|----------|-------------|----------|\n")
		for _, task := range data.TasksJSON.Tasks {
			taskFile := task.TaskFile
			if taskFile == "" {
				taskFile = "N/A"
			}
			commitHash := task.CommitHash
			if commitHash == "" {
				commitHash = "N/A"
			} else if len(commitHash) > 8 {
				commitHash = commitHash[:8] // Short hash
			}
			taskResults.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d |\n",
				task.TaskID, task.TaskName, task.Status, taskFile, commitHash, task.Attempts))
		}
	} else {
		taskResults.WriteString("无任务元信息\n")
	}

	// Set context variables
	builder.SetVariable("task_execution_results", taskResults.String())

	// Debug records
	if data.DebugContent != "" {
		builder.SetVariable("debug_records", data.DebugContent)
	} else {
		builder.SetVariable("debug_records", "无调试信息（任务执行顺利，无需调试）")
	}

	// OKR content
	if data.OKRContent != "" {
		builder.SetVariable("okr_content", data.OKRContent)
	} else {
		builder.SetVariable("okr_content", "（本 job 无 OKR.md）")
	}

	// Task MD content
	if data.TaskMDContent != "" {
		builder.SetVariable("task_md_content", data.TaskMDContent)
	} else {
		builder.SetVariable("task_md_content", "（本 job 无 task*.md 文件）")
	}

	// Build the prompt
	promptContent, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build learning prompt: %w", err)
	}

	return promptContent, nil
}
