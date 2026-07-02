package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateDreamPrompt generates the dream prompt content with placeholder paths (for dry-run).
func GenerateDreamPrompt(jobIDs []string, rickDir string) (string, error) {
	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("dream")
	if err != nil {
		return "", fmt.Errorf("failed to load dream template: %w", err)
	}
	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("pending_jobs", formatPendingJobs(jobIDs))
	builder.SetVariable("run_logs", loadRunLogs(rickDir))
	builder.SetVariable("loops_context", LoadLoopsContext(filepath.Join(rickDir, "loops")))
	builder.SetVariable("loops_dir", "<loops_dir>")
	builder.SetVariable("skills_dir", "<skills_dir>")
	builder.SetVariable("domain_dir", "<domain_dir>")
	builder.SetVariable("sense_skill_path", "<tmp>/rick-dream-skill-sense-*.md")
	builder.SetVariable("evolve_skills_skill_path", "<tmp>/rick-dream-skill-evolve-skills-*.md")
	builder.SetVariable("gen_skill_path", "<tmp>/rick-dream-skill-gen-skill-*.md")
	builder.SetVariable("gen_loop_path", "<tmp>/rick-dream-skill-gen-loop-*.md")
	builder.SetVariable("gen_domain_path", "<tmp>/rick-dream-skill-gen-domain-*.md")
	builder.SetVariable("rick_bin_path", "<rick>")
	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build dream prompt: %w", err)
	}
	actPathContent := loadActPaths(jobIDs, rickDir)
	if actPathContent != "" {
		content += "\n\n## 行为轨迹文件路径（按需读取）\n\n" + actPathContent
	}
	return content, nil
}

// GenerateDreamPromptFile generates the dream phase prompt and saves it to .rick/dream/prompts/.
// jobIDs is the list of jobs to process; rickDir is the .rick directory path.
// All files are persistent; no cleanup needed.
func GenerateDreamPromptFile(jobIDs []string, rickDir string) (string, []string, error) {
	mgr := NewPromptManager()

	tmpl, err := mgr.LoadTemplate("dream")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load dream template: %w", err)
	}

	dreamDir := filepath.Join(rickDir, "dream")
	promptsDir, err := EnsurePromptsDir(dreamDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create dream prompts dir: %w", err)
	}

	senseFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", nil, err
	}
	evolveFile, err := WriteSkillFile(promptsDir, "skill_evolve_skills.md", "evolve-skills")
	if err != nil {
		return "", nil, err
	}
	genSkillFile, err := WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return "", nil, err
	}
	genLoopFile, err := WriteSkillFile(promptsDir, "skill_gen_loop.md", "gen-loop")
	if err != nil {
		return "", nil, err
	}
	genDomainFile, err := WriteSkillFile(promptsDir, "skill_gen_domain.md", "gen-domain")
	if err != nil {
		return "", nil, err
	}

	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")
	domainDir := filepath.Join(rickDir, "domain")
	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("pending_jobs", formatPendingJobs(jobIDs))
	builder.SetVariable("run_logs", loadRunLogs(rickDir))
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("loops_dir", loopsDir)
	builder.SetVariable("skills_dir", skillsDir)
	builder.SetVariable("domain_dir", domainDir)
	builder.SetVariable("sense_skill_path", senseFile)
	builder.SetVariable("evolve_skills_skill_path", evolveFile)
	builder.SetVariable("gen_skill_path", genSkillFile)
	builder.SetVariable("gen_loop_path", genLoopFile)
	builder.SetVariable("gen_domain_path", genDomainFile)
	projectRoot, _ := os.Getwd()
	builder.SetVariable("rick_bin_path", filepath.Join(projectRoot, "bin", "rick"))

	content, err := builder.Build()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build dream prompt: %w", err)
	}

	// Append act-path context for each job
	actPathContent := loadActPaths(jobIDs, rickDir)
	if actPathContent != "" {
		content += "\n\n## 行为轨迹文件路径（按需读取）\n\n" + actPathContent
	}

	promptFile := filepath.Join(promptsDir, "dream_prompt.md")
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write dream prompt: %w", err)
	}

	return promptFile, nil, nil
}


func formatPendingJobs(jobIDs []string) string {
	if len(jobIDs) == 0 {
		return "（无待处理 jobs）"
	}
	var sb strings.Builder
	for _, id := range jobIDs {
		sb.WriteString("- " + id + "\n")
	}
	return sb.String()
}

func loadRunLogs(rickDir string) string {
	if rickDir == "" {
		return "（无历史 dream 记录）"
	}
	dreamDir := filepath.Join(rickDir, "dream")
	entries, err := os.ReadDir(dreamDir)
	if err != nil {
		return "（无历史 dream 记录）"
	}

	var logs []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, "dream_run_") && strings.HasSuffix(name, "_log.md") {
			logs = append(logs, name)
		}
	}
	if len(logs) == 0 {
		return "（无历史 dream 记录）"
	}
	sort.Strings(logs)

	var sb strings.Builder
	for _, name := range logs {
		content, err := os.ReadFile(filepath.Join(dreamDir, name))
		if err != nil {
			continue
		}
		sb.WriteString("### " + name + "\n\n")
		sb.Write(content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func loadActPaths(jobIDs []string, rickDir string) string {
	if len(jobIDs) == 0 || rickDir == "" {
		return ""
	}
	var sb strings.Builder
	for _, jobID := range jobIDs {
		doingTasksDir := filepath.Join(rickDir, "jobs", jobID, "doing", "tasks")
		entries, err := os.ReadDir(doingTasksDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			actPath := filepath.Join(doingTasksDir, e.Name(), "act-path.md")
			if _, err := os.Stat(actPath); err == nil {
				sb.WriteString(fmt.Sprintf("- `%s`\n", actPath))
			}
		}
	}
	return sb.String()
}
