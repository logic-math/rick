/**
 * subagent 显示名（Bug #8 2A 终版 human 反馈 2026-11-15：agent 名短一些——
 * researcher/worker 这类长英文名在侧边栏下拉/右侧栏/卡片里占位奇怪）。
 *
 * 常见 agent 名映射短中文；未知名截断（≤6 字符，title 全名）。
 */
const AGENT_SHORT_NAMES: Record<string, string> = {
  research: "调研",
  researcher: "调研",
  worker: "执行",
  think: "思考",
  reviewer: "评审",
  lesson: "教学",
  "lesson-writer": "教学",
  exporter: "导出",
};

export function shortAgentName(agent: string): string {
  const mapped = AGENT_SHORT_NAMES[agent];
  if (mapped) return mapped;
  return agent.length > 6 ? `${agent.slice(0, 6)}…` : agent;
}
