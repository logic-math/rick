package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const compliantLoopContent = `---
name: my-loop
trigger: when doing something
---

## 目标

Some goal

## 上下文管理

Context management

## 可调用工具

Tools

## 产出评估

Output evaluation

## 停止标准

Termination
`

const compliantSkillContent = `---
name: my-skill
description: does something
---

## When to Use

Use when...

## Procedure

Do this...

## Pitfalls

Watch out for...

## Verification

Verify by...
`

func makeLoopsDir(t *testing.T, rickDir string) string {
	t.Helper()
	dir := filepath.Join(rickDir, "loops")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func makeSkillsDir(t *testing.T, rickDir string) string {
	t.Helper()
	dir := filepath.Join(rickDir, "skills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestLoopsSkillsCheck groups all runLoopsAndSkillsCheck tests.

func TestLoopsSkillsCheck_NoDirectories(t *testing.T) {
	rickDir := t.TempDir()
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) != 0 {
		t.Errorf("expected no errors when dirs don't exist, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_CompliantLoop(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	if err := os.WriteFile(filepath.Join(loopsDir, "my_loop.md"), []byte(compliantLoopContent), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) != 0 {
		t.Errorf("expected no errors for compliant loop, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_LoopMissingTrigger(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	content := `---
name: my-loop
---

## 目标

## 上下文管理

## 可调用工具

## 产出评估

## 停止标准
`
	if err := os.WriteFile(filepath.Join(loopsDir, "bad_loop.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected errors for loop missing trigger")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "bad_loop.md") && strings.Contains(e, "trigger") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error mentioning bad_loop.md and trigger, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_LoopMissingSection(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	content := `---
name: my-loop
trigger: something
---

## 目标

## 上下文管理

## 可调用工具

## 停止标准
`
	// Missing ## 产出评估
	if err := os.WriteFile(filepath.Join(loopsDir, "bad_loop.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected errors for loop missing section")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "bad_loop.md") && strings.Contains(e, "产出评估") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error mentioning bad_loop.md and 产出评估, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_CompliantSkill(t *testing.T) {
	rickDir := t.TempDir()
	skillsDir := makeSkillsDir(t, rickDir)
	if err := os.WriteFile(filepath.Join(skillsDir, "my_skill.md"), []byte(compliantSkillContent), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) != 0 {
		t.Errorf("expected no errors for compliant skill, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_SkillMissingDescription(t *testing.T) {
	rickDir := t.TempDir()
	skillsDir := makeSkillsDir(t, rickDir)
	content := `---
name: my-skill
---

## When to Use

## Procedure

## Pitfalls

## Verification
`
	if err := os.WriteFile(filepath.Join(skillsDir, "bad_skill.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected errors for skill missing description")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "bad_skill.md") && strings.Contains(e, "description") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error mentioning bad_skill.md and description, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_SkillMissingSection(t *testing.T) {
	rickDir := t.TempDir()
	skillsDir := makeSkillsDir(t, rickDir)
	content := `---
name: my-skill
description: does something
---

## When to Use

## Pitfalls

## Verification
`
	// Missing ## Procedure
	if err := os.WriteFile(filepath.Join(skillsDir, "bad_skill.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected errors for skill missing Procedure section")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "bad_skill.md") && strings.Contains(e, "Procedure") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error mentioning bad_skill.md and Procedure, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_READMESkipped(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	// README.md has no frontmatter or required sections — should be skipped
	if err := os.WriteFile(filepath.Join(loopsDir, "README.md"), []byte("# Format Spec\nThis is the format spec."), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) != 0 {
		t.Errorf("expected README.md to be skipped, got errors: %v", errs)
	}
}

func TestLoopsSkillsCheck_MultipleErrors(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	// Missing trigger AND missing ## 产出评估
	content := `---
name: my-loop
---

## 目标

## 上下文管理

## 可调用工具

## 停止标准
`
	if err := os.WriteFile(filepath.Join(loopsDir, "multi_error.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors (trigger + section), got %d: %v", len(errs), errs)
	}
}

func TestLoopsSkillsCheck_LoopMissingFrontmatter(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := makeLoopsDir(t, rickDir)
	content := `# My Loop (no frontmatter)

## 目标

## 上下文管理

## 可调用工具

## 产出评估

## 停止标准
`
	if err := os.WriteFile(filepath.Join(loopsDir, "no_fm.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected error for loop missing frontmatter")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "no_fm.md") && strings.Contains(e, "frontmatter") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected frontmatter error for no_fm.md, got: %v", errs)
	}
}

func TestLoopsSkillsCheck_SkillMissingFrontmatter(t *testing.T) {
	rickDir := t.TempDir()
	skillsDir := makeSkillsDir(t, rickDir)
	content := `# My Skill (no frontmatter)

## When to Use

## Procedure

## Pitfalls

## Verification
`
	if err := os.WriteFile(filepath.Join(skillsDir, "no_fm.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	errs := runLoopsAndSkillsCheck(rickDir)
	if len(errs) == 0 {
		t.Fatal("expected error for skill missing frontmatter")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "no_fm.md") && strings.Contains(e, "frontmatter") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected frontmatter error for no_fm.md, got: %v", errs)
	}
}
