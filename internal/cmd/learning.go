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

			if len(args) > 0 {
				jobID = args[0]
			}
			if jobID == "" {
				jobID = GetJobID()
			}

			if GetDryRun() {
				return runLearningDryRun(jobID)
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick learning [job_id] or rick learning --job job_id")
			}

			if err := validateJobID(jobID); err != nil {
				return err
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Analyzing learnings for job: %s\n", jobID)
			}

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
	DebugContent string // raw content of debug.md, embedded directly in prompt
	TasksJSON    *executor.TasksJSON
	OKRContent   string
	TaskMDPaths  []string
	SpecPath     string
	ActPathFiles []string
}

// runLearningDryRun generates and prints the learning prompt without executing it.
func runLearningDryRun(jobID string) error {
	if jobID == "" {
		jobID = GetJobID()
	}
	if jobID == "" {
		jobID = "job_N"
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	jobDir := filepath.Join(rickDir, "jobs", jobID)
	learningDir := filepath.Join(jobDir, "learning")

	data := &ExecutionData{
		JobID:        jobID,
		SpecPath:     filepath.Join(rickDir, "SPEC.md"),
		TaskMDPaths:  []string{},
		ActPathFiles: []string{},
	}

	if content, err := os.ReadFile(filepath.Join(jobDir, "plan", "OKR.md")); err == nil {
		data.OKRContent = string(content)
	}
	doingDir := filepath.Join(jobDir, "doing")
	data.DebugContent = executor.LoadDebugContext(doingDir)

	promptsDir, _ := prompt.EnsurePromptsDir(learningDir)
	promptFile, err := buildLearningPrompt(data, learningDir, promptsDir)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate learning prompt: %v\n", err)
		return nil
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to read prompt file: %v\n", err)
		return nil
	}
	fmt.Printf("[DRY-RUN] Learning prompt (saved to: %s):\n\n", promptFile)
	fmt.Println(string(content))
	return nil
}

// executeLearningWorkflow executes the complete learning workflow
func executeLearningWorkflow(jobID string) error {
	fmt.Println("\n=== Learning Workflow ===")
	fmt.Println()

	fmt.Println("=== Step 1: Collecting execution data ===")
	data, err := collectExecutionData(jobID)
	if err != nil {
		return fmt.Errorf("failed to collect execution data: %w", err)
	}

	fmt.Println("\n=== Step 2: Analyzing with Claude ===")
	fmt.Println("Calling Claude Code CLI for analysis...")

	if err := callClaudeForAnalysis(data); err != nil {
		return fmt.Errorf("Claude analysis failed: %w", err)
	}

	fmt.Println("\n✅ Learning workflow completed!")
	return nil
}

// collectExecutionData collects execution data paths for learning
func collectExecutionData(jobID string) (*ExecutionData, error) {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get rick directory: %w", err)
	}

	jobDir := filepath.Join(rickDir, "jobs", jobID)
	doingDir := filepath.Join(jobDir, "doing")

	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("doing directory not found: %s (has the job been executed?)", doingDir)
	}

	data := &ExecutionData{
		JobID:    jobID,
		SpecPath: filepath.Join(rickDir, "SPEC.md"),
	}

	// Load debug context: prefers debug/ summaries, falls back to debug.md
	data.DebugContent = executor.LoadDebugContext(doingDir)
	fmt.Printf("✅ Loaded debug context (%d bytes)\n", len(data.DebugContent))

	// tasks.json
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

	// OKR.md — full content load
	planDir := filepath.Join(jobDir, "plan")
	if content, err := os.ReadFile(filepath.Join(planDir, "OKR.md")); err == nil {
		data.OKRContent = string(content)
		fmt.Printf("✅ Read OKR.md (%d bytes)\n", len(content))
	} else {
		fmt.Println("⚠ OKR.md not found (skipping)")
	}

	// task*.md paths
	taskFiles, err := filepath.Glob(filepath.Join(planDir, "task*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob task files: %w", err)
	}
	data.TaskMDPaths = taskFiles
	if len(taskFiles) > 0 {
		fmt.Printf("✅ Found %d task*.md files\n", len(taskFiles))
	} else {
		fmt.Println("⚠ No task*.md files found in plan directory")
	}

	// act-path.md paths
	actPathPattern := filepath.Join(doingDir, "tasks", "*", "act-path.md")
	actPathFiles, err := filepath.Glob(actPathPattern)
	if err == nil {
		data.ActPathFiles = actPathFiles
		fmt.Printf("✅ Found %d act-path.md files\n", len(actPathFiles))
	}

	return data, nil
}

