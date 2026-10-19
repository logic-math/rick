// session_test.go — job_48/task4：BuildSuperPrompt 工作区上下文块契约测试。
//
// 覆盖：工作区根推导（rickDir 上层）、super_env 绝对路径、env 生成约定、
// 观察手段（CLI + 直读 events.jsonl/archive.json/codes + pi session 可读）、
// 干预手段（event 队列 + 单写者）。用 fake rick 工作区（写入最小
// super-loop.md 与 gen_env_skill/skill.md），不依赖仓库真实 .rick。
package super

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRickWorkspace 建一个最小 rickDir fixture（loops/super-loop.md +
// skills/gen_env_skill/skill.md），返回 rickDir 与推导的 super ws 路径
// （<root>/.rick/super_env/<name>——job_48 存储模型）。
func fakeRickWorkspace(t *testing.T) (rickDir, ws string) {
	t.Helper()
	root := t.TempDir()
	rickDir = filepath.Join(root, ".rick")
	loopDir := filepath.Join(rickDir, "loops")
	skillDir := filepath.Join(rickDir, "skills", "gen_env_skill")
	for _, d := range []string{loopDir, skillDir, filepath.Join(rickDir, "super_env", "demo")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(loopDir, "super-loop.md"),
		[]byte("---\nname: super-loop\ntrigger: super 模式\nscope: 全局\n---\n\n# Loop: super-loop（中控 SOP）\n启动前门先证红确认。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "skill.md"),
		[]byte("# skill:gen-env\n\n## 核心内容\n\n门禁系统设计引导：任务定义→评分方向→门禁设计→参考实现→特征轴→初始解。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws = filepath.Join(rickDir, "super_env", "demo")
	return rickDir, ws
}

func TestBuildSuperPromptWorkspaceContext(t *testing.T) {
	rickDir, ws := fakeRickWorkspace(t)
	_, promptFile, err := BuildSuperPrompt(rickDir, ws)
	if err != nil {
		t.Fatalf("BuildSuperPrompt: %v", err)
	}
	data, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatal(err)
	}
	p := string(data)

	// 工作区上下文块存在
	if !strings.Contains(p, "## 工作区上下文") {
		t.Error("prompt 缺「## 工作区上下文」块")
	}
	// 工作区根 = rickDir 上层（绝对路径）
	workspaceRoot := filepath.Dir(rickDir)
	if !strings.Contains(p, workspaceRoot) {
		t.Errorf("prompt 缺工作区根 %q", workspaceRoot)
	}
	// super_env 绝对路径约定
	if !strings.Contains(p, ws) {
		t.Errorf("prompt 缺 super_env 工作区路径 %q", ws)
	}
	if !strings.Contains(p, "super_env") {
		t.Error("prompt 缺 super_env 路径约定表述")
	}
	// env 生成约定（gen_env 引导产出落 super_env/<name>/env/）
	if !strings.Contains(p, filepath.Join(ws, "env")) {
		t.Errorf("prompt 缺 env 生成路径 %q", filepath.Join(ws, "env"))
	}
	// 观察手段：CLI + 直读文件（真相源清单）
	for _, kw := range []string{"观察手段", "rick super status --json",
		"events.jsonl", "archive.json", "codes", "heartbeat.json", "pi agent"} {
		if !strings.Contains(p, kw) {
			t.Errorf("prompt 观察手段缺关键词 %q", kw)
		}
	}
	if !strings.Contains(p, "任意信息可读") {
		t.Error("prompt 缺「任意信息可读」授权表述")
	}
	// 干预手段：event 队列 + 单写者
	for _, kw := range []string{"干预手段", "rick super event --type", "单写者"} {
		if !strings.Contains(p, kw) {
			t.Errorf("prompt 干预手段缺关键词 %q", kw)
		}
	}
	// 严格加载证据：gen_env skill 全文内嵌（fixture 内容中的独特关键词）
	if !strings.Contains(p, "门禁系统设计引导") {
		t.Error("prompt 未内嵌 gen_env skill 全文（缺「门禁系统设计引导」）")
	}
	// 无未替换变量
	if strings.Contains(p, "{{") {
		t.Error("prompt 含未替换 {{...}} 变量")
	}
}
