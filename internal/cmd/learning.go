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
				fmt.Println("[DRY-RUN] Would execute learning")
				return nil
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

	return data, nil
}

// callClaudeForAnalysis calls Claude Code CLI for analysis
// Uses interactive mode so Claude can read git commits and create documentation
func callClaudeForAnalysis(data *ExecutionData) error {
	// Build simplified prompt: debug.md + task metadata
	prompt := buildLearningPrompt(data)

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

	fmt.Printf("\n📝 提示词已保存到: %s\n", tmpFile.Name())
	fmt.Println("🤖 启动 Claude Code CLI 交互模式...")
	fmt.Println("📌 Claude 将自动分析执行结果并更新文档")
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

	return nil
}

// buildLearningPrompt builds a simplified learning prompt in Chinese
// Only includes: debug.md content + task metadata (task file + commit info)
func buildLearningPrompt(data *ExecutionData) string {
	var prompt strings.Builder

	prompt.WriteString("# Learning 分析任务\n\n")
	prompt.WriteString(fmt.Sprintf("分析 Job %s 的执行结果并提取经验教训。\n\n", data.JobID))

	// Section 1: Debug Information
	prompt.WriteString("## 调试信息\n\n")
	if data.DebugContent != "" {
		prompt.WriteString(data.DebugContent)
	} else {
		prompt.WriteString("无调试信息（任务执行顺利，无需调试）\n")
	}
	prompt.WriteString("\n\n")

	// Section 2: Task Metadata
	prompt.WriteString("## 任务元信息\n\n")
	if data.TasksJSON != nil {
		prompt.WriteString("| Task ID | 任务名称 | 状态 | 任务文件 | Commit Hash | 重试次数 |\n")
		prompt.WriteString("|---------|---------|------|----------|-------------|----------|\n")
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
			prompt.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d |\n",
				task.TaskID, task.TaskName, task.Status, taskFile, commitHash, task.Attempts))
		}
	} else {
		prompt.WriteString("无任务元信息\n")
	}
	prompt.WriteString("\n\n")

	// Section 3: Instructions
	prompt.WriteString("## 执行指令\n\n")
	prompt.WriteString("基于上述调试信息和任务元信息，请执行以下操作：\n\n")
	prompt.WriteString("1. **分析执行过程**\n")
	prompt.WriteString("   - 使用 `git show <commit_hash>` 查看每个任务的代码变更\n")
	prompt.WriteString("   - 分析遇到的问题和解决方法（如果有）\n")
	prompt.WriteString("   - 识别关键洞察、模式和改进点\n\n")
	prompt.WriteString("2. **更新项目文档**（在 `.rick/` 目录下）\n")
	prompt.WriteString("   - `OKR.md` - 根据学到的经验更新项目目标\n")
	prompt.WriteString("   - `SPEC.md` - 如需要，更新开发规范\n")
	prompt.WriteString("   - `wiki/<主题>.md` - 为新概念创建或更新 wiki 页面\n")
	prompt.WriteString("   - `skills/<技能>.md` - 提取可复用的技能供未来任务使用\n\n")
	prompt.WriteString("3. **提交变更**\n")
	prompt.WriteString("   - 使用清晰的 commit message 提交你的文档更新\n")
	prompt.WriteString("   - Commit message 格式: `docs(learning): <简短描述>`\n\n")
	prompt.WriteString("**注意事项**：\n")
	prompt.WriteString("- 你拥有完整的 git、文件系统和所有工具的访问权限\n")
	prompt.WriteString("- 请提供全面的分析并自动更新文档\n")
	prompt.WriteString("- 重点关注可复用的经验和模式\n")
	prompt.WriteString("- 确保文档更新后的一致性和完整性\n")

	return prompt.String()
}