// callClaudeForAnalysis calls Claude Code CLI for analysis
func callClaudeForAnalysis(data *ExecutionData) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	learningDir := filepath.Join(rickDir, "jobs", data.JobID, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return fmt.Errorf("failed to create learning directory: %w", err)
	}

	fmt.Printf("✅ Created learning directory: %s\n\n", learningDir)

	promptsDir, err := prompt.EnsurePromptsDir(learningDir)
	if err != nil {
		return fmt.Errorf("failed to create prompts dir: %w", err)
	}

	promptFile, err := buildLearningPrompt(data, learningDir, promptsDir)
	if err != nil {
		return fmt.Errorf("failed to build learning prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	claudePath := cfg.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	fmt.Printf("📝 提示词已保存到: %s\n", promptFile)
	fmt.Println("🤖 启动 Claude Code CLI 交互模式...")
	fmt.Println("📌 Claude 将在 learning 目录下生成文档（等待人工审核后合并）")
	fmt.Println()

	cmd := exec.Command(claudePath, promptFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude Code CLI 执行失败: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Learning 阶段完成！")
	fmt.Printf("📁 执行摘要: %s/SUMMARY.md\n", learningDir)
	fmt.Println()

	return nil
}

// buildLearningPrompt builds learning prompt and saves to promptsDir/learning_prompt.md.
// Returns the prompt file path; all files are persistent, no cleanup needed.
func buildLearningPrompt(data *ExecutionData, learningDir, promptsDir string) (string, error) {
	promptMgr := prompt.NewPromptManager("")

	template, err := promptMgr.LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	builder := prompt.NewPromptBuilder(template)

	builder.SetVariable("job_id", data.JobID)
	builder.SetVariable("learning_dir", learningDir)

	// OKR — full content
	if data.OKRContent != "" {
		builder.SetVariable("okr_content", data.OKRContent)
	} else {
		builder.SetVariable("okr_content", "（本 job 无 OKR.md）")
	}

	// SPEC path and .rick/ sub-directories (learning writes directly here)
	builder.SetVariable("spec_path", data.SpecPath)
	rickDir := filepath.Dir(data.SpecPath) // .rick/SPEC.md → .rick/
	builder.SetVariable("wiki_dir", filepath.Join(rickDir, "wiki"))
	builder.SetVariable("tools_dir", filepath.Join(rickDir, "tools"))

	// debug.md — embed content directly
	if data.DebugContent != "" {
		builder.SetVariable("debug_content", data.DebugContent)
	} else {
		builder.SetVariable("debug_content", "（本次 job 无 debug.md 记录）")
	}

	// task*.md paths
	if len(data.TaskMDPaths) > 0 {
		var sb strings.Builder
		for _, p := range data.TaskMDPaths {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", p))
		}
		builder.SetVariable("task_md_files", strings.TrimRight(sb.String(), "\n"))
	} else {
		builder.SetVariable("task_md_files", "  （无 task*.md 文件）")
	}

	// act-path.md paths
	if len(data.ActPathFiles) > 0 {
		var sb strings.Builder
		for _, p := range data.ActPathFiles {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", p))
		}
		builder.SetVariable("act_path_files", strings.TrimRight(sb.String(), "\n"))
	} else {
		builder.SetVariable("act_path_files", "  （无 act-path.md 文件）")
	}

	// task execution results table
	var taskResults strings.Builder
	if data.TasksJSON != nil {
		taskResults.WriteString("| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |\n")
		taskResults.WriteString("|---------|---------|------|-------------|----------|\n")
		for _, task := range data.TasksJSON.Tasks {
			commitHash := task.CommitHash
			if commitHash == "" {
				commitHash = "N/A"
			} else if len(commitHash) > 8 {
				commitHash = commitHash[:8]
			}
			taskResults.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
				task.TaskID, task.TaskName, task.Status, commitHash, task.Attempts))
		}
	} else {
		taskResults.WriteString("无任务元信息\n")
	}
	builder.SetVariable("task_execution_results", taskResults.String())

	// rick_bin_path
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = "."
	}
	builder.SetVariable("rick_bin_path", filepath.Join(projectRoot, "bin", "rick"))

	// Write gen-skill to prompts/ dir, inject path
	genSkillFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return "", fmt.Errorf("failed to write gen-skill: %w", err)
	}
	builder.SetVariable("gen_skill_path", genSkillFile)

	promptFile := filepath.Join(promptsDir, "learning_prompt.md")
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", fmt.Errorf("failed to save learning prompt: %w", err)
	}

	return promptFile, nil
}
