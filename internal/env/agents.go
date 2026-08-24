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
	thinking       string // pi-subagents thinking 等级（可选；空则不写入 frontmatter，继承默认）
	timeoutMs      int64  // 单运行超时（可选；0 则不写入 frontmatter，用 pi 默认 30min）。glm-5.3 单轮 TTFB 慢 + 叶子扇出，research 实际最坏 ≈30-35min，必须显式放宽
}

// fullTools 是 rick subagent 的统一全量工具集（v4.1.1「默认不限制」策略）：
// pi 内置 7 件（read/grep/find/ls/bash/write/edit）+ web 扩展 2 件（web_search/
// fetch_content）+ subagent（fanout 授权——必须显式列出，省略 tools 会导致
// pi-subagents 不加载 fanout-child 扩展，think/research 将失去派发叶子能力）。
// 不列 intercom/contact_supervisor：launcher（intercom bridge / native supervisor
// channel）会按需自动注入；显式列 intercom 反而会在子会话未加载 pi-intercom
// 扩展时触发 0.51.0 的 strict 工具校验硬失败（"requested unavailable child
// tools: intercom"，job_35 验收期实测踩坑）。
const fullTools = "read, grep, find, ls, bash, write, edit, web_search, fetch_content, subagent"

// rickAgents 是 rick 经 env 职责 3 注册为 pi 自定义 agent 的清单
// （think/research/exporter）。system prompt = 对应源码 skill 的 wiki 正文。
// v4.1.1 起三 agent 工具全量开放（fullTools），行为约束交给各自 system prompt。
var rickAgents = []rickAgentSpec{
	{
		name:           "think",
		description:    "推理过程分析 + 4 维风险评估 + 启发性 3 问方法论（SENSE 的 S2 假设枚举与 EC 批判）",
		tools:          fullTools,
		defaultContext: "fresh",
		// 递归外包 + 自落盘：think 自己落盘简报，parent 读文件校验；重载思考可
		// 拆解为叶子外包（pi maxSubagentDepth=2 封底）。inline 交付降级为 parent
		// 侧安全网（连续失败时 resume 让其直接回复全文，parent 代写）。
		thinking:  "high",
		timeoutMs: 3600000, // 60min：glm-5.3 慢 TTFB + 思考外包等待，默认 30min 会掐死交付前一刻
	},
	{
		name:           "research",
		description:    "尽调树 + 信源加权方法论（澄清事实模糊性到极限；预估内容过多时拆叶子 worker 递归尽调）",
		tools:          fullTools,
		defaultContext: "fresh",
		// 递归外包 + 自落盘：research 自落盘调研报告（write 首块 + bash cat >> 分批
		// 追加），预估内容过多时拆叶子各写各的叶子文件；bash 支撑「运行时行为」信源验证。
		thinking:  "high",
		timeoutMs: 3600000, // 60min：实测 E 阶段 research（首轮 8.5min + 叶子扇出 14.4min）≈27-35min，默认 30min 超时恰在落盘前掐死
	},
	{
		name:           "exporter",
		description:    "教学综合 + RFC 固化（教师身份：第一性原理教学简报 → 启发式追问；最终大纲+内容两阶段 RFC）",
		tools:          fullTools,
		defaultContext: "fresh",
		// thinking high：exporter 起草中文 RFC 大文档，与 think 同一死亡模式。
		thinking:  "high",
		timeoutMs: 3600000,
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
	thinkingLine := ""
	if spec.thinking != "" {
		thinkingLine = fmt.Sprintf("thinking: %s\n", spec.thinking)
	}
	timeoutLine := ""
	if spec.timeoutMs > 0 {
		timeoutLine = fmt.Sprintf("timeoutMs: %d\n", spec.timeoutMs)
	}
	return fmt.Sprintf(`---
name: %s
description: %s
tools: %s
defaultContext: %s
%s%ssystemPromptMode: replace
rick-managed: true
---

%s`, spec.name, spec.description, spec.tools, spec.defaultContext, thinkingLine, timeoutLine, body)
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
