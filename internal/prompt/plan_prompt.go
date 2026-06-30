package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveRickBinPath returns the path to the rick binary.
func resolveRickBinPath() string {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "rick"
	}
	localBin := filepath.Join(projectRoot, "bin", "rick")
	if _, err := os.Stat(localBin); err == nil {
		return localBin
	}
	return "rick"
}

// extractJobID extracts the job ID (e.g. "job_1") from a plan or doing directory path.
func extractJobID(dirPath string) string {
	parts := strings.Split(filepath.ToSlash(dirPath), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasPrefix(parts[i], "job_") {
			return parts[i]
		}
	}
	return "job_N"
}

// GeneratePlanPrompt generates the plan prompt content with placeholder paths (for dry-run).
func GeneratePlanPrompt(requirement string, jobPlanDir string, rickDir string) (string, error) {
	if requirement == "" {
		return "", fmt.Errorf("requirement cannot be empty")
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("plan")
	if err != nil {
		return "", fmt.Errorf("failed to load plan template: %w", err)
	}

	loopsDir := filepath.Join(rickDir, "loops")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("user_requirement", requirement)
	builder.SetVariable("job_plan_dir", jobPlanDir)
	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("job_id", extractJobID(jobPlanDir))
	builder.SetVariable("grilling_skill_path", "<tmp>/rick-plan-prompts/skill_grilling.md")
	builder.SetVariable("write_spec_skill_path", "<tmp>/rick-plan-skill-write_spec-*.md")
	builder.SetVariable("tdd_skill_path", "<tmp>/rick-plan-skill-tdd-zh-*.md")
	builder.SetVariable("testing_anti_patterns_path", "<tmp>/rick-plan-skill-testing-anti-patterns-zh-*.md")
	builder.SetVariable("debug_skill_path", "<doing-prompts>/skill_debug_skill.md")

	return builder.Build()
}

// GeneratePlanPromptFile generates the plan prompt and saves to jobPlanDir/prompts/.
// All files are persistent; no cleanup needed by caller.
func GeneratePlanPromptFile(requirement string, jobPlanDir string, rickDir string) (string, []string, error) {
	if requirement == "" {
		return "", nil, fmt.Errorf("requirement cannot be empty")
	}

	promptsDir, err := EnsurePromptsDir(jobPlanDir)
	if err != nil {
		return "", nil, err
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("plan")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load plan template: %w", err)
	}

	grillingFile, err := WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")
	if err != nil {
		return "", nil, err
	}
	writeSpecFile, err := WriteSkillFile(promptsDir, "skill_write_spec.md", "write_spec")
	if err != nil {
		return "", nil, err
	}
	tddZhFile, err := WriteSkillFile(promptsDir, "skill_tdd_zh.md", "tdd-zh")
	if err != nil {
		return "", nil, err
	}
	testingAntiPatternsFile, err := WriteSkillFile(promptsDir, "skill_testing_anti_patterns_zh.md", "testing-anti-patterns-zh")
	if err != nil {
		return "", nil, err
	}

	loopsDir := filepath.Join(rickDir, "loops")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("user_requirement", requirement)
	builder.SetVariable("job_plan_dir", jobPlanDir)
	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("job_id", extractJobID(jobPlanDir))
	// Compute the doing/prompts path (will be created when doing runs)
	doingPromptsDir := filepath.Join(filepath.Dir(jobPlanDir), "doing", "prompts")
	debugSkillPath := filepath.Join(doingPromptsDir, "skill_debug_skill.md")

	builder.SetVariable("grilling_skill_path", grillingFile)
	builder.SetVariable("write_spec_skill_path", writeSpecFile)
	builder.SetVariable("tdd_skill_path", tddZhFile)
	builder.SetVariable("testing_anti_patterns_path", testingAntiPatternsFile)
	builder.SetVariable("debug_skill_path", debugSkillPath)

	promptFile := filepath.Join(promptsDir, "plan_prompt.md")
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", nil, fmt.Errorf("failed to save plan prompt: %w", err)
	}

	return promptFile, nil, nil
}



