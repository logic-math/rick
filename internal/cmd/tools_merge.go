package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/workspace"
)

// NewMergeCmd creates the merge subcommand
func NewMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <job_id>",
		Short: "Merge learning outputs into the main .rick context",
		Long: `Merge the learning phase outputs for a job into the main .rick context.

This command is primarily called by AI agents during the learning phase.

Flow:
  1. Check SUMMARY.md first line contains "APPROVED: true"
  2. Create branch learning/job_N from current HEAD
  3. Switch to that branch
  4. Copy learning/wiki/ → .rick/wiki/ (overwrite)
  5. Copy learning/skills/ → .rick/skills/ (overwrite)
  6. Copy learning/OKR.md → .rick/OKR.md (if exists)
  7. Copy learning/SPEC.md → .rick/SPEC.md (if exists)
  8. Regenerate .rick/wiki/README.md and .rick/skills/README.md
  9. git commit "learning: merge job_N knowledge"
  10. Switch back to original branch
  11. Print structured summary for AI agent

After this command, the AI agent should:
  - Show the diff to the human
  - After human approval: git merge --no-ff learning/job_N
  - Then: git branch -D learning/job_N

Arguments:
  job_id    Job identifier (e.g. job_1)

Exit codes:
  0  merge completed successfully
  1  error (not approved, git error, etc.)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]
			return runMerge(jobID)
		},
	}
}

func runMerge(jobID string) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	learningDir, err := workspace.GetJobLearningDir(jobID)
	if err != nil {
		return fmt.Errorf("failed to get learning directory: %w", err)
	}

	// Step 1: Check APPROVED: true in SUMMARY.md first line
	summaryPath := filepath.Join(learningDir, "SUMMARY.md")
	if err := checkApproved(summaryPath); err != nil {
		return err
	}

	// Step 2: Get current branch
	originalBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Step 3: Create and switch to learning branch
	branchName := fmt.Sprintf("learning/%s", jobID)
	if err := gitCreateAndSwitch(branchName); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Ensure we switch back on exit (best effort)
	defer func() {
		_ = gitCheckout(originalBranch)
	}()

	// Track what was merged
	var mergedItems []string

	// Step 4: Copy learning/wiki/ → .rick/wiki/
	learningWikiDir := filepath.Join(learningDir, "wiki")
	if info, err := os.Stat(learningWikiDir); err == nil && info.IsDir() {
		destWikiDir := filepath.Join(rickDir, workspace.WikiDirName)
		if err := os.MkdirAll(destWikiDir, 0755); err != nil {
			return fmt.Errorf("failed to create wiki directory: %w", err)
		}
		count, err := copyDir(learningWikiDir, destWikiDir)
		if err != nil {
			return fmt.Errorf("failed to copy wiki: %w", err)
		}
		// Regenerate wiki README
		if err := generateWikiREADME(rickDir); err != nil {
			return fmt.Errorf("failed to regenerate wiki README: %w", err)
		}
		mergedItems = append(mergedItems, fmt.Sprintf("wiki: %d files copied", count))
	}

	// Step 5: Copy learning/skills/ → .rick/skills/
	learningSkillsDir := filepath.Join(learningDir, "skills")
	if info, err := os.Stat(learningSkillsDir); err == nil && info.IsDir() {
		destSkillsDir := filepath.Join(rickDir, workspace.SkillsDirName)
		if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}
		count, err := copyDir(learningSkillsDir, destSkillsDir)
		if err != nil {
			return fmt.Errorf("failed to copy skills: %w", err)
		}
		// Regenerate skills README
		if err := workspace.GenerateSkillsREADME(rickDir); err != nil {
			return fmt.Errorf("failed to regenerate skills README: %w", err)
		}
		mergedItems = append(mergedItems, fmt.Sprintf("skills: %d files copied", count))
	}

	// Step 6: Copy learning/OKR.md → .rick/OKR.md
	learningOKR := filepath.Join(learningDir, "OKR.md")
	if _, err := os.Stat(learningOKR); err == nil {
		destOKR := filepath.Join(rickDir, workspace.OKRFileName)
		if err := copyFile(learningOKR, destOKR); err != nil {
			return fmt.Errorf("failed to copy OKR.md: %w", err)
		}
		mergedItems = append(mergedItems, "OKR.md: updated")
	}

	// Step 7: Copy learning/SPEC.md → .rick/SPEC.md
	learningSPEC := filepath.Join(learningDir, "SPEC.md")
	if _, err := os.Stat(learningSPEC); err == nil {
		destSPEC := filepath.Join(rickDir, workspace.SpecFileName)
		if err := copyFile(learningSPEC, destSPEC); err != nil {
			return fmt.Errorf("failed to copy SPEC.md: %w", err)
		}
		mergedItems = append(mergedItems, "SPEC.md: updated")
	}

	// Step 8: git add changed paths (only those that exist)
	pathsToAdd := []string{}
	wikiDir := filepath.Join(rickDir, workspace.WikiDirName)
	skillsDir := filepath.Join(rickDir, workspace.SkillsDirName)
	okrFile := filepath.Join(rickDir, workspace.OKRFileName)
	specFile := filepath.Join(rickDir, workspace.SpecFileName)

	for _, p := range []string{wikiDir, skillsDir, okrFile, specFile} {
		if _, err := os.Stat(p); err == nil {
			pathsToAdd = append(pathsToAdd, p)
		}
	}

	if len(pathsToAdd) > 0 {
		gitAddArgs := append([]string{"add"}, pathsToAdd...)
		if out, err := runGit(gitAddArgs...); err != nil {
			return fmt.Errorf("git add failed: %w\n%s", err, out)
		}
	}

	// Step 9: git commit
	commitMsg := fmt.Sprintf("learning: merge %s knowledge", jobID)
	if out, err := runGit("commit", "-m", commitMsg); err != nil {
		// If nothing to commit, that's ok
		if !strings.Contains(out, "nothing to commit") {
			return fmt.Errorf("git commit failed: %w\n%s", err, out)
		}
	}

	// Step 10: Switch back to original branch
	if err := gitCheckout(originalBranch); err != nil {
		return fmt.Errorf("failed to switch back to branch %s: %w", originalBranch, err)
	}

	// Re-apply the file copies to the working directory on the original branch
	// so the files are available as uncommitted changes for human review.
	if info, err := os.Stat(learningWikiDir); err == nil && info.IsDir() {
		destWikiDir := filepath.Join(rickDir, workspace.WikiDirName)
		_ = os.MkdirAll(destWikiDir, 0755)
		_, _ = copyDir(learningWikiDir, destWikiDir)
		_ = generateWikiREADME(rickDir)
	}
	if info, err := os.Stat(learningSkillsDir); err == nil && info.IsDir() {
		destSkillsDir := filepath.Join(rickDir, workspace.SkillsDirName)
		_ = os.MkdirAll(destSkillsDir, 0755)
		_, _ = copyDir(learningSkillsDir, destSkillsDir)
		_ = workspace.GenerateSkillsREADME(rickDir)
	}
	if _, err := os.Stat(learningOKR); err == nil {
		_ = copyFile(learningOKR, filepath.Join(rickDir, workspace.OKRFileName))
	}
	if _, err := os.Stat(learningSPEC); err == nil {
		_ = copyFile(learningSPEC, filepath.Join(rickDir, workspace.SpecFileName))
	}

	// Step 11: Print structured summary for AI agent
	printMergeSummary(jobID, branchName, originalBranch, mergedItems)

	return nil
}

// checkApproved verifies that SUMMARY.md first line is "APPROVED: true"
func checkApproved(summaryPath string) error {
	f, err := os.Open(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("SUMMARY.md not found at %s\nPlease create SUMMARY.md with 'APPROVED: true' on the first line", summaryPath)
		}
		return fmt.Errorf("failed to read SUMMARY.md: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		firstLine := strings.TrimSpace(scanner.Text())
		if firstLine == "APPROVED: true" {
			return nil
		}
	}

	return fmt.Errorf("merge rejected: SUMMARY.md first line must be 'APPROVED: true'\n" +
		"Please add 'APPROVED: true' to the top of %s and retry", summaryPath)
}

// getCurrentBranch returns the current git branch name
func getCurrentBranch() (string, error) {
	out, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// gitCreateAndSwitch creates a new branch from HEAD and switches to it
func gitCreateAndSwitch(branchName string) error {
	// Delete branch if it already exists (idempotent)
	_, _ = runGit("branch", "-D", branchName)

	out, err := runGit("checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

// gitCheckout switches to the given branch
func gitCheckout(branch string) error {
	out, err := runGit("checkout", branch)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

// runGit runs a git command in the current working directory and returns combined output
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// copyDir copies all files from src to dst (non-recursive, top-level only).
// Returns the count of files copied.
func copyDir(src, dst string) (int, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcFile := filepath.Join(src, e.Name())
		dstFile := filepath.Join(dst, e.Name())
		if err := copyFile(srcFile, dstFile); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// copyFile copies a single file from src to dst, overwriting dst if it exists
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	return nil
}

// generateWikiREADME scans rickDir/wiki/*.md and regenerates README.md
func generateWikiREADME(rickDir string) error {
	wikiDir := filepath.Join(rickDir, workspace.WikiDirName)
	entries, err := os.ReadDir(wikiDir)
	if err != nil {
		return fmt.Errorf("failed to read wiki directory: %w", err)
	}

	type wikiEntry struct {
		name    string
		title   string
		summary string
	}

	var wikiEntries []wikiEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" {
			continue
		}
		filePath := filepath.Join(wikiDir, e.Name())
		title, summary := extractWikiTitleAndSummary(filePath)
		wikiEntries = append(wikiEntries, wikiEntry{
			name:    strings.TrimSuffix(e.Name(), ".md"),
			title:   title,
			summary: summary,
		})
	}

	var sb strings.Builder
	sb.WriteString("# Wiki\n\n")
	sb.WriteString("| 文件 | 标题 | 摘要 |\n")
	sb.WriteString("|------|------|------|\n")
	for _, we := range wikiEntries {
		sb.WriteString(fmt.Sprintf("| %s.md | %s | %s |\n", we.name, we.title, we.summary))
	}

	readmePath := filepath.Join(wikiDir, "README.md")
	return os.WriteFile(readmePath, []byte(sb.String()), 0644)
}

// extractWikiTitleAndSummary extracts the first # heading and first paragraph from a markdown file
func extractWikiTitleAndSummary(filePath string) (title, summary string) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	titleFound := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !titleFound && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			titleFound = true
			continue
		}
		if titleFound && line != "" && !strings.HasPrefix(line, "#") {
			summary = line
			break
		}
	}
	return title, summary
}

// printMergeSummary prints an AI-friendly structured summary of the merge
func printMergeSummary(jobID, branchName, originalBranch string, mergedItems []string) {
	fmt.Println("## Merge Summary")
	fmt.Println()
	fmt.Printf("**Job**: %s\n", jobID)
	fmt.Printf("**Branch created**: `%s`\n", branchName)
	fmt.Printf("**Current branch**: `%s` (restored)\n", originalBranch)
	fmt.Println()
	fmt.Println("### Changes Merged")
	if len(mergedItems) == 0 {
		fmt.Println("- (no changes)")
	} else {
		for _, item := range mergedItems {
			fmt.Printf("- %s\n", item)
		}
	}
	fmt.Println()
	fmt.Println("### Next Steps for AI Agent")
	fmt.Println()
	fmt.Printf("1. Show the diff to the human:\n")
	fmt.Printf("   ```\n   git diff %s..%s\n   ```\n", originalBranch, branchName)
	fmt.Println()
	fmt.Println("2. After human approves, merge:")
	fmt.Printf("   ```\n   git merge --no-ff %s -m \"merge: integrate %s learning\"\n   ```\n", branchName, jobID)
	fmt.Println()
	fmt.Println("3. Delete the learning branch:")
	fmt.Printf("   ```\n   git branch -D %s\n   ```\n", branchName)
}
