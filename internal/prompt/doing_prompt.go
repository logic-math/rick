package prompt

import (
	"strings"
)

// loadDoingLoopContent reads the doing_loop skill, strips YAML frontmatter, and
// substitutes {{domain_dir}} and {{debug_skill_path}} with actual paths.
func loadDoingLoopContent(domainDir, debugSkillPath string) string {
	raw := LoadCoreSkills([]string{"doing_loop"})
	content := stripYAMLFrontmatter(raw)
	content = strings.ReplaceAll(content, "{{domain_dir}}", domainDir)
	return strings.ReplaceAll(content, "{{debug_skill_path}}", debugSkillPath)
}
