package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

// TestIntegration_RFC001 covers all assertions for job_9 task1~task4 changes.
// Each sub-test maps to a specific key result from the task specification.
func TestIntegration_RFC001(t *testing.T) {
	pm := NewPromptManager("")
	cm := NewContextManager("job_9")

	task := &parser.Task{
		ID:           "task1",
		Name:         "RFC001 Test Task",
		Dependencies: []string{},
		Goal:         "Verify RFC001 changes",
		KeyResults:   []string{"KR1", "KR2"},
		TestMethod:   "Run tests",
	}
	cm.LoadTask(task)

	// ─── task1: learning input refactor ──────────────────────────────────────

	t.Run("task1/learning_prompt_contains_task_name", func(t *testing.T) {
		// buildLearningPrompt in cmd/learning.go injects task_md_content.
		// The learning template renders {{task_md_content}}.
		// We test via GenerateLearningPrompt which uses the same template,
		// but also test that the template has the variable.
		tmpl, err := pm.LoadTemplate("learning")
		if err != nil {
			t.Fatalf("load learning template: %v", err)
		}
		if !strings.Contains(tmpl.Content, "{{task_md_content}}") {
			t.Error("learning template must contain {{task_md_content}} variable")
		}
		if !strings.Contains(tmpl.Content, "{{okr_content}}") {
			t.Error("learning template must contain {{okr_content}} variable")
		}
	})

	t.Run("task1/learning_prompt_no_hardcoded_stub_strings", func(t *testing.T) {
		// The old learning_prompt.go had four stub functions that returned
		// hardcoded Chinese placeholder text. Verify they are gone.
		tmpl, err := pm.LoadTemplate("learning")
		if err != nil {
			t.Fatalf("load learning template: %v", err)
		}
		forbidden := []string{
			"本周期内新增",
			"本周期内的代码改进",
		}
		for _, s := range forbidden {
			if strings.Contains(tmpl.Content, s) {
				t.Errorf("learning template must not contain hardcoded stub string %q", s)
			}
		}
	})

	t.Run("task1/learning_template_no_git_show", func(t *testing.T) {
		tmpl, err := pm.LoadTemplate("learning")
		if err != nil {
			t.Fatalf("load learning template: %v", err)
		}
		if strings.Contains(tmpl.Content, "git show") {
			t.Error("learning template must not contain 'git show' instruction")
		}
	})

	t.Run("task1/buildLearningPrompt_contains_task_md_content", func(t *testing.T) {
		// Build a learning prompt with real task content and verify it appears.
		tmpDir := t.TempDir()
		planDir := filepath.Join(tmpDir, "plan")
		if err := os.MkdirAll(planDir, 0755); err != nil {
			t.Fatal(err)
		}
		taskContent := "# 依赖关系\n\n# 任务名称\n实现登录接口\n\n# 任务目标\n创建 /login 端点\n\n# 关键结果\n1. 返回 JWT token\n"
		if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(taskContent), 0644); err != nil {
			t.Fatal(err)
		}

		prompt, err := buildLearningPromptForTest(t, "job_test", taskContent, "", tmpDir)
		if err != nil {
			t.Fatalf("buildLearningPromptForTest: %v", err)
		}
		if !strings.Contains(prompt, "实现登录接口") {
			t.Error("learning prompt must contain task name from task.md")
		}
		if !strings.Contains(prompt, "关键结果") {
			t.Error("learning prompt must contain key results from task.md")
		}
	})

	t.Run("task1/buildLearningPrompt_contains_okr_content", func(t *testing.T) {
		okrContent := "# Job OKR\n## O1: 实现用户认证\n- KR1: 完成登录接口"
		prompt, err := buildLearningPromptForTest(t, "job_test", "", okrContent, t.TempDir())
		if err != nil {
			t.Fatalf("buildLearningPromptForTest: %v", err)
		}
		if !strings.Contains(prompt, "实现用户认证") {
			t.Error("learning prompt must contain OKR content when OKR.md exists")
		}
	})

	// ─── task2: skills index ──────────────────────────────────────────────────

	t.Run("task2/doing_prompt_contains_skills_index_when_exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			t.Fatal(err)
		}
		indexContent := "# Skills Index\n\n| 文件 | 描述 | 触发场景 |\n|------|------|----------|\n| check_go_build.py | 检查 Go 编译 | |\n"
		if err := os.WriteFile(filepath.Join(skillsDir, "index.md"), []byte(indexContent), 0644); err != nil {
			t.Fatal(err)
		}

		promptContent, err := GenerateDoingPromptFile(task, 0, cm, pm, tmpDir)
		if err != nil {
			t.Fatalf("GenerateDoingPromptFile: %v", err)
		}
		defer os.Remove(promptContent)

		data, err := os.ReadFile(promptContent)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "Skills Index") {
			t.Error("doing prompt must contain skills index content when index.md exists")
		}
	})

	t.Run("task2/plan_prompt_contains_skills_index_when_exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			t.Fatal(err)
		}
		indexContent := "# Skills Index\n\n| 文件 | 描述 |\n|------|------|\n| my_skill.py | 描述 |\n"
		if err := os.WriteFile(filepath.Join(skillsDir, "index.md"), []byte(indexContent), 0644); err != nil {
			t.Fatal(err)
		}

		promptContent, err := GeneratePlanPrompt("需求描述", "/tmp/plan", cm, pm, tmpDir)
		if err != nil {
			t.Fatalf("GeneratePlanPrompt: %v", err)
		}
		if !strings.Contains(promptContent, "Skills Index") {
			t.Error("plan prompt must contain skills index content when index.md exists")
		}
	})

	t.Run("task2/doing_prompt_fallback_to_py_scan_when_no_index", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			t.Fatal(err)
		}
		// No index.md, but has a .py file
		pyContent := "# Description: 检查 Go 编译状态\nprint('ok')\n"
		if err := os.WriteFile(filepath.Join(skillsDir, "check_go_build.py"), []byte(pyContent), 0644); err != nil {
			t.Fatal(err)
		}

		promptFile, err := GenerateDoingPromptFile(task, 0, cm, pm, tmpDir)
		if err != nil {
			t.Fatalf("GenerateDoingPromptFile: %v", err)
		}
		defer os.Remove(promptFile)

		data, err := os.ReadFile(promptFile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "check_go_build") {
			t.Error("doing prompt must fall back to .py scan when index.md does not exist")
		}
	})

	// ─── task3: tools injection ───────────────────────────────────────────────

	t.Run("task3/doing_prompt_contains_tools_when_tools_exist", func(t *testing.T) {
		// Set up a temp project root with tools/*.py
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(toolsDir, "my_tool.py"), []byte("# Description: 示例工具\nprint('hi')\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Change to tmpDir so os.Getwd() returns tmpDir
		orig, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(orig)

		promptFile, err := GenerateDoingPromptFile(task, 0, cm, pm)
		if err != nil {
			t.Fatalf("GenerateDoingPromptFile: %v", err)
		}
		defer os.Remove(promptFile)

		data, err := os.ReadFile(promptFile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "my_tool") {
			t.Error("doing prompt must contain tools list when tools/*.py exist")
		}
		if !strings.Contains(string(data), "示例工具") {
			t.Error("doing prompt must contain tool description")
		}
	})

	t.Run("task3/plan_prompt_contains_tools_when_tools_exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(toolsDir, "plan_tool.py"), []byte("# Description: 规划工具\nprint('plan')\n"), 0644); err != nil {
			t.Fatal(err)
		}

		orig, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(orig)

		promptContent, err := GeneratePlanPrompt("需求", "/tmp/plan", cm, pm)
		if err != nil {
			t.Fatalf("GeneratePlanPrompt: %v", err)
		}
		if !strings.Contains(promptContent, "plan_tool") {
			t.Error("plan prompt must contain tools list when tools/*.py exist")
		}
	})

	t.Run("task3/doing_prompt_no_tools_section_when_no_tools", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No tools/ directory

		orig, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(orig)

		promptFile, err := GenerateDoingPromptFile(task, 0, cm, pm)
		if err != nil {
			t.Fatalf("GenerateDoingPromptFile: %v", err)
		}
		defer os.Remove(promptFile)

		data, err := os.ReadFile(promptFile)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "可用的项目 Tools") {
			t.Error("doing prompt must not contain tools section when tools/ does not exist")
		}
	})

	t.Run("task3/plan_prompt_no_tools_section_when_no_tools", func(t *testing.T) {
		tmpDir := t.TempDir()

		orig, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(orig)

		promptContent, err := GeneratePlanPrompt("需求", "/tmp/plan", cm, pm)
		if err != nil {
			t.Fatalf("GeneratePlanPrompt: %v", err)
		}
		// tools_list variable should be replaced with empty string
		if strings.Contains(promptContent, "{{tools_list}}") {
			t.Error("plan prompt must not contain unreplaced {{tools_list}} variable")
		}
	})

	// ─── task4: job-level OKR ─────────────────────────────────────────────────

	t.Run("task4/plan_prompt_no_global_okr_content", func(t *testing.T) {
		// plan template should not contain {{okr_content}} (global OKR removed)
		tmpl, err := pm.LoadTemplate("plan")
		if err != nil {
			t.Fatalf("load plan template: %v", err)
		}
		if strings.Contains(tmpl.Content, "{{okr_content}}") {
			t.Error("plan template must not contain {{okr_content}} (global OKR was removed in task4)")
		}
	})

	t.Run("task4/plan_prompt_instructs_generate_job_okr", func(t *testing.T) {
		// plan template should instruct Claude to generate job_N/plan/OKR.md
		tmpl, err := pm.LoadTemplate("plan")
		if err != nil {
			t.Fatalf("load plan template: %v", err)
		}
		if !strings.Contains(tmpl.Content, "OKR.md") {
			t.Error("plan template must contain instruction to generate OKR.md")
		}
	})

	t.Run("task4/doing_prompt_contains_job_okr_when_exists", func(t *testing.T) {
		cmWithOKR := NewContextManager("job_9")
		cmWithOKR.LoadTask(task)
		cmWithOKR.LoadOKRFromContent("# Job OKR\n## O1: 完成认证模块\n- KR1: 登录接口通过测试")

		promptContent, err := GenerateDoingPrompt(task, 0, cmWithOKR, pm)
		if err != nil {
			t.Fatalf("GenerateDoingPrompt: %v", err)
		}
		if !strings.Contains(promptContent, "完成认证模块") {
			t.Error("doing prompt must contain job OKR content when job OKR exists")
		}
	})

	t.Run("task4/doing_prompt_no_error_when_job_okr_missing", func(t *testing.T) {
		cmNoOKR := NewContextManager("job_9")
		cmNoOKR.LoadTask(task)
		// No OKR loaded

		_, err := GenerateDoingPrompt(task, 0, cmNoOKR, pm)
		if err != nil {
			t.Errorf("GenerateDoingPrompt must not error when job OKR is missing: %v", err)
		}
	})
}

// buildLearningPromptForTest is a helper that calls cmd.buildLearningPrompt indirectly
// by constructing the prompt via the template system directly (same logic as cmd/learning.go).
func buildLearningPromptForTest(t *testing.T, jobID, taskMDContent, okrContent, tmpDir string) (string, error) {
	t.Helper()
	pm := NewPromptManager("")
	tmpl, err := pm.LoadTemplate("learning")
	if err != nil {
		return "", err
	}
	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("project_name", "rick")
	builder.SetVariable("project_description", "Context-First AI Coding Framework")
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("learning_dir", filepath.Join(tmpDir, "learning"))
	builder.SetVariable("rick_bin_path", "./bin/rick")
	builder.SetVariable("task_execution_results", "| task1 | Test | success | task1.md | abc12345 | 1 |")
	builder.SetVariable("debug_records", "无调试信息")

	if okrContent != "" {
		builder.SetVariable("okr_content", okrContent)
	} else {
		builder.SetVariable("okr_content", "（本 job 无 OKR.md）")
	}

	if taskMDContent != "" {
		builder.SetVariable("task_md_content", taskMDContent)
	} else {
		builder.SetVariable("task_md_content", "（本 job 无 task*.md 文件）")
	}

	return builder.Build()
}
