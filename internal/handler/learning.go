package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// ExecutionData holds all execution information for learning (migrated from
// cmd.ExecutionData).
type ExecutionData struct {
	JobID        string
	DebugContent string // raw content of debug.md, embedded directly in prompt
	TasksJSON    *executor.TasksJSON
	TaskMDPaths  []string
	RickDir      string
	ActPathFiles []string
}

// Learning executes the complete learning workflow for a job (migrated from
// cmd.executeLearningWorkflow).
func Learning(jobID string, opts Options) error {
	fmt.Println("\n=== Learning Workflow ===")
	fmt.Println()

	fmt.Println("=== Step 1: Collecting execution data ===")
	data, err := collectExecutionData(jobID)
	if err != nil {
		return fmt.Errorf("failed to collect execution data: %w", err)
	}

	fmt.Println("\n=== Step 2: Analyzing with pi ===")
	fmt.Println("Calling pi for analysis...")

	if err := callAgentForAnalysis(data, opts.Verbose); err != nil {
		return fmt.Errorf("pi analysis failed: %w", err)
	}

	fmt.Println("\n✅ Learning workflow completed!")
	return nil
}

// LearningDryRun generates and prints the learning prompt without executing it
// (migrated from cmd.runLearningDryRun).
func LearningDryRun(jobID string) error {
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
		RickDir:      rickDir,
		TaskMDPaths:  []string{},
		ActPathFiles: []string{},
	}

	doingDir := filepath.Join(jobDir, "doing")
	data.DebugContent = executor.LoadDebugContext(doingDir)

	promptsDir := ""
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
	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")
	fmt.Printf("[DRY-RUN] Learning prompt (saved to: %s):\n", promptFile)
	fmt.Printf("[DRY-RUN] loops_dir=%s\n", loopsDir)
	fmt.Printf("[DRY-RUN] skills_dir=%s\n\n", skillsDir)
	fmt.Println(string(content))
	return nil
}

// collectExecutionData collects execution data paths for learning (migrated
// from cmd.collectExecutionData).
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
		JobID:   jobID,
		RickDir: rickDir,
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

	// task*.md paths
	planDir := filepath.Join(jobDir, "plan")
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

// callAgentForAnalysis drives the pi agent to analyze execution data and write
// learning docs (migrated from cmd.callAgentForAnalysis).
func callAgentForAnalysis(data *ExecutionData, verbose bool) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	learningDir := filepath.Join(rickDir, "jobs", data.JobID, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return fmt.Errorf("failed to create learning directory: %w", err)
	}

	fmt.Printf("✅ Created learning directory: %s\n\n", learningDir)

	promptFile, err := buildLearningPrompt(data, learningDir, "")
	if err != nil {
		return fmt.Errorf("failed to build learning prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("📝 提示词已保存到: %s\n", promptFile)
	fmt.Println("🤖 启动 pi 交互模式...")
	fmt.Println("📌 pi 将在 learning 目录下生成文档（等待人工审核后合并）")
	fmt.Println()

	if err := runtime.CallCLI(verbose, cfg, promptFile, runtime.ModeInteractive); err != nil {
		return fmt.Errorf("pi CLI 执行失败: %w", err)
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
	lp := builder.LearningParams{
		JobID:        data.JobID,
		RickDir:      data.RickDir,
		LearningDir:  learningDir,
		PromptsDir:   promptsDir,
		DebugContent: data.DebugContent,
		TaskMDPaths:  data.TaskMDPaths,
		ActPathFiles: data.ActPathFiles,
	}
	if data.RickDir != "" && data.JobID != "" {
		lp.DebugDir = filepath.Join(data.RickDir, "jobs", data.JobID, "doing", "debug")
	}
	if data.TasksJSON != nil {
		for _, task := range data.TasksJSON.Tasks {
			lp.TaskResults = append(lp.TaskResults, builder.LearningResult{
				TaskID:     task.TaskID,
				TaskName:   task.TaskName,
				Status:     task.Status,
				CommitHash: task.CommitHash,
				Attempts:   task.Attempts,
			})
		}
	}

	promptFile, _, err := builder.NewPIBuilder().SaveLearningPrompt(lp)
	if err != nil {
		return "", fmt.Errorf("failed to build learning prompt: %w", err)
	}
	return promptFile, nil
}
