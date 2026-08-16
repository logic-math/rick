package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/runtime"
)

// rickAgentSpec 描述一个 rick 自定义 agent：frontmatter 字段 + system prompt
// 来源（embedded templates/skills/{name}.md，即对应源码 skill 的 wiki 内容）。
type rickAgentSpec struct {
	name           string
	description    string
	tools          string // 逗号分隔的 pi 工具名（与 pi-subagents tools 字段对齐）
	defaultContext string
}

// rickAgents 是 rick 经 env 职责 3 注册为 pi 自定义 agent 的清单
// （think/research/exporter）。system prompt = 对应源码 skill 的 wiki 正文。
var rickAgents = []rickAgentSpec{
	{
		name:           "think",
		description:    "推理过程分析 + 4 维风险评估 + 启发性 3 问方法论（SENSE 的 S2 假设枚举与 EC 批判）",
		tools:          "read, grep, find, ls",
		defaultContext: "fresh",
	},
	{
		name:           "research",
		description:    "尽调树 + 信源加权方法论（澄清事实模糊性到极限）",
		tools:          "read, grep, find, ls, bash, web_search, fetch_content",
		defaultContext: "fresh",
	},
	{
		name:           "exporter",
		description:    "RFC 输出方法论（大纲 + 内容两阶段固化）",
		tools:          "read, write, bash",
		defaultContext: "fresh",
	},
}

// deployRickAgents 将 rick 自定义 agent 幂等写入 AgentDir()/agents/{name}.md。
// 覆盖语义：仅当目标文件不存在，或其 frontmatter 含 rick-managed: true 标记时
// 才写入；无标记的同名文件（用户自定义）跳过，避免覆盖用户内容。
func deployRickAgents() error {
	agentsDir := filepath.Join(runtime.AgentDir(), "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}
	for _, spec := range rickAgents {
		if err := deployRickAgent(agentsDir, spec); err != nil {
			return err
		}
	}
	return nil
}

// deployRickAgent 写入单个 agent 文件（幂等 + rick 标记覆盖语义）。
func deployRickAgent(agentsDir string, spec rickAgentSpec) error {
	target := filepath.Join(agentsDir, spec.name+".md")
	if data, err := os.ReadFile(target); err == nil {
		if !hasRickManagedMarker(string(data)) {
			return nil // 无 rick 标记的用户文件，不覆盖。
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", target, err)
	}

	body := prompt.LoadCoreSkills([]string{spec.name})
	if body == "" {
		return fmt.Errorf("embedded skill %q not found for agent %q", spec.name, spec.name)
	}
	content := renderRickAgentFile(spec, body)
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		return fmt.Errorf("write agent %s: %w", spec.name, err)
	}
	return nil
}

// renderRickAgentFile 渲染 agent 文件 = YAML frontmatter + system prompt 正文。
// systemPromptMode: replace 使正文整体替换 pi 默认骨架（system prompt = wiki 内容）。
func renderRickAgentFile(spec rickAgentSpec, body string) string {
	return fmt.Sprintf(`---
name: %s
description: %s
tools: %s
defaultContext: %s
systemPromptMode: replace
rick-managed: true
---

%s`, spec.name, spec.description, spec.tools, spec.defaultContext, body)
}

// hasRickManagedMarker 判断 agent 文件 frontmatter 是否含 rick-managed: true。
// frontmatter 由首行 --- 与随后的第一个 --- 行闭合；闭合前未找到标记即视为无。
// 解析对齐 pi 前端 frontmatter 语义：key/value 按 ":" 分割并 trim。
func hasRickManagedMarker(content string) bool {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for _, ln := range lines[1:] {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "---" {
			return false
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		kv := strings.SplitN(trimmed, ":", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "rick-managed" && strings.TrimSpace(kv[1]) == "true" {
			return true
		}
	}
	return false
}
