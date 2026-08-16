package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// DeployRickCustomizations 落盘 rick 自有定制到 pi 托管 agent 目录（职责 3）：
//
//   - rick-gates hook 扩展：<repo>/.rick/skills/rick-gates/ → AgentDir()/extensions/rick-gates/
//   - rick skills：<repo>/.rick/skills/*_skill/            → AgentDir()/skills/<name>/
//
// think/research/exporter agent frontmatter 不在此落盘（task9 经职责 3 写入
// AgentDir()/agents/），避免本 task 的占位文件被 task9 幂等复制跳过。
//
// 幂等：重复运行会重写目标文件（内容一致），不产生重复项。
func DeployRickCustomizations() error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("locate rick dir: %w", err)
	}
	return deployRickCustomizations(filepath.Join(rickDir, "skills"))
}

// deployRickCustomizations 从给定 skills 源目录复制 rick 定制，便于单测注入临时源目录。
func deployRickCustomizations(skillsDir string) error {
	// rick-gates hook 扩展 → extensions/rick-gates/。
	if err := copyDir(filepath.Join(skillsDir, "rick-gates"),
		filepath.Join(runtime.AgentDir(), "extensions", "rick-gates")); err != nil {
		return fmt.Errorf("deploy rick-gates: %w", err)
	}

	// rick skills（*_skill 目录，含 skill.md）→ skills/。
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 无 skills 目录 —— 无定制可部署。
		}
		return fmt.Errorf("read skills dir %s: %w", skillsDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "rick-gates" {
			continue
		}
		// 仅复制真正的 skill 目录（含 skill.md），跳过 README 等杂项。
		if !runtime.FileExists(filepath.Join(skillsDir, e.Name(), "skill.md")) {
			continue
		}
		if err := copyDir(filepath.Join(skillsDir, e.Name()),
			filepath.Join(runtime.AgentDir(), "skills", e.Name())); err != nil {
			return fmt.Errorf("deploy skill %s: %w", e.Name(), err)
		}
	}
	return nil
}

// copyDir 递归复制 src 目录到 dst，保留文件/目录权限。源不存在时返回 nil
// （幂等 + 容错：部分定制可缺失而不阻断整体部署）。
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	return filepath.Walk(src, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		mode := fi.Mode().Perm()
		if mode == 0 {
			mode = 0644
		}
		return os.WriteFile(target, data, mode)
	})
}
