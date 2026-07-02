package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLoopsContext_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "loop1.md"), "---\nname: 调试循环\ntrigger: 当出现 bug 时\nscope: doing\n---\n\nbody\n")
	writeFile(t, filepath.Join(tmpDir, "loop2.md"), "---\nname: 代码审查\ntrigger: 当 PR 需要审查时\nscope: doing\n---\n\nbody\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "## 可用的项目 Loops")
	check(t, result, "**调试循环**")
	check(t, result, "当出现 bug 时")
	check(t, result, "**代码审查**")
	check(t, result, "当 PR 需要审查时")
}

func TestLoadLoopsContext_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	result := LoadLoopsContext(tmpDir)
	check(t, result, "暂无项目 Loop 记录")
}

func TestLoadLoopsContext_NonExistentDir(t *testing.T) {
	result := LoadLoopsContext("/nonexistent/path/that/should/not/exist/abc123")
	check(t, result, "暂无项目 Loop 记录")
}

func TestLoadLoopsContext_MissingTrigger(t *testing.T) {
	tmpDir := t.TempDir()

	// File without trigger field - should be skipped
	writeFile(t, filepath.Join(tmpDir, "no_trigger.md"), "---\nname: 无触发词循环\nscope: doing\n---\n\nbody\n")
	// File with complete fields - should appear
	writeFile(t, filepath.Join(tmpDir, "complete.md"), "---\nname: 完整循环\ntrigger: 有触发词时\nscope: doing\n---\n\nbody\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "**完整循环**")
	checkAbsent(t, result, "无触发词循环")
}

func TestLoadLoopsContext_NonMdFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "loop.md"), "---\nname: 有效循环\ntrigger: 触发条件\n---\n")
	writeFile(t, filepath.Join(tmpDir, "README.txt"), "name: 无效文件\ntrigger: 不应出现\n")

	result := LoadLoopsContext(tmpDir)

	check(t, result, "**有效循环**")
	checkAbsent(t, result, "无效文件")
}

func TestLoadSkillsContext_MultipleSkills(t *testing.T) {
	tmpDir := t.TempDir()

	skill1Dir := filepath.Join(tmpDir, "verify_go_changes_skill")
	os.MkdirAll(skill1Dir, 0755)
	writeFile(t, filepath.Join(skill1Dir, "skill.md"), "# skill:verify-go-changes（验证 Go 代码修改）\n\n## 触发场景\n\n修改了 Go 源文件后，需要验证编译通过时使用。\n\n## 预期效果\n\n编译通过。\n")

	skill2Dir := filepath.Join(tmpDir, "mark_task_success_skill")
	os.MkdirAll(skill2Dir, 0755)
	writeFile(t, filepath.Join(skill2Dir, "skill.md"), "# skill:mark-task-success（tasks.json 两阶段提交）\n\n## 触发场景\n\ndoing_check 报 status != success 时修复。\n\n## 预期效果\n\ncheck pass。\n")

	result := LoadSkillsContext(tmpDir)

	check(t, result, "## 可用的项目 Skills")
	check(t, result, "**verify-go-changes**")
	check(t, result, "修改了 Go 源文件后")
	check(t, result, "**mark-task-success**")
	check(t, result, "doing_check 报 status != success 时修复")
}

func TestLoadSkillsContext_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	result := LoadSkillsContext(tmpDir)
	check(t, result, "暂无项目 Skill 记录")
}

func TestLoadSkillsContext_NonExistentDir(t *testing.T) {
	result := LoadSkillsContext("/nonexistent/path/that/should/not/exist/abc123")
	check(t, result, "暂无项目 Skill 记录")
}

func TestLoadSkillsContext_IgnoresNonSkillDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Should be ignored: not ending with _skill
	notSkillDir := filepath.Join(tmpDir, "README")
	os.MkdirAll(notSkillDir, 0755)
	writeFile(t, filepath.Join(notSkillDir, "skill.md"), "# skill:fake（应被忽略）\n\n## 触发场景\n\n不应出现。\n")

	// Should appear
	skillDir := filepath.Join(tmpDir, "real_skill")
	os.MkdirAll(skillDir, 0755)
	writeFile(t, filepath.Join(skillDir, "skill.md"), "# skill:real（真实）\n\n## 触发场景\n\n真实触发场景。\n")

	result := LoadSkillsContext(tmpDir)

	check(t, result, "**real**")
	checkAbsent(t, result, "不应出现")
}

func TestLoadSkillsContext_MissingTriggerSection(t *testing.T) {
	tmpDir := t.TempDir()

	// skill.md without ## 触发场景 - should be skipped
	noTriggerDir := filepath.Join(tmpDir, "no_trigger_skill")
	os.MkdirAll(noTriggerDir, 0755)
	writeFile(t, filepath.Join(noTriggerDir, "skill.md"), "# skill:no-trigger（无触发）\n\n## 预期效果\n\n无触发场景。\n")

	// valid skill
	validDir := filepath.Join(tmpDir, "valid_skill")
	os.MkdirAll(validDir, 0755)
	writeFile(t, filepath.Join(validDir, "skill.md"), "# skill:valid（有触发）\n\n## 触发场景\n\n有触发场景时使用。\n")

	result := LoadSkillsContext(tmpDir)

	check(t, result, "**valid**")
	checkAbsent(t, result, "no-trigger")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func check(t *testing.T, result, want string) {
	t.Helper()
	if !strings.Contains(result, want) {
		t.Errorf("expected %q in result, got:\n%s", want, result)
	}
}

func checkAbsent(t *testing.T, result, absent string) {
	t.Helper()
	if strings.Contains(result, absent) {
		t.Errorf("expected %q to be absent, got:\n%s", absent, result)
	}
}
