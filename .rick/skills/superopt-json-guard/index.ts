// superopt-json-guard — superopt agent 的 JSON 输出确定性 hook。
//
// 问题（env_11 实测）：agent 完成实验但最终回复缺少 JSON 契约块 →
// client 层标 valid=False → 触发 debug 重试 → 48% token 浪费。
//
// 方案：pi 进程内 hook——agent_settled 时检查最后一条 assistant 消息，
// 无 JSON 块 → 自动 sendUserMessage（「请只输出 JSON」）→ agent 在
// **同一进程同一会话**继续（不杀不重启），一次修复。
//
// 激活条件：系统提示词含 "superopt" 标记（rick super 的 agent 专用，
// 不影响其它 pi 用户）。JSON 检查：```json ... ``` 代码块或裸 JSON 对象。
//
// 部署：复制到 ~/.rick/pi/agent/extensions/superopt-json-guard/index.ts
// pi 自动发现 extensions/*/index.ts。

import * as fs from "node:fs";
import * as path from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/** 最多修复几次（避免无限循环） */
const MAX_JSON_REPAIR = 2;

/** 检查文本是否包含合法的 JSON 代码块或裸 JSON */
function hasJsonBlock(text: string): boolean {
	if (!text || !text.trim()) return false;
	// ```json ... ```
	const jsonBlock = /```json\s*\n[\s\S]*?\n\s*```/.exec(text);
	if (jsonBlock) {
		try {
			JSON.parse(jsonBlock[0].replace(/```json\s*\n?/, "").replace(/\n?\s*```/, ""));
			return true;
		} catch { /* parse fail = not valid */ }
	}
	// 裸 JSON 对象（agent 可能直接输出 {...} 不带 code fence）
	const trimmed = text.trim();
	if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
		try {
			JSON.parse(trimmed);
			return true;
		} catch { /* not valid */ }
	}
	return false;
}

/** 检查系统提示词是否含 superopt 标记 */
function isSuperoptAgent(ctx: { systemPrompt?: string }): boolean {
	return !!ctx.systemPrompt && ctx.systemPrompt.includes("superopt");
}

export function activate(api: ExtensionAPI): void {
	let repairCount = 0;

	api.on("agent_settled", (event) => {
		// 检查系统提示词（只有 superopt 的 agent 才激活）
		// 注意：agent_settled 不带 systemPrompt，改用 session 级检查
		// 从 messages 中取最后一条 assistant
		const messages = api.session?.messages ?? [];
		const lastAssistant = [...messages].reverse().find(m => m.role === "assistant");
		if (!lastAssistant) return;

		const text = typeof lastAssistant.content === "string"
			? lastAssistant.content
			: Array.isArray(lastAssistant.content)
				? lastAssistant.content.filter(c => c.type === "text").map(c => (c as any).text ?? "").join("")
				: "";

		// 有 JSON → 重置计数
		if (hasJsonBlock(text)) {
			repairCount = 0;
			return;
		}

		// 无 JSON + 未超修复上限 → 发送修复指令
		if (repairCount < MAX_JSON_REPAIR) {
			repairCount++;
			api.sendUserMessage(
				`你的最终回复缺少必需的 JSON 输出块。请按系统提示词的输出契约，只输出那个 JSON 代码块（\`\`\`json ... \`\`\`），不要任何其它内容。这是第 ${repairCount} 次提醒（上限 ${MAX_JSON_REPAIR} 次）。`,
			);
		}
	});
}
