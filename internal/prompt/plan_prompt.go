package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/workspace"
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

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("okr_path", loadOKRPath(rickDir))
	builder.SetVariable("spec_path", loadSpecPath(rickDir))
	builder.SetVariable("rfc_dir", loadRFCDir(rickDir))
	builder.SetVariable("rfc_paths", loadRFCPaths(rickDir))
	builder.SetVariable("user_requirement", requirement)
	builder.SetVariable("job_plan_dir", jobPlanDir)
	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("job_id", extractJobID(jobPlanDir))
	builder.SetVariable("sense_skill_path", "<tmp>/rick-plan-skill-sense-*.md")
	builder.SetVariable("write_spec_skill_path", "<tmp>/rick-plan-skill-write_spec-*.md")
	builder.SetVariable("tdd_skill_path", "<tmp>/rick-plan-skill-tdd-zh-*.md")
	builder.SetVariable("testing_anti_patterns_path", "<tmp>/rick-plan-skill-testing-anti-patterns-zh-*.md")

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

	senseFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
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

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("okr_path", loadOKRPath(rickDir))
	builder.SetVariable("spec_path", loadSpecPath(rickDir))
	builder.SetVariable("rfc_dir", loadRFCDir(rickDir))
	builder.SetVariable("rfc_paths", loadRFCPaths(rickDir))
	builder.SetVariable("user_requirement", requirement)
	builder.SetVariable("job_plan_dir", jobPlanDir)
	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("job_id", extractJobID(jobPlanDir))
	builder.SetVariable("sense_skill_path", senseFile)
	builder.SetVariable("write_spec_skill_path", writeSpecFile)
	builder.SetVariable("tdd_skill_path", tddZhFile)
	builder.SetVariable("testing_anti_patterns_path", testingAntiPatternsFile)

	promptFile := filepath.Join(promptsDir, "plan_prompt.md")
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", nil, fmt.Errorf("failed to save plan prompt: %w", err)
	}

	return promptFile, nil, nil
}

// loadOKRPath returns the path to .rick/OKR.md, or "暂无" if missing.
func loadOKRPath(rickDir string) string {
	if rickDir == "" {
		return "暂无"
	}
	p := filepath.Join(rickDir, "OKR.md")
	if _, err := os.Stat(p); err != nil {
		return "暂无"
	}
	return p
}

// loadSpecPath returns the path to .rick/SPEC.md, or "暂无" if missing.
func loadSpecPath(rickDir string) string {
	if rickDir == "" {
		return "暂无"
	}
	p := filepath.Join(rickDir, workspace.SpecFileName)
	if _, err := os.Stat(p); err != nil {
		return "暂无"
	}
	return p
}

// loadRFCDir returns the path to .rick/RFC/ directory, or "暂无" if rickDir is empty.
func loadRFCDir(rickDir string) string {
	if rickDir == "" {
		return "暂无"
	}
	return filepath.Join(rickDir, "RFC")
}

// loadRFCPaths returns a bullet list of paths to .md files under .rick/RFC/, or "暂无".
func loadRFCPaths(rickDir string) string {
	if rickDir == "" {
		return "暂无"
	}
	rfcDir := filepath.Join(rickDir, "RFC")
	entries, err := os.ReadDir(rfcDir)
	if err != nil || len(entries) == 0 {
		return "暂无"
	}

	var sb strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		sb.WriteString("- `" + filepath.Join(rfcDir, e.Name()) + "`\n")
	}
	if sb.Len() == 0 {
		return "暂无"
	}
	return sb.String()
}

