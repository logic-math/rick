package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunDoingCheck validates the doing directory structure.
// It checks tasks.json parseability and debug/ bug files format.
// Returns nil if all checks pass.
func RunDoingCheck(doingDir string) error {
	// 1. tasks.json exists and is parseable
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")
	_, err := LoadTasksJSON(tasksJSONPath)
	if err != nil {
		return fmt.Errorf("tasks.json not found or invalid: %w", err)
	}

	// 2. Validate debug/ directory and bug*.md files
	if err := CheckDebugDir(doingDir); err != nil {
		return err
	}

	return nil
}

// RunEasyCheck validates the doing directory for easy mode jobs.
// Easy mode has no task breakdown, so only the debug/ directory format is checked.
func RunEasyCheck(doingDir string) error {
	return CheckDebugDir(doingDir)
}

// CheckDebugDir validates debug/bug*.md files inside doingDir.
//
// File name rule:  bug{n}-{description}.md
// Required ## sections: Phase 1-6 (构建反馈回路/复现最小化/可证伪假设/插桩观察/修复回归/清理事后分析) + 结论
// Each ## 尝试N / ## 实验N block must contain: - 假设, - 改动, - 结果
// Frontmatter must have status: and must not be "🔄 进行中".
func CheckDebugDir(doingDir string) error {
	debugDir := filepath.Join(doingDir, "debug")
	info, err := os.Stat(debugDir)
	if os.IsNotExist(err) {
		return nil // no debug/ is fine
	}
	if err != nil {
		return fmt.Errorf("failed to stat debug/: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("debug/ exists but is not a directory")
	}

	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return fmt.Errorf("failed to read debug/: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := e.Name()

		if !isBugFileName(name) {
			return fmt.Errorf("debug/%s: file name must match bug{n}-{description}.md (e.g. bug1-nil-pointer.md)", name)
		}

		content, err := os.ReadFile(filepath.Join(debugDir, name))
		if err != nil {
			return fmt.Errorf("failed to read debug/%s: %w", name, err)
		}
		text := string(content)

		if !strings.Contains(text, "status:") {
			return fmt.Errorf("debug/%s: missing 'status:' in YAML frontmatter", name)
		}
		if strings.Contains(text, `"🔄 进行中"`) || strings.Contains(text, "'🔄 进行中'") {
			return fmt.Errorf("debug/%s: status is '🔄 进行中' — must be ✅ 已解决 or ❌ 无法修复", name)
		}

		for _, sec := range []string{
			"## Phase 1: 构建反馈回路",
			"## Phase 2: 复现最小化",
			"## Phase 3: 可证伪假设",
			"## Phase 4: 插桩观察",
			"## Phase 5: 修复回归",
			"## Phase 6: 清理事后分析",
			"## 结论",
		} {
			if !containsHeading(text, sec) {
				return fmt.Errorf("debug/%s: missing required section '%s'", name, sec)
			}
		}

		if err := checkAttemptBlocks(name, text); err != nil {
			return err
		}
	}
	return nil
}

func isBugFileName(name string) bool {
	if !strings.HasPrefix(name, "bug") {
		return false
	}
	rest := name[3:]
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(rest) || rest[i] != '-' {
		return false
	}
	desc := rest[i+1:]
	return len(desc) > 3 && strings.HasSuffix(desc, ".md")
}

func containsHeading(text, heading string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == heading {
			return true
		}
	}
	return false
}

func checkAttemptBlocks(fileName, text string) error {
	lines := strings.Split(text, "\n")
	inBlock := false
	blockName := ""
	has假设, has改动, has结果 := false, false, false

	flush := func() error {
		if !inBlock || blockName == "" {
			return nil
		}
		var missing []string
		if !has假设 {
			missing = append(missing, "- 假设")
		}
		if !has改动 {
			missing = append(missing, "- 改动")
		}
		if !has结果 {
			missing = append(missing, "- 结果")
		}
		if len(missing) > 0 {
			return fmt.Errorf("debug/%s: block '%s' missing: %s", fileName, blockName, strings.Join(missing, ", "))
		}
		return nil
	}

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## 尝试") || strings.HasPrefix(t, "## 实验") {
			if err := flush(); err != nil {
				return err
			}
			inBlock, blockName = true, t
			has假设, has改动, has结果 = false, false, false
			continue
		}
		if inBlock && (strings.HasPrefix(t, "## ") || strings.HasPrefix(t, "# ")) {
			if err := flush(); err != nil {
				return err
			}
			inBlock, blockName = false, ""
		}
		if inBlock {
			if strings.HasPrefix(t, "- 假设") {
				has假设 = true
			}
			if strings.HasPrefix(t, "- 改动") {
				has改动 = true
			}
			if strings.HasPrefix(t, "- 结果") {
				has结果 = true
			}
		}
	}
	return flush()
}
